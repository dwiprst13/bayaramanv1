package worker

import (
	"context"
	"log"
	"time"

	shippingSvc "github.com/prast13/bayaraman/internal/service/shipping"
)

// StartShipmentSyncWorker periodically polls the shipping aggregator
// for status updates on active shipments that haven't been updated via webhook.
func StartShipmentSyncWorker(shippingService shippingSvc.ShippingService) {
	ticker := time.NewTicker(30 * time.Minute)
	for {
		<-ticker.C
		log.Println("[SHIPMENT_SYNC_WORKER] Syncing active shipments...")

		ctx := context.Background()
		if err := shippingService.SyncActiveShipments(ctx); err != nil {
			log.Printf("[SHIPMENT_SYNC_WORKER ERROR] %v\n", err)
		}
	}
}
