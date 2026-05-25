package indexer

import (
	"context"
	"nft-auction-backend/internal/config"
	"nft-auction-backend/internal/logger"
	"nft-auction-backend/internal/service/svc"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

type Indexer struct {
	ec         *ethclient.Client
	svcCtx     *svc.ServerCtx
	stopChan   chan struct{}
	batchSize  uint64 // 每次 getLogs 允许的最大区块跨度（例如 2000）
	blockDelay uint64 // 区块确认延迟（例如 6 个块），防止区块回滚引发数据不一致
}

func NewIndexer(c *config.Config, svcCtx *svc.ServerCtx) (*Indexer, error) {
	ec, err := ethclient.Dial(c.EthCfg.RpcUrl)
	if err != nil {
		return nil, err
	}
	return &Indexer{
		ec:         ec,
		svcCtx:     svcCtx,
		stopChan:   make(chan struct{}),
		batchSize:  2000,
		blockDelay: 6,
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
	// 假设你通过 go-ethereum 的 client 获取：i.svcCtx.EthClient.BlockNumber(ctx)
	chainLatestBlock, err := i.getChainLatestBlock(ctx)
	if err != nil {
		return err
	}

	// 3. 计算安全的结束区块（减去延迟，规避轻微回滚）
	if chainLatestBlock <= i.blockDelay {
		return nil
	}
	safeLatestBlock := chainLatestBlock - i.blockDelay

	startBlock := lastSyncedBlock + 1
	if startBlock > safeLatestBlock {
		// 已经追到最新高度，无需同步
		return nil
	}

	// 4. 计算本次同步的结束位置（防止单次跨度过大被 RPC 限流）
	endBlock := startBlock + i.batchSize - 1
	if endBlock > safeLatestBlock {
		endBlock = safeLatestBlock
	}

	logger.S().Infof("Syncing blocks from %d to %d", startBlock, endBlock)

	// 5. 调用 RPC 的 getLogs 并处理业务逻辑
	// 具体的解析逻辑放入 i.parseAndSaveLogs(startBlock, endBlock)
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
	return 0, nil
}

func (i *Indexer) getChainLatestBlock(ctx context.Context) (uint64, error) {
	return 0, nil
}

func (i *Indexer) updateSyncedBlockInDB(ctx context.Context, blockNum uint64) error {

	return nil
}

func (i *Indexer) processLogsInBlockRange(ctx context.Context, start, end uint64) error {
	// 1. 构建 ethereum.FilterQuery { FromBlock: start, ToBlock: end, Addresses: [...] }
	// 2. 调用 logs, err := i.svcCtx.EthClient.FilterLogs(ctx, query)
	// 3. 循环 logs，利用 ABI Unpack 解析事件
	// 4. 调用业务 Service 将解析后的数据写入 Auction 表（建议放在同一个 DB 事务中）
	return nil
}
