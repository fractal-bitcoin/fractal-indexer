package serial

import (
	"fractal-indexer/logger"
	"fractal-indexer/model"
	scriptDecoder "fractal-indexer/parser/script"
	"fractal-indexer/store"

	"go.uber.org/zap"
)

// SyncBlockRevert writes revert data for the block.
// op=0: spent UTXOs (to restore on reorg) — marshaled TxoData
// op=1: new outputs (to delete on reorg) — outpoint key only
// op=2: new inscription IDs (to delete on reorg) — binId in outpoint field
func SyncBlockRevert(block *model.Block) {
	ins := store.RevertInserter()
	height := uint32(block.Height)

	// op=0: spent UTXOs (SpentUtxoDataMap excludes coinbase inputs)
	// Standard dust UTXOs (P2WPKH=294, P2TR=330, P2PKH=546, no inscriptions) are never
	// written to Pika, so there is nothing to restore on reorg — the read path re-infers
	// satoshi from the spending input's scriptSig/witness.
	// marshalBuf is reused across iterations — safe because ins.PutString copies data immediately.
	var marshalBuf []byte
	for outpointKey, txoData := range block.ParseData.SpentUtxoDataMap {
		if txoData.IsStandardDustStub {
			continue // inferred from input witness/scriptSig, never in Pika
		}
		if len(txoData.CreatePointOfNFTs) == 0 && scriptDecoder.ClassifyStandardDust(txoData.ScriptType, txoData.Satoshi) {
			continue // self-spent standard dust (from NewUtxoDataMap), also never in Pika
		}
		size := txoData.MarshalBufSize()
		if cap(marshalBuf) < size {
			marshalBuf = make([]byte, size)
		} else {
			marshalBuf = marshalBuf[:size]
		}
		zbuf, _ := txoData.Marshal(marshalBuf)

		ins.Lock()
		ins.PutUInt32(height)
		ins.PutUInt8(0)
		ins.PutString([]byte(model.TrimOutpointKey(outpointKey)))
		ins.PutString(zbuf)
		if err := ins.EndRow(); err != nil {
			logger.Log.Error("sync-revert-spent-err",
				zap.Uint32("height", height),
				zap.Error(err),
			)
			model.NeedStop = true
		}
	}

	// op=1: new outputs (skip unspendable zero-value outputs and standard dust outputs).
	// Standard dust outputs are intentionally absent from Pika, so the reorg DEL is a no-op;
	// omitting op=1 rows for them reduces WAL size without affecting correctness.
	for _, tx := range block.Txs[1:] {
		for idx := range tx.TxOuts {
			output := &tx.TxOuts[idx]
			if output.LockingScriptUnspendable && output.Satoshi == 0 {
				continue
			}
			// SyncBlockRevert runs in the End stage, after inscription tracking has
			// populated output.CreatePointOfNFTs with the final NFT list. Reading the
			// output directly also covers the self-spend case where the entry has
			// already been removed from NewUtxoDataMap during the serial stage.
			if scriptDecoder.ClassifyStandardDust(output.ScriptType, output.Satoshi) && len(output.CreatePointOfNFTs) == 0 {
				continue
			}
			ins.Lock()
			ins.PutUInt32(height)
			ins.PutUInt8(1)
			ins.PutString([]byte(model.TrimOutpointKey(output.OutpointKey)))
			ins.PutString(nil) // empty utxo_data
			if err := ins.EndRow(); err != nil {
				logger.Log.Error("sync-revert-new-err",
					zap.Uint32("height", height),
					zap.Error(err),
				)
				model.NeedStop = true
			}
		}
	}

	// op=2: inscription IDs (for deleting pika NFT ID entries on reorg)
	for _, nft := range block.ParseData.NewInscriptions {
		if nft.CreatePoint.IsBRC20Mint {
			continue
		}
		binId := scriptDecoder.GetNFTBinIdFromTxIdAndIdx(nft.TxId, int(nft.IdxInTx))
		ins.Lock()
		ins.PutUInt32(height)
		ins.PutUInt8(2)
		ins.PutString([]byte(binId))
		ins.PutString(nil) // empty utxo_data
		if err := ins.EndRow(); err != nil {
			logger.Log.Error("sync-revert-nftid-err",
				zap.Uint32("height", height),
				zap.Error(err),
			)
			model.NeedStop = true
		}
	}
}
