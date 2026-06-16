package task

import (
	"context"
	"fractal-indexer/constant"
	"fractal-indexer/loader"
	"fractal-indexer/logger"
	"fractal-indexer/model"
	"fractal-indexer/rdb"
	rdbUtils "fractal-indexer/rdb/utils"
	"fractal-indexer/store"
	"fractal-indexer/task/parallel"
	"fractal-indexer/task/serial"
	"fractal-indexer/utils"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
)

var ctx = context.Background()

// ParseBlockParallel first analyzes blocks in parallel: different blocks run in parallel, while work within the same block is serial.
func ParseBlockParallel(block *model.Block) {
	// pre-allocate TxoData slab for all outputs in the block
	totalOuts := 0
	for i := range block.Txs {
		totalOuts += int(block.Txs[i].TxOutCnt)
	}
	txoSlab := make([]model.TxoData, totalOuts)
	txoIdx := 0

	for txIdx := range block.Txs {
		tx := &block.Txs[txIdx]
		isCoinbase := txIdx == 0
		parallel.ParseTxFirst(tx, isCoinbase, block.ParseData)

		// Prepare UTXO spend relationship data.
		// UTXO records spent by all txins.
		parallel.ParseUpdateTxoSpendByTxParallel(tx, isCoinbase, block.ParseData)
		// UTXO records created by all txouts.
		parallel.ParseUpdateNewUtxoInTxParallel(uint32(txIdx), tx, block.ParseData, txoSlab[txoIdx:])
		txoIdx += int(tx.TxOutCnt)
	}

	parallel.ParseBlockMetricsParallel(block)

	// Prefetch spent UTXOs from Redis in parallel with other blocks' goroutines.
	// Passes the full block so inference from TxIn scriptSig/witness is possible.
	parallel.ParseGetSpentUtxoDataFromRedisParallel(block)
}

// ParseBlockSerialStart then processes the block serially.
func ParseBlockSerialStart(nftStartNumber, nftCursedStartNumber int64, block *model.Block) {
	// Fill in UTXO data spent by all transactions in the current block from Redis.
	valid := serial.ParseGetSpentUtxoDataFromRedisSerial(block.ParseData)
	if !valid {
		model.MissingUTXO = true
		if !model.SkipMissingUTXO {
			model.NeedStop.Store(true)
			return
		}
	}

	// Update NFT tracking data on input/output records and in UTXOs. Depends on UTXOs fetched from Redis.
	serial.ParseBlockTxNFTsInAndOutSerial(block)

	// Update numbers.
	serial.ParseBlockUpdateNFTNumberSerial(nftStartNumber, nftCursedStartNumber, block.ParseData.NewInscriptions)

	// Update txin DB data after prior and current block txouts are processed. Depends on UTXOs fetched from Redis, but does not require txout DB updates to be complete.
	serial.SyncBlockTxInputDetail(block)

	// Update txout DB data; depends on txin completion.
	serial.SyncBlockTxOutputInfo(block)

	// Must be serial: apply current block UTXO changes to the in-memory program cache.
	serial.UpdateUtxoInMapSerial(block.ParseData)
}

// ParseBlockParallelEnd then processes the block in parallel.
func ParseBlockParallelEnd(block *model.Block) {
	// Update block DB data; depends on txout and txin completion to calculate block fees.
	serial.SyncBlock(block)
	serial.SyncBlockMetrics(block)
	// Update NFT event DB data; depends on txout and txin completion.
	serial.SyncBlockEvent(block.ParseData.NewEventInscriptions)

	// Persisting directly here keeps each block's inscription end number immediately available after rollback.
	// Each batch does not use the intermediate block values written by this parallel DB write.
	// Normal persistence should happen in Submit; ideally this should also be written once in Submit.
	serial.SyncBlockNFTEndNumber(constant.ORDINALS_INSCRIPTION_COUNTS_BY_HEIGHT, block.Height, block.ParseData.NftEndNumber)              // Write directly to Pika.
	serial.SyncBlockNFTEndNumber(constant.ORDINALS_INSCRIPTION_CURSED_COUNTS_BY_HEIGHT, block.Height, block.ParseData.NftCursedEndNumber) // Write directly to Pika.

	// write NFT ID -> createPoint mappings to pika per-block (moved from SubmitBlocks)
	serial.SyncBlockNFTID(block.ParseData.NewInscriptions)

	// write revert data for reorg support (only in WAL mode)
	if model.EnableWAL {
		serial.SyncBlockRevert(block)
	}

	releaseBlockResources(block)
}

