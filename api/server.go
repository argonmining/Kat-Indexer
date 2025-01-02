// api/server.go
package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"kasplex-executor/api/handlers"
	"kasplex-executor/api/middleware"
)

type Server struct {
	port           int
	allowedOrigins []string
	logger         *log.Logger
	server         *http.Server
}

func NewServer(port int, allowedOrigins []string) *Server {
	logger := log.New(os.Stdout, "[API] ", log.LstdFlags)

	return &Server{
		port:           port,
		allowedOrigins: allowedOrigins,
		logger:         logger,
	}
}

func (s *Server) Start(stop chan struct{}) error {
	mux := http.NewServeMux()

	// Initialize middleware
	cors := middleware.NewCorsMiddleware(s.allowedOrigins)
	rateLimiter := middleware.NewRateLimiter(100, time.Minute)
	logger := middleware.NewLogMiddleware(s.logger)

	// Debug logging
	s.logger.Printf("Starting route registration...")

	// Register snapshot endpoint first
	snapshotHandler := http.HandlerFunc(handlers.GetTokenSnapshot)
	mux.Handle("/api/v1/token/snapshot", snapshotHandler)
	s.logger.Printf("Registered: /api/v1/token/snapshot")

	// Other endpoints
	mux.HandleFunc("/api/v1/token/balances", handlers.GetTokenBalances)
	mux.HandleFunc("/api/v1/token/info", handlers.GetTokenInfo)
	mux.HandleFunc("/api/v1/address/balances", handlers.GetAddressBalances)
	mux.HandleFunc("/api/v1/token/holders", handlers.GetTokenHolders)
	mux.HandleFunc("/api/v1/token/operations", handlers.GetTokenOperations)
	mux.HandleFunc("/api/v1/tokens", handlers.GetAllTokens)
	mux.HandleFunc("/api/v1/holders/top", handlers.GetTopHolders)

	s.logger.Printf("All routes registered")

	// Apply middleware chain
	handler := logger.Handler(
		rateLimiter.Handler(
			cors.Handler(mux),
		),
	)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: handler,
	}

	// Graceful shutdown goroutine
	go func() {
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		s.logger.Printf("Shutting down API server...")
		if err := s.server.Shutdown(ctx); err != nil {
			s.logger.Printf("API server shutdown error: %v", err)
		}
	}()

	return s.server.ListenAndServe()
}
