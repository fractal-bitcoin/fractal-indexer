package parser

import (
	"fractal-indexer/constant"
	"fractal-indexer/loader"
	"fractal-indexer/logger"
	"fractal-indexer/model"
	"fractal-indexer/rdb"
	"fractal-indexer/task"
	utilsTask "fractal-indexer/task/utils"
	"fractal-indexer/utils"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
)

type Blockchain struct {
	Blocks                map[string]*model.BlockIndex     // Synced blocks
	BlocksOfChainById     map[string]struct{}              // blkid main-chain check, fetched from RPC; only contains 100 blocks before and 500 blocks after the latest locally synced height
	BlocksOfChainByHeight map[uint32]*model.BlockIndexInfo // Main-chain blocks by height
	MainChainHeight       uint32
	MainChainBlockIdHex   string
	ReorgBlockByRpc       int
	m                     sync.Mutex
}

func NewBlockchain(reorgBlock int) (bc *Blockchain, err error) {
	bc = new(Blockchain)
	bc.Blocks = make(map[string]*model.BlockIndex, 0)

	bc.ReorgBlockByRpc = reorgBlock

	return
}

// ParseLongestChain traverses blocks in two passes. It fetches headers first, then traverses blocks.
func (bc *Blockchain) ParseLongestChain(startBlockHeight, endBlockHeight uint32, nftStartNumber, nftCursedStartNumber int64) (lastBlockId []byte, lastHeight uint32, txCount int) {
	blocksReady := make(chan *model.Block, 64)

	blocksStage := make(chan *model.Block, 64)

	// Decode blocks in parallel, as the producer.
	go bc.InitLongestChainBlockByHeader(blocksReady, startBlockHeight, endBlockHeight)

	// Consume decoded blocks in order.
	go bc.ParseLongestChainBlockStart(blocksReady, blocksStage, startBlockHeight, endBlockHeight, nftStartNumber, nftCursedStartNumber)

	// Consume processed blocks in parallel.
	return bc.ParseLongestChainBlockEnd(blocksStage)
}

