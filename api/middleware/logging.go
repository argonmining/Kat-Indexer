package middleware

import (
	"log"
	"net/http"
	"time"
)

type LogMiddleware struct {
	logger *log.Logger
}

func NewLogMiddleware(logger *log.Logger) *LogMiddleware {
	return &LogMiddleware{
		logger: logger,
	}
}

func (m *LogMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom response writer to capture status code
		rw := &responseWriter{w, http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		m.logger.Printf(
			"%s %s %s %d %v",
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			rw.status,
			duration,
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
