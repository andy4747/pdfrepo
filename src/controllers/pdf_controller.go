package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"text/template"
	"time"

	unidoc "github.com/unidoc/unipdf/v3/model"

	"pdfrepo-backend/models"
)

func PDFReaderHandler(w http.ResponseWriter, r *http.Request) {
	pdfURL := r.URL.Query().Get("url")
	if pdfURL == "" {
		http.Error(w, "Missing PDF URL", http.StatusBadRequest)
		return
	}

	// Validate URL (as before) ...

	tmpl, err := template.ParseFiles("views/pdf_reader.html") // Parse ONLY pdf_reader.html
	if err != nil {
		http.Error(w, "Failed to load PDF reader template", http.StatusInternalServerError)
		log.Printf("Failed to load PDF reader template: %v", err)
		return
	}

	data := map[string]interface{}{
		"Title":  "PDF Reader",
		"PDFURL": pdfURL,
	}

	if err := tmpl.Execute(w, data); err != nil { // Execute ONLY pdf_reader.html template
		http.Error(w, "Failed to render PDF reader template", http.StatusInternalServerError)
		log.Printf("Failed to render PDF reader template: %v", err)
	}
}
func FetchPDFMetadata(pdfURL string) (models.PDFMetadata, error) {
	parsedURL, err := url.ParseRequestURI(pdfURL)
	if err != nil {
		return models.PDFMetadata{Error: "Invalid URL format"}, fmt.Errorf("invalid URL format: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return models.PDFMetadata{Error: "Invalid URL scheme"}, fmt.Errorf("invalid URL scheme: %s", parsedURL.Scheme)
	}

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(pdfURL)
	if err != nil {
		return models.PDFMetadata{Error: "Failed to fetch PDF"}, fmt.Errorf("failed to fetch PDF: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.PDFMetadata{Error: "Failed to fetch PDF, server status: " + resp.Status}, fmt.Errorf("failed to fetch PDF, server status: %s", resp.Status)
	}

	pdfData, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.PDFMetadata{Error: "Failed to read PDF body"}, fmt.Errorf("failed to read PDF body: %w", err)
	}

	pdfReader, err := unidoc.NewPdfReader(bytes.NewReader(pdfData))
	if err != nil {
		return models.PDFMetadata{Error: "Failed to parse PDF"}, fmt.Errorf("failed to parse PDF: %w", err)
	}

	isEncrypted, err := pdfReader.IsEncrypted()
	if err != nil {
		log.Printf("Error checking encryption: %v, URL: %s", err, pdfURL)
		isEncrypted = false
	}

	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return models.PDFMetadata{Error: "Failed to get page count"}, fmt.Errorf("failed to get page count: %w", err)
	}

	info, err := pdfReader.GetPdfInfo()
	var title string
	var author string

	if err == nil && info != nil {
		if t := info.Title; t != nil {
			title = t.String()
		}
		if a := info.Author; a != nil {
			author = a.String()
		}
	} else {
		log.Println("Failed to get PDF Info from URL:", pdfURL, ", error:", err)
		title = ""
		author = ""
	}

	return models.PDFMetadata{
		PageCount: numPages,
		Encrypted: isEncrypted,
		Title:     title,
		Author:    author,
	}, nil
}

func PDFMetaHandler(w http.ResponseWriter, r *http.Request) {
	urlStr := r.URL.Query().Get("url")
	if urlStr == "" {
		http.Error(w, "Missing PDF URL", http.StatusBadRequest)
		return
	}

	metadata, err := FetchPDFMetadata(urlStr)
	if err != nil {
		log.Printf("Error fetching PDF metadata: %v", err)
		// Error details are already in metadata.Error
	}

	w.Header().Set("Content-Type", "application/json") // Still return JSON for meta endpoint
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
		log.Printf("Failed to encode JSON response: %v", err)
	}
}
