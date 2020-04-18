package server

import (
	"net/http"
	"time"
)

func redirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://"+r.Host+r.URL.String(), http.StatusMovedPermanently)
}

// NewRedirectServer creates a new http.Server that redirects all requests
// to the respective https equivalent.
func NewRedirectServer() *http.Server {
	return &http.Server{
		Handler:      http.HandlerFunc(redirect),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
}
