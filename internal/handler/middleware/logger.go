package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// responseData stores response status and payload size for access logging.
type responseData struct {
	status int
	size   int
}

// loggingResponseWriter captures response metadata while delegating writes to the original writer.
type loggingResponseWriter struct {
	http.ResponseWriter
	responseData *responseData
	wroteHeader  bool
}

// WriteHeader records the status code and writes it once to the underlying response.
func (w *loggingResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}

	w.responseData.status = statusCode
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(statusCode)
}

// Write records the number of bytes written and sends data to the underlying response.
func (w *loggingResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	size, err := w.ResponseWriter.Write(b)
	w.responseData.size += size

	return size, err
}

// Logger logs request method, URI, status, size and duration.
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			data := &responseData{status: http.StatusOK}
			lw := &loggingResponseWriter{
				ResponseWriter: w,
				responseData:   data,
			}

			next.ServeHTTP(lw, r)

			logger.Info(
				"request",
				"method", r.Method,
				"uri", r.RequestURI,
				"status", data.status,
				"size", data.size,
				"duration", time.Since(start),
			)
		})
	}
}
