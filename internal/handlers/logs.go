package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mule-ai/mule/pkg/log"
)

/*
This package implements an HTTP handler for viewing and filtering logs in a web interface. Let me break it down comprehensively:
Core Purpose
The package provides a web-based log viewer that:

Reads log entries from a JSON-formatted log file
Groups logs by conversation ID
Allows filtering by search terms and log levels
Presents logs in a user-friendly web interface or as JSON for AJAX requests

Key Components
Data Structures

LogEntry
Represents a single log entry with fields like level, timestamp, message, content, model, error, etc.
Maps directly to the JSON structure in the log file
Includes a computed Time field derived from the timestamp

Conversation
Groups related log entries by their conversation ID
Tracks metadata like start time, message count, and status
Status is determined by the level of the most recent message

LogsData
Container for rendering the template with conversation data
Includes page identification and the list of conversations

Main Handler Function: HandleLogs
This function processes HTTP requests to view logs and:

Handles Query Parameters
search: Text to search for in log messages
level: Filter by log level (error, info, etc.)
limit: Number of conversations to display (defaults to 10)

Detects Request Type
Determines if it's an AJAX request or direct browser request
Returns JSON for AJAX, HTML for direct browser access

Processes Log Files
Opens and reads the log file specified in log.LogFile
Handles potentially very large lines with a 1MB limit
Parses each line as JSON into a LogEntry

Applies Security Measures
Uses html.EscapeString() to prevent XSS attacks by escaping HTML characters

Organizes Data
Groups entries by conversation ID
Sorts messages chronologically within each conversation
Sorts conversations by start time (newest first)

Applies Filters
Filters logs by level if specified
Filters logs by search term if specified

Returns Results
For AJAX: Returns JSON data
For direct browser requests: Renders the "layout.html" template with log data

Notable Features

Efficient Reading
Handles large log files by reading line by line
Can process partial lines for extremely long entries

Content Protection
Truncates extremely large content fields
Escapes HTML in all user-visible fields

Flexible Output
Supports both HTML and JSON output formats
Enables both direct viewing and programmatic access

Pagination and Filtering
Limits results to a configurable number
Supports text search and log level filtering

Integration Points
Uses the log package from the same project to access the log file location
Renders results using a template system (referenced but not defined in this code)
Designed to work as part of a web application with AJAX capabilities
*/

type LogEntry struct {
	Level     string  `json:"level"`
	TimeStamp float64 `json:"ts"`
	Time      time.Time
	Logger    string `json:"logger"`
	Caller    string `json:"caller"`
	Message   string `json:"msg"`
	Content   string `json:"content,omitempty"`
	Model     string `json:"model,omitempty"`
	Error     string `json:"error,omitempty"`
	ID        string `json:"id,omitempty"`
}

type Conversation struct {
	ID           string
	StartTime    time.Time
	Messages     []LogEntry
	MessageCount int
	Status       string // Status based on last message level
}

type LogsData struct {
	Page          string
	Conversations []Conversation
}

func HandleLogs(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	level := r.URL.Query().Get("level")
	limitStr := r.URL.Query().Get("limit")
	isAjax := r.Header.Get("X-Requested-With") == "XMLHttpRequest"

	// Parse limit parameter, default to 10 if not specified or invalid
	limit := 10
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
		}
	}

	// Read and parse log file
	file, err := os.Open(log.LogFile)
	if err != nil {
		errString := fmt.Sprintf("Error reading log file: %v", err)
		if isAjax {
			http.Error(w, `{"error": "`+errString+`"}`, http.StatusInternalServerError)
		} else {
			http.Error(w, errString, http.StatusInternalServerError)
		}
		return
	}
	defer file.Close()

	// Map to store conversations by ID
	conversations := make(map[string]*Conversation)
	reader := bufio.NewReader(file)

	const maxLineLength = 1024 * 1024 // 1MB

	for {
		// ReadLine returns line, isPrefix, error
		var fullLine []byte
		for {
			line, isPrefix, err := reader.ReadLine()
			if err != nil {
				if err.Error() == "EOF" {
					break
				}
				errString := fmt.Sprintf("Error reading line: %v", err)
				if isAjax {
					http.Error(w, `{"error": "`+errString+`"}`, http.StatusInternalServerError)
				} else {
					http.Error(w, errString, http.StatusInternalServerError)
				}
				return
			}

			fullLine = append(fullLine, line...)
			if !isPrefix {
				break
			}
		}

		// Break the outer loop if we've reached EOF
		if len(fullLine) == 0 {
			break
		}

		var entry LogEntry
		if len(fullLine) > maxLineLength {
			// Try to parse the JSON we have to get the metadata
			if err := json.Unmarshal(fullLine, &entry); err != nil {
				continue // Skip if we can't parse the JSON
			}
			// Only truncate the content field if it exists and is too long
			if entry.Content != "" && len(entry.Content) > maxLineLength {
				entry.Content = fmt.Sprintf("[Content exceeds %d bytes and has been truncated]", maxLineLength)
			}
		} else {
			if err := json.Unmarshal(fullLine, &entry); err != nil {
				continue // Skip invalid JSON entries
			}
		}
		entry.Time = time.Unix(int64(entry.TimeStamp), 0)

		// HTML escape the content and message fields
		entry.Message = html.EscapeString(entry.Message)
		entry.Content = html.EscapeString(entry.Content)
		entry.Error = html.EscapeString(entry.Error)

		// Apply filters
		if level != "" && !strings.EqualFold(entry.Level, level) {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(entry.Message), strings.ToLower(search)) {
			continue
		}

		// Group by conversation ID
		if entry.ID != "" {
			if conv, exists := conversations[entry.ID]; exists {
				conv.Messages = append(conv.Messages, entry)
				conv.MessageCount++
			} else {
				conversations[entry.ID] = &Conversation{
					ID:           entry.ID,
					StartTime:    entry.Time,
					Messages:     []LogEntry{entry},
					MessageCount: 1,
				}
			}
		}
	}

	// Convert map to slice and sort by start time
	var sortedConversations []Conversation
	for _, conv := range conversations {
		// Sort messages within conversation by timestamp
		sort.Slice(conv.Messages, func(i, j int) bool {
			return conv.Messages[i].Time.Before(conv.Messages[j].Time)
		})

		// Set status based on last message level
		if len(conv.Messages) > 0 {
			conv.Status = conv.Messages[len(conv.Messages)-1].Level
		}

		sortedConversations = append(sortedConversations, *conv)
	}

	// Sort conversations by start time, most recent first
	sort.Slice(sortedConversations, func(i, j int) bool {
		return sortedConversations[i].StartTime.After(sortedConversations[j].StartTime)
	})

	// Apply conversation limit if greater than 0 (0 means no limit)
	if limit > 0 && len(sortedConversations) > limit {
		sortedConversations = sortedConversations[:limit]
	}

	if isAjax {
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(sortedConversations)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	data := LogsData{
		Page:          "logs",
		Conversations: sortedConversations,
	}

	err = templates.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
