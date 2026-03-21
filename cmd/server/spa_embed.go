//go:build frontend

package main

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
)

//go:embed web/dist
var spaFS embed.FS

func serveSPA(mux *http.ServeMux) {
	dist, err := fs.Sub(spaFS, "web/dist")
	if err != nil {
		panic("spa: failed to sub web/dist: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(dist))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try serving the file directly. If it doesn't exist, serve index.html
		// so the SPA's client-side router can handle the route.
		f, err := dist.Open(r.URL.Path[1:]) // strip leading /
		if err != nil {
			serveIndex(w, dist)
			return
		}
		f.Close()
		fileServer.ServeHTTP(w, r)
	}))
}

func serveIndex(w http.ResponseWriter, dist fs.FS) {
	f, err := dist.Open("index.html")
	if err != nil {
		http.Error(w, "index.html not found", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.Copy(w, f) //nolint:errcheck
}
