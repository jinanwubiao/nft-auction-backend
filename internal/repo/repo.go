package repo

import (
	"nft-auction-backend/internal/model"

	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.SyncCheckpoint{}, &model.Nft{}, &model.Auction{},
		&model.Bid{}, &model.PlatformMetric{},
	)
}
