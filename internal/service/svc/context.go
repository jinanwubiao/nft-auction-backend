package svc

import (
	"nft-auction-backend/internal/config"

	"gorm.io/gorm"
)

type ServerCtx struct {
	C  *config.Config
	DB *gorm.DB
}

func NewServiceContext(c *config.Config) (*ServerCtx, error) {
	var err error
	db, err := gdb.NewDB(&c.DB)
	if err != nil {
		return nil, err
	}
}
