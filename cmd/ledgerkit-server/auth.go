package main

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// requireToken wraps h so every request must present the configured bearer
// token. Comparison uses subtle.ConstantTimeCompare to avoid leaking the
// token's value through response-timing differences.
func requireToken(token string, h http.Handler) http.Handler {
	want := []byte(token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || subtle.ConstantTimeCompare([]byte(got), want) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// withCORS allows any origin to call the API. This is safe here because
// auth is a bearer token the client sends explicitly (in a header), not a
// cookie — an open CORS policy can't be abused to ride a victim's existing
// session the way it could with cookie auth. This is what lets the
// Capacitor Android app (a different origin from the server) call the API.
func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}
