package router

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/gorilla/sessions"

	"vtg-store/internal/config"
)

func TestDevelopmentAdminLoginRequiresCSRFAndCanReachDashboard(t *testing.T) {
	t.Setenv("DEMO_ADMIN_EMAIL", config.DefaultAdminEmail)
	t.Setenv("DEMO_ADMIN_PASSWORD", config.DefaultAdminPassword)
	db, err := config.OpenDatabase(filepath.Join(t.TempDir(), "router.db"), "development")
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	handler := New(db, sessions.NewCookieStore([]byte("01234567890123456789012345678901")), []byte("12345678901234567890123456789012"), false, t.TempDir(), true)

	missingToken := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("email=admin%40vtg.local"))
	missingToken.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	blocked := httptest.NewRecorder()
	handler.ServeHTTP(blocked, missingToken)
	if blocked.Code != http.StatusForbidden {
		t.Fatalf("POST without CSRF token status = %d, want %d", blocked.Code, http.StatusForbidden)
	}

	loginPage := httptest.NewRecorder()
	handler.ServeHTTP(loginPage, httptest.NewRequest(http.MethodGet, "/login", nil))
	if loginPage.Code != http.StatusOK || !strings.Contains(loginPage.Body.String(), config.DefaultAdminEmail) {
		t.Fatalf("development login page missing demo account; status = %d", loginPage.Code)
	}
	match := regexp.MustCompile(`name="gorilla\.csrf\.Token" value="([^"]+)"`).FindStringSubmatch(loginPage.Body.String())
	if len(match) != 2 {
		t.Fatal("login page did not render a CSRF token")
	}
	var csrfCookie *http.Cookie
	for _, cookie := range loginPage.Result().Cookies() {
		if strings.Contains(cookie.Name, "csrf") {
			csrfCookie = cookie
		}
	}
	if csrfCookie == nil {
		t.Fatal("login page did not set a CSRF cookie")
	}
	form := url.Values{
		"email":              {config.DefaultAdminEmail},
		"password":           {config.DefaultAdminPassword},
		"gorilla.csrf.Token": {match[1]},
	}
	loginRequest := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	loginRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginRequest.AddCookie(csrfCookie)
	loginResponse := httptest.NewRecorder()
	handler.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusSeeOther {
		t.Fatalf("admin login status = %d, want %d; body=%s", loginResponse.Code, http.StatusSeeOther, loginResponse.Body.String())
	}

	dashboardRequest := httptest.NewRequest(http.MethodGet, "/admin", nil)
	for _, cookie := range loginResponse.Result().Cookies() {
		dashboardRequest.AddCookie(cookie)
	}
	dashboardResponse := httptest.NewRecorder()
	handler.ServeHTTP(dashboardResponse, dashboardRequest)
	if dashboardResponse.Code != http.StatusOK || !strings.Contains(dashboardResponse.Body.String(), "แดชบอร์ดผู้ดูแล") {
		t.Fatalf("admin dashboard status = %d, want %d", dashboardResponse.Code, http.StatusOK)
	}
}
