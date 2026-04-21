package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
)

//go:embed static/*
var embeddedStaticFiles embed.FS

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}

	staticFiles, err := fs.Sub(embeddedStaticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}

	fileServer := http.FileServer(http.FS(staticFiles))
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			http.ServeFileFS(w, r, staticFiles, "index.html")
			return
		}

		fileServer.ServeHTTP(w, r)
	})

	addr := ":" + port
	log.Printf("serving static site on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
