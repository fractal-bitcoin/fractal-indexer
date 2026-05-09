package serial

import (
	"fractal-indexer/logger"
	"fractal-indexer/model"
	"fractal-indexer/utils"

	"go.uber.org/zap"
)

// SyncBlockTxInputDetail all tx input info
func SyncBlockTxInputDetail(block *model.Block) {
	var commonObjData *model.TxoData = &model.TxoData{
		Satoshi: utils.CalcBlockSubsidy(block.Height),
	}

	for txIdx := range block.Txs {
		tx := &block.Txs[txIdx]
		isCoinbase := (txIdx == 0)

		for vin, input := range tx.TxIns {
			objData := commonObjData
			if !isCoinbase {
				objData.Satoshi = 0
				if obj, ok := block.ParseData.SpentUtxoDataMap[input.InputOutpointKey]; ok {
					objData = obj
				} else {
					logger.Log.Info("tx-input-err",
						zap.String("txin", "input missing utxo"),
						zap.String("txid", tx.TxIdHex()),
						zap.Int("vin", vin),

						zap.String("utxid", input.InputHashHex()),
						zap.Uint32("vout", input.InputVout),
					)
				}
			}
			tx.InputsValue += objData.Satoshi
			tx.NFTInputsCnt += uint64(len(objData.CreatePointOfNFTs))
		}
	}
}