func releaseBlockResources(block *model.Block) {
	block.Txs = nil
	block.ParseData = nil

	// return slabs to pool
	model.PutNFTDataSlabs(block.NFTDataSlabs)
	block.NFTDataSlabs = nil
	model.PutTxSlabs(block.TxSlab, block.TxInSlab, block.TxOutSlab)
	block.TxSlab = nil
	block.TxInSlab = nil
	block.TxOutSlab = nil

	// return rawblock buffer to pool
	loader.PutRawBlock(block.RawBlockBuf)
	block.RawBlockBuf = nil
}

// ParseEnd performs the final processing step (business data only, revert already committed).
func ParseEnd(isFull bool) bool {
	// Commit DB writes (blk + event only).
	if ok := store.CommitBusinessCk(); !ok {
		return false
	}

	// Execute additional DB data updates.
	if isFull {
		return store.ProcessAllSyncCk()
	}
	return store.ProcessPartSyncCk()
}

// CheckAndRecover checks height consistency across all stores.
// If revert_height > business heights, rolls back to a clean state using RemoveBlocksForReorg.
func CheckAndRecover() bool {
	revertH, err := loader.GetInfoHeight(rdb.RdbClient, constant.TASK_REVERT_HEIGHT)
	if err != nil {
		logger.Log.Error("CheckAndRecover: read revert_height failed", zap.Error(err))
		return false
	}

	revertLastH, err := loader.GetInfoHeight(rdb.RdbClient, constant.TASK_REVERT_LAST_HEIGHT)
	if err != nil {
		logger.Log.Error("CheckAndRecover: read revert_last_height failed", zap.Error(err))
		return false
	}

	if revertLastH > revertH {
		startBlockHeight := uint32(revertLastH)
		if !store.RemoveOrphanPartSyncCk(startBlockHeight, true) {
			logger.Log.Error("CheckAndRecover: remove stale revert data failed",
				zap.Uint32("height", startBlockHeight))
			model.NeedStop.Store(true)
			return false
		}
		if _, err := rdb.RdbClient.HSet(ctx, constant.TASK_INFO_KEYNAME,
			constant.TASK_REVERT_LAST_HEIGHT, revertH).Result(); err != nil {
			logger.Log.Error("CheckAndRecover: update revert_last_height failed", zap.Error(err))
			model.NeedStop.Store(true)
			return false
		}
	}

	blockH, err := loader.GetInfoHeight(rdb.RdbClient, constant.TASK_BLOCK_HEIGHT)
	if err != nil {
		logger.Log.Error("CheckAndRecover: read block_height failed", zap.Error(err))
		return false
	}

	nftIdH, err := loader.GetInfoHeight(rdb.RdbClient, constant.TASK_NFT_ID)
	if err != nil {
		logger.Log.Error("CheckAndRecover: read nft_id_height failed", zap.Error(err))
		return false
	}

	var ckH uint32
	var ckBlockID string
	blkRsp, err := loader.GetLatestBlockFromDB()
	if err != nil {
		ckH = 0
	} else {
		ckH = blkRsp.Height
		ckBlockID = utils.HashString(blkRsp.BlockId)
	}

	logger.Log.Info("height check",
		zap.Uint32("revert", revertH),
		zap.Uint32("block", blockH),
		zap.Uint32("nft_id", nftIdH),
		zap.Uint32("ck", ckH),
	)

	// All consistent
	if revertH == blockH && blockH == nftIdH && blockH == ckH {
		if !syncBestBlockHash(ckBlockID) {
			return false
		}
		logger.Log.Info("CheckAndRecover: all heights consistent", zap.Uint32("height", blockH))
		return true
	}

	// Legacy: no revert_height yet, seed it if others are consistent
	if revertH == 0 && (blockH != 0 || nftIdH != 0 || ckH != 0) {
		if blockH == nftIdH && blockH == ckH {
			logger.Log.Info("CheckAndRecover: seeding revert_height from legacy state", zap.Uint32("height", blockH))
			if _, err := rdb.RdbClient.HMSet(ctx, constant.TASK_INFO_KEYNAME,
				constant.TASK_REVERT_HEIGHT, blockH,
				constant.TASK_REVERT_LAST_HEIGHT, blockH).Result(); err != nil {
				logger.Log.Error("CheckAndRecover: seed revert_height failed", zap.Error(err))
				model.NeedStop.Store(true)
				return false
			}
			if !syncBestBlockHash(ckBlockID) {
				return false
			}
			return true
		}
		logger.Log.Error("CheckAndRecover: legacy inconsistency, cannot auto-recover",
			zap.Uint32("block", blockH),
			zap.Uint32("nft_id", nftIdH),
			zap.Uint32("ck", ckH),
		)
		return false
	}

	// Find the lowest business height
	safeH := blockH
	if nftIdH < safeH {
		safeH = nftIdH
	}
	if ckH < safeH {
		safeH = ckH
	}

	if revertH > safeH {
		logger.Log.Warn("CheckAndRecover: incomplete write detected, rolling back",
			zap.Uint32("revert", revertH),
			zap.Uint32("safe", safeH),
		)

		// Reuse existing reorg logic to roll back to safeH
		if !RemoveBlocksForReorg(safeH + 1) {
			logger.Log.Error("CheckAndRecover: recovery failed")
			return false
		}

		// Update block hash in info to match safeH
		blkInfo, err := loader.GetBlockInfoFromDB(safeH)
		if err != nil {
			logger.Log.Error("CheckAndRecover: failed to get block info at safe height",
				zap.Uint32("height", safeH), zap.Error(err))
			return false
		}
		blockID := utils.HashString(blkInfo.BlockId)
		if _, err := rdb.RdbClient.HSet(ctx, constant.TASK_INFO_KEYNAME,
			constant.TASK_BLOCK, blockID,
		).Result(); err != nil {
			logger.Log.Error("CheckAndRecover: update block hash failed",
				zap.Uint32("height", safeH), zap.Error(err))
			model.NeedStop.Store(true)
			return false
		}
		logger.Log.Info("CheckAndRecover: recovered to consistent state",
			zap.Uint32("height", safeH),
			zap.String("blockId", blockID),
		)

		return true
	}

	// Unexpected: revert_height < business height
	logger.Log.Error("CheckAndRecover: unexpected state, revert_height < business height",
		zap.Uint32("revert", revertH),
		zap.Uint32("block", blockH),
		zap.Uint32("nft_id", nftIdH),
		zap.Uint32("ck", ckH),
	)
	return false
}

