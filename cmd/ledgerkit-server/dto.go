package main

import "ledgerkit/internal/journal"

// DTOs live here, separate from internal/journal's core types, so the
// domain model stays free of JSON tags and presentation concerns (mirrors
// how internal/render is kept separate and only used by cmd/ledgerkit).

type amountDTO struct {
	Commodity string `json:"commodity"`
	Value     string `json:"value"`   // decimal string, e.g. "120.50" or "-3.00"
	Sign      int    `json:"sign"`    // -1, 0, or 1
	Display   string `json:"display"` // pre-formatted for direct display, e.g. "$120" or "120.50 TWD"
}

func amountDTOFrom(a journal.Amount) amountDTO {
	return amountDTO{
		Commodity: a.Commodity,
		Value:     a.Value.String(),
		Sign:      a.Value.Sign(),
		Display:   a.String(),
	}
}

func amountsDTOFrom(a journal.Amounts) []amountDTO {
	out := make([]amountDTO, 0, len(a.Values))
	for _, c := range a.Commodities() {
		out = append(out, amountDTO{
			Commodity: c,
			Value:     a.Values[c].String(),
			Sign:      a.Values[c].Sign(),
			Display:   a.Format(c),
		})
	}
	return out
}

type balanceRowDTO struct {
	Name        string      `json:"name"`
	FullAccount string      `json:"fullAccount"`
	Depth       int         `json:"depth"`
	Amounts     []amountDTO `json:"amounts"`
}

type registerEntryDTO struct {
	Date        string      `json:"date"`
	Description string      `json:"description"`
	Account     string      `json:"account"`
	Amount      amountDTO   `json:"amount"`
	Running     []amountDTO `json:"running"`
}

type postingRequest struct {
	Account string `json:"account"`
	Amount  string `json:"amount"` // "" = elided, inferred to balance the transaction
}

type transactionRequest struct {
	Date        string           `json:"date"` // "YYYY-MM-DD"
	Description string           `json:"description"`
	Postings    []postingRequest `json:"postings"`
}
