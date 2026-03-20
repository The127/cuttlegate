//go:build frontend

package main

import (
	"embed"
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
	mux.Handle("/", http.FileServer(http.FS(dist)))
}