func syncBestBlockHash(ckBlockID string) bool {
	if ckBlockID == "" {
		return true
	}
	blockID, err := loader.GetBestBlockIdFromRedis()
	if err != nil {
		logger.Log.Error("CheckAndRecover: read block hash failed", zap.Error(err))
		model.NeedStop.Store(true)
		return false
	}
	if blockID == ckBlockID {
		return true
	}
	logger.Log.Warn("CheckAndRecover: block hash mismatch, correcting",
		zap.String("redisBlockId", blockID),
		zap.String("ckBlockId", ckBlockID),
	)
	if _, err := rdb.RdbClient.HSet(ctx, constant.TASK_INFO_KEYNAME,
		constant.TASK_BLOCK, ckBlockID,
	).Result(); err != nil {
		logger.Log.Error("CheckAndRecover: correct block hash failed", zap.Error(err))
		model.NeedStop.Store(true)
		return false
	}
	return true
}

// RemoveBlocksForReorg
func RemoveBlocksForReorg(startBlockHeight uint32) bool {
	// All ClickHouse writes (RowBinary HTTP insert, INSERT INTO...SELECT) are synchronous.
	// The only async op is ALTER TABLE DELETE, which is awaited inside RemoveOrphanPartSyncCk.
	// No need to poll for data presence.

	// Before updating, remove data for blocks that were imported last time but are now orphaned.
	logger.Log.Info("remove...")
	utxoToRestore, err := loader.GetSpentUTXOFromRevert(startBlockHeight)
	if err != nil {
		logger.Log.Error("get utxo to restore failed", zap.Error(err))
		return false
	}
	utxoToRemoveKeys, err := loader.GetNewUTXOKeysFromRevert(startBlockHeight)
	if err != nil {
		logger.Log.Error("get utxo to remove failed", zap.Error(err))
		return false
	}
	nftInscriptionIdsToRemove, err := loader.GetInscriptionIdsAfterBlockHeight(startBlockHeight)
	if err != nil {
		logger.Log.Error("get nft inscriptionIds to remove failed", zap.Error(err))
		return false
	}

	// remove common keys from both maps
	for key := range utxoToRemoveKeys {
		if _, ok := utxoToRestore[key]; ok {
			delete(utxoToRemoveKeys, key)
			delete(utxoToRestore, key)
		}
	}

	var wg sync.WaitGroup
	var failed atomic.Bool

	// ck
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Clear DB.
		if !store.RemoveOrphanPartSyncCk(startBlockHeight, false) {
			failed.Store(true)
		}
	}()

	// utxo
	wg.Add(1)
	go func() {
		defer wg.Done()

		removeSlice := make([]string, 0, len(utxoToRemoveKeys))
		for k := range utxoToRemoveKeys {
			removeSlice = append(removeSlice, k)
		}
		if ok := rdbUtils.UpdateUtxoInPikaDel(removeSlice); !ok {
			failed.Store(true)
			return
		}
		// Add the new UTXOs first, then delete the old UTXOs.
		if ok := rdbUtils.UpdateUtxoInPikaAddRaw(utxoToRestore); !ok {
			failed.Store(true)
			return
		}
		if _, err := rdb.RdbClient.HSet(ctx, constant.TASK_INFO_KEYNAME,
			constant.TASK_BLOCK_HEIGHT, startBlockHeight-1).Result(); err != nil {
			logger.Log.Error("RemoveBlocksForReorg failed, HSET block_height err", zap.Error(err))
			failed.Store(true)
		}
		logger.Log.Debug("utxo updated")
	}()

	// nft id
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Update the Redis NFT ID list.
		pipe := rdb.RdbClient.Pipeline()
		defer pipe.Close()
		for _, inscriptionId := range nftInscriptionIdsToRemove {
			pipe.Del(ctx, inscriptionId)
		}
		pipe.HSet(ctx, constant.TASK_INFO_KEYNAME, constant.TASK_NFT_ID, startBlockHeight-1)
		if _, err := pipe.Exec(ctx); err != nil {
			logger.Log.Error("pika nft id exec failed", zap.Error(err))
			failed.Store(true)
		}
		logger.Log.Debug("inscription id removed")
	}()

	wg.Wait()

	if failed.Load() {
		model.NeedStop.Store(true)
		return false
	}

	// Clear DB; make sure this deletion happens last.
	if _, err := rdb.RdbClient.HMSet(ctx, constant.TASK_INFO_KEYNAME,
		constant.TASK_REVERT_LAST_HEIGHT, startBlockHeight,
		constant.TASK_REVERT_HEIGHT, startBlockHeight-1).Result(); err != nil {
		logger.Log.Error("RemoveBlocksForReorg failed, HSET block_height err", zap.Error(err))
		model.NeedStop.Store(true)
	} else {
		// If current ClickHouse data exceeds revert_height, delete it directly back to revert_height.
		// Pika data should normally never exceed revert_height.
		if !store.RemoveOrphanPartSyncCk(startBlockHeight, true) {
			logger.Log.Error("RemoveBlocksForReorg failed, remove revert data failed",
				zap.Uint32("height", startBlockHeight))
			model.NeedStop.Store(true)
			return false
		}
	}
	if _, err := rdb.RdbClient.HMSet(ctx, constant.TASK_INFO_KEYNAME,
		constant.TASK_REVERT_LAST_HEIGHT, startBlockHeight-1,
		constant.TASK_REVERT_HEIGHT, startBlockHeight-1).Result(); err != nil {
		logger.Log.Error("RemoveBlocksForReorg failed, HSET block_height err", zap.Error(err))
		model.NeedStop.Store(true)
	}
	if model.NeedStop.Load() {
		return false
	}

	return true
}

