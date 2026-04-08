// Package main implements the Pulsarity Health Collector service.
// It polls registered endpoints and records their health status.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	port         string
	targets      = make(map[string]*Target)
	targetsMu    sync.RWMutex
	logger       *log.Logger
)

// Target represents a monitored endpoint.
type Target struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	IntervalSec int       `json:"interval_sec"`
	Status      string    `json:"status"`
	LastChecked string    `json:"last_checked"`
	ResponseMs  int64     `json:"response_ms"`
	CreatedAt   string    `json:"created_at"`
}

// HealthResponse is returned by the /health endpoint.
type HealthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
}

// TargetListResponse wraps the list of targets.
type TargetListResponse struct {
	Targets []*Target `json:"targets"`
	Count   int       `json:"count"`
}

// ErrorResponse represents an error.
type ErrorResponse struct {
	Error string `json:"error"`
}

// MessageResponse represents a success message.
type MessageResponse struct {
	Message string `json:"message"`
}

// CheckResult is returned after a health check.
type CheckResult struct {
	TargetID   string `json:"target_id"`
	Status     string `json:"status"`
	ResponseMs int64  `json:"response_ms"`
	CheckedAt  string `json:"checked_at"`
}

func init() {
	port = os.Getenv("COLLECTOR_PORT")
	if port == "" {
		port = "8002"
	}
	logger = log.New(os.Stdout, "[health-collector] ", log.LstdFlags)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	logger.Println("Health check requested")
	writeJSON(w, http.StatusOK, HealthResponse{
		Status:    "ok",
		Service:   "health-collector",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func listTargetsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "Method not allowed"})
		return
	}
	targetsMu.RLock()
	defer targetsMu.RUnlock()

	list := make([]*Target, 0, len(targets))
	for _, t := range targets {
		list = append(list, t)
	}
	logger.Printf("Listing %d targets", len(list))
	writeJSON(w, http.StatusOK, TargetListResponse{Targets: list, Count: len(list)})
}

func createTargetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "Method not allowed"})
		return
	}

	var input struct {
		Name        string `json:"name"`
		URL         string `json:"url"`
		IntervalSec int    `json:"interval_sec"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		logger.Printf("Invalid request body: %v", err)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid JSON body"})
		return
	}

	if input.Name == "" || input.URL == "" {
		logger.Printf("Missing required fields: name=%s url=%s", input.Name, input.URL)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Fields 'name' and 'url' are required"})
		return
	}

	if input.IntervalSec <= 0 {
		input.IntervalSec = 30
	}

	id := fmt.Sprintf("t-%d", time.Now().UnixNano()%100000000)
	target := &Target{
		ID:          id,
		Name:        input.Name,
		URL:         input.URL,
		IntervalSec: input.IntervalSec,
		Status:      "pending",
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	targetsMu.Lock()
	targets[id] = target
	targetsMu.Unlock()

	logger.Printf("Created target: id=%s name=%s url=%s", id, input.Name, input.URL)
	writeJSON(w, http.StatusCreated, target)
}

func deleteTargetHandler(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "Method not allowed"})
		return
	}

	targetsMu.Lock()
	defer targetsMu.Unlock()

	if _, exists := targets[id]; !exists {
		logger.Printf("Target not found: %s", id)
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "Target not found"})
		return
	}

	delete(targets, id)
	logger.Printf("Deleted target: %s", id)
	writeJSON(w, http.StatusOK, MessageResponse{Message: "Target deleted"})
}

func checkTargetHandler(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "Method not allowed"})
		return
	}

	targetsMu.RLock()
	target, exists := targets[id]
	targetsMu.RUnlock()

	if !exists {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "Target not found"})
		return
	}

	start := time.Now()
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(target.URL)
	elapsed := time.Since(start).Milliseconds()

	targetsMu.Lock()
	target.LastChecked = time.Now().UTC().Format(time.RFC3339)
	target.ResponseMs = elapsed
	if err != nil || resp.StatusCode >= 400 {
		target.Status = "unhealthy"
	} else {
		target.Status = "healthy"
	}
	if resp != nil {
		resp.Body.Close()
	}
	targetsMu.Unlock()

	logger.Printf("Checked target %s: status=%s response_ms=%d", id, target.Status, elapsed)
	writeJSON(w, http.StatusOK, CheckResult{
		TargetID:   id,
		Status:     target.Status,
		ResponseMs: elapsed,
		CheckedAt:  target.LastChecked,
	})
}

func targetsRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// POST /targets or GET /targets
	if path == "/targets" {
		switch r.Method {
		case http.MethodGet:
			listTargetsHandler(w, r)
		case http.MethodPost:
			createTargetHandler(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "Method not allowed"})
		}
		return
	}

	// Extract ID from /targets/{id} or /targets/{id}/check
	var id, action string
	rest := path[len("/targets/"):]
	for i, c := range rest {
		if c == '/' {
			id = rest[:i]
			action = rest[i+1:]
			break
		}
	}
	if id == "" {
		id = rest
	}

	if action == "check" {
		checkTargetHandler(w, r, id)
		return
	}
	if action == "" {
		if r.Method == http.MethodDelete {
			deleteTargetHandler(w, r, id)
			return
		}
		// GET single target
		targetsMu.RLock()
		t, exists := targets[id]
		targetsMu.RUnlock()
		if !exists {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "Target not found"})
			return
		}
		writeJSON(w, http.StatusOK, t)
		return
	}

	writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "Not found"})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/targets", targetsRouter)
	mux.HandleFunc("/targets/", targetsRouter)

	logger.Printf("Starting Health Collector on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		logger.Fatalf("Server failed: %v", err)
	}
}
