package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/rpgo/retirement-calculator/internal/config"
	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/rpgo/retirement-calculator/pkg/api"
)

// Server represents the web server instance
type Server struct {
	apiService *api.Service
	router     *mux.Router
	port       string
}

// NewServer creates a new web server instance
func NewServer(port string) *Server {
	apiService := api.NewService()
	router := mux.NewRouter()

	server := &Server{
		apiService: apiService,
		router:     router,
		port:       port,
	}

	server.setupRoutes()
	server.setupMiddleware()

	return server
}

// setupMiddleware configures middleware for the server
func (s *Server) setupMiddleware() {
	// CORS middleware
	s.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
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

	// Configuration endpoints
	apiV1.HandleFunc("/configurations", s.handleGetConfigurations).Methods(http.MethodGet)
	apiV1.HandleFunc("/configurations", s.handleCreateConfiguration).Methods(http.MethodPost)
	apiV1.HandleFunc("/configurations/{id}", s.handleGetConfiguration).Methods(http.MethodGet)
	apiV1.HandleFunc("/configurations/{id}", s.handleUpdateConfiguration).Methods(http.MethodPut)
	apiV1.HandleFunc("/configurations/{id}", s.handleDeleteConfiguration).Methods(http.MethodDelete)

	// Scenario endpoints
	apiV1.HandleFunc("/scenarios/run", s.handleRunScenario).Methods(http.MethodPost)

	// Example configuration endpoint
	apiV1.HandleFunc("/configurations/example", s.handleGetExampleConfiguration).Methods(http.MethodGet)
}

// Run starts the server and listens for requests
func (s *Server) Run() error {
	addr := fmt.Sprintf(":%s", s.port)
	log.Printf("Server starting on %s", addr)

	server := &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		<-quit
		log.Println("Server shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed to start: %w", err)
	}

	return nil
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

// handleRunScenario runs a scenario calculation
func (s *Server) handleRunScenario(w http.ResponseWriter, r *http.Request) {
	var cfg domain.Configuration
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
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
