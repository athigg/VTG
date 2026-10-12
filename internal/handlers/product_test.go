package handlers

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"

	"vtg-store/internal/middleware"
	"vtg-store/internal/models"
)

func handlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "handlers.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Product{}); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func TestEditProductHidesAnotherUsersListing(t *testing.T) {
	db := handlerTestDB(t)
	owner := models.User{Name: "Owner", Email: "owner@example.test", PasswordHash: "hash", Role: models.RoleUser, Active: true}
	viewer := models.User{Name: "Viewer", Email: "viewer@example.test", PasswordHash: "hash", Role: models.RoleUser, Active: true}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&viewer).Error; err != nil {
		t.Fatal(err)
	}
	product := models.Product{UserID: owner.ID, Title: "Test table", Description: "A lovingly restored table.", PriceCents: 1200, Category: "Furniture", Condition: "Good"}
	if err := db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	store := sessions.NewCookieStore([]byte("01234567890123456789012345678901"))
	session := sessions.NewSession(store, "vtg-session")
	session.Values["user_id"] = viewer.ID
	initialRequest := httptest.NewRequest("GET", "/", nil)
	initialResponse := httptest.NewRecorder()
	if err := store.Save(initialRequest, initialResponse, session); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/products/"+strconv.Itoa(int(product.ID))+"/edit", nil)
	request.SetPathValue("id", strconv.Itoa(int(product.ID)))
	request.AddCookie(initialResponse.Result().Cookies()[0])
	response := httptest.NewRecorder()
	app := New(db, store, t.TempDir(), false)
	middleware.LoadUser(store, db, httpHandler(app.EditProduct)).ServeHTTP(response, request)
	if response.Code != 404 {
		t.Fatalf("status = %d, want 404", response.Code)
	}
	var unchanged models.Product
	if err := db.First(&unchanged, product.ID).Error; err != nil {
		t.Fatal(err)
	}
	if unchanged.Title != product.Title {
		t.Fatalf("product title changed to %q", unchanged.Title)
	}
}

type httpHandler func(http.ResponseWriter, *http.Request)

func (handler httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) { handler(w, r) }

type testImageFile struct{ *bytes.Reader }

func (testImageFile) Close() error { return nil }

func TestStoreImageAcceptsPNGAndRejectsInvalidContent(t *testing.T) {
	var encoded bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 180, A: 255})
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	app := New(nil, nil, directory, false)
	name, err := app.storeImage(testImageFile{bytes.NewReader(encoded.Bytes())}, &multipart.FileHeader{Filename: "item.png", Size: int64(encoded.Len())})
	if err != nil {
		t.Fatalf("store valid png: %v", err)
	}
	if _, err := os.Stat(filepath.Join(directory, filepath.Base(name))); err != nil {
		t.Fatalf("stored image missing: %v", err)
	}
	_, err = app.storeImage(testImageFile{bytes.NewReader([]byte("not an image"))}, &multipart.FileHeader{Filename: "item.png", Size: 12})
	if err == nil {
		t.Fatal("expected invalid image content to be rejected")
	}
}
