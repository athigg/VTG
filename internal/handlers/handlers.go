package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"

	"vtg-store/internal/middleware"
	"vtg-store/internal/models"
	"vtg-store/templates"
)

type App struct {
	DB         *gorm.DB
	Sessions   sessions.Store
	UploadDir  string
	DemoLogins bool
}

type PageData struct {
	Title        string
	User         *models.User
	Products     []models.Product
	Product      *models.Product
	Users        []models.User
	Profile      *models.User
	Query        string
	Category     string
	Flash        string
	Error        string
	DemoEmail    string
	DemoPassword string
	DemoLogins   bool
	CSRFField    template.HTML
	Categories   []string
}

func New(db *gorm.DB, store sessions.Store, uploadDir string, demoLogins bool) *App {
	return &App{DB: db, Sessions: store, UploadDir: uploadDir, DemoLogins: demoLogins}
}

func (a *App) render(w http.ResponseWriter, r *http.Request, page string, data PageData) {
	data.User = middleware.CurrentUser(r)
	data.CSRFField = csrf.TemplateField(r)
	data.DemoLogins = a.DemoLogins
	session, err := a.Sessions.Get(r, "vtg-session")
	if err == nil {
		if flash, ok := session.Values["flash"].(string); ok {
			data.Flash = flash
			delete(session.Values, "flash")
			_ = a.Sessions.Save(r, w, session)
		}
	}
	t, err := template.New("base").Funcs(template.FuncMap{
		"money":   func(cents int64) string { return fmt.Sprintf("฿%.2f", float64(cents)/100) },
		"decimal": func(cents int64) string { return fmt.Sprintf("%.2f", float64(cents)/100) },
		"categoryLabel": func(value string) string {
			labels := map[string]string{"Furniture": "เฟอร์นิเจอร์", "Clothing": "เสื้อผ้าและเครื่องแต่งกาย", "Electronics": "เครื่องใช้ไฟฟ้า", "Homeware": "ของใช้ในบ้าน", "Collectibles": "ของสะสม", "Other": "อื่น ๆ"}
			if label, ok := labels[value]; ok {
				return label
			}
			return value
		},
		"conditionLabel": func(value string) string {
			labels := map[string]string{"Like new": "เหมือนใหม่", "Very good": "ดีมาก", "Good": "ดี", "Well loved": "ผ่านการใช้งาน"}
			if label, ok := labels[value]; ok {
				return label
			}
			return value
		},
	}).ParseFS(templates.Files, "layouts/base.html", page)
	if err != nil {
		log.Printf("parse template %s: %v", page, err)
		http.Error(w, "ไม่สามารถแสดงหน้านี้ได้ในขณะนี้", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "ไม่สามารถแสดงหน้านี้ได้ในขณะนี้", http.StatusInternalServerError)
	}
}

func (a *App) setFlash(w http.ResponseWriter, r *http.Request, message string) {
	session, err := a.Sessions.Get(r, "vtg-session")
	if err != nil {
		return
	}
	session.Values["flash"] = message
	_ = a.Sessions.Save(r, w, session)
}

func (a *App) currentUserID(r *http.Request) uint {
	user := middleware.CurrentUser(r)
	if user == nil {
		return 0
	}
	return user.ID
}
