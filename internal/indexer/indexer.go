package indexer

import (
	"context"
	"math/big"
	"nft-auction-backend/internal/config"
	"nft-auction-backend/internal/logger"
	"nft-auction-backend/internal/service/svc"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Indexer struct {
	ec       *ethclient.Client
	svcCtx   *svc.ServerCtx
	cfg      *config.EthConf
	abi      abi.ABI
	addr     common.Address
	stopChan chan struct{}
}

var testLastBlockNum uint64 = 10910860

func NewIndexer(cfg *config.EthConf, svcCtx *svc.ServerCtx) (*Indexer, error) {
	ec, err := ethclient.Dial(cfg.RpcUrl)
	if err != nil {
		return nil, err
	}
	abi := mustParseABI()
	addr := common.HexToAddress(cfg.AuctionAddr)
	return &Indexer{
		ec:       ec,
		cfg:      cfg,
		svcCtx:   svcCtx,
		abi:      abi,
		addr:     addr,
		stopChan: make(chan struct{}),
	}, nil
}

func (i *Indexer) Start() {
	logger.S().Info("Indexer service started")
	ticker := time.NewTicker(12 * time.Second) // Sepolia/Ethereum 约 12 秒一个块
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := i.syncBlocks(context.Background()); err != nil {
				logger.S().Errorf("Sync blocks error: %v", err)
			}
		case <-i.stopChan:
			logger.S().Info("Indexer service stopped smoothly")
			return
		}
	}
}

func (i *Indexer) Stop() {
	close(i.stopChan)
}

func (i *Indexer) syncBlocks(ctx context.Context) error {
	// 1. 从数据库读取上次同步到的区块
	// 实际代码中建议在 service 层封装：i.svcCtx.SyncService.GetLastSyncedBlock(ctx, i.contractName)
	lastSyncedBlock, err := i.getLastSyncedBlockFromDB(ctx)
	if err != nil {
		return err
	}

	// 2. 从 RPC 节点获取当前最新区块高度
	chainLatestBlock, err := i.getChainLatestBlock(ctx)
	if err != nil {
		return err
	}

	// 3. 计算安全的结束区块（减去延迟，规避轻微回滚）
	if chainLatestBlock <= i.cfg.BLockDelay {
		return nil
	}
	safeLatestBlock := chainLatestBlock - i.cfg.BLockDelay

	startBlock := lastSyncedBlock + 1
	if startBlock > safeLatestBlock {
		// 已经追到最新高度，无需同步
		logger.S().Info("已经追到最新高度，无需同步")
		return nil
	}

	// 4. 计算本次同步的结束位置（防止单次跨度过大被 RPC 限流）
	endBlock := startBlock + i.cfg.BatchSize - 1
	if endBlock > safeLatestBlock {
		endBlock = safeLatestBlock
	}
	logger.S().Infof("chainLatestBlock:%d", chainLatestBlock)
	logger.S().Infof("safeLatestBlock:%d", safeLatestBlock)
	logger.S().Infof("Syncing blocks from %d to %d", startBlock, endBlock)

	// 5. 调用 RPC 的 getLogs 并处理业务逻辑
	if err := i.processLogsInBlockRange(ctx, startBlock, endBlock); err != nil {
		return err
	}

	// 6. 成功后，更新断点进度
	if err := i.updateSyncedBlockInDB(ctx, endBlock); err != nil {
		return err
	}

	return nil
}

func (i *Indexer) getLastSyncedBlockFromDB(ctx context.Context) (uint64, error) {
	return testLastBlockNum, nil
}

func (i *Indexer) getChainLatestBlock(ctx context.Context) (uint64, error) {
	latest, err := i.ec.BlockNumber(ctx)
	if err != nil {
		return 0, err
	}
	return latest, nil
}

func (i *Indexer) updateSyncedBlockInDB(ctx context.Context, blockNum uint64) error {
	testLastBlockNum = blockNum
	return nil
}

func (i *Indexer) processLogsInBlockRange(ctx context.Context, start, end uint64) error {
	// 1. 构建 ethereum.FilterQuery { FromBlock: start, ToBlock: end, Addresses: [...] }
	topics := []common.Hash{
		i.abi.Events["AuctionCreated"].ID,
		i.abi.Events["BidPlaced"].ID,
		i.abi.Events["AuctionSettled"].ID,
		i.abi.Events["AuctionCancelled"].ID,
	}
	q := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(start)),
		ToBlock:   big.NewInt(int64(end)),
		Addresses: []common.Address{i.addr},
		Topics:    [][]common.Hash{topics},
	}
	// 2. 调用 logs, err := i.svcCtx.EthClient.FilterLogs(ctx, query)
	logs, err := i.ec.FilterLogs(ctx, q)
	if err != nil {
		return err
	}
	// 3. 循环 logs，利用 ABI Unpack 解析事件
	for _, lg := range logs {
		if err := i.HandleLog(ctx, lg); err != nil {
			logger.S().Infof(
				"[indexer] handle log error block=%d tx=%s err=%v",
				lg.BlockNumber,
				lg.TxHash.Hex(),
				err,
			)
		}
	}
	// 4. 调用业务 Service 将解析后的数据写入 Auction 表（建议放在同一个 DB 事务中）
	return nil
}

