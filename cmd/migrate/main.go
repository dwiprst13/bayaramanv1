package main

import (
	"log"

	"github.com/prast13/bayaraman/config"
	"github.com/prast13/bayaraman/internal/model"
)

func main() {
	log.Println("Starting database migration...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := config.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	err = db.AutoMigrate(
		&model.User{},
		&model.Session{},
		&model.AuditLog{},
		&model.EscrowTransaction{},
		&model.Payment{},
		&model.Wallet{},
		&model.WalletTransaction{},
		&model.Payout{},
		&model.Message{},
	)

	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Database migration completed successfully!")
}
