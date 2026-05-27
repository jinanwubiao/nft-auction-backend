package model

import (
	"database/sql/driver"
	"fmt"
	"time"
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

// 1. SyncPointer 区块同步指针表
// 作用：记录每条链上每个合约当前同步到的最高区块号，支持多节点部署的行级分布式锁（FOR UPDATE）。
type SyncPointer struct {
	ID              uint64   `gorm:"primaryKey;autoIncrement;column:id"`
	ChainID         int      `gorm:"uniqueIndex:uq_sync_chain_contract;not null;column:chain_id"`
	ContractAddress string   `gorm:"uniqueIndex:uq_sync_chain_contract;type:varchar(42);not null;column:contract_address"`
	LastSyncedBlock uint64   `gorm:"not null;default:0;column:last_synced_block"`
	UpdatedAt       JSONTime `gorm:"autoUpdateTime;column:updated_at"`
}

func (SyncPointer) TableName() string { return "sync_pointers" }

// 2. BlockchainEvent 原始事件流水账表
// 作用：第一阶段存证层。拉取的所有原始日志直接写入此表。通过唯一索引防重，提供数据重放（Replay）和防分叉回滚（Reorg）的底座。
type BlockchainEvent struct {
	ID              uint64   `gorm:"primaryKey;autoIncrement;column:id"`
	ChainID         int      `gorm:"uniqueIndex:uq_events_chain_tx_log;index:idx_events_chain_block;not null;column:chain_id"`
	BlockNumber     uint64   `gorm:"index:idx_events_chain_block;not null;column:block_number"`
	TxHash          string   `gorm:"uniqueIndex:uq_events_chain_tx_log;type:varchar(66);not null;column:tx_hash"`
	LogIndex        uint     `gorm:"uniqueIndex:uq_events_chain_tx_log;not null;column:log_index"` // 复合唯一索引拦截重复日志
	EventName       string   `gorm:"type:varchar(50);not null;column:event_name"`
	ContractAddress string   `gorm:"type:varchar(42);not null;column:contract_address"`
	RawData         string   `gorm:"type:text;not null;column:raw_data"` // 存储原始 Log JSON 字符串
	CreatedAt       JSONTime `gorm:"autoCreateTime;column:created_at"`
}

func (BlockchainEvent) TableName() string { return "blockchain_events" }

// 3. Auction 拍卖项目业务主表
// 作用：核心业务表，提供前端拍卖列表的高效过滤、分页与排序。引入了多币种和 USD 影子字段补零优化。
type Auction struct {
	ID              uint64 `gorm:"primaryKey;autoIncrement;column:id"`
	ChainID         int    `gorm:"uniqueIndex:uq_auctions_chain_auction;index:idx_auctions_chain_status;index:idx_auctions_chain_end_time;index:idx_auctions_chain_nft_token;not null;column:chain_id"`
	AuctionId       uint64 `gorm:"uniqueIndex:uq_auctions_chain_auction;not null;column:auction_id"` // 对齐事件 auctionId
	ContractAddress string `gorm:"type:varchar(42);not null;column:contract_address"`
	NftContract     string `gorm:"type:varchar(42);index:idx_auctions_chain_nft_token;not null;column:nft_contract"` // 对齐事件 nft
	TokenId         string `gorm:"type:varchar(78);index:idx_auctions_chain_nft_token;not null;column:token_id"`     // 对齐事件 tokenId
	Seller          string `gorm:"type:varchar(42);not null;column:seller"`                                          // 对齐事件 seller

	// ---- 对齐 AuctionCreated 的起拍 USD 最低价 ----
	MinUsdValue    string `gorm:"type:varchar(78);not null;column:min_usd_value"`                                               // 对应事件 minUsdValue (1e18大数)
	MinUsdValuePad string `gorm:"type:varchar(30);index:idx_auctions_min_usd_pad;not null;default:'';column:min_usd_value_pad"` // 用于起拍价排序的影子字段

	// ---- 对齐 BidPlaced & AuctionSettled 的动态最高出价数据 ----
	PayToken         string `gorm:"type:varchar(42);not null;default:'';column:pay_token"`                                      // 对应事件 token
	HighestBid       string `gorm:"type:varchar(78);not null;default:'0';column:highest_bid"`                                   // 对应事件 rawAmount
	HighestBidder    string `gorm:"type:varchar(42);default:null;column:highest_bidder"`                                        // 对应事件 bidder 或 winner
	HighestBidUSD    string `gorm:"type:varchar(78);not null;default:'0';column:highest_bid_usd"`                               // 对应事件 usdAmount (1e18大数)
	HighestBidUSDPad string `gorm:"type:varchar(30);index:idx_auctions_usd_pad;not null;default:'';column:highest_bid_usd_pad"` // 极致性能排序影子字段

	StartTime JSONTime `gorm:"not null;column:start_time"`                                              // 对应事件 startTime
	EndTime   JSONTime `gorm:"index:idx_auctions_chain_end_time;not null;column:end_time"`              // 对应事件 endTime
	Status    string   `gorm:"type:varchar(20);index:idx_auctions_chain_status;not null;column:status"` // CREATED, ACTIVE, CANCELLED, SETTLED
	CreatedAt JSONTime `gorm:"autoCreateTime;column:created_at"`
	UpdatedAt JSONTime `gorm:"autoUpdateTime;column:updated_at"`

	NftMetadata *NftMetadataCache `gorm:"foreignKey:ChainID,NftContract,TokenId;references:ChainID,ContractAddress,TokenID"`
}

func (Auction) TableName() string { return "auctions" }

// 4. BidHistory 出价历史记录表
// 作用：流水账表，记录所有成功的链上出价。同样引入了出价时的 USD 价值对齐，用于未来平台总交易额审计。
type BidHistory struct {
	ID          uint64   `gorm:"primaryKey;autoIncrement;column:id"`
	ChainID     int      `gorm:"uniqueIndex:uq_bids_chain_tx_log;index:idx_bids_auction_amount;not null;column:chain_id"`
	AuctionId   uint64   `gorm:"index:idx_bids_auction_amount,priority:1;not null;column:auction_id"` // 对应事件 auctionId
	Bidder      string   `gorm:"type:varchar(42);not null;column:bidder"`                             // 对应事件 bidder
	PayToken    string   `gorm:"type:varchar(42);not null;column:pay_token"`                          // 对应事件 token
	Amount      string   `gorm:"type:varchar(78);not null;column:amount"`                             // 对应事件 rawAmount
	AmountUSD   string   `gorm:"type:varchar(78);not null;default:'0';column:amount_usd"`             // 对应事件 usdAmount
	TxHash      string   `gorm:"uniqueIndex:uq_bids_chain_tx_log;type:varchar(66);not null;column:tx_hash"`
	LogIndex    uint     `gorm:"uniqueIndex:uq_bids_chain_tx_log;not null;column:log_index"`
	BlockNumber uint64   `gorm:"not null;column:block_number"`
	BidTime     JSONTime `gorm:"not null;column:bid_time"`
	CreatedAt   JSONTime `gorm:"autoCreateTime;column:created_at"`
}

func (BidHistory) TableName() string { return "bid_histories" }

// 5. WithdrawHistory 资金/资产退回历史记录表
// 作用：单独追踪竞拍结束后，未中签用户调用 withdraw() 取回退款或卖家取回收益的事件与时间，实现闭环。
type WithdrawHistory struct {
	ID              uint64   `gorm:"primaryKey;autoIncrement;column:id"`
	ChainID         int      `gorm:"uniqueIndex:uq_withdraw_chain_tx_log;not null;column:chain_id"`
	ContractAddress string   `gorm:"type:varchar(42);not null;column:contract_address"`
	UserAddress     string   `gorm:"type:varchar(42);index:idx_withdraw_user;not null;column:user_address"` // 对应事件 user
	PayToken        string   `gorm:"type:varchar(42);not null;column:pay_token"`                            // 对应事件 token
	Amount          string   `gorm:"type:varchar(78);not null;column:amount"`                               // 对应事件 amount
	TxHash          string   `gorm:"uniqueIndex:uq_withdraw_chain_tx_log;type:varchar(66);not null;column:tx_hash"`
	LogIndex        uint     `gorm:"uniqueIndex:uq_withdraw_chain_tx_log;not null;column:log_index"`
	BlockNumber     uint64   `gorm:"not null;column:block_number"`
	WithdrawTime    JSONTime `gorm:"not null;column:withdraw_time"`
	CreatedAt       JSONTime `gorm:"autoCreateTime;column:created_at"`
}

func (WithdrawHistory) TableName() string { return "withdraw_histories" }

// 6. FailedEvent 异常事件暂存表
// 作用：容错和重试。第二阶段由于业务逻辑导致处理失败的日志丢入该表，不阻塞整个区块同步。
type FailedEvent struct {
	ID           uint64   `gorm:"primaryKey;autoIncrement;column:id"`
	ChainID      int      `gorm:"index:idx_failed_events_chain_status;not null;column:chain_id"`
	BlockNumber  uint64   `gorm:"not null;column:block_number"`
	TxHash       string   `gorm:"type:varchar(66);not null;column:tx_hash"`
	EventName    string   `gorm:"type:varchar(50);not null;column:event_name"`
	RawData      string   `gorm:"type:text;not null;column:raw_data"`
	ErrorMessage string   `gorm:"type:text;column:error_message"`
	RetryCount   int      `gorm:"not null;default:0;column:retry_count"`
	Status       string   `gorm:"type:varchar(20);index:idx_failed_events_chain_status;not null;default:'PENDING';column:status"` // PENDING, FIXED, FAILED
	CreatedAt    JSONTime `gorm:"autoCreateTime;column:created_at"`
	UpdatedAt    JSONTime `gorm:"autoUpdateTime;column:updated_at"`
}

func (FailedEvent) TableName() string { return "failed_events" }

// 7. NftMetadataCache 单件 NFT 元数据缓存表 (Token 级粒度)
// 作用：异步缓存单件 NFT 的独一无二的属性与图片。更新频率极低（终生基本不变）。
type NftMetadataCache struct {
	ID              uint64   `gorm:"primaryKey;autoIncrement;column:id"`
	ChainID         int      `gorm:"uniqueIndex:uq_nft_token;not null;column:chain_id"`
	ContractAddress string   `gorm:"type:varchar(42);uniqueIndex:uq_nft_token;not null;column:contract_address"` // NFT 合约地址
	TokenID         string   `gorm:"type:varchar(78);uniqueIndex:uq_nft_token;not null;column:token_id"`
	Name            string   `gorm:"type:varchar(255);column:name"`
	ImageURL        string   `gorm:"type:text;column:image_url"`
	Attributes      string   `gorm:"type:text;column:attributes"` // 存储 Traits 的 JSON 字符串
	UpdatedAt       JSONTime `gorm:"autoUpdateTime;column:updated_at"`

	// ---- GORM 代码层逻辑关联 ----
	// 顺着该 NFT 所在的合约，逻辑联动查询该 NFT 整个系列在二级市场上的整体地板价
	FloorInfo *NftFloorPrice `gorm:"foreignKey:ChainID,ContractAddress;references:ChainID,ContractAddress"`
}

func (NftMetadataCache) TableName() string { return "nft_metadata_caches" }

// 8. NftFloorPrice 三方系列地板价缓存表 (Collection 级粒度)
// 作用：解耦存储整个 NFT 系列的动态挂单地板价。高频每 10 分钟利用定时器请求 OpenSea/Blur 覆盖刷新。
type NftFloorPrice struct {
	ID              uint64   `gorm:"primaryKey;autoIncrement;column:id"`
	ChainID         int      `gorm:"uniqueIndex:uq_floor_chain_contract;not null;column:chain_id"`
	ContractAddress string   `gorm:"type:varchar(42);uniqueIndex:uq_floor_chain_contract;not null;column:contract_address"`
	FloorPrice      float64  `gorm:"type:numeric(20,4);not null;column:floor_price"` // 保留 4 位小数的地板价
	Currency        string   `gorm:"type:varchar(10);not null;default:'ETH';column:currency"`
	UpdatedAt       JSONTime `gorm:"autoUpdateTime;column:updated_at"`
}

func (NftFloorPrice) TableName() string { return "nft_floor_prices" }

// 9. UserBalance 用户合约内沉淀余额表 (新增账本表)
// 作用：核心记账层。实时映射并缓存用户在合约公共资金池中沉淀的、可提现的未中签退款或者拍卖资金。
type UserBalance struct {
	ID               uint64   `gorm:"primaryKey;autoIncrement;column:id"`
	ChainID          int      `gorm:"uniqueIndex:uq_user_token;not null;column:chain_id"`
	ContractAddress  string   `gorm:"uniqueIndex:uq_user_token;type:varchar(42);not null;column:contract_address"`
	UserAddress      string   `gorm:"uniqueIndex:uq_user_token;type:varchar(42);not null;column:user_address"` // 用户钱包地址
	PayToken         string   `gorm:"uniqueIndex:uq_user_token;type:varchar(42);not null;column:pay_token"`    // 代币合约地址
	AvailableBalance string   `gorm:"type:varchar(78);not null;default:'0';column:available_balance"`          // 当前合约内剩余可提现金额 (Wei)
	UpdatedAt        JSONTime `gorm:"autoUpdateTime;column:updated_at"`
}

func (UserBalance) TableName() string { return "user_balances" }
