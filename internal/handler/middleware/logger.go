package middleware

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type LogResponseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (w *LogResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *LogResponseWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	w.size += n
	return n, err
}

func Logger(logger *logrus.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		return func(next http.Handler) http.Handler { return next }
	}

	logger.SetLevel(logrus.InfoLevel)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			lw := &LogResponseWriter{ResponseWriter: w}
			next.ServeHTTP(lw, r)

			if lw.status == 0 {
				lw.status = http.StatusOK
			}

			logger.Infof(
				"request method=%s uri=%s duration=%s status=%d size=%d",
				r.Method, r.RequestURI, time.Since(start).String(), lw.status, lw.size,
			)
		})
	}
}