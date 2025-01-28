package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
)

type Config struct {
	Port            string
	LogLevel        slog.Level
	ServiceAURL     string
	ServiceAAuthKey string
}

type ServiceAResponse struct {
	Message   string    `json:"message"`
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
}

type Response struct {
	ServiceAMessage string    `json:"service_a_message"`
	ServiceBMessage string    `json:"service_b_message"`
	RequestID       string    `json:"request_id"`
	Timestamp       time.Time `json:"timestamp"`
}

type Server struct {
	config *Config
	logger *slog.Logger
	server *http.Server
	client *http.Client
}

func NewServer(config *Config) *Server {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: config.LogLevel,
	}))

	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	s := &Server{
		config: config,
		logger: logger,
		server: srv,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	mux.HandleFunc("/process", s.handleProcess)
	mux.HandleFunc("/health", s.handleHealth)

	return s
}

func (s *Server) handleProcess(w http.ResponseWriter, r *http.Request) {
	requestID := uuid.New().String()
	ctx := context.WithValue(r.Context(), "request_id", requestID)
	clientIP := r.RemoteAddr

	// Log incoming request details
	s.logger.Info("Processing service interaction request",
		slog.String("request_id", requestID),
		slog.String("client_ip", clientIP),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
	)

	startTime := time.Now()

	// Call Service A
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/greet", s.config.ServiceAURL), nil)
	if err != nil {
		s.handleError(w, r, err, "Failed to create request to Service A", http.StatusInternalServerError)
		return
	}
	req.Header.Set("X-Auth-Key", s.config.ServiceAAuthKey)
	req.Header.Set("X-Request-ID", requestID)

	// Log outgoing request to Service A
	s.logger.Info("Calling Service A",
		slog.String("request_id", requestID),
		slog.String("service_a_url", s.config.ServiceAURL),
		slog.String("auth_key_masked", maskAuthKey(s.config.ServiceAAuthKey)),
	)

	resp, err := s.client.Do(req)
	if err != nil {
		s.handleError(w, r, err, "Failed to call Service A", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	// Log Service A response details
	s.logger.Info("Received response from Service A",
		slog.String("request_id", requestID),
		slog.Int("status_code", resp.StatusCode),
	)

	if resp.StatusCode != http.StatusOK {
		s.handleError(w, r,
			fmt.Errorf("service A returned status %d", resp.StatusCode),
			"Service A request failed",
			resp.StatusCode,
		)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.handleError(w, r, err, "Failed to read Service A response", http.StatusInternalServerError)
		return
	}

	var serviceAResp ServiceAResponse
	if err := json.Unmarshal(body, &serviceAResp); err != nil {
		s.handleError(w, r, err, "Failed to parse Service A response", http.StatusInternalServerError)
		return
	}

	// Create combined response
	response := Response{
		ServiceAMessage: serviceAResp.Message,
		ServiceBMessage: "Hello from Service B!",
		RequestID:       requestID,
		Timestamp:       time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(http.StatusOK)

	// Log response preparation
	s.logger.Info("Preparing response",
		slog.String("request_id", requestID),
		slog.String("service_a_message", serviceAResp.Message),
		slog.String("service_b_message", response.ServiceBMessage),
	)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Log encoding error with detailed context
		s.logger.Error("Failed to encode response",
			slog.String("error", err.Error()),
			slog.String("request_id", requestID),
			slog.String("client_ip", clientIP),
			slog.Duration("processing_time", time.Since(startTime)),
		)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Log successful response
	s.logger.Info("Process request completed successfully",
		slog.String("request_id", requestID),
		slog.String("client_ip", clientIP),
		slog.String("service_a_request_id", serviceAResp.RequestID),
		slog.Duration("processing_time", time.Since(startTime)),
	)
}

func (s *Server) handleError(w http.ResponseWriter, r *http.Request, err error, logMessage string, statusCode int) {
	requestID := r.Context().Value("request_id").(string)
	clientIP := r.RemoteAddr

	// Enhanced error logging with more context
	s.logger.Error(logMessage,
		slog.String("error", err.Error()),
		slog.String("request_id", requestID),
		slog.String("client_ip", clientIP),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Int("status_code", statusCode),
	)

	http.Error(w, logMessage, statusCode)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	requestID := uuid.New().String()
	startTime := time.Now()

	// Log health check request
	s.logger.Info("Health check received",
		slog.String("request_id", requestID),
		slog.String("client_ip", r.RemoteAddr),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
	)

	// Perform basic health checks
	status := map[string]string{
		"status":      "healthy",
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"request_id":  requestID,
		"server_port": s.config.Port,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)

	// Log health check response
	s.logger.Info("Health check completed",
		slog.String("request_id", requestID),
		slog.Duration("processing_time", time.Since(startTime)),
	)
}

func (s *Server) Start() error {
	// Log server start with configuration details
	s.logger.Info("Initializing server",
		slog.String("port", s.config.Port),
		slog.String("log_level", s.config.LogLevel.String()),
		slog.String("service_a_url", s.config.ServiceAURL),
		slog.Bool("service_a_auth_configured", s.config.ServiceAAuthKey != ""),
	)

	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.logger.Error("Server startup failed",
			slog.String("error", err.Error()),
			slog.String("port", s.config.Port),
		)
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	// Log graceful shutdown initiation
	s.logger.Info("Initiating graceful shutdown",
		slog.String("timeout", ctx.Value("timeout").(string)),
	)

	if err := s.server.Shutdown(ctx); err != nil {
		// Log shutdown errors
		s.logger.Error("Graceful shutdown failed",
			slog.String("error", err.Error()),
		)
		return err
	}

	s.logger.Info("Server shutdown completed successfully")
	return nil
}

// maskAuthKey masks the authentication key for logging
func maskAuthKey(key string) string {
	if len(key) > 4 {
		return key[:2] + "****" + key[len(key)-2:]
	}
	return "****"
}

func main() {
	config := &Config{
		Port:            os.Getenv("PORT"),
		LogLevel:        slog.LevelInfo,
		ServiceAURL:     os.Getenv("SERVICE_A_URL"),
		ServiceAAuthKey: os.Getenv("SERVICE_A_AUTH_KEY"),
	}

	if config.Port == "" {
		config.Port = "8081"
		// Log default port selection
		slog.Info("No port specified, using default",
			slog.String("default_port", config.Port),
		)
	}

	if config.ServiceAURL == "" {
		config.ServiceAURL = "http://service-a:8080"
		// Log default Service A URL selection
		slog.Warn("No Service A URL specified, using default",
			slog.String("default_url", config.ServiceAURL),
		)
	}

	if config.ServiceAAuthKey == "" {
		config.ServiceAAuthKey = "default-secret-key"
		// Log security warning about default auth key
		slog.Warn("No authentication key provided, using default. This is NOT recommended for production!",
			slog.String("default_key", maskAuthKey(config.ServiceAAuthKey)),
		)
	}

	server := NewServer(config)

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			server.logger.Error("Server start error",
				slog.String("error", err.Error()),
				slog.String("port", config.Port),
			)
			os.Exit(1)
		}
	}()

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ctx = context.WithValue(ctx, "timeout", "10s")

	if err := server.Shutdown(ctx); err != nil {
		server.logger.Error("Server shutdown error",
			slog.String("error", err.Error()),
			slog.String("timeout", "10s"),
		)
	}

	server.logger.Info("Service B shutdown complete")
}
