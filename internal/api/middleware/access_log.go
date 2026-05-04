package middleware

import (
	"log/slog"
	"net/http"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
)

// AccessLog logs one slog record per request, similar to a classic web-server
// access log. Fields: method, path, query, status, bytes, dur_ms, remote, ua,
// request_id, referer.
//
// Status >= 500 logs at WARN; everything else at INFO.
func AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		dur := time.Since(start)
		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}

		attrs := []slog.Attr{
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("query", r.URL.RawQuery),
			slog.Int("status", status),
			slog.Int("bytes", ww.BytesWritten()),
			slog.Int64("dur_ms", dur.Milliseconds()),
			slog.String("remote", r.RemoteAddr),
			slog.String("ua", r.UserAgent()),
			slog.String("request_id", chimw.GetReqID(r.Context())),
		}
		if ref := r.Referer(); ref != "" {
			attrs = append(attrs, slog.String("referer", ref))
		}

		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelWarn
		}
		slog.LogAttrs(r.Context(), level, "http", attrs...)
	})
}
