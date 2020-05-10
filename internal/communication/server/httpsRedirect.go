package server

import (
	"fmt"
	"net/http"
	"time"
)

func generateRedirect(rootURL string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(rootURL + r.URL.String())
		http.Redirect(w, r, rootURL+r.URL.String(), http.StatusMovedPermanently)
	}
}

// NewRedirectServer creates a new http.Server that redirects all requests
// to the respective https equivalent.
func NewRedirectServer(rootURL string) *http.Server {
	return &http.Server{
		Handler:      http.HandlerFunc(generateRedirect(rootURL)),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
}
