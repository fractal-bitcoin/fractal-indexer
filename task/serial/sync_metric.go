package serial

import (
	"fractal-indexer/logger"
	"fractal-indexer/model"
	"fractal-indexer/store"

	"go.uber.org/zap"
)

func SyncBlockMetrics(block *model.Block) {
	if len(block.ParseData.BlockMetrics) == 0 {
		return
	}

	ins := store.MetricInserter()
	for metric, value := range block.ParseData.BlockMetrics {
		if value == 0 {
			continue
		}
		ins.Lock()
		ins.PutUInt32(block.Height)
		ins.PutUInt32(block.BlockTime)
		ins.PutUInt32(metric)
		ins.PutUInt32(value)
		if err := ins.EndRow(); err != nil {
			logger.Log.Info("sync-block-metric-err",
				zap.Uint32("height", block.Height),
				zap.Uint32("metric", metric),
				zap.String("err", err.Error()),
			)
			model.NeedStop.Store(true)
		}
	}
}
