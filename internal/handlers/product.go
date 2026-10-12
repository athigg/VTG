package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"vtg-store/internal/models"
)

var categories = []string{"Furniture", "Clothing", "Electronics", "Homeware", "Collectibles", "Other"}
var conditions = []string{"Like new", "Very good", "Good", "Well loved"}

func (a *App) Products(w http.ResponseWriter, r *http.Request) {
	query := a.DB.Preload("User").Order("created_at DESC")
	if category := strings.TrimSpace(r.URL.Query().Get("category")); category != "" {
		query = query.Where("category = ?", category)
	}
	if search := strings.TrimSpace(r.URL.Query().Get("q")); search != "" {
		like := "%" + strings.ReplaceAll(search, "%", "") + "%"
		query = query.Where("title LIKE ? OR description LIKE ?", like, like)
	}
	var products []models.Product
	if err := query.Limit(100).Find(&products).Error; err != nil {
		http.Error(w, "ไม่สามารถโหลดรายการสินค้าได้", http.StatusInternalServerError)
		return
	}
	a.render(w, r, "products/index.html", PageData{Title: "สินค้าแนะนำ", Products: products, Categories: categories, Query: r.URL.Query().Get("q"), Category: r.URL.Query().Get("category")})
}

func (a *App) NewProduct(w http.ResponseWriter, r *http.Request) {
	a.render(w, r, "products/create.html", PageData{Title: "ลงขายสินค้า", Categories: categories})
}

func (a *App) CreateProduct(w http.ResponseWriter, r *http.Request) {
	product, err := productFromForm(w, r)
	if err != nil {
		a.render(w, r, "products/create.html", PageData{Title: "ลงขายสินค้า", Error: err.Error(), Categories: categories})
		return
	}
	imagePath, err := a.saveUpload(w, r, "image")
	if err != nil {
		a.render(w, r, "products/create.html", PageData{Title: "ลงขายสินค้า", Error: err.Error(), Categories: categories})
		return
	}
	product.UserID = a.currentUserID(r)
	product.ImagePath = imagePath
	if err := a.DB.Create(&product).Error; err != nil {
		http.Error(w, "ไม่สามารถบันทึกสินค้าได้", http.StatusInternalServerError)
		return
	}
	a.setFlash(w, r, "ลงประกาศขายสินค้าเรียบร้อยแล้ว")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) EditProduct(w http.ResponseWriter, r *http.Request) {
	product, ok := a.ownedProduct(w, r)
	if !ok {
		return
	}
	a.render(w, r, "products/edit.html", PageData{Title: "แก้ไขประกาศขาย", Product: &product, Categories: categories})
}

func (a *App) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	product, ok := a.ownedProduct(w, r)
	if !ok {
		return
	}
	updated, err := productFromForm(w, r)
	if err != nil {
		a.render(w, r, "products/edit.html", PageData{Title: "แก้ไขประกาศขาย", Product: &product, Error: err.Error(), Categories: categories})
		return
	}
	imagePath, err := a.saveUpload(w, r, "image")
	if err != nil {
		a.render(w, r, "products/edit.html", PageData{Title: "แก้ไขประกาศขาย", Product: &product, Error: err.Error(), Categories: categories})
		return
	}
	if imagePath != "" {
		a.removeUpload(product.ImagePath)
		product.ImagePath = imagePath
	}
	product.Title, product.Description = updated.Title, updated.Description
	product.PriceCents, product.Category, product.Condition = updated.PriceCents, updated.Category, updated.Condition
	if err := a.DB.Save(&product).Error; err != nil {
		http.Error(w, "ไม่สามารถแก้ไขสินค้าได้", http.StatusInternalServerError)
		return
	}
	a.setFlash(w, r, "บันทึกการแก้ไขประกาศเรียบร้อยแล้ว")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	product, ok := a.ownedProduct(w, r)
	if !ok {
		return
	}
	if err := a.DB.Delete(&product).Error; err != nil {
		http.Error(w, "ไม่สามารถลบสินค้าได้", http.StatusInternalServerError)
		return
	}
	a.removeUpload(product.ImagePath)
	a.setFlash(w, r, "ลบประกาศขายเรียบร้อยแล้ว")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) ownedProduct(w http.ResponseWriter, r *http.Request) (models.Product, bool) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil || id == 0 {
		http.NotFound(w, r)
		return models.Product{}, false
	}
	var product models.Product
	err = a.DB.First(&product, "id = ? AND user_id = ?", id, a.currentUserID(r)).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.NotFound(w, r)
		return models.Product{}, false
	}
	if err != nil {
		http.Error(w, "ไม่สามารถโหลดสินค้าได้", http.StatusInternalServerError)
		return models.Product{}, false
	}
	return product, true
}