// SubmitBlocks writes data in WAL-first order:
// Phase 1 (serial): flush revert data to ClickHouse, mark revert_height in pika
// Phase 2 (parallel): write business data (CH blk/event, pika UTXO, pika NFT ID)
// Phase 3: update height markers
func SubmitBlocks(isFull bool, stageBlockHeight uint32) bool {
	// Phase 1: Write-Ahead Revert.
	if model.EnableWAL {
		if ok := store.CommitRevertCk(); !ok {
			model.NeedStop.Store(true)
			return false
		}
		if !isFull {
			if ok := store.ProcessRevertPartSyncCk(); !ok {
				model.NeedStop.Store(true)
				return false
			}
		}
	}
	if _, err := rdb.RdbClient.HMSet(ctx, constant.TASK_INFO_KEYNAME,
		constant.TASK_REVERT_LAST_HEIGHT, stageBlockHeight,
		constant.TASK_REVERT_HEIGHT, stageBlockHeight).Result(); err != nil {
		logger.Log.Error("SubmitBlocks failed, HSET revert_height err", zap.Error(err))
		model.NeedStop.Store(true)
		return false
	}

	// Phase 2: Business data in parallel.
	var wg sync.WaitGroup
	var failed atomic.Bool

	// ck
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Run final processing.
		if ok := ParseEnd(isFull); !ok {
			failed.Store(true)
			return
		}
		if model.EnableWAL {
			store.DropOldRevertPartitions(stageBlockHeight)
		}
	}()

	// utxo data + nftpointer
	wg.Add(1)
	go func() {
		defer wg.Done()

		if ok := rdbUtils.UpdateUtxoInPikaDel(model.GlobalDeleteUtxoKeysMap); !ok {
			failed.Store(true)
			return
		}

		if ok := rdbUtils.UpdateUtxoInPikaAdd(model.GlobalNewUtxoDataMap); !ok {
			failed.Store(true)
			return
		}

		if _, err := rdb.RdbClient.HSet(ctx, constant.TASK_INFO_KEYNAME, constant.TASK_BLOCK_HEIGHT, stageBlockHeight).Result(); err != nil {
			logger.Log.Error("SubmitBlocks failed, HSET block_height err", zap.Error(err))
			failed.Store(true)
		}
		logger.Log.Debug("utxo updated")
	}()

	// nft id height update (NFT ID data already written per-block in ParseBlockParallelEnd)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err := rdb.RdbClient.HSet(ctx, constant.TASK_INFO_KEYNAME,
			constant.TASK_NFT_ID, stageBlockHeight).Result(); err != nil {
			logger.Log.Error("pika nft id height update failed", zap.Error(err))
			failed.Store(true)
		}
		logger.Log.Debug("inscription id updated")
	}()
	wg.Wait()

	if failed.Load() {
		model.NeedStop.Store(true)
		return false
	}

	// Clear local map memory.
	model.SnapshotBatchSizes()
	model.CleanUtxoMap()
	return true
}
