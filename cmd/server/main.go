package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/sessions"

	"vtg-store/internal/config"
	"vtg-store/internal/router"
)

func main() {
	appEnv := env("APP_ENV", "development")
	development := strings.EqualFold(appEnv, "development")
	secureCookies := !development
	sessionKey, err := secret("SESSION_KEY", "vtg-local-session-key-change-before-production-32")
	if err != nil {
		log.Fatal(err)
	}
	csrfKey, err := secret("CSRF_KEY", "vtg-local-csrf-key-change-before-production-32")
	if err != nil {
		log.Fatal(err)
	}
	databaseConfig := env("DATABASE_DSN", env("DATABASE_PATH", "data/vtg-store.db"))
	db, err := config.OpenDatabase(databaseConfig, appEnv)
	if err != nil {
		log.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	sessionStore := sessions.NewCookieStore([]byte(sessionKey))
	sessionStore.Options = &sessions.Options{
		Path: "/", MaxAge: 86400 * 7, HttpOnly: true, Secure: secureCookies, SameSite: http.SameSiteLaxMode,
	}
	addr := env("ADDR", ":8080")
	server := &http.Server{
		Addr: addr, Handler: router.New(db, sessionStore, []byte(csrfKey), secureCookies, env("UPLOAD_DIR", "static/uploads"), development),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
	}
	go func() {
		log.Printf("VTG Store listening on %s (%s)", addr, appEnv)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal(fmt.Errorf("server shutdown: %w", err))
	}
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func secret(key, developmentFallback string) (string, error) {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		if len(value) < 32 {
			return "", fmt.Errorf("%s must be at least 32 characters", key)
		}
		return value, nil
	}
	if !strings.EqualFold(env("APP_ENV", "development"), "development") {
		return "", fmt.Errorf("%s must be configured in production", key)
	}
	return developmentFallback, nil
}
