package main

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"path"
)

// Dir returns a sub-filesystem rooted at dir inside the given embed.FS,
// suitable for passing to http.FileServer or ServeFromEmbed.
//
// Example:
//
//	//go:embed static
//	var staticFS embed.FS
//
//	sub := server.Dir(staticFS, "static")
func Dir(fsys embed.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		// dir not found in the embed — return an empty FS so callers
		// fail gracefully at request time rather than at startup.
		return emptyFS{}
	}
	return sub
}

// ServeFromEmbed writes a file from fsys to w, executing it as an
// html/template with data if the extension is .html, or serving it raw
// otherwise.  file is a slash-separated path relative to the FS root.
//
// Example:
//
//	server.ServeFromEmbed(w, r, staticFS, "index.html", pageData)
func ServeFromEmbed(w http.ResponseWriter, r *http.Request, fsys fs.FS, file string, data any) {
	ext := path.Ext(file)
	if ext == ".html" {
		tmpl, err := template.ParseFS(fsys, file)
		if err != nil {
			http.Error(w, "template not found: "+file, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, data); err != nil {
			// Header already written; log rather than re-write.
			return
		}
		return
	}

	// Non-template: serve the raw file through http.FileServer semantics.
	http.ServeFileFS(w, r, fsys, file)
}

// emptyFS is a no-op fs.FS returned when Dir cannot locate the sub-tree.
type emptyFS struct{}

func (emptyFS) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}
