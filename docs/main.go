package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gen2brain/go-fitz"
	unidoc "github.com/unidoc/unipdf/v3/model"
)

//go:embed pdf.worker.min.js
var workerFS embed.FS

func main() {
	http.Handle("/pdf.worker.min.js",
		http.FileServer(http.FS(workerFS)))

	// Routes
	http.HandleFunc("/api/github", handleGitHubProxy)
	http.HandleFunc("/api/pdf/meta", handlePDFMeta)
	http.HandleFunc("/api/pdf/preview", handlePDFPreview)

	log.Println("Server starting on port 8080")
	http.ListenAndServe(":8080", nil)
}

func handlePDFMeta(w http.ResponseWriter, r *http.Request) {
	urlStr := r.URL.Query().Get("url")
	if urlStr == "" {
		http.Error(w, "Missing PDF URL", http.StatusBadRequest)
		return
	}

	// 1. Validate URL (same as before)
	parsedURL, err := url.ParseRequestURI(urlStr)
	if err != nil {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		log.Printf("Invalid URL format: %v, error: %v", urlStr, err)
		return
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		http.Error(w, "Invalid URL scheme, must be http or https", http.StatusBadRequest)
		log.Printf("Invalid URL scheme: %v, scheme: %v", urlStr, parsedURL.Scheme)
		return
	}

	// 2. Fetch PDF with timeout (same as before)
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(urlStr)
	if err != nil {
		http.Error(w, "Failed to fetch PDF", http.StatusBadGateway)
		log.Printf("Failed to fetch PDF from URL: %v, error: %v", urlStr, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Failed to fetch PDF, server returned "+resp.Status, http.StatusBadGateway)
		log.Printf("Failed to fetch PDF, server returned status: %v, URL: %v", resp.Status, urlStr)
		return
	}

	// **Correction: Read resp.Body into memory and create io.ReadSeeker**
	pdfData, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read PDF body", http.StatusInternalServerError)
		log.Printf("Failed to read PDF body from URL: %v, error: %v", urlStr, err)
		return
	}
	pdfReader, err := unidoc.NewPdfReader(bytes.NewReader(pdfData)) // Use bytes.NewReader
	if err != nil {
		http.Error(w, "Failed to parse PDF", http.StatusUnprocessableEntity)
		log.Printf("Failed to parse PDF from URL: %v, error: %v", urlStr, err)
		return
	}

	isEncrypted, err := pdfReader.IsEncrypted()
	if err != nil {
		log.Printf("Error checking encryption: %v, URL: %v", err, urlStr)
		isEncrypted = false
	}

	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		http.Error(w, "Failed to get page count", http.StatusInternalServerError)
		log.Printf("Failed to get page count from URL: %v, error: %v", urlStr, err)
		return
	}

	// 4. Extract metadata (Title, Author)
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
		log.Println("Failed to get PDF Info or Info dictionary is nil from URL:", urlStr, ", error:", err)
		title = ""
		author = ""
	}

	// 5. Create metadata map and encode JSON response
	metadata := map[string]interface{}{
		"page_count": numPages,
		"encrypted":  isEncrypted,
		"title":      title,
		"author":     author,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
		log.Printf("Failed to encode JSON response, error: %v", err)
	}
}

