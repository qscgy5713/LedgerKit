package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"ledgerkit/internal/journal"
	"ledgerkit/internal/ledger"
)

type api struct {
	store *store
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// loadOrRespond loads the journal, translating a missing file into a
// friendly 404 instead of a raw filesystem error.
func (a *api) loadOrRespond(w http.ResponseWriter) (*journal.Journal, bool) {
	j, err := a.store.load()
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "journal file does not exist yet; create one with `ledgerkit init`")
			return nil, false
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return nil, false
	}
	return j, true
}

func filterFromQuery(r *http.Request) (ledger.Filter, error) {
	f := ledger.Filter{AccountPattern: r.URL.Query().Get("account")}
	t, err := journal.ParseOptionalDate(r.URL.Query().Get("from"))
	if err != nil {
		return f, fmt.Errorf("from: %w", err)
	}
	f.From = t
	t, err = journal.ParseOptionalDate(r.URL.Query().Get("to"))
	if err != nil {
		return f, fmt.Errorf("to: %w", err)
	}
	f.To = t
	return f, nil
}

func (a *api) handleAccounts(w http.ResponseWriter, r *http.Request) {
	j, ok := a.loadOrRespond(w)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, j.Accounts())
}

func (a *api) handleBalance(w http.ResponseWriter, r *http.Request) {
	j, ok := a.loadOrRespond(w)
	if !ok {
		return
	}
	f, err := filterFromQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	root := ledger.BuildTree(j.Transactions, f)
	rows := ledger.FlattenBalance(root)

	out := make([]balanceRowDTO, len(rows))
	for i, row := range rows {
		out[i] = balanceRowDTO{
			Name:        row.Name,
			FullAccount: row.FullAccount,
			Depth:       row.Depth,
			Amounts:     amountsDTOFrom(row.Amounts),
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"rows":  out,
		"total": amountsDTOFrom(root.Total),
	})
}

func (a *api) handleRegister(w http.ResponseWriter, r *http.Request) {
	j, ok := a.loadOrRespond(w)
	if !ok {
		return
	}
	f, err := filterFromQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	entries := ledger.Register(j.Transactions, f)
	out := make([]registerEntryDTO, len(entries))
	for i, e := range entries {
		out[i] = registerEntryDTO{
			Date:        e.Date.Format("2006-01-02"),
			Description: e.Description,
			Account:     e.Account,
			Amount:      amountDTOFrom(e.Amount),
			Running:     amountsDTOFrom(e.Running),
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *api) handleCheck(w http.ResponseWriter, r *http.Request) {
	j, ok := a.loadOrRespond(w)
	if !ok {
		return
	}
	errs := j.Validate()
	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Error()
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":               len(errs) == 0,
		"transactionCount": len(j.Transactions),
		"errors":           msgs,
	})
}

func (a *api) handleCreateTransaction(w http.ResponseWriter, r *http.Request) {
	if !a.store.exists() {
		writeError(w, http.StatusNotFound, "journal file does not exist yet; create one with `ledgerkit init`")
		return
	}

	var req transactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid date, want YYYY-MM-DD")
		return
	}
	if req.Description == "" {
		writeError(w, http.StatusBadRequest, "description is required")
		return
	}
	if len(req.Postings) < 2 {
		writeError(w, http.StatusBadRequest, "need at least 2 postings")
		return
	}

	txn := journal.Transaction{Date: date, Status: '*', Description: req.Description}
	for _, p := range req.Postings {
		if p.Account == "" {
			writeError(w, http.StatusBadRequest, "posting missing account")
			return
		}
		posting := journal.Posting{Account: p.Account}
		if p.Amount != "" {
			amt, err := journal.ParseAmount(p.Amount)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			posting.Amount = &amt
		}
		txn.Postings = append(txn.Postings, posting)
	}
	if err := journal.ResolveElided(&txn); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := a.store.append(txn); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})
}
