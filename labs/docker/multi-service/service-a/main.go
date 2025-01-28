package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
)

type Config struct {
	Port     string
	LogLevel slog.Level
	AuthKey  string
}

type Response struct {
	Message   string    `json:"message"`
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
}

type Server struct {
	config *Config
	logger *slog.Logger
	server *http.Server
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
	}

	mux.HandleFunc("/greet", s.authMiddleware(s.handleGreet))
	mux.HandleFunc("/health", s.handleHealth)

	return s
}

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.New().String()
		ctx := context.WithValue(r.Context(), "request_id", requestID)
		r = r.WithContext(ctx)

		authKey := r.Header.Get("X-Auth-Key")
		clientIP := r.RemoteAddr

		// Log authentication attempt
		s.logger.Info("Authentication attempt",
			slog.String("request_id", requestID),
			slog.String("client_ip", clientIP),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)

		if subtle.ConstantTimeCompare([]byte(authKey), []byte(s.config.AuthKey)) != 1 {
			// Enhanced unauthorized access logging
			s.logger.Warn("Authentication failed",
				slog.String("request_id", requestID),
				slog.String("client_ip", clientIP),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("auth_key_provided", maskAuthKey(authKey)),
			)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Log successful authentication
		s.logger.Info("Authentication successful",
			slog.String("request_id", requestID),
			slog.String("client_ip", clientIP),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)

		next.ServeHTTP(w, r)
	}
}

// maskAuthKey masks the authentication key for logging
func maskAuthKey(key string) string {
	if len(key) > 4 {
		return key[:2] + "****" + key[len(key)-2:]
	}
	return "****"
}

func (s *Server) handleGreet(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id").(string)
	clientIP := r.RemoteAddr

	// Log incoming request details
	s.logger.Info("Processing greeting request",
		slog.String("request_id", requestID),
		slog.String("client_ip", clientIP),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
	)

	response := Response{
		Message:   "Hello from Service A!",
		RequestID: requestID,
		Timestamp: time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(http.StatusOK)

	startTime := time.Now()
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
	s.logger.Info("Greeting request processed successfully",
		slog.String("request_id", requestID),
		slog.String("client_ip", clientIP),
		slog.Duration("processing_time", time.Since(startTime)),
		slog.String("response_message", response.Message),
	)
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
		slog.Bool("auth_configured", s.config.AuthKey != ""),
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

func main() {
	config := &Config{
		Port:     os.Getenv("PORT"),
		LogLevel: slog.LevelInfo,
		AuthKey:  os.Getenv("AUTH_KEY"),
	}

	if config.Port == "" {
		config.Port = "8080"
		// Log default port selection
		slog.Info("No port specified, using default",
			slog.String("default_port", config.Port),
		)
	}

	if config.AuthKey == "" {
		config.AuthKey = "default-secret-key"
		// Log security warning about default auth key
		slog.Warn("No authentication key provided, using default. This is NOT recommended for production!",
			slog.String("default_key", maskAuthKey(config.AuthKey)),
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

	server.logger.Info("Service A shutdown complete")
}
