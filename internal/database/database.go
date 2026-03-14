package database

import (
	"go-entegra/internal/config"
	"go-entegra/internal/models"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(cfg *config.Config) (*gorm.DB, error) {
	logLevel := logger.Silent
	if cfg.App.Env == "development" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(cfg.DB.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, err
	}

	DB = db
	return db, nil
}

func Migrate(db *gorm.DB) error {
	log.Println("Running database migrations...")

	err := db.AutoMigrate(
		&models.Business{},
		&models.Restaurant{},
		&models.Provider{},
		&models.RestaurantProvider{},
		&models.Order{},
		&models.CreditTransaction{},
	)
	if err != nil {
		return err
	}

	// Seed default providers
	seedProviders(db)

	log.Println("Migrations completed.")
	return nil
}

func seedProviders(db *gorm.DB) {
	providers := []models.Provider{
		{Name: "Getir Yemek", Slug: "getir", Status: "active"},
		{Name: "Yemeksepeti", Slug: "yemeksepeti", Status: "active"},
		{Name: "Trendyol Yemek", Slug: "trendyol", Status: "active"},
		{Name: "Migros Yemek", Slug: "migros", Status: "active"},
	}

	for _, p := range providers {
		db.Where(models.Provider{Slug: p.Slug}).FirstOrCreate(&p)
	}
}
