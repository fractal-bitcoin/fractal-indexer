package store

import (
	"fractal-indexer/loader/clickhouse"
	"fractal-indexer/logger"

	"go.uber.org/zap"
)

var (
	blkInserter    *clickhouse.RowBinaryInserter
	eventInserter  *clickhouse.RowBinaryInserter
	revertInserter *clickhouse.RowBinaryInserter
)

func prepareSyncCk(isFull bool) bool {
	blkTable := "blk_height_new"
	eventTable := "blkevent_height_new"
	revertTable := "blkrevert_height_new"
	if isFull {
		blkTable = "blk_height"
		eventTable = "blkevent_height"
		revertTable = "blkrevert_height"
	}

	if blkInserter == nil {
		blkInserter = clickhouse.NewRowBinaryInserter(blkTable)
	} else {
		blkInserter.ResetForTable(blkTable)
	}
	if eventInserter == nil {
		eventInserter = clickhouse.NewRowBinaryInserter(eventTable)
	} else {
		eventInserter.ResetForTable(eventTable)
	}
	if revertInserter == nil {
		revertInserter = clickhouse.NewRowBinaryInserter(revertTable)
	} else {
		revertInserter.ResetForTable(revertTable)
	}
	return true
}

func PrepareFullSyncCk() bool {
	return prepareSyncCk(true)
}

func PreparePartSyncCk() bool {
	return prepareSyncCk(false)
}

// CommitRevertCk flushes only the revert inserter (WAL phase).
// Must succeed before any business data is written.
func CommitRevertCk() bool {
	if err := revertInserter.Flush(); err != nil {
		logger.Log.Error("sync-commit-revert-wal", zap.Error(err))
		return false
	}
	logger.Log.Debug("wal-revert-committed", zap.Int("revert_count", revertInserter.Count()))
	return true
}

// CommitBusinessCk flushes blk and event inserters (business data phase).
// Called after revert WAL is confirmed.
func CommitBusinessCk() bool {
	isOK := true

	if err := blkInserter.Flush(); err != nil {
		logger.Log.Error("sync-commit-blk", zap.Error(err))
		isOK = false
	}
	if err := eventInserter.Flush(); err != nil {
		logger.Log.Error("sync-commit-nftevent", zap.Error(err))
		isOK = false
	}

	logger.Log.Debug("batch-commit-stats",
		zap.Int("blk_count", blkInserter.Count()),
		zap.Int("event_count", eventInserter.Count()),
	)

	return isOK
}

func CommitSyncCk() bool {
	isOK := true

	if err := blkInserter.Flush(); err != nil {
		logger.Log.Error("sync-commit-blk", zap.Error(err))
		isOK = false
	}
	if err := eventInserter.Flush(); err != nil {
		logger.Log.Error("sync-commit-nftevent", zap.Error(err))
		isOK = false
	}
	if err := revertInserter.Flush(); err != nil {
		logger.Log.Error("sync-commit-revert", zap.Error(err))
		isOK = false
	}

	logger.Log.Debug("batch-commit-stats",
		zap.Int("blk_count", blkInserter.Count()),
		zap.Int("event_count", eventInserter.Count()),
		zap.Int("revert_count", revertInserter.Count()),
	)

	return isOK
}

// BlkInserter returns the block inserter for direct RowBinary writes.
func BlkInserter() *clickhouse.RowBinaryInserter {
	return blkInserter
}

// EventInserter returns the event inserter for direct RowBinary writes.
func EventInserter() *clickhouse.RowBinaryInserter {
	return eventInserter
}

// RevertInserter returns the revert inserter for direct RowBinary writes.
func RevertInserter() *clickhouse.RowBinaryInserter {
	return revertInserter
}
