package web

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/rpgo/retirement-calculator/internal/config"
	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/rpgo/retirement-calculator/pkg/api"
	"gopkg.in/yaml.v3"
)

//go:embed frontend/*
var frontendFS embed.FS

// Default idle timeout before auto-shutdown when no browser clients are connected.
const defaultIdleTimeout = 30 * time.Second

// Server represents the web server instance
type Server struct {
	apiService         *api.Service
	router             *mux.Router
	port               string
	allowedCORSOrigins []string
	maxBodyBytes       int64
	idleShutdown       bool          // enable auto-shutdown when no clients connected
	idleTimeout        time.Duration // how long to wait after last heartbeat

	// heartbeat tracking
	mu            sync.Mutex
	lastHeartbeat time.Time
}

// NewServer creates a new web server instance.
// By default, idle-shutdown is enabled so the process exits when no browser
// is connected. Call SetIdleShutdown(false) to disable (e.g. for development).
func NewServer(port string) *Server {
	apiService := api.NewService()
	router := mux.NewRouter()

	server := &Server{
		apiService:         apiService,
		router:             router,
		port:               port,
		allowedCORSOrigins: parseCORSOrigins(os.Getenv("FERSCALC_CORS_ORIGINS")),
		maxBodyBytes:       parseMaxBodyBytes(os.Getenv("FERSCALC_MAX_BODY_BYTES")),
		idleShutdown:       true,
		idleTimeout:        defaultIdleTimeout,
		lastHeartbeat:      time.Now(), // give the browser time to connect
	}

	server.setupRoutes()
	server.setupMiddleware()

	return server
}

// SetIdleShutdown enables or disables automatic shutdown when no browser
// clients are connected.
func (s *Server) SetIdleShutdown(enabled bool) {
	s.idleShutdown = enabled
}

// setupMiddleware configures middleware for the server
func (s *Server) setupMiddleware() {
	// CORS middleware
	s.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if originAllowed(origin, s.allowedCORSOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Request logging middleware
	s.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			log.Printf("Request: %s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
			log.Printf("Response: %s %s - %v", r.Method, r.URL.Path, time.Since(start))
		})
	})
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// API v1 routes
	apiV1 := s.router.PathPrefix("/api/v1").Subrouter()

	// Metadata endpoints
	apiV1.HandleFunc("/meta/states", s.handleGetStates).Methods(http.MethodGet)
	apiV1.HandleFunc("/meta/tsp-strategies", s.handleGetTSPStrategies).Methods(http.MethodGet)
	apiV1.HandleFunc("/meta/tsp-funds", s.handleGetTSPFunds).Methods(http.MethodGet)
	apiV1.HandleFunc("/meta/employment-types", s.handleGetEmploymentTypes).Methods(http.MethodGet)
	apiV1.HandleFunc("/meta/annuity-options", s.handleGetAnnuityOptions).Methods(http.MethodGet)

	// Configuration endpoints - specific routes first
	apiV1.HandleFunc("/configurations", s.handleGetConfigurations).Methods(http.MethodGet)
	apiV1.HandleFunc("/configurations", s.handleCreateConfiguration).Methods(http.MethodPost)
	apiV1.HandleFunc("/configurations/example", s.handleGetExampleConfiguration).Methods(http.MethodGet)
	apiV1.HandleFunc("/configurations/export-yaml", s.handleExportYAML).Methods(http.MethodPost)
	apiV1.HandleFunc("/configurations/parse-yaml", s.handleParseYAML).Methods(http.MethodPost)
	apiV1.HandleFunc("/configurations/{id}", s.handleGetConfiguration).Methods(http.MethodGet)
	apiV1.HandleFunc("/configurations/{id}", s.handleUpdateConfiguration).Methods(http.MethodPut)
	apiV1.HandleFunc("/configurations/{id}", s.handleDeleteConfiguration).Methods(http.MethodDelete)

	// Scenario endpoints
	apiV1.HandleFunc("/scenarios/run", s.handleRunScenario).Methods(http.MethodPost)

	// Heartbeat — browser pings this to keep the server alive
	apiV1.HandleFunc("/heartbeat", s.handleHeartbeat).Methods(http.MethodPost)

	// SPA frontend — serve embedded static files with fallback to index.html
	s.router.PathPrefix("/").Handler(spaHandler())
}

