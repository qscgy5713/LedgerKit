// Command ledgerkit-server exposes a LedgerKit journal over an HTTP JSON
// API (for the web UI and the Capacitor Android app), reusing the same
// internal/journal and internal/ledger packages as the ledgerkit CLI and
// reading/writing the same journal file.
package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"
)

//go:embed web
var webFS embed.FS

func main() {
	journalPath := os.Getenv("LEDGERKIT_JOURNAL")
	if journalPath == "" {
		journalPath = "ledger.journal"
	}

	token := os.Getenv("LEDGERKIT_API_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "ledgerkit-server: LEDGERKIT_API_TOKEN must be set (e.g. `openssl rand -hex 32`); refusing to start unauthenticated")
		os.Exit(1)
	}

	addr := os.Getenv("LEDGERKIT_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	a := &api{store: newStore(journalPath)}

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("GET /api/accounts", a.handleAccounts)
	apiMux.HandleFunc("GET /api/balance", a.handleBalance)
	apiMux.HandleFunc("GET /api/register", a.handleRegister)
	apiMux.HandleFunc("GET /api/check", a.handleCheck)
	apiMux.HandleFunc("POST /api/transactions", a.handleCreateTransaction)

	staticFS, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatal(err)
	}

	root := http.NewServeMux()
	root.Handle("/api/", withCORS(requireToken(token, apiMux)))
	root.Handle("/", http.FileServer(http.FS(staticFS)))

	srv := &http.Server{
		Addr:              addr,
		Handler:           root,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("ledgerkit-server listening on %s (journal: %s)", addr, journalPath)
	log.Fatal(srv.ListenAndServe())
}