// InitLongestChainBlock decodes blocks as the producer.
func (bc *Blockchain) InitLongestChainBlockByHeader(blocksReady chan *model.Block, startBlockHeight, endBlockHeight uint32) {
	var wg sync.WaitGroup
	var stopped atomic.Bool

	blocksTotal := bc.MainChainHeight + 1

	if endBlockHeight == 0 {
		endBlockHeight = blocksTotal
	}

	blockLimit := make(chan struct{}, model.BlockDecodeConcurrency)

	parseBlock := func(blockInfo *model.BlockIndexInfo) bool {
		blockLimit <- struct{}{}
		if model.NeedStop.Load() || stopped.Load() {
			<-blockLimit
			return false
		}

		wg.Add(1)
		go func(blockInfo *model.BlockIndexInfo) {
			defer wg.Done()
			defer func() {
				<-blockLimit
			}()

			if model.NeedStop.Load() {
				return
			}

			rawblock, err := loader.GetRawBlock(blockInfo.HashHex)
			if err != nil {
				logger.Log.Error("get block error",
					zap.Uint32("height", blockInfo.Height),
					zap.String("blkId", blockInfo.HashHex),
					zap.Error(err))
				stopped.Store(true)
				return
			}
			if len(rawblock) < 80+9 { // block header + txn
				logger.Log.Warn("raw block too short, skip block",
					zap.Uint32("height", blockInfo.Height),
					zap.String("blkId", blockInfo.HashHex),
					zap.Int("rawBlockLen", len(rawblock)))
				loader.PutRawBlock(rawblock)
				return
			}

			block := &model.Block{
				Height:      blockInfo.Height,
				RawBlockBuf: rawblock,
			}
			InitBlock(block, rawblock)
			if blockInfo.HashHex != block.HashHex {
				logger.Log.Warn("blkId not match hash(rawblk)",
					zap.Uint32("height", blockInfo.Height),
					zap.String("blkId", blockInfo.HashHex),
					zap.String("blkHash", block.HashHex))
				loader.PutRawBlock(rawblock)
				stopped.Store(true)
				return
			}
			if blockInfo.Height > 0 {
				if prevBlock, ok := bc.BlocksOfChainByHeight[blockInfo.Height-1]; ok && block.ParentHex != prevBlock.HashHex {
					logger.Log.Error("block parent not continuous",
						zap.Uint32("height", blockInfo.Height),
						zap.String("blkId", block.HashHex),
						zap.String("parent", block.ParentHex),
						zap.String("expectedParent", prevBlock.HashHex))
					loader.PutRawBlock(rawblock)
					stopped.Store(true)
					return
				}
			}

			if txs, txSlab, inSlab, outSlab, nftSlabs, ok := NewTxs(block.Raw[80:], block.Height); ok {
				block.Txs = txs
				block.TxSlab = txSlab
				block.TxInSlab = inSlab
				block.TxOutSlab = outSlab
				block.NFTDataSlabs = nftSlabs
			} else {
				logger.Log.Fatal("blkId txs not valid",
					zap.Uint32("height", block.Height),
					zap.String("blkHash", block.HashHex))
				return
			}

			processBlock := &model.ProcessBlock{
				Height:               uint32(block.Height),
				NewUtxoDataMap:       make(map[string]*model.TxoData, block.TxCnt),
				SpentUtxoDataMap:     make(map[string]*model.TxoData, block.TxCnt),
				SpentUtxoKeysMap:     make(map[string]struct{}, block.TxCnt),
				NewInscriptions:      make([]*model.NewInscriptionInfo, 0), // order in block/mempool  nft: nftpoint/nftid
				NewEventInscriptions: make([]*model.NewInscriptionInfo, 0),
			}
			block.ParseData = processBlock
			// First analyze blocks in parallel. This can run independent preprocessing tasks within each block; different blocks run in parallel and out of order.
			task.ParseBlockParallel(block)

			if model.NeedStop.Load() {
				loader.PutRawBlock(block.RawBlockBuf)
				return
			}

			block.Raw = nil
			blocksReady <- block
		}(blockInfo)
		return true
	}

	for nextBlockHeight := startBlockHeight; nextBlockHeight < endBlockHeight; nextBlockHeight++ {
		if model.NeedStop.Load() || stopped.Load() {
			break
		}

		blockInfo, ok := bc.BlocksOfChainByHeight[nextBlockHeight]
		if !ok {
			// Exit if this is not a main-chain block.
			logger.Log.Warn("main chain block header not found, stop parsing range",
				zap.Uint32("height", nextBlockHeight),
				zap.Uint32("start", startBlockHeight),
				zap.Uint32("end", endBlockHeight),
				zap.Uint32("mainChainHeight", bc.MainChainHeight),
				zap.Int("loadedHeaders", len(bc.BlocksOfChainByHeight)))
			break
		}

		if !parseBlock(blockInfo) {
			break
		}
	}
	wg.Wait()

	close(blocksReady)
}

