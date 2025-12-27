package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/storo/lettice/pkg/core"
	"github.com/storo/lettice/pkg/mesh"
	"github.com/storo/lettice/pkg/security"
)

// Server provides an HTTP API for the mesh.
type Server struct {
	mesh   *mesh.Mesh
	auth   *security.Auth
	mux    *http.ServeMux
	server *http.Server
}

// ServerOption configures the server.
type ServerOption func(*Server)

// NewServer creates a new HTTP server for the mesh.
func NewServer(m *mesh.Mesh, opts ...ServerOption) *Server {
	s := &Server{
		mesh: m,
		mux:  http.NewServeMux(),
	}

	for _, opt := range opts {
		opt(s)
	}

	s.registerRoutes()

	return s
}

// WithAuth enables authentication on the server.
func WithAuth(auth *security.Auth) ServerOption {
	return func(s *Server) {
		s.auth = auth
	}
}

// registerRoutes sets up the HTTP routes.
func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/agents", s.handleAgents)
	s.mux.HandleFunc("/agents/", s.handleAgent)
	s.mux.HandleFunc("/mesh/run", s.handleMeshRun)
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Skip auth for health check
	if r.URL.Path == "/health" {
		s.mux.ServeHTTP(w, r)
		return
	}

	// Check authentication if configured
	if s.auth != nil {
		claims, err := s.auth.AuthenticateRequest(r.Context(), r)
		if err != nil {
			s.writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		// Add claims to context
		ctx := context.WithValue(r.Context(), claimsKey, claims)
		r = r.WithContext(ctx)
	}

	s.mux.ServeHTTP(w, r)
}

// ListenAndServe starts the server.
func (s *Server) ListenAndServe(addr string) error {
	s.server = &http.Server{
		Addr:         addr,
		Handler:      s,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	return s.server.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// handleHealth responds to health checks.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// handleAgents lists all agents.
func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()
	agents, err := s.mesh.ListAgents(ctx)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	infos := make([]AgentInfo, 0, len(agents))
	for _, a := range agents {
		infos = append(infos, agentToInfo(a))
	}

	s.writeJSON(w, http.StatusOK, ListAgentsResponse{Agents: infos})
}

// handleAgent handles individual agent routes.
func (s *Server) handleAgent(w http.ResponseWriter, r *http.Request) {
	// Extract agent ID from path: /agents/{id} or /agents/{id}/run
	path := strings.TrimPrefix(r.URL.Path, "/agents/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		s.writeError(w, http.StatusNotFound, "agent not found")
		return
	}

	agentID := parts[0]

	if len(parts) == 1 {
		// GET /agents/{id}
		s.handleGetAgent(w, r, agentID)
	} else if len(parts) == 2 && parts[1] == "run" {
		// POST /agents/{id}/run
		s.handleRunAgent(w, r, agentID)
	} else {
		s.writeError(w, http.StatusNotFound, "not found")
	}
}

// handleGetAgent returns agent info.
func (s *Server) handleGetAgent(w http.ResponseWriter, r *http.Request, agentID string) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()
	a, err := s.mesh.GetAgent(ctx, agentID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "agent not found")
		return
	}

	s.writeJSON(w, http.StatusOK, agentToInfo(a))
}

// handleRunAgent executes a specific agent.
func (s *Server) handleRunAgent(w http.ResponseWriter, r *http.Request, agentID string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx := r.Context()
	result, err := s.mesh.RunAgent(ctx, agentID, req.Input)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, resultToResponse(result))
}

// handleMeshRun executes on the mesh (auto-select agent).
func (s *Server) handleMeshRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx := r.Context()
	result, err := s.mesh.Run(ctx, req.Input)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, resultToResponse(result))
}

// writeJSON writes a JSON response.
func (s *Server) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response.
func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, ErrorResponse{Error: message})
}

// agentToInfo converts an agent to AgentInfo.
func agentToInfo(a core.Agent) AgentInfo {
	caps := make([]string, 0, len(a.Provides()))
	for _, c := range a.Provides() {
		caps = append(caps, string(c))
	}

	needs := make([]string, 0, len(a.Needs()))
	for _, c := range a.Needs() {
		needs = append(needs, string(c))
	}

	return AgentInfo{
		ID:          a.ID(),
		Name:        a.Name(),
		Description: a.Description(),
		Provides:    caps,
		Needs:       needs,
	}
}

// resultToResponse converts a Result to RunResponse.
func resultToResponse(r *core.Result) RunResponse {
	return RunResponse{
		Output:    r.Output,
		TokensIn:  r.TokensIn,
		TokensOut: r.TokensOut,
		Duration:  r.Duration.String(),
		TraceID:   r.TraceID,
	}
}

// Context key for claims
type contextKey string

const claimsKey contextKey = "claims"
