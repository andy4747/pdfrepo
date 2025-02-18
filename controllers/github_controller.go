package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"pdfrepo-backend/models"
	"text/template"
)

func FetchGitHubContents(owner, repo, path, token string) ([]models.GitHubItem, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, path)
	ghReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub API request: %w", err)
	}

	if token != "" {
		ghReq.Header.Add("Authorization", "token "+token)
	}

	resp, err := http.DefaultClient.Do(ghReq)
	if err != nil {
		return nil, fmt.Errorf("github API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API returned status: %s", resp.Status)
	}

	var items []models.GitHubItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("failed to decode github API response: %w", err)
	}

	filteredItems := []models.GitHubItem{}
	for _, item := range items {
		if item.Type == "dir" || (item.Type == "file" && item.Name != "" && item.Name[len(item.Name)-4:] == ".pdf") { // Basic PDF check
			filteredItems = append(filteredItems, item)
		}
	}

	fmt.Println("Testing for empty dirs")
	fmt.Printf("%+v\n", filteredItems)

	return filteredItems, nil
}

func GitHubBrowserHandler(w http.ResponseWriter, r *http.Request) {
	owner := r.FormValue("owner")
	repo := r.FormValue("repo")
	token := r.FormValue("token")
	currentPath := r.FormValue("path")
	isPrivate := r.FormValue("isPrivate")
	var err error
	tmpl := template.New("index.html").Funcs(funcMap)
	tmpl, err = tmpl.ParseFiles("views/index.html", "views/file_list.html")
	if err != nil {
		http.Error(w, "Failed to load templates", http.StatusInternalServerError)
		return
	}
	items, err := FetchGitHubContents(owner, repo, currentPath, token)
	if err != nil {
		log.Printf("Error fetching GitHub contents: %v", err)
		data := map[string]interface{}{
			"Title":     "PDFRepo",
			"Owner":     owner,
			"Repo":      repo,
			"Token":     token,
			"Path":      currentPath,
			"IsPrivate": isPrivate,
			"ListFiles": "",
			"Error":     "Failed to fetch repository contents",
		}
		if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
		return
	}
	if len(items) == 0 {
		fmt.Println("Items empty and trying to send a nopdf message")
		data := map[string]interface{}{
			"Title":     "PDFRepo",
			"Owner":     owner,
			"Repo":      repo,
			"Token":     token,
			"Path":      currentPath,
			"IsPrivate": isPrivate,
			"ListFiles": "-",
			"NoPdf":     "No pdf files present in this directory.",
		}
		if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
			fmt.Println(err)
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
		return
	}

	if isPrivate == "1" && token == "" {
		data := map[string]interface{}{
			"Title":     "PDFRepo",
			"Owner":     owner,
			"Repo":      repo,
			"Token":     token,
			"Path":      currentPath,
			"IsPrivate": isPrivate,
			"ListFiles": "-",
			"Error":     "please provide a token",
		}
		if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
		return
	}
	if owner == "" || repo == "" {
		http.Error(w, "Owner and Repo are required", http.StatusBadRequest)
		return
	}
	data := map[string]interface{}{
		"Title":       "PDFRepo",
		"Owner":       owner,
		"Repo":        repo,
		"Token":       token,
		"Path":        currentPath,
		"IsPrivate":   isPrivate,
		"ListFiles":   "-",
		"GitHubItems": items,
	}
	if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
		fmt.Println(err)
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
	fmt.Println("One Process Done")
}
