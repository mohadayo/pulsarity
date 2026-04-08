package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/targets", targetsRouter)
	mux.HandleFunc("/targets/", targetsRouter)
	return mux
}

func clearTargets() {
	targetsMu.Lock()
	defer targetsMu.Unlock()
	for k := range targets {
		delete(targets, k)
	}
}

func TestHealthEndpoint(t *testing.T) {
	clearTargets()
	mux := setupMux()
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp HealthResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "ok" {
		t.Fatalf("expected status ok, got %s", resp.Status)
	}
	if resp.Service != "health-collector" {
		t.Fatalf("expected service health-collector, got %s", resp.Service)
	}
}

func TestListTargetsEmpty(t *testing.T) {
	clearTargets()
	mux := setupMux()
	req := httptest.NewRequest("GET", "/targets", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp TargetListResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Count != 0 {
		t.Fatalf("expected 0 targets, got %d", resp.Count)
	}
}

func TestCreateTarget(t *testing.T) {
	clearTargets()
	mux := setupMux()
	body := map[string]interface{}{
		"name": "Test Service",
		"url":  "http://example.com/health",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/targets", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var target Target
	json.NewDecoder(w.Body).Decode(&target)
	if target.Name != "Test Service" {
		t.Fatalf("expected name 'Test Service', got %s", target.Name)
	}
	if target.IntervalSec != 30 {
		t.Fatalf("expected default interval 30, got %d", target.IntervalSec)
	}
}

func TestCreateTargetMissingFields(t *testing.T) {
	clearTargets()
	mux := setupMux()
	body := map[string]interface{}{"name": "No URL"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/targets", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateTargetInvalidJSON(t *testing.T) {
	clearTargets()
	mux := setupMux()
	req := httptest.NewRequest("POST", "/targets", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDeleteTarget(t *testing.T) {
	clearTargets()
	mux := setupMux()

	// Create first
	body := map[string]interface{}{"name": "Delete Me", "url": "http://example.com"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/targets", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var created Target
	json.NewDecoder(w.Body).Decode(&created)

	// Delete
	req = httptest.NewRequest("DELETE", "/targets/"+created.ID, nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestDeleteTargetNotFound(t *testing.T) {
	clearTargets()
	mux := setupMux()
	req := httptest.NewRequest("DELETE", "/targets/nonexistent", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetSingleTarget(t *testing.T) {
	clearTargets()
	mux := setupMux()

	body := map[string]interface{}{"name": "Get Me", "url": "http://example.com"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/targets", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var created Target
	json.NewDecoder(w.Body).Decode(&created)

	req = httptest.NewRequest("GET", "/targets/"+created.ID, nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var fetched Target
	json.NewDecoder(w.Body).Decode(&fetched)
	if fetched.Name != "Get Me" {
		t.Fatalf("expected name 'Get Me', got %s", fetched.Name)
	}
}

func TestGetSingleTargetNotFound(t *testing.T) {
	clearTargets()
	mux := setupMux()
	req := httptest.NewRequest("GET", "/targets/doesnotexist", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCheckTarget(t *testing.T) {
	clearTargets()
	mux := setupMux()

	// Start a tiny test server to check against
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	body := map[string]interface{}{"name": "Check Me", "url": ts.URL}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/targets", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var created Target
	json.NewDecoder(w.Body).Decode(&created)

	req = httptest.NewRequest("POST", "/targets/"+created.ID+"/check", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result CheckResult
	json.NewDecoder(w.Body).Decode(&result)
	if result.Status != "healthy" {
		t.Fatalf("expected healthy, got %s", result.Status)
	}
}

func TestCheckTargetNotFound(t *testing.T) {
	clearTargets()
	mux := setupMux()
	req := httptest.NewRequest("POST", "/targets/nope/check", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestListTargetsAfterCreate(t *testing.T) {
	clearTargets()
	mux := setupMux()

	for i := 0; i < 3; i++ {
		body := map[string]interface{}{
			"name": "Svc",
			"url":  "http://example.com",
		}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/targets", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
	}

	req := httptest.NewRequest("GET", "/targets", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var resp TargetListResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Count != 3 {
		t.Fatalf("expected 3, got %d", resp.Count)
	}
}
