package handlers

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/mule-ai/mule/internal/settings"
	"github.com/mule-ai/mule/internal/state"
	"github.com/mule-ai/mule/pkg/remote/types"
)

// Here's a breakdown of its key components and functions:

// LocalPageData struct: This struct defines the structure of data that will likely be used to render a
// web page related to a local repository. It includes information about the Repository,
// lists of Issues and PullRequests, the current Page identifier ("local"), and the application Settings.

// HandleLocalProviderPage: This handler retrieves information (issues and pull requests) for a 
// specific local repository identified by a path query parameter. It fetches the repository from
// the application's state, retrieves the relevant data using the repository's Remote interface, 
// and then likely renders an HTML template ("layout.html") with the gathered LocalPageData.

// HandleCreateLocalIssue: This handler processes requests to create a new issue in a local repository. 
// It expects a JSON request body containing the path of the repository, the issue title, and body. It retrieves the repository, 
// uses the repo.Remote.CreateIssue method to create the issue, and returns the new issue number.

// HandleAddLocalComment: This handler adds a comment to either an issue or a pull request in a local repository. 
// It takes a JSON request body with the repository path, the resourceId (issue or PR number), 
// resourceType ("issue" or "pr"), the comment body, and an optional diffHunk. 
// It calls the appropriate repo.Remote.CreateIssueComment or repo.Remote.CreatePRComment method.

// HandleAddLocalReaction: This handler adds a reaction to a comment on a local repository resource. 
// It expects a JSON body with the repository path, the commentId, and the reaction type. 
// It uses repo.Remote.AddCommentReaction to perform the action.

// HandleGetLocalDiff: This handler retrieves the diff for a specific pull request in a local repository. 
// It requires path and prNumber as query parameters, fetches the diff using repo.Remote.FetchDiffs, 
// and returns the diff as plain text.

// HandleAddLocalLabel: This handler adds a label to an issue in a local repository. 
// It takes a JSON body with the repository path, issueNumber, and the label to add, 
// using repo.Remote.AddLabelToIssue.

// HandleUpdateLocalIssueState: This handler updates the state (open or closed) of an issue in a local repository. 
// It expects a JSON body with the repository path, issueNumber, and the desired state, 
// calling repo.Remote.UpdateIssueState. It includes validation for the state value.

// HandleUpdateLocalPullRequestState: This handler updates the state of a pull request in a local repository. 
// It takes a JSON body with the repository path, prNumber, and the target state, 
// using repo.Remote.UpdatePullRequestState.

// HandleDeleteLocalIssue: This handler deletes an issue from a local repository, 
// requiring the repository path and issueNumber in a JSON request body and calling repo.Remote.DeleteIssue.

// HandleDeleteLocalPullRequest: This handler deletes a pull request from a local repository, 
// taking the repository path and prNumber in a JSON body and using repo.Remote.DeletePullRequest.

// getRepository (Helper Function): Although not explicitly shown in the provided snippet, 
// the repeated use of getRepository(req.Path) suggests there's a helper function 
// (likely within this package or a related internal package) that standardizes the process of retrieving a 
// repository from the global state.State.Repositories map based on its absolute path, 
// handling potential errors like the repository not being found. 
// It also appears to handle locking the repository's mutex before performing operations.

// In summary, this package provides a set of HTTP endpoints that allow interacting with issues, 
// pull requests, comments, reactions, and labels within local repositories managed by the mule-ai/mule application,
// acting as a bridge between HTTP requests and the underlying "local" remote provider implementation.


type LocalPageData struct {
	Repository   interface{}
	Issues       []types.Issue
	PullRequests []types.PullRequest
	Page         string
	Settings     settings.Settings
}