func handlePDFPreview(w http.ResponseWriter, r *http.Request) {
	urlStr := r.URL.Query().Get("url")
	if urlStr == "" {
		http.Error(w, "Missing PDF URL", http.StatusBadRequest)
		return
	}

	// 1. Validate URL (same as before)
	parsedURL, err := url.ParseRequestURI(urlStr)
	if err != nil {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		log.Printf("Invalid URL format for preview: %v, error: %v", urlStr, err)
		return
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		http.Error(w, "Invalid URL scheme for preview, must be http or https", http.StatusBadRequest)
		log.Printf("Invalid URL scheme for preview: %v, scheme: %v", urlStr, parsedURL.Scheme)
		return
	}

	// 2. Fetch PDF with timeout (same as before)
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(urlStr)
	if err != nil {
		http.Error(w, "Failed to fetch PDF for preview", http.StatusBadGateway)
		log.Printf("Failed to fetch PDF for preview from URL: %v, error: %v", urlStr, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Failed to fetch PDF for preview, server returned "+resp.Status, http.StatusBadGateway)
		log.Printf("Failed to fetch PDF for preview, server returned status: %v, URL: %v", resp.Status, urlStr)
		return
	}

	// **Correction for handlePDFPreview as well (if needed for go-fitz - check fitz.NewFromReader docs)**
	pdfData, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read PDF body for preview", http.StatusInternalServerError)
		log.Printf("Failed to read PDF body for preview from URL: %v, error: %v", urlStr, err)
		return
	}

	// 3. Render first page using go-fitz - Check if go-fitz also needs io.ReadSeeker
	doc, err := fitz.NewFromReader(bytes.NewReader(pdfData)) // Use bytes.NewReader for go-fitz too, if needed
	if err != nil {
		http.Error(w, "Failed to parse PDF for preview", http.StatusUnprocessableEntity)
		log.Printf("Failed to parse PDF for preview from URL: %v, error: %v", urlStr, err)
		return
	}
	defer doc.Close()

	// 4. Get first page image
	img, err := doc.Image(0)
	if err != nil {
		http.Error(w, "Failed to render preview image", http.StatusInternalServerError)
		log.Printf("Failed to render preview image from URL: %v, error: %v", urlStr, err)
		return
	}

	// 5. Set Content-Type and encode JPEG response
	w.Header().Set("Content-Type", "image/jpeg")
	if err := jpeg.Encode(w, img, &jpeg.Options{Quality: 85}); err != nil {
		http.Error(w, "Failed to encode JPEG for preview", http.StatusInternalServerError)
		log.Printf("Failed to encode JPEG for preview, error: %v", err)
	}
}

// Fixed error handling and added token forwarding
func handleGitHubProxy(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")
	repo := r.URL.Query().Get("repo")
	path := r.URL.Query().Get("path")
	token := r.URL.Query().Get("token")

	if owner == "" || repo == "" {
		http.Error(w, "Missing owner/repo parameters", http.StatusBadRequest)
		log.Println("GitHub Proxy: Missing owner/repo parameters") // Added log
		return
	}

	// Construct GitHub API URL
	ghURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, path)

	// Validate URL (important for proxy to prevent unexpected URLs)
	parsedURL, err := url.ParseRequestURI(ghURL)
	if err != nil {
		http.Error(w, "Invalid GitHub URL format", http.StatusBadRequest)
		log.Printf("GitHub Proxy: Invalid GitHub URL format: %v, error: %v", ghURL, err)
		return
	}
	if parsedURL.Scheme != "https" { // Enforce HTTPS for security
		http.Error(w, "Invalid GitHub URL scheme, must be https", http.StatusBadRequest)
		log.Printf("GitHub Proxy: Invalid GitHub URL scheme: %v, scheme: %v", ghURL, parsedURL.Scheme)
		return
	}

	ghReq, err := http.NewRequest("GET", ghURL, nil) // Capture error from NewRequest
	if err != nil {
		http.Error(w, "Failed to create GitHub API request", http.StatusInternalServerError)
		log.Printf("GitHub Proxy: Failed to create GitHub API request, error: %v", err)
		return
	}

	if token != "" {
		ghReq.Header.Add("Authorization", "token "+token) // Forward token
	}

	resp, err := http.DefaultClient.Do(ghReq)
	if err != nil {
		http.Error(w, "GitHub API request failed", http.StatusBadGateway)
		log.Printf("GitHub Proxy: GitHub API request failed, error: %v", err)
		return
	}
	defer resp.Body.Close()

	// Forward status code and headers
	w.WriteHeader(resp.StatusCode)
	for k, v := range resp.Header {
		w.Header()[k] = v
	}

	_, copyErr := io.Copy(w, resp.Body) // Capture error from io.Copy
	if copyErr != nil {
		log.Printf("GitHub Proxy: Error copying response body: %v", copyErr) // Log copy error, but don't fail request
		// In a real MVP, you might want to handle this more robustly, maybe return an error to the client if critical.
	}
}
