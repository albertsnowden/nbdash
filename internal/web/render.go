// Package web owns the dashboard's static assets: the React SPA's built
// output and everything else served under /static/. Everything is
// embedded, so the dashboard ships as a single binary with no runtime file
// dependencies — the SPA's own build (ui/, via Vite) collapses into this
// package the same way the Next.js dashboard's out/ directory plus nginx
// used to collapse into the old htmx server before it.
package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static
var files embed.FS

// Static serves the embedded assets under /static/, including the SPA's own
// built JS/CSS (ui/vite.config.ts builds straight into static/app/, so
// there is nothing SPA-specific to wire up here beyond this already
// covering it).
func Static() (http.Handler, error) {
	sub, err := fs.Sub(files, "static")
	if err != nil {
		return nil, err
	}
	return http.StripPrefix("/static/", http.FileServer(noDirFS{http.FS(sub)})), nil
}

// AppShell serves the SPA's index.html for any path not otherwise claimed
// (see internal/api.Server.Routes' catch-all registration), letting
// react-router own client-side routing from "/" down.
//
// index.html is read once at startup rather than on every request: it is
// embedded, immutable for the life of the process, and small.
func AppShell() (http.HandlerFunc, error) {
	index, err := fs.ReadFile(files, "static/app/index.html")
	if err != nil {
		return nil, err
	}
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(index)
	}, nil
}

// noDirFS turns a request for a directory into "not found", so http.FileServer
// cannot render its built-in index listing. Nothing here is secret — it is the
// same CSS and JS every visitor already loads — but enumerating an origin's
// assets is a free first step for anyone probing it, and there is no reason to
// answer. Requests for actual files are untouched.
type noDirFS struct{ fs http.FileSystem }

func (n noDirFS) Open(name string) (http.File, error) {
	f, err := n.fs.Open(name)
	if err != nil {
		return nil, err
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if info.IsDir() {
		f.Close()
		return nil, fs.ErrNotExist
	}

	return f, nil
}
