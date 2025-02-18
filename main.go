package main

import (
	"embed"
	"log"
	"net/http"

	"pdfrepo-backend/controllers" // Import your controllers
)

//go:embed views/*.html
var templateFS embed.FS

//go:embed public/*
var publicFS embed.FS

func main() {
	// static files (CSS)
	http.Handle("/public/", http.FileServer(http.FS(publicFS)))

	// Routes
	http.HandleFunc("/", controllers.HomeHandler)
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	http.HandleFunc("/api/browse", controllers.GitHubBrowserHandler)
	http.HandleFunc("/api/pdf/meta", controllers.PDFMetaHandler)
	http.HandleFunc("/api/pdf/reader", controllers.PDFReaderHandler)

	log.Println("Server starting on port 8080")
	http.ListenAndServe(":8080", nil)
}
