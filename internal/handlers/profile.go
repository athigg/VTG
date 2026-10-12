package handlers

import (
	"net/http"
	"strings"

	"vtg-store/internal/middleware"
)

func (a *App) EditProfile(w http.ResponseWriter, r *http.Request) {
	user := middleware.CurrentUser(r)
	a.render(w, r, "profile/edit.html", PageData{Title: "โปรไฟล์ของฉัน", Profile: user})
}

func (a *App) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	user := middleware.CurrentUser(r)
	if err := r.ParseForm(); err != nil {
		a.render(w, r, "profile/edit.html", PageData{Title: "โปรไฟล์ของฉัน", Profile: user, Error: "ข้อมูลในแบบฟอร์มไม่ถูกต้อง โปรดตรวจสอบแล้วลองอีกครั้ง"})
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	bio := strings.TrimSpace(r.FormValue("bio"))
	if len(name) < 2 || len(name) > 100 || len(bio) > 500 {
		a.render(w, r, "profile/edit.html", PageData{Title: "โปรไฟล์ของฉัน", Profile: user, Error: "ชื่อต้องมีความยาว 2 ถึง 100 ตัวอักษร และข้อความแนะนำตัวต้องไม่เกิน 500 ตัวอักษร"})
		return
	}
	if err := a.DB.Model(user).Updates(map[string]interface{}{"name": name, "bio": bio}).Error; err != nil {
		http.Error(w, "ไม่สามารถแก้ไขโปรไฟล์ได้", http.StatusInternalServerError)
		return
	}
	a.setFlash(w, r, "บันทึกโปรไฟล์เรียบร้อยแล้ว")
	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}
