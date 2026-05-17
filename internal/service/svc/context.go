package svc

import (
	"nft-auction-backend/internal/config"
	"nft-auction-backend/internal/logger"
	"nft-auction-backend/internal/store/gdb"

	"gorm.io/gorm"
)

type ServerCtx struct {
	C  *config.Config
	DB *gorm.DB
}

func NewServiceContext(c *config.Config) (*ServerCtx, error) {
	_, err := logger.Init(c.Log)
	if err != nil {
		return nil, err
	}
	db, err := gdb.NewDB(c.DB)
	if err != nil {
		return nil, err
	}
	return &ServerCtx{
		c,
		db,
	}, nil
}