// ParseLongestChainBlock consumes decoded blocks in order.
func (bc *Blockchain) ParseLongestChainBlockStart(blocksReady, blocksStage chan *model.Block, startBlockHeight, maxBlockHeight uint32, nftStartNumber, nftCursedStartNumber int64) {
	defer close(blocksStage)

	if maxBlockHeight == 0 || maxBlockHeight > bc.MainChainHeight+1 {
		maxBlockHeight = bc.MainChainHeight + 1
	}
	if maxBlockHeight <= startBlockHeight {
		logger.Log.Warn("empty block parse range",
			zap.Uint32("start", startBlockHeight),
			zap.Uint32("end", maxBlockHeight),
			zap.Uint32("mainChainHeight", bc.MainChainHeight))
		return
	}

	nextBlockHeight := startBlockHeight
	blockParseBufferBlock := make([]*model.Block, maxBlockHeight-startBlockHeight)
	for block := range blocksReady {
		// Buffer the block temporarily.
		if block.Height < maxBlockHeight {
			blockParseBufferBlock[block.Height-startBlockHeight] = block
		}

		// In order.
		if block.Height != nextBlockHeight {
			continue
		}
		for nextBlockHeight < maxBlockHeight {
			block = blockParseBufferBlock[nextBlockHeight-startBlockHeight]
			if block == nil { // Check whether it is ready.
				break
			}

			if model.ShouldIndexBusiness() {
				// Then analyze blocks serially. This can run tasks that strictly require ordered processing; blocks run serially in sequence.
				// When serial execution reaches a block, all tasks for previous blocks and preprocessing tasks for this block have completed.
				task.ParseBlockSerialStart(nftStartNumber, nftCursedStartNumber, block)
				if !model.SkipMissingUTXO && model.MissingUTXO {
					return
				}
				// update inscription number
				for _, nft := range block.ParseData.NewInscriptions {
					if nft.NFTData.IsCursed {
						nftCursedStartNumber++
					} else {
						nftStartNumber++
					}
				}
				block.ParseData.NftEndNumber = nftStartNumber
				block.ParseData.NftCursedEndNumber = nftCursedStartNumber

				// block speed
				utilsTask.ParseBlockSpeed(len(block.Txs), len(model.GlobalNewUtxoDataMap), model.GlobalSpentUtxoCount,
					len(block.ParseData.NewInscriptions), block.ParseData.NftTransferCount,
					block.Height, maxBlockHeight)
			} else {
				utilsTask.ParseBlockSpeed(len(block.Txs), 0, 0, 0, 0, block.Height, maxBlockHeight)
			}

			blocksStage <- block

			nextBlockHeight++
		}
		if nextBlockHeight >= maxBlockHeight {
			break
		}
	}
	if nextBlockHeight < maxBlockHeight && !model.NeedStop.Load() && !model.MissingUTXO {
		logger.Log.Warn("block parse pipeline stopped before next sequential block",
			zap.Uint32("nextHeight", nextBlockHeight),
			zap.Uint32("start", startBlockHeight),
			zap.Uint32("end", maxBlockHeight),
			zap.Uint32("mainChainHeight", bc.MainChainHeight))
	}
}

// ParseLongestChainBlockEnd processes blocks sequentially to ensure ordered
// ClickHouse writes. Runs concurrently with the serial phase (pipeline parallelism).
func (bc *Blockchain) ParseLongestChainBlockEnd(blocksStage chan *model.Block) (lastBlockId []byte, lastHeight uint32, txCount int) {
	var wg sync.WaitGroup
	blocksLimit := make(chan struct{}, 64)
	for block := range blocksStage {
		lastBlockId = block.Hash
		lastHeight = block.Height
		txCount += int(block.TxCnt)
		blocksLimit <- struct{}{}
		wg.Add(1)
		go func(block *model.Block) {
			defer wg.Done()
			task.ParseBlockParallelEnd(block)
			<-blocksLimit
		}(block)
	}
	wg.Wait()
	return lastBlockId, lastHeight, txCount
}

// InitLatestBlockFromDB
func (bc *Blockchain) InitLatestBlockFromDB() bool {
	blkRsp, err := loader.GetLatestBlockFromDB()
	if err != nil {
		return false
	}
	var startHeight uint32 = 0
	if blkRsp.Height > uint32(bc.ReorgBlockByRpc) {
		startHeight = blkRsp.Height - uint32(bc.ReorgBlockByRpc)
	}
	endHeight := blkRsp.Height + 1
	blksRsp, err := loader.GetBlockListFromDB(startHeight, endHeight)
	if err != nil {
		return false
	}

	var parent []byte = make([]byte, 32)
	for _, blk := range blksRsp {
		hashHex := utils.HashString(blk.BlockId)
		bc.Blocks[hashHex] = &model.BlockIndex{
			Height:    blk.Height,
			HashHex:   hashHex,
			ParentHex: utils.HashString(parent),
		}
		parent = blk.BlockId
	}
	return true
}

