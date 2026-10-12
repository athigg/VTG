package middleware

import (
	"net/http"

	"github.com/gorilla/csrf"
)

func SecurityHeaders(secure bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		header.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: images.unsplash.com; style-src 'self'; form-action 'self'; frame-ancestors 'none'; base-uri 'self'")
		if secure {
			header.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

func CSRF(key []byte, secure bool, next http.Handler) http.Handler {
	return csrf.Protect(key, csrf.Secure(secure), csrf.Path("/"), csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "ไม่พบโทเคน CSRF หรือโทเคนไม่ถูกต้อง", http.StatusForbidden)
	})))(next)
}
