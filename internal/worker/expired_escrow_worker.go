package worker

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"
	"github.com/prast13/bayaraman/internal/service/config"
)

// StartExpiredEscrowWorker periodically checks for and cancels expired pending escrows
func StartExpiredEscrowWorker(db *gorm.DB, configService config.ConfigService) {
	ticker := time.NewTicker(5 * time.Minute)
	for {
		<-ticker.C
		log.Println("[EXPIRED_ESCROW_WORKER] Checking for expired escrows...")

		ctx := context.Background()
		expiryHours := configService.GetEscrowExpiryHours(ctx)
		// Tambahkan grace period 15 menit agar webhook yang terlambat masuk tidak balapan dengan worker
		cutoffTime := time.Now().Add(-time.Duration(expiryHours) * time.Hour).Add(-15 * time.Minute)

		// Update all pending escrows that are older than cutoffTime to cancelled
		result := db.Table("escrow_transactions").
			Where("status = ?", "pending").
			Where("created_at < ?", cutoffTime).
			Update("status", "cancelled")

		if result.Error != nil {
			log.Printf("[EXPIRED_ESCROW_WORKER ERROR] Failed to cancel expired escrows: %v\n", result.Error)
		} else if result.RowsAffected > 0 {
			log.Printf("[EXPIRED_ESCROW_WORKER] Successfully cancelled %d expired escrows.\n", result.RowsAffected)
		}
	}
}