func (i *Indexer) HandleLog(ctx context.Context, lg types.Log) error {
	logger.S().Infof(
		"HandleLog: tx=%s topic0=%s block=%d",
		lg.TxHash.Hex(),
		lg.Topics[0].Hex(),
		lg.BlockNumber,
	)

	if len(lg.Topics) == 0 {
		return nil
	}

	event, err := i.abi.EventByID(lg.Topics[0])
	if err != nil {
		return nil // 不是我们关心的事件
	}

	switch event.Name {
	case "AuctionCreated":
		return i.handleAuctionCreated(ctx, lg)
	case "BidPlaced":
		return i.handleBidPlaced(ctx, lg)
	case "AuctionSettled":
		return i.handleAuctionSettled(ctx, lg)
	case "AuctionCancelled":
		return i.handleAuctionCancelled(ctx, lg)
	default:
		return nil
	}
}

func (i *Indexer) handleAuctionCreated(ctx context.Context, lg types.Log) error {
	// 1. 解 data（非 indexed）
	var data struct {
		TokenId     *big.Int
		MinUsdValue *big.Int
		StartTime   *big.Int
		EndTime     *big.Int
	}
	if err := i.abi.UnpackIntoInterface(&data, "AuctionCreated", lg.Data); err != nil {
		logger.S().Errorf("unpack AuctionCreated data failed: %w", err)
		return err
	}
	// 2. topics：indexed 参数
	if len(lg.Topics) < 4 {
		logger.S().Errorf("unpack AuctionCreated data failed: %w", len(lg.Topics))
		return nil
	}
	auctionId := new(big.Int).SetBytes(lg.Topics[1].Bytes()).Uint64()
	seller := common.BytesToAddress(lg.Topics[2].Bytes())
	nft := common.BytesToAddress(lg.Topics[3].Bytes())
	// 3. 防御性校验
	if data.TokenId == nil || data.EndTime == nil || data.MinUsdValue == nil {
		logger.S().Errorf("unpack AuctionCreated data failed: %w", len(lg.Topics))
		return nil
	}
	logger.S().Info("parse AuctionCreated event:")
	logger.S().Infof("auctionId:%d", auctionId)
	logger.S().Infof("seller:%d", seller)
	logger.S().Infof("nft:%d", nft)
	logger.S().Infof("TokenId:%d", data.TokenId)
	logger.S().Infof("MinUsdValue:%d", data.MinUsdValue)
	logger.S().Infof("StartTime:%d", data.StartTime)
	logger.S().Infof("EndTime:%d", data.EndTime)
	return nil
}

func (i *Indexer) handleBidPlaced(ctx context.Context, lg types.Log) error {
	// 1. 解非 indexed data
	var data struct {
		Token     common.Address
		RawAmount *big.Int
		UsdAmount *big.Int
	}

	if err := i.abi.UnpackIntoInterface(&data, "BidPlaced", lg.Data); err != nil {
		logger.S().Errorf("unpack BidPlaced failed: %w", err)
		return nil
	}

	// 2. 解 indexed topics
	if len(lg.Topics) < 3 {
		logger.S().Errorf("invalid BidPlaced topics len=%d", len(lg.Topics))
		return nil
	}

	auctionId := new(big.Int).SetBytes(lg.Topics[1].Bytes()).Uint64()
	bidder := common.BytesToAddress(lg.Topics[2].Bytes())

	// 3. 防御性校验
	if data.RawAmount == nil || data.UsdAmount == nil {
		logger.S().Errorf("BidPlaced data has nil field: %+v", data)
		return nil
	}
	logger.S().Infof("auctionId:%d", auctionId)
	logger.S().Infof("bidder:%d", bidder)
	logger.S().Infof("Token:%d", data.Token)
	logger.S().Infof("RawAmount:%d", data.RawAmount)
	logger.S().Infof("UsdAmount:%d", data.UsdAmount)
	return nil
}

func (i *Indexer) handleAuctionSettled(ctx context.Context, lg types.Log) error {
	// 1. 解非 indexed data
	var data struct {
		Amount *big.Int
		Token  common.Address
	}

	if err := i.abi.UnpackIntoInterface(&data, "AuctionSettled", lg.Data); err != nil {
		logger.S().Errorf("unpack AuctionSettled failed: %w", err)
		return nil
	}

	// 2. 解 indexed topics
	if len(lg.Topics) < 3 {
		logger.S().Errorf("invalid BidPlaced topics len=%d", len(lg.Topics))
		return nil
	}

	auctionId := new(big.Int).SetBytes(lg.Topics[1].Bytes()).Uint64()
	winner := common.BytesToAddress(lg.Topics[2].Bytes())

	// 3. 防御性校验
	if data.Amount == nil {
		logger.S().Errorf("AuctionSettled data has nil field: %+v", data)
		return nil
	}
	logger.S().Infof("auctionId:%d", auctionId)
	logger.S().Infof("winner:%d", winner)
	logger.S().Infof("Token:%d", data.Token)
	logger.S().Infof("Amount:%d", data.Amount)
	logger.S().Infof("Token:%s", data.Token)
	return nil
}

func (i *Indexer) handleAuctionCancelled(ctx context.Context, lg types.Log) error {
	// 1. 解 indexed topics
	if len(lg.Topics) < 2 {
		logger.S().Errorf("invalid AuctionCancelled topics len=%d", len(lg.Topics))
		return nil
	}

	auctionId := new(big.Int).SetBytes(lg.Topics[1].Bytes()).Uint64()

	logger.S().Infof("auctionId:%d", auctionId)
	return nil
}