// spaHandler returns an http.Handler that serves the embedded frontend files.
// For any path that does not match a real file, it serves index.html (SPA fallback).
func spaHandler() http.Handler {
	// Strip the "frontend" prefix so "/" maps to "frontend/index.html"
	subFS, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		log.Fatalf("Failed to create sub filesystem for frontend: %v", err)
	}
	fileServer := http.FileServer(http.FS(subFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the actual file
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Check if file exists in the embedded FS
		filePath := strings.TrimPrefix(path, "/")
		if _, err := fs.Stat(subFS, filePath); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html for any unknown path
		indexData, err := fs.ReadFile(subFS, "index.html")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexData)
	})
}

// Run starts the server and listens for requests
func (s *Server) Run() error {
	addr := fmt.Sprintf(":%s", s.port)
	log.Printf("Server starting on %s", addr)
	log.Printf("Open http://localhost:%s in your browser", s.port)

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// shutdown is used by both signal handler and idle watchdog
	shutdown := func(reason string) {
		log.Printf("Server shutting down (%s)...", reason)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}

	// Graceful shutdown on Ctrl+C / SIGINT
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	go func() {
		<-quit
		shutdown("interrupt signal")
	}()

	// Idle-shutdown watchdog: exit when no browser client has sent a
	// heartbeat for longer than the idle timeout.
	if s.idleShutdown {
		log.Printf("Idle auto-shutdown enabled (timeout: %s)", s.idleTimeout)
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				s.mu.Lock()
				idle := time.Since(s.lastHeartbeat)
				s.mu.Unlock()
				if idle > s.idleTimeout {
					shutdown(fmt.Sprintf("no browser connected for %s", s.idleTimeout))
					return
				}
			}
		}()
	}

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed to start: %w", err)
	}

	return nil
}

// handleHeartbeat records that a browser client is still connected.
// The frontend sends a POST here every ~10 seconds.
func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.lastHeartbeat = time.Now()
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// handleGetStates returns a list of valid states for tax calculations
func (s *Server) handleGetStates(w http.ResponseWriter, r *http.Request) {
	states := []string{
		"Pennsylvania",
		"New Jersey",
		"Maryland",
		"Virginia",
		"Washington D.C.",
		"Delaware",
		"New York",
	}
	s.sendJSONResponse(w, states, http.StatusOK)
}

// handleGetTSPStrategies returns a list of valid TSP withdrawal strategies
func (s *Server) handleGetTSPStrategies(w http.ResponseWriter, r *http.Request) {
	strategies := []string{
		"4_percent_rule",
		"need_based",
		"variable_percentage",
		"fixed_annuity",
		"floor_ceiling",
	}
	s.sendJSONResponse(w, strategies, http.StatusOK)
}

// handleGetTSPFunds returns a list of valid TSP funds
func (s *Server) handleGetTSPFunds(w http.ResponseWriter, r *http.Request) {
	funds := []string{"c_fund", "s_fund", "i_fund", "f_fund", "g_fund"}
	s.sendJSONResponse(w, funds, http.StatusOK)
}

// handleGetEmploymentTypes returns a list of valid employment types
func (s *Server) handleGetEmploymentTypes(w http.ResponseWriter, r *http.Request) {
	types := []string{
		"federal",
		"non-federal",
	}
	s.sendJSONResponse(w, types, http.StatusOK)
}

// handleGetAnnuityOptions returns a list of valid annuity options
func (s *Server) handleGetAnnuityOptions(w http.ResponseWriter, r *http.Request) {
	options := map[string]interface{}{
		"payout_rates":     []float64{0.04, 0.045, 0.05, 0.055, 0.06, 0.065, 0.07},
		"cola_rates":       []float64{0.00, 0.01, 0.02, 0.03},
		"guaranteed_years": []int{0, 5, 10, 15, 20},
	}
	s.sendJSONResponse(w, options, http.StatusOK)
}

// handleGetConfigurations returns all saved configurations
func (s *Server) handleGetConfigurations(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement actual configuration retrieval
	sendErrorResponse(w, "Not implemented yet", http.StatusNotImplemented)
}

// handleCreateConfiguration creates a new configuration
func (s *Server) handleCreateConfiguration(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement actual configuration creation
	sendErrorResponse(w, "Not implemented yet", http.StatusNotImplemented)
}

// handleGetConfiguration returns a specific configuration by ID
func (s *Server) handleGetConfiguration(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement actual configuration retrieval
	sendErrorResponse(w, "Not implemented yet", http.StatusNotImplemented)
}

