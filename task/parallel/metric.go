package parallel

import (
	"fractal-indexer/constant"
	"fractal-indexer/model"
	scriptDecoder "fractal-indexer/parser/script"
)

func ParseBlockMetricsParallel(block *model.Block) {
	metrics := make(map[uint32]uint32, 6)
	for txIdx := range block.Txs {
		tx := &block.Txs[txIdx]

		if tx.WitOffset > 0 {
			metrics[constant.BlockMetricTxWithWitness]++
		}
		if len(tx.NewNFTDataCreated) > 0 {
			metrics[constant.BlockMetricTxWithInscription]++
		}

		hasTacit := false
		for inIdx := range tx.TxIns {
			if scriptDecoder.WitnessHasTacitEnvelope(tx.TxIns[inIdx].ScriptWitness) {
				hasTacit = true
				break
			}
		}
		if hasTacit {
			metrics[constant.BlockMetricTxWithTacit]++
		}

		hasOpReturn := false
		hasRunestone := false
		hasEtching := false
		for outIdx := range tx.TxOuts {
			pkScript := tx.TxOuts[outIdx].PkScript
			if !hasOpReturn && scriptDecoder.IsOpreturn(pkScript) {
				hasOpReturn = true
			}
			if !hasRunestone && scriptDecoder.IsRunestone(pkScript) {
				hasRunestone = true
			}
			if !hasEtching && scriptDecoder.IsRunesEtching(pkScript) {
				hasEtching = true
			}
		}
		if hasOpReturn {
			metrics[constant.BlockMetricTxWithOpReturn]++
		}
		if hasRunestone {
			metrics[constant.BlockMetricTxWithRunesRunestone]++
		}
		if hasEtching {
			metrics[constant.BlockMetricTxWithRunesEtching]++
		}
	}
	block.ParseData.BlockMetrics = metrics
}
