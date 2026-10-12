package router

import (
	"net/http"

	"github.com/gorilla/sessions"
	"gorm.io/gorm"

	"vtg-store/internal/handlers"
	"vtg-store/internal/middleware"
)

func New(db *gorm.DB, store sessions.Store, csrfKey []byte, secureCookies bool, uploadDir string, demoLogins bool) http.Handler {
	app := handlers.New(db, store, uploadDir, demoLogins)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", app.Products)
	mux.HandleFunc("GET /login", app.LoginPage)
	mux.HandleFunc("POST /login", app.Login)
	mux.HandleFunc("GET /register", app.RegisterPage)
	mux.HandleFunc("POST /register", app.Register)
	mux.Handle("POST /logout", middleware.RequireAuth(http.HandlerFunc(app.Logout)))
	mux.Handle("GET /products/new", middleware.RequireAuth(http.HandlerFunc(app.NewProduct)))
	mux.Handle("POST /products", middleware.RequireAuth(http.HandlerFunc(app.CreateProduct)))
	mux.Handle("GET /products/{id}/edit", middleware.RequireAuth(http.HandlerFunc(app.EditProduct)))
	mux.Handle("POST /products/{id}", middleware.RequireAuth(http.HandlerFunc(app.UpdateProduct)))
	mux.Handle("POST /products/{id}/delete", middleware.RequireAuth(http.HandlerFunc(app.DeleteProduct)))
	mux.Handle("GET /profile", middleware.RequireAuth(http.HandlerFunc(app.EditProfile)))
	mux.Handle("POST /profile", middleware.RequireAuth(http.HandlerFunc(app.UpdateProfile)))
	mux.Handle("GET /admin", middleware.RequireAdmin(http.HandlerFunc(app.AdminDashboard)))
	mux.Handle("POST /admin/users/{id}/role", middleware.RequireAdmin(http.HandlerFunc(app.UpdateUserRole)))
	mux.Handle("POST /admin/users/{id}/status", middleware.RequireAdmin(http.HandlerFunc(app.UpdateUserStatus)))
	mux.Handle("POST /admin/products/{id}/delete", middleware.RequireAdmin(http.HandlerFunc(app.AdminDeleteProduct)))
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	secured := middleware.CSRF(csrfKey, secureCookies, mux)
	secured = middleware.SecurityHeaders(secureCookies, secured)
	return middleware.LoadUser(store, db, secured)
}
