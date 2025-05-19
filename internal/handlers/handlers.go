package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"

	"github.com/jbutlerdev/genai/tools"
	"github.com/mule-ai/mule/pkg/validation"
)
// Used to check if all the tools mentioned on the codebase are valid and
// we have all the tools mentioned

var templates *template.Template

func InitTemplates(t *template.Template) {
	templates = t
}

func HandleTools(w http.ResponseWriter, r *http.Request) {
	// These should match the tools defined in your codebase
	tools := tools.Tools()
	err := json.NewEncoder(w).Encode(tools)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func HandleValidationFunctions(w http.ResponseWriter, r *http.Request) {
	// Get validation functions from the validation package
	validationFuncs := validation.Validations()

	err := json.NewEncoder(w).Encode(validationFuncs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
