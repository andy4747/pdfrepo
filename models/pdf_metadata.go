package models

type PDFMetadata struct {
	PageCount int    `json:"page_count"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	Encrypted bool   `json:"encrypted"`
	Error     string `json:"error,omitempty"` // To pass errors to templates
}