func (bc *Blockchain) InitLatestMetricBlockFromRedis() bool {
	height, err := loader.GetInfoHeight(rdb.RdbClient, constant.TASK_METRIC_HEIGHT)
	if err != nil {
		logger.Log.Error("read metric height failed", zap.Error(err))
		return false
	}

	blockID, err := loader.GetInfoStringFromRedis(constant.TASK_METRIC_BLOCK)
	if err != nil || blockID == "" {
		if height == 0 {
			return true
		}
		logger.Log.Error("metric block missing", zap.Uint32("height", height), zap.Error(err))
		return false
	}

	bc.Blocks[blockID] = &model.BlockIndex{
		Height:    height,
		HashHex:   blockID,
		ParentHex: "",
	}
	return true
}

// InitLatestBlockFromRPC
// Read the existing synced blocks from CK, then fetch the main-chain block list from the node for reorg detection.
// Only needs to fetch 100 blocks before and after the synced height.
func (bc *Blockchain) InitLatestBlockFromRPC(batch uint32) (uint32, bool) {
	blockIdHex, err := loader.GetBestBlockIdFromRedis()
	if err != nil {
		panic("sync check by GetBestBlockIdFromRedis, but failed.")
	}
	return bc.initLatestBlockFromRPCByBlockID(batch, blockIdHex)
}

func (bc *Blockchain) InitLatestMetricBlockFromRPC(batch uint32) (uint32, bool) {
	blockIdHex, err := loader.GetInfoStringFromRedis(constant.TASK_METRIC_BLOCK)
	if err != nil {
		panic("metric sync check by GetInfoStringFromRedis, but failed.")
	}
	return bc.initLatestBlockFromRPCByBlockID(batch, blockIdHex)
}

func (bc *Blockchain) initLatestBlockFromRPCByBlockID(batch uint32, blockIdHex string) (uint32, bool) {
	var heightPoint uint32 = 0

	if blockIdHex != "" {
		blk, ok := bc.Blocks[blockIdHex]
		if !ok {
			panic("sync check by GetBestBlockIdFromRedis, but failed.")
		}
		heightPoint = blk.Height
	}
	// logger.Log.Info("load block header from rpc", zap.Uint32("height", heightPoint))

	bc.BlocksOfChainById = make(map[string]struct{}, 0)
	bc.BlocksOfChainByHeight = make(map[uint32]*model.BlockIndexInfo, 0)

	var startHeight uint32 = 0
	if heightPoint > uint32(bc.ReorgBlockByRpc) {
		startHeight = heightPoint - uint32(bc.ReorgBlockByRpc)
	}
	endHeight := heightPoint + 1 + batch
	blockInfos, ok := loader.GetBlockIndexRangeStandardRPC(startHeight, endHeight, heightPoint, blockIdHex)
	if !ok {
		return heightPoint, false
	}
	var parentHex string
	for _, blk := range blockInfos {
		block := &model.BlockIndexInfo{
			Height:  blk.Height,
			HashHex: blk.HashHex,
		}
		bc.BlocksOfChainById[blk.HashHex] = struct{}{}
		bc.BlocksOfChainByHeight[blk.Height] = block

		bc.MainChainHeight = blk.Height
		bc.MainChainBlockIdHex = blk.HashHex

		if _, ok := bc.Blocks[blk.HashHex]; !ok {
			// update block index
			bc.Blocks[blk.HashHex] = &model.BlockIndex{
				Height:    blk.Height,
				HashHex:   blk.HashHex,
				ParentHex: parentHex,
			}
		}
		parentHex = blk.HashHex
	}
	return heightPoint, true
}

