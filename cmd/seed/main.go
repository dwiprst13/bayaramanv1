package main

import (
	"log"

	"github.com/google/uuid"
	"github.com/prast13/bayaraman/config"
	"github.com/prast13/bayaraman/internal/model"
	"github.com/prast13/bayaraman/pkg/hash"
)

func main() {
	log.Println("Starting database seeder...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := config.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	adminEmail := "admin@bayaraman.com"
	adminPassword := "AdminSecret123!"

	var count int64
	db.Model(&model.User{}).Where("email = ?", adminEmail).Count(&count)

	if count > 0 {
		log.Println("Admin user already exists. Skipping seeder.")
		return
	}

	hashedPassword, err := hash.HashPassword(adminPassword)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	adminUser := model.User{
		ID:              uuid.New(),
		Email:           adminEmail,
		PasswordHash:    hashedPassword,
		Role:            "admin",
		IsEmailVerified: true,
		IsPhoneVerified: true,
		KYCStatus:       "verified",
	}

	if err := db.Create(&adminUser).Error; err != nil {
		log.Fatalf("Failed to seed admin user: %v", err)
	}

	log.Println("==================================================")
	log.Println("Admin User successfully created!")
	log.Printf("Email    : %s\n", adminEmail)
	log.Printf("Password : %s\n", adminPassword)
	log.Println("==================================================")
}
