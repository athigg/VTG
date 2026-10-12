package middleware

import (
	"context"
	"net/http"

	"github.com/gorilla/sessions"
	"gorm.io/gorm"

	"vtg-store/internal/models"
)

type contextKey string

const userKey contextKey = "current-user"

func CurrentUser(r *http.Request) *models.User {
	user, _ := r.Context().Value(userKey).(*models.User)
	return user
}

func LoadUser(store sessions.Store, db *gorm.DB, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "vtg-session")
		if err == nil {
			if userID, ok := session.Values["user_id"].(uint); ok && userID > 0 {
				var user models.User
				if db.First(&user, userID).Error == nil && user.Active {
					r = r.WithContext(context.WithValue(r.Context(), userKey, &user))
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if CurrentUser(r) == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := CurrentUser(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if user.Role != models.RoleAdmin {
			http.Error(w, "คุณไม่มีสิทธิ์เข้าถึงหน้านี้", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
