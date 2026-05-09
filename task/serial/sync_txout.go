package serial

import (
	"fractal-indexer/model"
)

// SyncBlockTxOutputInfo all tx output info
func SyncBlockTxOutputInfo(block *model.Block) {
	for i := range block.Txs {
		tx := &block.Txs[i]
		for _, output := range tx.TxOuts {
			tx.NFTOutputsCnt += uint64(len(output.CreatePointOfNFTs))
			tx.OutputsValue += output.Satoshi
		}
	}
}
