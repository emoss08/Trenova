package viewer

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static
var files embed.FS

func Handler() http.Handler {
	root, err := fs.Sub(files, "static")
	if err != nil {
		panic(err)
	}
	server := http.FileServerFS(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		server.ServeHTTP(w, r)
	})
}
