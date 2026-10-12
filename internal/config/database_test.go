package config

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"vtg-store/internal/models"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{})
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
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
	return db
}

func TestSeedDatabaseDevelopmentCreatesDemoAccountsAndProducts(t *testing.T) {
	t.Setenv("DEMO_ADMIN_EMAIL", DefaultAdminEmail)
	t.Setenv("DEMO_ADMIN_PASSWORD", DefaultAdminPassword)
	db := testDB(t)
	if err := SeedDatabase(db, "development"); err != nil {
		t.Fatal(err)
	}
	var admin models.User
	if err := db.Where("email = ?", DefaultAdminEmail).First(&admin).Error; err != nil {
		t.Fatalf("expected demo admin: %v", err)
	}
	if admin.Role != models.RoleAdmin {
		t.Fatalf("admin role = %q, want %q", admin.Role, models.RoleAdmin)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(DefaultAdminPassword)); err != nil {
		t.Fatalf("demo admin password was not hashed correctly: %v", err)
	}
	var productCount int64
	if err := db.Model(&models.Product{}).Count(&productCount).Error; err != nil {
		t.Fatal(err)
	}
	if productCount == 0 {
		t.Fatal("expected seeded products")
	}
}

func TestSeedDatabaseProductionDoesNotCreateDemoAccounts(t *testing.T) {
	db := testDB(t)
	if err := SeedDatabase(db, "production"); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("production seed created %d users, want none", count)
	}
}
