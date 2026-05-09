package utils

import (
	"fmt"
	"fractal-indexer/logger"
	"time"

	"go.uber.org/zap"
)

var (
	lastLogTime              time.Time
	lastBlockHeight          uint32
	lastBlockTxCount         int
	lastNewInscriptions      int
	lastTransferInscriptions int
)

func byteCountBinary(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func ParseBlockSpeed(nTx, lenGlobalNewUtxoDataMap, globalSpentUtxoCount, nNewInscriptions, nTransferInscriptions int, nextBlockHeight, maxBlockHeight uint32) {
	msg := "block"
	if nTx == 0 {
		msg = "header"
	}
	lastBlockTxCount += nTx
	lastNewInscriptions += nNewInscriptions
	lastTransferInscriptions += nTransferInscriptions

	if nextBlockHeight != maxBlockHeight-1 && time.Since(lastLogTime) < time.Second {
		return
	}

	if nextBlockHeight < lastBlockHeight {
		lastBlockHeight = 0
	}

	lastLogTime = time.Now()

	logger.Log.Info(msg,
		zap.Uint32("h", nextBlockHeight),
		zap.Uint32("nblk", nextBlockHeight-lastBlockHeight),
		zap.Int("ntx", lastBlockTxCount),
		zap.Int("txo", globalSpentUtxoCount),
		zap.Int("utxo", lenGlobalNewUtxoDataMap),
		zap.Int("mint", lastNewInscriptions),
		zap.Int("xfer", lastTransferInscriptions),
	)
	lastBlockHeight = nextBlockHeight
	lastBlockTxCount = 0
	lastNewInscriptions = 0
	lastTransferInscriptions = 0
}
