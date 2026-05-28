package repo

import (
	"nft-auction-backend/internal/model"

	"gorm.io/gorm"
)

// Repo 统一持有 DB 实例
type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{
		db: db,
	}
}

func (r *Repo) AutoMigrate() error {
	return r.db.AutoMigrate(
		&model.SyncPointer{}, &model.BlockchainEvent{}, &model.Auction{},
		&model.BidHistory{}, &model.FailedEvent{}, &model.UserBalance{},
		&model.WithdrawHistory{}, &model.NftMetadataCache{}, &model.NftFloorPrice{},
	)
}
