package handlers

import (
	"log"
	"net/http"
	"strings"
)

func LoggingHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lw := &loggingResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(lw, r)
		log.Printf("%s %s %d %d", r.Method, r.URL.Path, lw.status, lw.size)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (w *loggingResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *loggingResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.size += n
	return n, err
}

func ProxyIPHeadersHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			parts := strings.SplitN(fwd, ",", 2)
			r.RemoteAddr = strings.TrimSpace(parts[0])
		} else if realIP := r.Header.Get("X-Real-Ip"); realIP != "" {
			r.RemoteAddr = strings.TrimSpace(realIP)
		}
		next.ServeHTTP(w, r)
	})
}
