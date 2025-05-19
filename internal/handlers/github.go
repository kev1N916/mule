package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/mule-ai/mule/internal/state"
	"github.com/mule-ai/mule/pkg/repository"
)

/*
This Go package, handlers, appears to be responsible for handling HTTP requests related to GitHub data
within the mule-ai/mule application. It provides two main handler functions:

HandleGitHubRepositories: This function handles requests to fetch a list of GitHub repositories. 
It interacts with a remote.GitHub object (likely a client for the GitHub API stored in the application's state)
to retrieve the repositories and then encodes the result as a JSON response.

HandleGitHubIssues: This function handles requests to fetch issues for a specific GitHub repository. 
It requires a path query parameter in the request URL to identify the local path of the repository. 
It then looks up the repository in the application's state, checks for a configured GitHub token, 
updates the issues for that repository using the repo.UpdateIssues() method, and finally encodes the 
fetched issues as a JSON response.

*/
// Uses the github apis to get information

func HandleGitHubRepositories(w http.ResponseWriter, r *http.Request) {
	state.State.Mu.RLock()
	remote := state.State.Remote
	state.State.Mu.RUnlock()

	repos, err := remote.GitHub.FetchRepositories()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching repositories: %v", err), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(repos)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func HandleGitHubIssues(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Path parameter is required", http.StatusBadRequest)
		return
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	state.State.Mu.RLock()
	repo, exists := state.State.Repositories[absPath]
	state.State.Mu.RUnlock()

	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	state.State.Mu.RLock()
	token := state.State.Settings.GitHubToken
	state.State.Mu.RUnlock()

	if token == "" {
		http.Error(w, "GitHub token not configured", http.StatusBadRequest)
		return
	}

	err = repo.UpdateIssues()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching issues: %v", err), http.StatusInternalServerError)
		return
	}

	issues := make([]repository.Issue, 0, len(repo.Issues))
	for _, issue := range repo.Issues {
		issues = append(issues, *issue)
	}

	err = json.NewEncoder(w).Encode(issues)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