// handleUpdateConfiguration updates a specific configuration by ID
func (s *Server) handleUpdateConfiguration(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement actual configuration update
	sendErrorResponse(w, "Not implemented yet", http.StatusNotImplemented)
}

// handleDeleteConfiguration deletes a specific configuration by ID
func (s *Server) handleDeleteConfiguration(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement actual configuration deletion
	sendErrorResponse(w, "Not implemented yet", http.StatusNotImplemented)
}

// handleGetExampleConfiguration returns an example configuration
func (s *Server) handleGetExampleConfiguration(w http.ResponseWriter, r *http.Request) {
	parser := config.NewInputParser()
	example := parser.CreateExampleConfiguration()
	s.sendJSONResponse(w, example, http.StatusOK)
}

// handleExportYAML accepts a JSON configuration and returns it as YAML
func (s *Server) handleExportYAML(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.maxBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendErrorResponse(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Parse JSON into a generic map to preserve structure
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		sendErrorResponse(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		sendErrorResponse(w, fmt.Sprintf("YAML marshal failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=ferscalc_config.yaml")
	w.WriteHeader(http.StatusOK)
	w.Write(yamlBytes)
}

// handleParseYAML accepts raw YAML and returns it as parsed JSON
func (s *Server) handleParseYAML(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.maxBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendErrorResponse(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Parse YAML into a generic map
	var data interface{}
	if err := yaml.Unmarshal(body, &data); err != nil {
		sendErrorResponse(w, fmt.Sprintf("Invalid YAML: %v", err), http.StatusBadRequest)
		return
	}

	// Convert YAML map keys from interface{} to string for JSON compatibility
	data = convertYAMLToJSON(data)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

// convertYAMLToJSON recursively converts map[interface{}]interface{} to map[string]interface{}
// which is needed because yaml.v3 uses interface{} keys by default
func convertYAMLToJSON(val interface{}) interface{} {
	switch v := val.(type) {
	case map[interface{}]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			result[fmt.Sprint(key)] = convertYAMLToJSON(value)
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			result[key] = convertYAMLToJSON(value)
		}
		return result
	case []interface{}:
		for i, item := range v {
			v[i] = convertYAMLToJSON(item)
		}
		return v
	default:
		return v
	}
}

// handleRunScenario runs a scenario calculation
func (s *Server) handleRunScenario(w http.ResponseWriter, r *http.Request) {
	var cfg domain.Configuration
	r.Body = http.MaxBytesReader(w, r.Body, s.maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			sendErrorResponse(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		sendErrorResponse(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	results, err := s.apiService.RunScenario(ctx, &cfg, false)
	if err != nil {
		sendAPIErrorResponse(w, err)
		return
	}

	s.sendJSONResponse(w, results, http.StatusOK)
}

func parseCORSOrigins(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		val := strings.TrimSpace(p)
		if val != "" {
			out = append(out, val)
		}
	}
	return out
}

func parseMaxBodyBytes(raw string) int64 {
	const defaultMax = int64(2 << 20) // 2 MB
	if strings.TrimSpace(raw) == "" {
		return defaultMax
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || parsed <= 0 {
		return defaultMax
	}
	return parsed
}

func originAllowed(origin string, allowed []string) bool {
	if origin == "" {
		return false
	}
	for _, entry := range allowed {
		if entry == "*" {
			return true
		}
		if strings.EqualFold(origin, entry) {
			return true
		}
	}
	return false
}

// sendJSONResponse sends a JSON response
func (s *Server) sendJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to encode response"})
	}
}

// sendErrorResponse sends an error response
func sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// sendAPIErrorResponse sends an API error response
func sendAPIErrorResponse(w http.ResponseWriter, err error) {
	var statusCode int

	switch err.(type) {
	case *api.AppError:
		apiErr := err.(*api.AppError)
		switch apiErr.Kind {
		case api.ErrorKindValidation:
			statusCode = http.StatusBadRequest
		case api.ErrorKindConfig:
			statusCode = http.StatusBadRequest
		case api.ErrorKindCalculation:
			statusCode = http.StatusInternalServerError
		default:
			statusCode = http.StatusInternalServerError
		}
		sendErrorResponse(w, apiErr.Error(), statusCode)
	default:
		sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
	}
}
