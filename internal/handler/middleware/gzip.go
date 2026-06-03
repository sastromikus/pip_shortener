package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// gzipResponseWriter wraps http.ResponseWriter and gzip-compresses response bodies when enabled.
type gzipResponseWriter struct {
	http.ResponseWriter

	gw          *gzip.Writer
	writer      io.Writer
	wroteHeader bool
	shouldGzip  bool
	status      int
}

// WriteHeader writes the response status and prepares gzip compression when the content type is supported.
func (w *gzipResponseWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}

	w.wroteHeader = true
	w.status = code

	ct := w.Header().Get("Content-Type")
	w.shouldGzip = shouldCompressContentType(ct)

	if w.shouldGzip {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")

		w.gw = gzip.NewWriter(w.ResponseWriter)
		w.writer = w.gw
	} else {
		w.writer = w.ResponseWriter
	}

	w.ResponseWriter.WriteHeader(code)
}

// Write writes response data through the selected plain or gzip writer.
func (w *gzipResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	return w.writer.Write(p)
}

// Close flushes and closes the gzip writer when it was created.
func (w *gzipResponseWriter) Close() error {
	if w.gw != nil {
		return w.gw.Close()
	}

	return nil
}

func shouldCompressContentType(ct string) bool {
	ct = strings.ToLower(ct)

	return strings.HasPrefix(ct, "application/json") || strings.HasPrefix(ct, "text/html") || strings.HasPrefix(ct, "text/plain")
}

// Gzip enables gzip decoding for requests and gzip encoding for supported responses.
func Gzip() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(strings.ToLower(r.Header.Get("Content-Encoding")), "gzip") {
				gr, err := gzip.NewReader(r.Body)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				defer gr.Close()
				r.Body = io.NopCloser(gr)
			}

			if !strings.Contains(strings.ToLower(r.Header.Get("Accept-Encoding")), "gzip") {
				next.ServeHTTP(w, r)
				return
			}

			gzw := &gzipResponseWriter{ResponseWriter: w}
			defer gzw.Close()

			next.ServeHTTP(gzw, r)
		})
	}
}
