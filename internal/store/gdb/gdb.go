package gdb

import (
	"fmt"
	"nft-auction-backend/internal/config"
	"nft-auction-backend/internal/logger"
	"time"

	"github.com/pkg/errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func NewDB(c *config.DBConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.User, c.Password, c.Host, c.Port, c.Database)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.NewGormZap(logger.L())})
	if err != nil {
		return nil, errors.WithMessage(err, "gdb: open database connection err")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, errors.WithMessage(err, "gdb: get database instance err")
	}
	sqlDB.SetMaxIdleConns(c.MaxIdleConns)
	sqlDB.SetMaxOpenConns(c.MaxOpenConns)
	if c.MaxConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(time.Second * time.Duration(c.MaxConnMaxLifetime))
	}
	return db, nil
}
