//go:build !frontend

package main

import "net/http"

func serveSPA(mux *http.ServeMux) {
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
}
