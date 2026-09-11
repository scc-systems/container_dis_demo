package web

import (
	"embed"
	"io/fs"
	"net/http"
	"text/template"
)

//go:embed static
var staticFS embed.FS

var wallTmpl = func() *template.Template {
	data, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		panic(err)
	}
	return template.Must(template.New("index.html").Parse(string(data)))
}()

func StaticHandler() http.Handler {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(sub))
}

// WallHandler serves the /wall entry point with the base path injected.
func WallHandler(basePath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_ = wallTmpl.Execute(w, map[string]string{"BasePath": basePath})
	})
}
