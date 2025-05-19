package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mule-ai/mule/internal/config"
	"github.com/mule-ai/mule/internal/settings"
	"github.com/mule-ai/mule/internal/state"
	"github.com/mule-ai/mule/pkg/agent"
)

// Understood

// Core Purpose
// The package provides RESTful API endpoints that:

// Retrieve and update application settings
// Fetch template values for agent prompts
// Provide metadata about workflow input/output fields
// Connect the web interface with the application's state management

// Key Components
// HTTP Handler Functions

// HandleGetSettings
// Responds to GET requests for retrieving current application settings
// Returns the current settings as JSON from the application state
// Uses read-only locking to ensure thread safety


// HandleUpdateSettings
// Processes POST requests to update application settings
// Decodes JSON from the request body into a settings struct
// Calls handleSettingsChange to apply and persist the changes


// handleSettingsChange (internal helper)
// Updates the settings in the application state
// Triggers updates to agents and workflows
// Persists the changes to a configuration file


// HandleTemplateValues
// Returns available prompt template values for agents
// Provides metadata for template construction in the UI

// HandleWorkflowOutputFields
// Returns a list of available output fields for workflow steps
// These are standardized fields that can be passed between workflow steps

// HandleWorkflowInputMappings
// Returns a list of ways to map outputs from previous steps to inputs
// Defines how data flows between workflow components


// Important Data Elements

// Workflow Output Fields
// generatedText: Raw text output from an agent
// extractedCode: Code extracted from generated text
// summary: Content summary
// actionItems: Extracted action items
// suggestedChanges: Code change suggestions
// reviewComments: Code review comments
// testCases: Generated test cases
// documentationText: Generated documentation


// Workflow Input Mappings
// useAsPrompt: Use output directly as prompt
// appendToPrompt: Append output to existing prompt
// useAsContext: Use output as context information
// useAsInstructions: Use output as agent instructions
// useAsCodeInput: Use output as code to process
// useAsReviewTarget: Use output as review target

// Integration and Dependencies
// State Management: Uses state.State for accessing and mutating application state
// Settings: Works with the settings package for structure definitions
// Configuration: Uses config package to persist settings to disk
// Agent System: Interacts with the agent package for template values

// System Architecture Insights
// Based on this code, we can infer:

// Concurrent Design: The application uses mutex locks for thread safety, indicating it's designed for concurrent use
// Agent-Based Architecture: The system appears to use an agent-based approach with configurable prompts and workflows
// Workflow Orchestration: The system supports complex workflows where outputs from one step can be mapped as inputs to another step
// AI Integration: Fields like "generatedText" and the overall structure suggest this is part of an AI application, likely using language models
// Configuration Management: The application maintains persistent settings in the user's home directory


// returns the current settings
func HandleGetSettings(w http.ResponseWriter, r *http.Request) {
	state.State.Mu.RLock()
	defer state.State.Mu.RUnlock()

	err := json.NewEncoder(w).Encode(state.State.Settings)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func HandleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var settings settings.Settings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := handleSettingsChange(settings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Used to change the settings of agents and workflows using the agents
func handleSettingsChange(newSettings settings.Settings) error {
	state.State.Mu.Lock()
	state.State.Settings = newSettings
	state.State.Mu.Unlock()

	// Update agents and workflows after settings are updated.
	if err := state.State.UpdateAgents(); err != nil {
		return err
	}
	if err := state.State.UpdateWorkflows(); err != nil {
		return err
	}

	configPath, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configPath = filepath.Join(configPath, config.ConfigPath)
	if err := config.SaveConfig(configPath); err != nil {
		return fmt.Errorf("error saving config: %v", err)
	}
	return nil
}

func HandleTemplateValues(w http.ResponseWriter, r *http.Request) {
	values := agent.GetPromptTemplateValues()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(values); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandleWorkflowOutputFields returns the available output fields for workflow steps
func HandleWorkflowOutputFields(w http.ResponseWriter, r *http.Request) {
	// These are the fields that can be used as outputs from one agent to another
	outputFields := []string{
		"generatedText",     // The raw generated text from an agent
		"extractedCode",     //Code extracted from the generated text
		"summary",           // A summary of the generated content 
		"actionItems",       // Action items extracted from the content
		"suggestedChanges",  // Suggested code changes
		"reviewComments",    // Code review comments
		"testCases",         // Generated test cases
		"documentationText", // Generated documentation
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(outputFields); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandleWorkflowInputMappings returns the available input mappings for workflow steps
func HandleWorkflowInputMappings(w http.ResponseWriter, r *http.Request) {
	// These are the ways to map outputs from previous steps to inputs for the next step
	inputMappings := []string{
		"useAsPrompt",       // Use the output directly as the prompt
		"appendToPrompt",    // Append the output to the existing prompt
		"useAsContext",      // Use the output as context information
		"useAsInstructions", // Use the output as instructions for the agent
		"useAsCodeInput",    // Use the output as code to be processed
		"useAsReviewTarget", // Use the output as the target for a review
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(inputMappings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
