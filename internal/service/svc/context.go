package svc

import (
	"nft-auction-backend/internal/config"
	"nft-auction-backend/internal/logger"
	"nft-auction-backend/internal/repo"
	"nft-auction-backend/internal/store/gdb"

	"gorm.io/gorm"
)

type ServerCtx struct {
	C    *config.Config
	DB   *gorm.DB
	Repo *repo.Repo
}

func NewServiceContext(c *config.Config) (*ServerCtx, error) {
	//初始化日志
	_, err := logger.Init(c.Log)
	if err != nil {
		return nil, err
	}
	//初始化数据库连接
	db, err := gdb.NewDB(c.DB)
	if err != nil {
		return nil, err
	}
	// 初始化万能 Repo
	repository := repo.NewRepo(db)
	// 处理自动迁移
	if err := repository.AutoMigrate(); err != nil {
		return nil, err
	}
	return &ServerCtx{
		c,
		db,
		repository,
	}, nil
}
