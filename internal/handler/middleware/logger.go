package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type LogResponseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (w *LogResponseWriter) WriteHeader(code int) {
	if w.status != 0 {
		return
	}
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *LogResponseWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(p)
	w.size += n
	return n, err
}

func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			lw := &LogResponseWriter{ResponseWriter: w}
			next.ServeHTTP(lw, r)

			if lw.status == 0 {
				lw.status = http.StatusOK
			}

			logger.Info("request",
				"method", r.Method,
				"uri", r.RequestURI,
				"duration", time.Since(start).String(),
				"status", lw.status,
				"size", lw.size,
			)
		})
	}
}