// GetBlockSyncCommonBlockHeight gets the common block height where block sync starts.
func (bc *Blockchain) GetBlockSyncCommonBlockHeight(endBlockHeight uint32) (heigth, orphanCount, newblock uint32, ok bool) {
	blockIdHex, err := loader.GetBestBlockIdFromRedis()
	if err != nil {
		panic("sync check by GetBestBlockIdFromRedis, but failed.")
	}
	return bc.getSyncCommonBlockHeight(blockIdHex, endBlockHeight)
}

func (bc *Blockchain) GetMetricSyncCommonBlockHeight(endBlockHeight uint32) (heigth, orphanCount, newblock uint32, ok bool) {
	blockIdHex, err := loader.GetInfoStringFromRedis(constant.TASK_METRIC_BLOCK)
	if err != nil {
		panic("metric sync check by GetInfoStringFromRedis, but failed.")
	}
	if blockIdHex == "" {
		if endBlockHeight == 0 || endBlockHeight > bc.MainChainHeight+1 {
			endBlockHeight = bc.MainChainHeight + 1
		}
		if endBlockHeight == 0 {
			return ^uint32(0), 0, 0, true
		}
		return ^uint32(0), 0, endBlockHeight, true
	}
	if commonHeight, orphanCount, newBlocks, ok := bc.getSyncCommonBlockHeight(blockIdHex, endBlockHeight); ok {
		return commonHeight, orphanCount, newBlocks, true
	}

	metricHeight, err := loader.GetInfoHeight(rdb.RdbClient, constant.TASK_METRIC_HEIGHT)
	if err != nil {
		logger.Log.Error("read metric height failed", zap.Error(err))
		return 0, 0, 0, false
	}
	if endBlockHeight == 0 || endBlockHeight > bc.MainChainHeight+1 {
		endBlockHeight = bc.MainChainHeight + 1
	}
	commonHeight := ^uint32(0)
	syncStartHeight := uint32(0)
	if metricHeight > uint32(bc.ReorgBlockByRpc) {
		commonHeight = metricHeight - uint32(bc.ReorgBlockByRpc)
		syncStartHeight = commonHeight + 1
	}
	newblock = endBlockHeight - syncStartHeight
	logger.Log.Warn("metric block not in main-chain window, replaying safe range",
		zap.String("metricBlock", blockIdHex),
		zap.Uint32("metricHeight", metricHeight),
		zap.Uint32("replayHeight", syncStartHeight),
		zap.Uint32("newBlock", newblock))
	return commonHeight, metricHeight - syncStartHeight + 1, newblock, true
}

func (bc *Blockchain) getSyncCommonBlockHeight(blockIdHex string, endBlockHeight uint32) (heigth, orphanCount, newblock uint32, ok bool) {
	if endBlockHeight == 0 || endBlockHeight > bc.MainChainHeight+1 {
		endBlockHeight = bc.MainChainHeight + 1
	}

	orphanCount = 0
	for {
		block, ok := bc.Blocks[blockIdHex]
		if !ok {
			logger.Log.Error("local blockId not load, too long reorg", zap.String("blkId", blockIdHex))
			break
		}

		if _, ok := bc.BlocksOfChainById[blockIdHex]; ok {
			newblock = endBlockHeight - 1 - block.Height
			logger.Log.Info("should sync block",
				zap.String("lastBlockId", block.HashHex),
				zap.Uint32("lastHeight", block.Height),
				zap.Uint32("nOrphan", orphanCount),
				zap.Uint32("nBlkNew", newblock))
			return block.Height, orphanCount, newblock, true
		}

		orphanCount++
		logger.Log.Info("orphan block happen",
			zap.String("blkIdNotInMain", blockIdHex),
			zap.Uint32("orphan", orphanCount),
		)
		blockIdHex = block.ParentHex
	}
	return 0, 0, 0, false
}
