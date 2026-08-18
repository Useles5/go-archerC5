package main

import (
	"crypto/subtle"
	"fmt"
	"net/http"
)

func (app *application) requireAPIKey(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providedKey := r.Header.Get("X-API-Key")
		if providedKey == "" {
			msg := fmt.Sprintf("%s: Missing X-API-Key in header", http.StatusText(http.StatusUnauthorized))
			app.writeJSON(w, http.StatusUnauthorized, nil, msg)
			return
		}

		// compare the keys
		// use subtle.ConstantTimeCompare() to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(providedKey), []byte(app.config.apiKey)) != 1 {
			msg := fmt.Sprintf("%s: Invalid API Key", http.StatusText(http.StatusUnauthorized))
			app.writeJSON(w, http.StatusUnauthorized, nil, msg)
			return
		}

		next(w, r)
	}
}
