package controllers

import (
	"html/template"
	"log"
	"net/http"
	"strings"
)

var count int

// Create template functions map with direct function reference
var funcMap = template.FuncMap{
	"split": strings.Split, // Register split function directly
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	count++
	log.Printf("Request #%d - Method: %s, URL: %s, User-Agent: %s",
		count,
		r.Method,
		r.URL.Path,
		r.Header.Get("User-Agent"))

	// Create a new template with functions
	tmpl := template.New("index.html").Funcs(funcMap)

	// Parse template files
	tmpl, err := tmpl.ParseFiles("views/index.html", "views/file_list.html", "views/pdf_reader.html")
	if err != nil {
		log.Printf("Template parsing error: %v", err)
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}

	// Get path and ensure it starts with /
	path := r.URL.Query().Get("path")
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	data := map[string]interface{}{
		"Title":            "PDFRepo",
		"TitleDescription": "PDF Repository Explorer",
		"Author":           "andy (codeangel.tech)",
		"Owner":            "",
		"Repo":             "",
		"IsPrivate":        "0",
		"Path":             path,
		"Token":            "",
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}
