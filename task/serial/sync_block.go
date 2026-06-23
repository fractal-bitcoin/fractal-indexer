package serial

import (
	"fractal-indexer/logger"
	"fractal-indexer/model"
	"fractal-indexer/store"

	"go.uber.org/zap"
)

// SyncBlock writes block data in RowBinary format.
func SyncBlock(block *model.Block) {
	coinbase := &block.Txs[0]
	coinbaseOut := coinbase.OutputsValue

	txInputsValue := uint64(0)
	txOutputsValue := uint64(0)

	nftInputsCnt := uint64(0)
	nftOutputsCnt := uint64(0)

	// Number of NFTs dropped in regular transactions and collected in coinbase.
	nftLostCnt := block.Txs[0].NFTInputsCnt

	nftNewCnt := uint64(0) // Includes NFTs from invalid creations.

	witSize := uint32(0)

	for _, tx := range block.Txs[1:] {
		txInputsValue += tx.InputsValue
		txOutputsValue += tx.OutputsValue

		nftNewCnt += uint64(len(tx.NewNFTDataCreated))
		nftInputsCnt += tx.NFTInputsCnt
		nftOutputsCnt += tx.NFTOutputsCnt

		if tx.WitOffset > 0 {
			witSize += (tx.Size - tx.WitOffset - 4)
		}
	}

	ins := store.BlkInserter()
	ins.Lock()
	ins.PutUInt32(uint32(block.Height))
	ins.PutUInt32(block.Version)
	ins.PutFixedString(block.Hash, 32)
	ins.PutFixedString(block.Parent, 32)
	ins.PutFixedString(block.MerkleRoot, 32)
	ins.PutUInt32(block.TxCnt)
	ins.PutUInt64(nftNewCnt)
	ins.PutUInt64(nftInputsCnt)
	ins.PutUInt64(nftOutputsCnt)
	ins.PutUInt64(nftLostCnt)
	ins.PutUInt64(txInputsValue)
	ins.PutUInt64(txOutputsValue)
	ins.PutUInt64(coinbaseOut)
	ins.PutUInt32(block.BlockTime)
	ins.PutUInt32(block.Bits)
	ins.PutUInt32(block.Size)
	ins.PutUInt32(witSize)
	if err := ins.EndRow(); err != nil {
		logger.Log.Info("sync-block-err",
			zap.String("blkid", block.HashHex),
			zap.String("err", err.Error()),
		)
		model.NeedStop.Store(true)
	}
}
