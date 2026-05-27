package middleware

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type LogResponseWriter struct {
	http.ResponseWriter
	status      int
	size        int
	wroteHeader bool
}

func (w *LogResponseWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}

	w.wroteHeader = true
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *LogResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	n, err := w.ResponseWriter.Write(p)
	w.size += n
	return n, err
}

func Logger(logger *logrus.Logger) func(http.Handler) http.Handler {
	logger.SetLevel(logrus.InfoLevel)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			lw := &LogResponseWriter{ResponseWriter: w}
			next.ServeHTTP(lw, r)

			if lw.status == 0 {
				lw.status = http.StatusOK
			}

			logger.WithFields(logrus.Fields{
				"method":   r.Method,
				"uri":      r.RequestURI,
				"duration": time.Since(start).String(),
				"status":   lw.status,
				"size":     lw.size,
			}).Info("request")
		})
	}
}
