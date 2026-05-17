package model

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

type JSONTime time.Time

// 1.实现MarshalJSON:用于gin返回JSON时格式化
func (t JSONTime) MarshalJSON() ([]byte, error) {
	formatted := fmt.Sprintf("\"%s\"", time.Time(t).Format("2006-01-02 15:04:05"))
	return []byte(formatted), nil
}

// 2.实现Value:写入时转成JSONTime
func (t JSONTime) Value() (driver.Value, error) {
	tTime := time.Time(t)
	if tTime.IsZero() {
		return nil, nil
	}
	return tTime, nil
}

// 3.实现Scan:从数据库中读取时调用
func (t *JSONTime) Scan(v interface{}) error {
	if value, ok := v.(JSONTime); ok {
		*t = JSONTime(value)
		return nil
	}
	return fmt.Errorf("can not convert %v to JSONTime", v)
}

// SyncCheckpoint 记录同步进度
type SyncCheckpoint struct {
	TaskKey            string   `gorm:"primaryKey;type:varchar(100)"` // 格式: "chain_id:contract_address"
	ChainID            int      `gorm:"not null"`
	ContractAddress    string   `gorm:"type:varchar(42);not null"`
	LastProcessedBlock int64    `gorm:"not null"`
	UpdatedAt          JSONTime `gorm:"autoUpdateTime"`
}

// Nft NFT 资产缓存表
type Nft struct {
	ContractAddress string    `gorm:"primaryKey;type:varchar(42)"`
	TokenID         string    `gorm:"primaryKey;type:varchar(78)"` // 使用 string 兼容 uint256
	OwnerAddress    string    `gorm:"type:varchar(42);index;not null"`
	Name            string    `gorm:"type:varchar(255)"`
	ImageURL        string    `gorm:"type:text"`
	Metadata        string    `gorm:"type:jsonb"` // PostgreSQL 的 JSONB 格式
	LastSyncedAt    JSONTime  `gorm:"autoCreateTime"`
	Auctions        []Auction `gorm:"foreignKey:NftContract,TokenID;references:ContractAddress,TokenID"`
}

// Auction 拍卖主表
type Auction struct {
	AuctionID      int64           `gorm:"primaryKey;autoIncrement:false"`
	Seller         string          `gorm:"type:varchar(42);not null"`
	NftContract    string          `gorm:"type:varchar(42);not null"`
	TokenID        string          `gorm:"type:varchar(78);not null"`
	StartPrice     decimal.Decimal `gorm:"type:numeric(78,0);not null"`
	HighestBid     decimal.Decimal `gorm:"type:numeric(78,0);default:0"`
	HighestBidder  string          `gorm:"type:varchar(42)"`
	StartTime      JSONTime        `gorm:"not null"`
	EndTime        JSONTime        `gorm:"index;not null"`
	Status         string          `gorm:"type:varchar(20);index;default:'active'"` // active, ended, canceled
	CreatedAtBlock int64           `gorm:"not null"`
	TxHash         string          `gorm:"type:varchar(66);not null"`
	UpdatedAt      JSONTime        `gorm:"autoUpdateTime"`

	// 关联
	Nft  Nft   `gorm:"foreignKey:NftContract,TokenID;references:ContractAddress,TokenID"`
	Bids []Bid `gorm:"foreignKey:AuctionID"`
}

// Bid 出价记录表
type Bid struct {
	ID          uint            `gorm:"primaryKey"`
	AuctionID   int64           `gorm:"not null;index"`
	Bidder      string          `gorm:"type:varchar(42);not null"`
	Amount      decimal.Decimal `gorm:"type:numeric(78,0);not null"`
	TxHash      string          `gorm:"type:varchar(66);not null;uniqueIndex:idx_tx_log"`
	LogIndex    int             `gorm:"not null;uniqueIndex:idx_tx_log"` // 联合唯一索引保证幂等
	BlockNumber int64           `gorm:"not null"`
	CreatedAt   JSONTime        `gorm:"autoCreateTime"`
}

// PlatformMetric 平台统计表
type PlatformMetric struct {
	MetricKey   string          `gorm:"primaryKey;type:varchar(50)"`
	MetricValue decimal.Decimal `gorm:"type:numeric(78,0);default:0"`
	UpdatedAt   JSONTime        `gorm:"autoUpdateTime"`
}
