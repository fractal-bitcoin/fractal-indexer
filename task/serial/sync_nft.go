package serial

import (
	"context"
	"fmt"
	"fractal-indexer/constant"
	"fractal-indexer/logger"
	"fractal-indexer/model"
	scriptDecoder "fractal-indexer/parser/script"
	"fractal-indexer/rdb"
	"fractal-indexer/store"
	"fractal-indexer/utils"

	redis "github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

func ParseBlockUpdateNFTNumberSerial(nftStartNumber, nftCursedStartNumber int64, nfts []*model.NewInscriptionInfo) {
	if len(nfts) == 0 {
		return
	}

	var orderNFTs []*model.NewInscriptionInfo
	for _, nft := range nfts {
		// When an NFT is paid as a fee, or when the input sat is 0, its number is deferred.
		// If it only has an unrecognized even tag, it is not deferred.
		// If both cases occur, for example an even-tag input sat is 0, it should be deferred.
		// This logic only runs for blocks; the mempool does not handle it yet.
		if nft.IsDefer { // inscribe as fee, and have no even tag, need defer number
			orderNFTs = append(orderNFTs, nft)
			continue
		}

		if nft.NFTData.IsCursed {
			nft.Number = -(nftCursedStartNumber + 1)
			nftCursedStartNumber++
		} else {
			nft.Number = nftStartNumber
			nftStartNumber++
		}
	}
	for _, nft := range orderNFTs {
		if nft.NFTData.IsCursed {
			nft.Number = -(nftCursedStartNumber + 1)
			nftCursedStartNumber++
		} else {
			nft.Number = nftStartNumber
			nftStartNumber++
		}
	}
}

// SyncBlockEvent writes all events in RowBinary format. Zero string conversions.
func SyncBlockEvent(events []*model.NewInscriptionInfo) {
	for eventIdx, event := range events {

		nftheight := event.CreatePoint.Height
		transferNftHeight := event.Height
		if transferNftHeight != 0 {
			// transfer
			transferNftHeight, nftheight = nftheight, transferNftHeight
		}

		nfttype := uint8(0)
		if event.CreatePoint.IsText {
			nfttype = 1
			if event.CreatePoint.IsBRC20 {
				nfttype = 3
			}
		}

		if event.CreatePoint.IsCursed {
			nfttype += 0x80
		}
		if event.CreatePoint.IsVindicate {
			nfttype += 0x40
		}

		contentCode, contentBody := constant.EncodeNFTContentForDB(event.NFTData.ContentBody)

		ins := store.EventInserter()
		ins.Lock()
		ins.PutFixedString(event.TxId, 32)
		ins.PutUInt32(event.IdxInTx)
		ins.PutUInt32(event.NFTData.InTxVin)
		ins.PutUInt32(event.InTxVout)
		ins.PutUInt64(event.CreatePoint.Offset)
		ins.PutUInt64(event.Satoshi)
		ins.PutString(event.PkScript)
		ins.PutString(event.PkScriptFrom)
		ins.PutString(event.NFTData.TapScriptPk)
		ins.PutUInt32(event.InputIdx)
		ins.PutUInt64(event.InputsValue)
		ins.PutUInt64(event.OutputsValue)
		ins.PutUInt64(event.NFTData.Pointer)
		ins.PutUInt32(event.NFTData.GetFlags())
		ins.PutString(event.NFTData.GetParentsData())
		ins.PutString(event.NFTData.Deligate)
		ins.PutString(event.NFTData.MetaProtocol)
		ins.PutString(event.NFTData.Metadata)
		ins.PutString(event.NFTData.ContentEncoding)
		ins.PutString(event.NFTData.ContentType)
		ins.PutUInt32(uint32(len(event.NFTData.ContentBody)))
		ins.PutUInt8(contentCode)
		ins.PutString(contentBody)
		ins.PutUInt32(nftheight)
		ins.PutUInt64(uint64(eventIdx))
		ins.PutUInt32(event.TxIdx)
		ins.PutUInt32(event.BlockTime)
		ins.PutUInt32(event.CreatePoint.IdxInBlock)
		ins.PutInt64(event.Number)
		ins.PutUInt32(transferNftHeight)
		ins.PutUInt16(event.CreatePoint.Sequence)
		ins.PutUInt8(nfttype)
		if err := ins.EndRow(); err != nil {
			logger.Log.Info("sync-event-err",
				zap.String("eventid", utils.HashString(event.TxId)),
				zap.String("err", err.Error()),
			)
			model.NeedStop.Store(true)
		}
	}
}

// SyncBlockNFTID writes inscription ID -> createPoint mappings to pika per-block.
// BRC-20 mint inscriptions are intentionally excluded from the pika NFT ID index.
func SyncBlockNFTID(nfts []*model.NewInscriptionInfo) {
	if len(nfts) == 0 {
		return
	}
	pipe := rdb.RdbClient.Pipeline()
	defer pipe.Close()
	ctx := context.Background()
	queued := false
	for _, nft := range nfts {
		if nft.CreatePoint.IsBRC20Mint {
			continue
		}
		binId := scriptDecoder.GetNFTBinIdFromTxIdAndIdx(nft.TxId, int(nft.IdxInTx))
		pipe.Set(ctx, "i"+binId, nft.CreatePoint.GetCreateIdxUint64(), 0)
		queued = true
	}
	if !queued {
		return
	}
	if _, err := pipe.Exec(ctx); err != nil {
		logger.Log.Error("SyncBlockNFTID failed", zap.Error(err))
		model.NeedStop.Store(true)
	}
}

func SyncBlockNFTEndNumber(key string, height uint32, nftEndNumber int64) {
	member := &redis.Z{
		Score:  float64(nftEndNumber),
		Member: fmt.Sprintf("%v", height),
	}
	_, err := rdb.RdbClient.ZAdd(context.Background(), key, member).Result()
	if err != nil {
		logger.Log.Error("SyncBlockNFTEndNumber failed",
			zap.String("err", err.Error()),
		)
		model.NeedStop.Store(true)
	}
}
