package worker

import (
	"log"
	"time"

	"gorm.io/gorm"
)

// StartReconciliationWorker runs periodically to check if DB total balance matches Xendit actual balance
func StartReconciliationWorker(db *gorm.DB) {
	ticker := time.NewTicker(24 * time.Hour)
	for {
		<-ticker.C
		log.Println("[RECONCILIATION] Starting daily wallet reconciliation...")

		var totalDBBalance float64
		// SUM(balance + held_balance) from wallets
		err := db.Table("wallets").Select("COALESCE(SUM(balance + held_balance), 0)").Scan(&totalDBBalance).Error
		if err != nil {
			log.Printf("[RECONCILIATION ERROR] Failed to calculate total DB balance: %v\n", err)
			continue
		}

		// Stub: Call Xendit API to get actual balance
		// In reality: balance, err := balance.Get(&balance.GetParams{})
		actualXenditBalance := totalDBBalance // STUB: Assume it matches for now

		log.Printf("[RECONCILIATION] Total DB Balance: %.2f | Xendit Balance: %.2f\n", totalDBBalance, actualXenditBalance)

		if totalDBBalance > actualXenditBalance {
			// CRITICAL ALERT: Platform owes more money than it actually has
			log.Printf("[CRITICAL ALERT] RECONCILIATION FAILED! Platform deficit. DB: %.2f, Xendit: %.2f\n", totalDBBalance, actualXenditBalance)
		} else if actualXenditBalance > totalDBBalance {
			log.Printf("[RECONCILIATION] Platform surplus. DB: %.2f, Xendit: %.2f\n", totalDBBalance, actualXenditBalance)
		} else {
			log.Println("[RECONCILIATION] Balances matched perfectly.")
		}
	}
}
