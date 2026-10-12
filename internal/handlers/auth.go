package handlers

import (
	"net/http"
	"strings"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"

	"vtg-store/internal/models"
)

func (a *App) LoginPage(w http.ResponseWriter, r *http.Request) {
	data := PageData{Title: "เข้าสู่ระบบ"}
	if a.DemoLogins {
		switch r.URL.Query().Get("demo") {
		case "admin":
			data.DemoEmail, data.DemoPassword = "admin@vtg.local", "VintageAdmin123!"
		case "user":
			data.DemoEmail, data.DemoPassword = "demo@vtg.local", "VintageUser123!"
		}
	}
	a.render(w, r, "auth/login.html", data)
}

func (a *App) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.render(w, r, "auth/login.html", PageData{Title: "เข้าสู่ระบบ", Error: "ข้อมูลในแบบฟอร์มไม่ถูกต้อง โปรดตรวจสอบแล้วลองอีกครั้ง"})
		return
	}
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	password := r.FormValue("password")
	var user models.User
	if err := a.DB.Where("email = ?", email).First(&user).Error; err != nil || !user.Active || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		a.render(w, r, "auth/login.html", PageData{Title: "เข้าสู่ระบบ", Error: "อีเมลหรือรหัสผ่านไม่ถูกต้อง"})
		return
	}
	session, err := a.Sessions.Get(r, "vtg-session")
	if err != nil {
		http.Error(w, "ไม่สามารถเริ่มต้นเซสชันได้", http.StatusInternalServerError)
		return
	}
	session.Values["user_id"] = user.ID
	if err := a.Sessions.Save(r, w, session); err != nil {
		http.Error(w, "ไม่สามารถเริ่มต้นเซสชันได้", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) RegisterPage(w http.ResponseWriter, r *http.Request) {
	a.render(w, r, "auth/register.html", PageData{Title: "สมัครสมาชิก"})
}

func (a *App) Register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.render(w, r, "auth/register.html", PageData{Title: "สมัครสมาชิก", Error: "ข้อมูลในแบบฟอร์มไม่ถูกต้อง โปรดตรวจสอบแล้วลองอีกครั้ง"})
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	password := r.FormValue("password")
	if len(name) < 2 || len(name) > 100 || !strings.Contains(email, "@") || len(email) > 255 || len(password) < 10 || len(password) > 72 {
		a.render(w, r, "auth/register.html", PageData{Title: "สมัครสมาชิก", Error: "โปรดระบุชื่อ อีเมลที่ถูกต้อง และรหัสผ่านความยาว 10 ถึง 72 ตัวอักษร"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "ไม่สามารถสร้างบัญชีได้", http.StatusInternalServerError)
		return
	}
	user := models.User{Name: name, Email: email, PasswordHash: string(hash), Role: models.RoleUser, Active: true}
	if err := a.DB.Create(&user).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") || strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			a.render(w, r, "auth/register.html", PageData{Title: "สมัครสมาชิก", Error: "อีเมลนี้มีบัญชีผู้ใช้อยู่แล้ว"})
			return
		}
		http.Error(w, "ไม่สามารถสร้างบัญชีได้", http.StatusInternalServerError)
		return
	}
	session, err := a.Sessions.Get(r, "vtg-session")
	if err == nil {
		session.Values["user_id"] = user.ID
		if a.Sessions.Save(r, w, session) == nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}
	a.render(w, r, "auth/login.html", PageData{Title: "เข้าสู่ระบบ", Flash: "สร้างบัญชีเรียบร้อยแล้ว โปรดเข้าสู่ระบบ"})
}

func (a *App) Logout(w http.ResponseWriter, r *http.Request) {
	session, err := a.Sessions.Get(r, "vtg-session")
	if err == nil {
		session.Values = make(map[interface{}]interface{})
		session.Options = &sessions.Options{Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode}
		_ = a.Sessions.Save(r, w, session)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
