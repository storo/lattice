package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/storo/lettice/pkg/agent"
	"github.com/storo/lettice/pkg/core"
	"github.com/storo/lettice/pkg/mesh"
	"github.com/storo/lettice/pkg/provider"
	"github.com/storo/lettice/pkg/security"
)

func setupTestMesh() *mesh.Mesh {
	mockProvider := provider.NewMockWithResponse("Test response from agent")

	a := agent.New("test-agent").
		Model(mockProvider).
		Provides(core.CapResearch).
		Build()

	m := mesh.New()
	m.Register(a)

	return m
}

func TestServer_HealthCheck(t *testing.T) {
	m := setupTestMesh()
	server := NewServer(m)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", resp["status"])
	}
}

func TestServer_ListAgents(t *testing.T) {
	m := setupTestMesh()
	server := NewServer(m)

	req := httptest.NewRequest("GET", "/agents", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp ListAgentsResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(resp.Agents))
	}
}

func TestServer_GetAgent(t *testing.T) {
	m := setupTestMesh()
	server := NewServer(m)

	// First get the agent ID
	agents, _ := m.ListAgents(context.Background())
	agentID := agents[0].ID()

	req := httptest.NewRequest("GET", "/agents/"+agentID, nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp AgentInfo
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.ID != agentID {
		t.Errorf("expected agent ID '%s', got '%s'", agentID, resp.ID)
	}
}

func TestServer_GetAgentNotFound(t *testing.T) {
	m := setupTestMesh()
	server := NewServer(m)

	req := httptest.NewRequest("GET", "/agents/nonexistent", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestServer_RunAgent(t *testing.T) {
	m := setupTestMesh()
	server := NewServer(m)

	agents, _ := m.ListAgents(context.Background())
	agentID := agents[0].ID()

	body := RunRequest{Input: "Test input"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/agents/"+agentID+"/run", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp RunResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Output != "Test response from agent" {
		t.Errorf("expected 'Test response from agent', got '%s'", resp.Output)
	}
}

func TestServer_RunMesh(t *testing.T) {
	m := setupTestMesh()
	server := NewServer(m)

	body := RunRequest{Input: "Test input"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/mesh/run", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp RunResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Output == "" {
		t.Error("expected non-empty output")
	}
}

func TestServer_WithAuth(t *testing.T) {
	m := setupTestMesh()

	apiKeyAuth := security.NewAPIKeyAuth()
	apiKeyAuth.RegisterKey("valid-key", &security.KeyEntry{
		AgentID: "test",
		Roles:   []string{"admin"},
	})

	auth := security.NewAuth(security.WithAPIKeyAuth(apiKeyAuth))
	server := NewServer(m, WithAuth(auth))

	// Request without auth
	req := httptest.NewRequest("GET", "/agents", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 without auth, got %d", w.Code)
	}

	// Request with valid auth
	req = httptest.NewRequest("GET", "/agents", nil)
	req.Header.Set("X-API-Key", "valid-key")
	w = httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 with valid auth, got %d", w.Code)
	}
}

func TestServer_WithAuth_HealthNoAuth(t *testing.T) {
	m := setupTestMesh()

	apiKeyAuth := security.NewAPIKeyAuth()
	auth := security.NewAuth(security.WithAPIKeyAuth(apiKeyAuth))
	server := NewServer(m, WithAuth(auth))

	// Health check should work without auth
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("health check should not require auth, got status %d", w.Code)
	}
}

func TestServer_InvalidJSON(t *testing.T) {
	m := setupTestMesh()
	server := NewServer(m)

	req := httptest.NewRequest("POST", "/mesh/run", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid JSON, got %d", w.Code)
	}
}

func TestServer_MethodNotAllowed(t *testing.T) {
	m := setupTestMesh()
	server := NewServer(m)

	req := httptest.NewRequest("DELETE", "/agents", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}
