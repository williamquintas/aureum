package middleware

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/pkg/auth"
)

type auditLogger struct {
	pool *pgxpool.Pool
}

func NewAuditLogger(pool *pgxpool.Pool) *auditLogger {
	return &auditLogger{pool: pool}
}

func (l *auditLogger) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := auth.GetClaims(r.Context())
		userID := ""
		if claims != nil {
			userID = claims.Subject
		}

		lw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(lw, r)

		if lw.status < 400 {
			return
		}

		l.log(r.Context(), userID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent(), lw.status)
	})
}

func (l *auditLogger) log(ctx context.Context, userID, method, path, ip, ua string, status int) {
	if l.pool == nil {
		return
	}
	_, _ = l.pool.Exec(ctx,
		`INSERT INTO audit_logs (user_id, event_type, ip_address, user_agent, details, created_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())`,
		userID, method+" "+path, ip, ua,
		map[string]interface{}{"status": status, "method": method, "path": path},
	)
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
