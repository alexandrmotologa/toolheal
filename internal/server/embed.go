package server

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed web_dist/*
var embeddedConsoleFS embed.FS

// ConsoleHandler serves the embedded React 19 single page application.
type ConsoleHandler struct {
	fileServer http.Handler
}

// NewConsoleHandler initializes SPA asset routing.
func NewConsoleHandler() *ConsoleHandler {
	distSub, err := fs.Sub(embeddedConsoleFS, "web_dist")
	if err != nil {
		panic(err)
	}
	return &ConsoleHandler{
		fileServer: http.FileServer(http.FS(distSub)),
	}
}

// ServeHTTP handles requests, serving static assets or falling back to index.html for SPA routing.
func (h *ConsoleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.URL.Path, ".") && r.URL.Path != "/" {
		r.URL.Path = "/"
	}
	h.fileServer.ServeHTTP(w, r)
}
