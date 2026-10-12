package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"vtg-store/internal/models"
)

const (
	DefaultAdminEmail    = "admin@vtg.local"
	DefaultAdminPassword = "VintageAdmin123!"
	DefaultDemoEmail     = "demo@vtg.local"
	DefaultDemoPassword  = "VintageUser123!"
)

func OpenDatabase(path, appEnv string) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	// หากเป็น MySQL DSN ให้เชื่อมต่อ MySQL
	if strings.Contains(path, "@tcp(") {
		db, err = gorm.Open(mysql.Open(path), &gorm.Config{})
	} else {
		// หากเป็นเส้นทางไฟล์ ให้ใช้ SQLite สำหรับการทดสอบ
		if path != ":memory:" && filepath.Dir(path) != "." {
			if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
				return nil, fmt.Errorf("create database directory: %w", err)
			}
		}

		db, err = gorm.Open(sqlite.Open(path), &gorm.Config{})
	}

	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.AutoMigrate(&models.User{}, &models.Product{}); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	if err := SeedDatabase(db, appEnv); err != nil {
		return nil, err
	}

	return db, nil
}

func SeedDatabase(db *gorm.DB, appEnv string) error {
	if !isDevelopment(appEnv) {
		return nil
	}
	if err := seedUser(db, "VTG Demo", DefaultDemoEmail, DefaultDemoPassword, models.RoleUser); err != nil {
		return err
	}
	adminEmail := envOr("DEMO_ADMIN_EMAIL", DefaultAdminEmail)
	adminPassword := envOr("DEMO_ADMIN_PASSWORD", DefaultAdminPassword)
	if adminEmail == DefaultAdminEmail && adminPassword == DefaultAdminPassword {
		if err := seedUser(db, "VTG Demo Admin", adminEmail, adminPassword, models.RoleAdmin); err != nil {
			return err
		}
	} else if adminEmail == "" || adminPassword == "" {
		return errors.New("DEMO_ADMIN_EMAIL and DEMO_ADMIN_PASSWORD must both be set")
	} else {
		if err := seedUser(db, "VTG Demo Admin", adminEmail, adminPassword, models.RoleAdmin); err != nil {
			return err
		}
	}
	return seedProducts(db)
}

func isDevelopment(appEnv string) bool {
	return strings.EqualFold(strings.TrimSpace(appEnv), "development") || strings.TrimSpace(appEnv) == ""
}

func seedUser(db *gorm.DB, name, email, password, role string) error {
	var existing models.User
	err := db.Where("email = ?", strings.ToLower(email)).First(&existing).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("find seed user: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash seed password: %w", err)
	}
	user := models.User{Name: name, Email: strings.ToLower(email), PasswordHash: string(hash), Role: role}
	if err := db.Create(&user).Error; err != nil {
		return fmt.Errorf("create seed user: %w", err)
	}
	return nil
}

func seedProducts(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.Product{}).Count(&count).Error; err != nil {
		return fmt.Errorf("count products: %w", err)
	}
	if count > 0 {
		return nil
	}
	var seller models.User
	if err := db.Where("email = ?", DefaultDemoEmail).First(&seller).Error; err != nil {
		return fmt.Errorf("find demo seller: %w", err)
	}
	products := []models.Product{
		{UserID: seller.ID, Title: "1970s teak side table", Description: "Warm grain and a simple, enduring profile. Lightly restored and ready for a new home.", PriceCents: 14500, Category: "Furniture", Condition: "Good", ImagePath: "https://images.unsplash.com/photo-1538688525198-9b88f6f53126?auto=format&fit=crop&w=900&q=85"},
		{UserID: seller.ID, Title: "Vintage film camera", Description: "A tactile 35mm camera with a clean lens and tested shutter.", PriceCents: 8900, Category: "Electronics", Condition: "Good", ImagePath: "https://images.unsplash.com/photo-1516035069371-29a1b244cc32?auto=format&fit=crop&w=900&q=85"},
		{UserID: seller.ID, Title: "Hand-thrown ceramic vase", Description: "A one-of-a-kind studio piece in a soft moss glaze.", PriceCents: 4200, Category: "Homeware", Condition: "Very good", ImagePath: "https://images.unsplash.com/photo-1578749556568-bc2c40e68b61?auto=format&fit=crop&w=900&q=85"},
	}
	if err := db.Create(&products).Error; err != nil {
		return fmt.Errorf("create sample products: %w", err)
	}
	return nil
}

func envOr(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return strings.TrimSpace(value)
	}
	return fallback
}
