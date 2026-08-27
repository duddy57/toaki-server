package shared

import (
	"net/http"
	"uuid"

	"github.com/alexedwards/scs/v2"
	"gorm.io/gorm"
)

func AuthMiddleware(sessionManager *scs.SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !sessionManager.Exists(r.Context(), "user_id") {
				EncodeJson(w, r, http.StatusUnauthorized, map[string]any{
					"message": "must be logged in",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func ForTenant(tenantID uuid.UUID) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("tenant_id = ?", tenantID)
	}
}
