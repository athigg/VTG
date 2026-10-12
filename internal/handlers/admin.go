package handlers

import (
	"net/http"
	"strconv"

	"vtg-store/internal/models"
)

func (a *App) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	var users []models.User
	var products []models.Product
	if err := a.DB.Order("created_at DESC").Limit(200).Find(&users).Error; err != nil {
		http.Error(w, "ไม่สามารถโหลดบัญชีผู้ใช้ได้", http.StatusInternalServerError)
		return
	}
	if err := a.DB.Preload("User").Order("created_at DESC").Limit(200).Find(&products).Error; err != nil {
		http.Error(w, "ไม่สามารถโหลดประกาศขายได้", http.StatusInternalServerError)
		return
	}
	a.render(w, r, "admin/dashboard.html", PageData{Title: "แดชบอร์ดผู้ดูแล", Users: users, Products: products})
}

func (a *App) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	user, ok := a.adminTargetUser(w, r)
	if !ok {
		return
	}
	if user.ID == a.currentUserID(r) {
		http.Error(w, "คุณไม่สามารถเปลี่ยนบทบาทผู้ดูแลของบัญชีตนเองได้", http.StatusBadRequest)
		return
	}
	role := r.FormValue("role")
	if role != models.RoleUser && role != models.RoleAdmin {
		http.Error(w, "บทบาทผู้ใช้ไม่ถูกต้อง", http.StatusBadRequest)
		return
	}
	if err := a.DB.Model(&user).Update("role", role).Error; err != nil {
		http.Error(w, "ไม่สามารถแก้ไขบทบาทผู้ใช้ได้", http.StatusInternalServerError)
		return
	}
	a.setFlash(w, r, "บันทึกบทบาทผู้ใช้เรียบร้อยแล้ว")
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (a *App) UpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	user, ok := a.adminTargetUser(w, r)
	if !ok {
		return
	}
	if user.ID == a.currentUserID(r) {
		http.Error(w, "คุณไม่สามารถระงับบัญชีของตนเองได้", http.StatusBadRequest)
		return
	}
	active := r.FormValue("active") == "true"
	if err := a.DB.Model(&user).Update("active", active).Error; err != nil {
		http.Error(w, "ไม่สามารถแก้ไขบัญชีผู้ใช้ได้", http.StatusInternalServerError)
		return
	}
	a.setFlash(w, r, "บันทึกสถานะบัญชีผู้ใช้เรียบร้อยแล้ว")
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (a *App) AdminDeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil || id == 0 {
		http.NotFound(w, r)
		return
	}
	var product models.Product
	if err := a.DB.First(&product, id).Error; err != nil {
		http.NotFound(w, r)
		return
	}
	if err := a.DB.Delete(&product).Error; err != nil {
		http.Error(w, "ไม่สามารถลบประกาศขายได้", http.StatusInternalServerError)
		return
	}
	a.removeUpload(product.ImagePath)
	a.setFlash(w, r, "ลบประกาศขายเรียบร้อยแล้ว")
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (a *App) adminTargetUser(w http.ResponseWriter, r *http.Request) (models.User, bool) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil || id == 0 {
		http.NotFound(w, r)
		return models.User{}, false
	}
	var user models.User
	if err := a.DB.First(&user, id).Error; err != nil {
		http.NotFound(w, r)
		return models.User{}, false
	}
	return user, true
}