func HandleLocalProviderPage(w http.ResponseWriter, r *http.Request) {
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

	issues, err := repo.Remote.FetchIssues(absPath, types.IssueFilterOptions{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pullRequests, err := repo.Remote.FetchPullRequests(absPath, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := LocalPageData{
		Repository:   repo,
		Page:         "local",
		Settings:     state.State.Settings,
		Issues:       issues,
		PullRequests: pullRequests,
	}

	err = templates.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func HandleCreateLocalIssue(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path  string `json:"path"`
		Title string `json:"title"`
		Body  string `json:"body"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	repo, err := getRepository(req.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	repo.Mu.Lock()
	defer repo.Mu.Unlock()

	issueNumber, err := repo.Remote.CreateIssue(types.Issue{
		Title:     req.Title,
		Body:      req.Body,
		State:     "open",
		CreatedAt: time.Now().String(),
		Comments:  make([]*types.Comment, 0),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte(strconv.Itoa(issueNumber)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func HandleAddLocalComment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path         string `json:"path"`
		ResourceID   int    `json:"resourceId"`
		ResourceType string `json:"resourceType"`
		Body         string `json:"body"`
		DiffHunk     string `json:"diffHunk,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	repo, err := getRepository(req.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	repo.Mu.Lock()
	defer repo.Mu.Unlock()

	comment := &types.Comment{
		ID:        time.Now().Unix(),
		Body:      req.Body,
		DiffHunk:  req.DiffHunk,
		Reactions: types.Reactions{},
	}

	switch req.ResourceType {
	case "issue":
		err = repo.Remote.CreateIssueComment(req.Path, req.ResourceID, *comment)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case "pr":
		err = repo.Remote.CreatePRComment(req.Path, req.ResourceID, *comment)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
}

func HandleAddLocalReaction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path      string `json:"path"`
		CommentID int64  `json:"commentId"`
		Reaction  string `json:"reaction"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	repo, err := getRepository(req.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	repo.Mu.Lock()
	defer repo.Mu.Unlock()

	err = repo.Remote.AddCommentReaction(req.Path, req.Reaction, req.CommentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func HandleGetLocalDiff(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	prNumber := r.URL.Query().Get("pr")
	if path == "" || prNumber == "" {
		http.Error(w, "Path and PR number are required", http.StatusBadRequest)
		return
	}

	prNum, err := strconv.Atoi(prNumber)
	if err != nil {
		http.Error(w, "Invalid PR number", http.StatusBadRequest)
		return
	}

	repo, err := getRepository(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	diff, err := repo.Remote.FetchDiffs("", "", prNum)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	_, err = w.Write([]byte(diff))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func HandleAddLocalLabel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path        string `json:"path"`
		IssueNumber int    `json:"issueNumber"`
		Label       string `json:"label"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	repo, err := getRepository(req.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	repo.Mu.Lock()
	defer repo.Mu.Unlock()

	err = repo.Remote.AddLabelToIssue(req.IssueNumber, req.Label)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func HandleUpdateLocalIssueState(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path        string `json:"path"`
		IssueNumber int    `json:"issueNumber"`
		State       string `json:"state"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.State != "open" && req.State != "closed" {
		http.Error(w, "Invalid state. Must be 'open' or 'closed'", http.StatusBadRequest)
		return
	}

	repo, err := getRepository(req.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	repo.Mu.Lock()
	defer repo.Mu.Unlock()

	err = repo.Remote.UpdateIssueState(req.IssueNumber, req.State)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func HandleUpdateLocalPullRequestState(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path     string `json:"path"`
		PRNumber int    `json:"prNumber"`
		State    string `json:"state"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	repo, err := getRepository(req.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	repo.Mu.Lock()
	defer repo.Mu.Unlock()

	err = repo.Remote.UpdatePullRequestState(req.Path, req.PRNumber, req.State)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func HandleDeleteLocalIssue(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path        string `json:"path"`
		IssueNumber int    `json:"issueNumber"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	repo, err := getRepository(req.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	repo.Mu.Lock()
	defer repo.Mu.Unlock()

	err = repo.Remote.DeleteIssue(req.Path, req.IssueNumber)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func HandleDeleteLocalPullRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path     string `json:"path"`
		PRNumber int    `json:"prNumber"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	repo, err := getRepository(req.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	repo.Mu.Lock()
	defer repo.Mu.Unlock()

	err = repo.Remote.DeletePullRequest(req.Path, req.PRNumber)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func HandleUpdateLocalIssue(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path        string `json:"path"`
		IssueNumber int    `json:"issueNumber"`
		Title       string `json:"title"`
		Body        string `json:"body"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	repo, err := getRepository(req.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	repo.Mu.Lock()
	defer repo.Mu.Unlock()

	err = repo.Remote.UpdateIssue(req.IssueNumber, req.Title, req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