func productFromForm(w http.ResponseWriter, r *http.Request) (models.Product, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 6<<20)
	if err := r.ParseMultipartForm(6 << 20); err != nil && !errors.Is(err, http.ErrNotMultipart) {
		return models.Product{}, errors.New("แบบฟอร์มหรือรูปภาพมีขนาดใหญ่เกินไป")
	}
	title := strings.TrimSpace(r.FormValue("title"))
	description := strings.TrimSpace(r.FormValue("description"))
	category := strings.TrimSpace(r.FormValue("category"))
	condition := strings.TrimSpace(r.FormValue("condition"))
	price, err := strconv.ParseFloat(strings.TrimSpace(r.FormValue("price")), 64)
	if len(title) < 3 || len(title) > 120 || len(description) < 10 || len(description) > 2000 || !contains(categories, category) || !contains(conditions, condition) || err != nil || price <= 0 || price > 1000000 {
		return models.Product{}, errors.New("โปรดระบุชื่อและรายละเอียดสินค้า เลือกหมวดหมู่กับสภาพสินค้า และกำหนดราคาระหว่าง 0.01 ถึง 1,000,000 บาท")
	}
	return models.Product{Title: title, Description: description, PriceCents: int64(price*100 + 0.5), Category: category, Condition: condition}, nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (a *App) saveUpload(w http.ResponseWriter, r *http.Request, field string) (string, error) {
	if r.MultipartForm == nil {
		if err := r.ParseMultipartForm(6 << 20); err != nil {
			if errors.Is(err, http.ErrNotMultipart) || errors.Is(err, http.ErrMissingBoundary) {
				return "", nil
			}
			return "", errors.New("ไฟล์รูปภาพมีขนาดใหญ่เกินไปหรือไม่ถูกต้อง")
		}
	}
	file, header, err := r.FormFile(field)
	if errors.Is(err, http.ErrMissingFile) {
		return "", nil
	}
	if err != nil {
		return "", errors.New("ไม่สามารถอ่านไฟล์รูปภาพได้")
	}
	defer file.Close()
	return a.storeImage(file, header)
}

func (a *App) storeImage(file multipart.File, header *multipart.FileHeader) (string, error) {
	if header.Size > 5<<20 {
		return "", errors.New("รูปภาพต้องมีขนาดไม่เกิน 5 MB")
	}
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", errors.New("ไม่สามารถอ่านไฟล์รูปภาพได้")
	}
	contentType := http.DetectContentType(buffer[:n])
	ext := ""
	switch contentType {
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	default:
		return "", errors.New("โปรดอัปโหลดรูปภาพ JPEG หรือ PNG")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", errors.New("ไม่สามารถตรวจสอบไฟล์รูปภาพได้")
	}
	config, _, err := image.DecodeConfig(file)
	if err != nil || config.Width < 1 || config.Height < 1 || config.Width > 8000 || config.Height > 8000 || int64(config.Width)*int64(config.Height) > 32_000_000 {
		return "", errors.New("ไฟล์ที่อัปโหลดไม่ใช่รูปภาพที่ถูกต้อง")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", errors.New("ไม่สามารถบันทึกไฟล์รูปภาพได้")
	}
	if err := os.MkdirAll(a.UploadDir, 0750); err != nil {
		return "", fmt.Errorf("ไม่สามารถบันทึกรูปภาพได้: %w", err)
	}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", errors.New("ไม่สามารถสร้างชื่อไฟล์รูปภาพได้")
	}
	name := hex.EncodeToString(random) + ext
	destination, err := os.OpenFile(filepath.Join(a.UploadDir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0640)
	if err != nil {
		return "", errors.New("ไม่สามารถบันทึกไฟล์รูปภาพได้")
	}
	if _, err := io.Copy(destination, io.LimitReader(file, 5<<20+1)); err != nil {
		_ = destination.Close()
		_ = os.Remove(destination.Name())
		return "", errors.New("ไม่สามารถบันทึกไฟล์รูปภาพได้")
	}
	if err := destination.Close(); err != nil {
		_ = os.Remove(destination.Name())
		return "", errors.New("ไม่สามารถบันทึกไฟล์รูปภาพให้เสร็จสมบูรณ์ได้")
	}
	return "/static/uploads/" + name, nil
}

func (a *App) removeUpload(path string) {
	if !strings.HasPrefix(path, "/static/uploads/") {
		return
	}
	name := filepath.Base(path)
	if name != "." && name != string(filepath.Separator) {
		_ = os.Remove(filepath.Join(a.UploadDir, name))
	}
}
