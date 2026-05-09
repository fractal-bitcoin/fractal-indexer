package serial

import (
	"encoding/hex"
	"fractal-indexer/logger"
	"fractal-indexer/model"

	"go.uber.org/zap"
)

// ParseGetSpentUtxoDataFromRedisSerial consumes UTXO prefetch results completed in the parallel phase.
// Redis queries and deserialization are completed in parallel goroutines; this function only classifies and fills data.
func ParseGetSpentUtxoDataFromRedisSerial(block *model.ProcessBlock) (valid bool) {
	valid = true
	fetched := block.SpentUtxoPrefetch

	for outpointKey := range block.SpentUtxoKeysMap {
		// Self-spend within the block.
		if data, ok := block.NewUtxoDataMap[outpointKey]; ok {
			block.SpentUtxoDataMap[outpointKey] = data
			delete(block.NewUtxoDataMap, outpointKey)
			continue
		}
		// Cross-block self-spend, created by a previous block in this batch and not yet written to Pika.
		if data, ok := model.GlobalNewUtxoDataMap[outpointKey]; ok {
			block.SpentUtxoDataMap[outpointKey] = data
			delete(model.GlobalNewUtxoDataMap, outpointKey)
			continue
		}
		// Data fetched from Pika during the parallel phase, or an inferred standard dust stub.
		if d, ok := fetched[outpointKey]; ok {
			block.SpentUtxoDataMap[outpointKey] = d
			if d.IsStandardDustStub {
				// Never written to Pika, so no DEL needed.
				continue
			}
			model.GlobalSpentUtxoCount++
			model.GlobalDeleteUtxoKeysMap = append(model.GlobalDeleteUtxoKeysMap, outpointKey)
			continue
		}
		// Truly missing.
		logger.Log.Error("parse block, but missing utxo from redis",
			zap.String("outpoint", hex.EncodeToString([]byte(outpointKey))))
		valid = false
		if !model.SkipMissingUTXO {
			return valid
		}
	}
	return valid
}

// UpdateUtxoInMapSerial sequentially applies current block UTXO changes to the program-wide cache.
func UpdateUtxoInMapSerial(block *model.ProcessBlock) {
	// Update local new-UTXO storage.
	for outpointKey, data := range block.NewUtxoDataMap {
		model.GlobalNewUtxoDataMap[outpointKey] = data
	}
}
