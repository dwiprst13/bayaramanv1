package worker

import (
	"context"
	"log"
	"time"

	"github.com/prast13/bayaraman/internal/service/storage"
	"gorm.io/gorm"
)

func StartVideoCleanupWorker(db *gorm.DB, storageSvc storage.StorageService) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	log.Println("[Worker] Starting Video Cleanup Worker...")
	cleanupVideos(db, storageSvc)

	for {
		<-ticker.C
		cleanupVideos(db, storageSvc)
	}
}

func cleanupVideos(db *gorm.DB, storageSvc storage.StorageService) {
	ctx := context.Background()
	log.Println("[Worker] Running cleanup job for old videos...")

	sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour)

	var oldEscrows []struct {
		ID               string
		PackingVideoURL  string
		UnboxingVideoURL string
	}

	err := db.Table("escrow_transactions").
		Select("id, packing_video_url, unboxing_video_url").
		Where("status = ? AND updated_at < ? AND (packing_video_url != '' OR unboxing_video_url != '')", "completed", sevenDaysAgo).
		Scan(&oldEscrows).Error

	if err != nil {
		log.Printf("[Worker] Failed to query old escrows: %v", err)
		return
	}

	for _, e := range oldEscrows {
		if e.PackingVideoURL != "" {
			err := storageSvc.DeleteFile(ctx, e.PackingVideoURL)
			if err != nil {
				log.Printf("[Worker] Failed to delete packing video %s: %v", e.PackingVideoURL, err)
			} else {
				log.Printf("[Worker] Deleted packing video: %s", e.PackingVideoURL)
			}
		}

		if e.UnboxingVideoURL != "" {
			err := storageSvc.DeleteFile(ctx, e.UnboxingVideoURL)
			if err != nil {
				log.Printf("[Worker] Failed to delete unboxing video %s: %v", e.UnboxingVideoURL, err)
			} else {
				log.Printf("[Worker] Deleted unboxing video: %s", e.UnboxingVideoURL)
			}
		}

		db.Table("escrow_transactions").Where("id = ?", e.ID).Updates(map[string]interface{}{
			"packing_video_url":  "",
			"unboxing_video_url": "",
		})
	}

	log.Printf("[Worker] Video cleanup finished. Processed %d escrows.", len(oldEscrows))
}
