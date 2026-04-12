package server

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
)

var webAssets embed.FS

func SetWebAssets(assets embed.FS) {
	webAssets = assets
}

func Run(port int, apiKey string) {
	if err := EnsureDirs(); err != nil {
		log.Fatalf("error creating directories: %v", err)
	}

	h := &handlers{apiKey: apiKey}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/projects", h.listProjects)
	mux.HandleFunc("GET /api/projects/{slug}", h.getProject)
	mux.HandleFunc("POST /api/projects", h.createProject)
	mux.HandleFunc("PUT /api/projects/{slug}", h.updateProject)
	mux.HandleFunc("DELETE /api/projects/{slug}", h.deleteProject)
	mux.HandleFunc("POST /api/projects/{slug}/generate", h.generateDocs)
	mux.HandleFunc("GET /api/docs/{slug}", h.checkDocs)
	mux.HandleFunc("GET /api/settings", h.getSettings)

	mux.Handle("/docs/", http.StripPrefix("/docs/", http.FileServer(http.Dir(DocsDir))))

	mux.Handle("/", spaHandler())

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Servidor iniciado en http://localhost:%d", port)
	log.Fatal(http.ListenAndServe(addr, corsMiddleware(mux)))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func spaHandler() http.Handler {
	distFS, err := fs.Sub(webAssets, "web/dist")
	if err != nil {
		log.Printf("warning: web assets not embedded, serving placeholder: %v", err)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<!DOCTYPE html><html><body><h1>apidocgen</h1><p>Frontend not built. Run <code>cd web && npm run build</code></p></body></html>`)
		})
	}

	fileServer := http.FileServer(http.FS(distFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "index.html"
		} else {
			path = strings.TrimPrefix(path, "/")
		}

		if _, err := fs.Stat(distFS, path); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html for any unknown route
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
