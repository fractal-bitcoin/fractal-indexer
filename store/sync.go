package store

import (
	"fractal-indexer/loader/clickhouse"
	"fractal-indexer/logger"

	"go.uber.org/zap"
)

var (
	blkInserter    *clickhouse.RowBinaryInserter
	metricInserter *clickhouse.RowBinaryInserter
	eventInserter  *clickhouse.RowBinaryInserter
	revertInserter *clickhouse.RowBinaryInserter
)

func prepareSyncCk(isFull, withMetric bool) bool {
	blkTable := "blk_height_new"
	metricTable := "blkmetric_height_new"
	eventTable := "blkevent_height_new"
	revertTable := "blkrevert_height_new"
	if isFull {
		blkTable = "blk_height"
		metricTable = "blkmetric_height"
		eventTable = "blkevent_height"
		revertTable = "blkrevert_height"
	}

	if blkInserter == nil {
		blkInserter = clickhouse.NewRowBinaryInserter(blkTable)
	} else {
		blkInserter.ResetForTable(blkTable)
	}
	if withMetric {
		if metricInserter == nil {
			metricInserter = clickhouse.NewRowBinaryInserter(metricTable)
		} else {
			metricInserter.ResetForTable(metricTable)
		}
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
	return prepareSyncCk(true, false)
}

func PrepareFullSyncCkWithMetric() bool {
	return prepareSyncCk(true, true)
}

func PreparePartSyncCk() bool {
	return prepareSyncCk(false, false)
}

func PreparePartSyncCkWithMetric() bool {
	return prepareSyncCk(false, true)
}

func PrepareMetricSyncCk(isFull bool) bool {
	metricTable := "blkmetric_height_new"
	if isFull {
		metricTable = "blkmetric_height"
	}
	if metricInserter == nil {
		metricInserter = clickhouse.NewRowBinaryInserter(metricTable)
	} else {
		metricInserter.ResetForTable(metricTable)
	}
	return true
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
	metricCount := 0
	if metricInserter != nil {
		if err := metricInserter.Flush(); err != nil {
			logger.Log.Error("sync-commit-blkmetric", zap.Error(err))
			isOK = false
		}
		metricCount = metricInserter.Count()
	}

	logger.Log.Debug("batch-commit-stats",
		zap.Int("blk_count", blkInserter.Count()),
		zap.Int("metric_count", metricCount),
		zap.Int("event_count", eventInserter.Count()),
	)

	return isOK
}

func CommitMetricCk() bool {
	if metricInserter == nil {
		return true
	}
	if err := metricInserter.Flush(); err != nil {
		logger.Log.Error("sync-commit-blkmetric", zap.Error(err))
		return false
	}
	logger.Log.Debug("metric-commit-stats", zap.Int("metric_count", metricInserter.Count()))
	return true
}

func CommitSyncCk() bool {
	isOK := true

	if err := blkInserter.Flush(); err != nil {
		logger.Log.Error("sync-commit-blk", zap.Error(err))
		isOK = false
	}
	metricCount := 0
	if metricInserter != nil {
		if err := metricInserter.Flush(); err != nil {
			logger.Log.Error("sync-commit-blkmetric", zap.Error(err))
			isOK = false
		}
		metricCount = metricInserter.Count()
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
		zap.Int("metric_count", metricCount),
		zap.Int("event_count", eventInserter.Count()),
		zap.Int("revert_count", revertInserter.Count()),
	)

	return isOK
}

// BlkInserter returns the block inserter for direct RowBinary writes.
func BlkInserter() *clickhouse.RowBinaryInserter {
	return blkInserter
}

// MetricInserter returns the block metric inserter for direct RowBinary writes.
func MetricInserter() *clickhouse.RowBinaryInserter {
	return metricInserter
}

// EventInserter returns the event inserter for direct RowBinary writes.
func EventInserter() *clickhouse.RowBinaryInserter {
	return eventInserter
}

// RevertInserter returns the revert inserter for direct RowBinary writes.
func RevertInserter() *clickhouse.RowBinaryInserter {
	return revertInserter
}
