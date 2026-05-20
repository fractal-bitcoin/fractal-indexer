package service

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"fractal-indexer/api/constant"
	"fractal-indexer/api/dao/rdb"
	scriptDecoder "fractal-indexer/api/lib/blkparser/script"
	"fractal-indexer/api/lib/utils"
	"fractal-indexer/api/model"
	mtx "fractal-indexer/lib/midware"
	"fractal-indexer/logger"
	indexerUtils "fractal-indexer/utils"

	redis "github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

const MAX_UTXO_LIMIT = 5000

// getNonTokenUtxoFromRedisReturnAddressOnly gets UTXO details from a list of UTXO keys.
func getNonTokenUtxoFromRedisReturnAddressOnly(caller string, utxoOutpoints []string) (txOutsRsp []*model.TxStandardOutResp, err error) {
	logger.Log.Info("getNonTokenUtxoFromRedisReturnAddressOnly", zap.Int("nOutpoints", len(utxoOutpoints)))
	if len(utxoOutpoints) > 1000000 {
		logger.Log.Warn("getNonTokenUtxoFromRedisReturnAddressOnly outpoints too many", zap.Int("nOutpoints", len(utxoOutpoints)))
	}

	txOutsRsp = make([]*model.TxStandardOutResp, 0, len(utxoOutpoints))
	const pageSize = 10000
	for i := 0; i < len(utxoOutpoints); i += pageSize {
		pageOutpoints := utxoOutpoints[i:min(i+pageSize, len(utxoOutpoints))]
		txOutsRspPage, err := getNonTokenUtxoFromRedisReturnAddressOnlyWithPagination(caller, pageOutpoints)
		if err != nil {
			logger.Log.Error("getNonTokenUtxoFromRedisReturnAddressOnly failed to get result", zap.Error(err), zap.String("caller", caller))
			return make([]*model.TxStandardOutResp, 0), err
		}
		txOutsRsp = append(txOutsRsp, txOutsRspPage...)
	}
	return txOutsRsp, nil
}

// getNonTokenUtxoFromRedisReturnAddressOnlyWithPagination gets UTXO owner addresses from a list of UTXO keys.
func getNonTokenUtxoFromRedisReturnAddressOnlyWithPagination(caller string, utxoOutpoints []string) (txOutsRsp []*model.TxStandardOutResp, err error) {
	txOutsRsp = make([]*model.TxStandardOutResp, 0, len(utxoOutpoints))

	pipeline := rdb.RdbRedisClient.Pipeline()
	defer pipeline.Close()
	outpointsCmd := make([]*redis.StringCmd, 0, len(utxoOutpoints))
	for _, outpoint := range utxoOutpoints {
		if len(outpoint) == 0 {
			outpointsCmd = append(outpointsCmd, nil)
			continue
		}
		outpointsCmd = append(outpointsCmd, pipeline.Get(ctx, "u"+outpoint))
	}
	_, err = pipeline.Exec(ctx)
	if err != nil && err != redis.Nil {
		mtx.ReportError(mtx.ErrorRedis)
		panic(err)
	}

	for outpointIdx, data := range outpointsCmd {
		if data == nil {
			txOutsRsp = append(txOutsRsp, &model.TxStandardOutResp{})
			continue
		}
		outpoint := utxoOutpoints[outpointIdx]
		res, err := data.Result()
		if err == redis.Nil {
			txIdHex := indexerUtils.HashString([]byte(outpoint[:32]))
			txIndex := int(binary.LittleEndian.Uint32([]byte(outpoint[32:])))
			txOutsRsp = append(txOutsRsp, &model.TxStandardOutResp{
				TxIdHex: txIdHex,
				Vout:    txIndex,
			})
			logger.Log.Info("nft holders not found by utxo",
				zap.String("caller", caller),
				zap.String("utxid", txIdHex),
				zap.Int("vout", txIndex),
			)
			continue
		} else if err != nil {
			mtx.ReportError(mtx.ErrorRedis)
			panic(err)
		}

		txout := model.NewTxoDataWithoutInscriptions([]byte(outpoint), []byte(res))
		addressData := scriptDecoder.ExtractPkScriptForTxo(txout.PkScript, txout.ScriptType)
		txOutsRsp = append(txOutsRsp, &model.TxStandardOutResp{
			Address: utils.EncodeAddressByCodeType(txout.PkScript, addressData.CodeType),
			Height:  int(txout.BlockHeight),
		})
	}
	return txOutsRsp, nil
}

func getNonTokenUtxoFromRedis(caller string, utxoOutpoints []string, withNFT bool) ([]*model.TxStandardOutResp, error) {
	logger.Log.Info("getNonTokenUtxoFromRedis", zap.Int("nOutpoints", len(utxoOutpoints)), zap.Bool("withNFT", withNFT))

	txOutsRsp := make([]*model.TxStandardOutResp, 0, len(utxoOutpoints))

	pipelineForUtxo := rdb.RdbRedisClient.Pipeline()
	defer pipelineForUtxo.Close()
	utxoCmds := make([]*redis.StringCmd, 0, len(utxoOutpoints))
	for _, outpoint := range utxoOutpoints {
		if len(outpoint) == 0 {
			utxoCmds = append(utxoCmds, nil)
			continue
		}
		utxoCmds = append(utxoCmds, pipelineForUtxo.Get(ctx, "u"+outpoint))
	}

	if _, err := pipelineForUtxo.Exec(ctx); err != nil && err != redis.Nil {
		mtx.ReportError(mtx.ErrorRedis)
		logger.Log.Error("getNonTokenUtxoFromRedis pipelineForUtxo.Exec err", zap.Error(err))
		return nil, err
	}

	for idx, utxoCmd := range utxoCmds {
		if utxoCmd == nil {
			txOutsRsp = append(txOutsRsp, &model.TxStandardOutResp{})
			continue
		}
		outpoint := utxoOutpoints[idx]
		data, err := utxoCmd.Result()
		if err == redis.Nil {
			txIdHex := indexerUtils.HashString([]byte(outpoint[:32]))
			txIndex := int(binary.LittleEndian.Uint32([]byte(outpoint[32:])))
			txOutsRsp = append(txOutsRsp, &model.TxStandardOutResp{
				TxIdHex: txIdHex,
				Vout:    txIndex,
			})
			logger.Log.Info("getNonTokenUtxoFromRedis utxo not found from redis", zap.String("caller", caller), zap.String("utxid", txIdHex), zap.Int("vout", txIndex))
			continue
		}
		if err != nil {
			mtx.ReportError(mtx.ErrorRedis)
			logger.Log.Error("getNonTokenUtxoFromRedis failed to get result", zap.Error(err), zap.String("caller", caller))
			return nil, err
		}

		txout := model.NewTxoData([]byte(outpoint), []byte(data))
		addressData := scriptDecoder.ExtractPkScriptForTxo(txout.PkScript, txout.ScriptType)
		if len(txout.CreatePointOfNFTs) > 1000 {
			logger.Log.Warn("getNonTokenUtxoFromRedis utxo has too many inscriptions, will limit to 500", zap.String("caller", caller), zap.String("utxid", indexerUtils.HashString(txout.UTxid)), zap.Uint32("vout", txout.Vout), zap.Int("inscriptions_count", len(txout.CreatePointOfNFTs)))
		}
		var limitedCreatePointOfNFTs []*model.NFTCreatePoint
		for _, nftpoint := range txout.CreatePointOfNFTs {
			if nftpoint.IsStrip {
				continue
			}
			if len(limitedCreatePointOfNFTs) >= 500 {
				break
			}
			limitedCreatePointOfNFTs = append(limitedCreatePointOfNFTs, nftpoint)
		}
		txOutsRsp = append(txOutsRsp, &model.TxStandardOutResp{
			TxIdHex:  indexerUtils.HashString(txout.UTxid),
			Vout:     int(txout.Vout),
			Satoshi:  int(txout.Satoshi),
			CodeType: int(addressData.CodeType),
			Address:  utils.EncodeAddressByCodeType(txout.PkScript, addressData.CodeType),

			ScriptTypeHex: hex.EncodeToString(txout.ScriptType),
			ScriptPkHex:   hex.EncodeToString(txout.PkScript),
			Height:        int(txout.BlockHeight),
			TxIdx:         int(txout.TxIdx),
			OpInRBF:       txout.OpInRBF,

			CreatePointOfNFTs: limitedCreatePointOfNFTs,
			InscriptionsCount: len(txout.CreatePointOfNFTs),
		})
	}

	for _, utxo := range txOutsRsp {
		utxo.Inscriptions = make([]*model.NFTData, 0)
		if len(utxo.CreatePointOfNFTs) == 0 || !withNFT {
			continue
		}
		nftsCreateIdx := make([]uint64, 0, len(utxo.CreatePointOfNFTs))
		for _, nftpoint := range utxo.CreatePointOfNFTs {
			nftsCreateIdx = append(nftsCreateIdx, nftpoint.GetCreateIdxKey())
		}

		nftsRsp, err := GetInscriptionBodyFromRedis(nftsCreateIdx)
		if err != nil {
			logger.Log.Info("redis nftsId not found")
			continue
		}

		for idx, nftpoint := range utxo.CreatePointOfNFTs {
			utxo.Inscriptions = append(utxo.Inscriptions, &model.NFTData{
				InscriptionNumber: nftsRsp[idx].InscriptionNumber,
				InscriptionId:     nftsRsp[idx].InscriptionId,
				Offset:            nftpoint.Offset,
				IsStrip:           nftpoint.IsStrip,
				HasMoved:          nftpoint.Sequence > 0,
				Sequence:          nftpoint.Sequence,
				IsCursed:          nftpoint.IsCursed,
				IsVindicate:       nftpoint.IsVindicate,
				IsBRC20Tran:       nftpoint.IsBRC20Tran,
				IsBRC20Mint:       nftpoint.IsBRC20Mint,
				IsBRC20Ext:        nftpoint.IsBRC20Ext,
				IsBRC20:           nftpoint.IsBRC20,
			})
		}
	}

	return txOutsRsp, nil
}

func GetUtxoByTxIdAndIdx(ctx context.Context, txId []byte, txIdHex string, txIndex int) (*model.TxStandardOutResp, error) {
	var outpoint [36]byte
	copy(outpoint[:], txId)
	binary.LittleEndian.PutUint32(outpoint[32:], uint32(txIndex))

	redisKeyUtxo := "u" + string(outpoint[:])

	txOutRsp := &model.TxStandardOutResp{
		TxIdHex: indexerUtils.HashString(txId),
		Vout:    txIndex,
	}
	var addressData *scriptDecoder.AddressData

	res, err := rdb.RdbRedisClient.Get(ctx, redisKeyUtxo).Result()
	if err == redis.Nil {
		logger.Log.Info("query utxo by api, but not found(can fail)", zap.String("utxid", txIdHex), zap.Int("vout", txIndex))
		return nil, nil
	}
	if err != nil {
		mtx.ReportError(mtx.ErrorRedis)
		logger.Log.Error("GetUtxoByTxIdAndIdx redis get outpoint err", zap.Error(err), zap.String("utxid", txIdHex), zap.Int("vout", txIndex), zap.Error(err))
		return nil, err
	}

	txout := model.NewTxoData(outpoint[:], []byte(res))
	addressData = scriptDecoder.ExtractPkScriptForTxo(txout.PkScript, txout.ScriptType)
	if len(txout.CreatePointOfNFTs) > 1000 {
		logger.Log.Warn("GetUtxoByTxIdAndIdx utxo has too many inscriptions, will limit to 500", zap.String("utxid", txIdHex), zap.Uint32("vout", txout.Vout), zap.Int("inscriptions_count", len(txout.CreatePointOfNFTs)))
	}
	var limitedCreatePointOfNFTs []*model.NFTCreatePoint
	for _, nftpoint := range txout.CreatePointOfNFTs {
		if nftpoint.IsStrip {
			continue
		}
		if len(limitedCreatePointOfNFTs) >= 500 {
			break
		}
		limitedCreatePointOfNFTs = append(limitedCreatePointOfNFTs, nftpoint)
	}

	txOutRsp = &model.TxStandardOutResp{
		TxIdHex:  indexerUtils.HashString(txout.UTxid),
		Vout:     int(txout.Vout),
		Satoshi:  int(txout.Satoshi),
		CodeType: int(addressData.CodeType),
		Address:  utils.EncodeAddressByCodeType(txout.PkScript, addressData.CodeType),

		ScriptTypeHex: hex.EncodeToString(txout.ScriptType),
		ScriptPkHex:   hex.EncodeToString(txout.PkScript),
		Height:        int(txout.BlockHeight),
		TxIdx:         int(txout.TxIdx),
		OpInRBF:       txout.OpInRBF,

		CreatePointOfNFTs: limitedCreatePointOfNFTs,
		InscriptionsCount: len(txout.CreatePointOfNFTs),
	}

	txOutRsp.Inscriptions = make([]*model.NFTData, 0)
	if len(txOutRsp.CreatePointOfNFTs) == 0 {
		return txOutRsp, nil
	}
	nftsCreateIdx := make([]uint64, 0, len(txOutRsp.CreatePointOfNFTs))
	for _, nftpoint := range txOutRsp.CreatePointOfNFTs {
		nftsCreateIdx = append(nftsCreateIdx, nftpoint.GetCreateIdxKey())
	}

	nftsRsp, err := GetInscriptionBodyFromRedis(nftsCreateIdx)
	if err != nil {
		logger.Log.Info("redis nftsId not found")
		return nil, err
	}

	for idx, nftpoint := range txOutRsp.CreatePointOfNFTs {
		txOutRsp.Inscriptions = append(txOutRsp.Inscriptions, &model.NFTData{
			InscriptionNumber: nftsRsp[idx].InscriptionNumber,
			InscriptionId:     nftsRsp[idx].InscriptionId,
			Offset:            nftpoint.Offset,
			IsStrip:           nftpoint.IsStrip,
			HasMoved:          nftpoint.Sequence > 0,
			Sequence:          nftpoint.Sequence,
			IsCursed:          nftpoint.IsCursed,
			IsVindicate:       nftpoint.IsVindicate,
			IsBRC20Tran:       nftpoint.IsBRC20Tran,
			IsBRC20Mint:       nftpoint.IsBRC20Mint,
			IsBRC20Ext:        nftpoint.IsBRC20Ext,
			IsBRC20:           nftpoint.IsBRC20,
		})
	}

	return txOutRsp, nil

}

// GetUtxoNftOffsetByTxIdAndIdx gets the UTXO NFT offset list.
func GetUtxoNftOffsetByTxIdAndIdx(txId []byte, txIdHex string, txIndex int) (offsetList []uint64, err error) {
	offsetList = []uint64{}
	var outpoint [36]byte
	copy(outpoint[:], txId)
	binary.LittleEndian.PutUint32(outpoint[32:], uint32(txIndex))
	// Check the length first.
	redisKeyUtxo := "u" + string(outpoint[:])
	dataSize, err := rdb.RdbRedisClient.StrLen(ctx, redisKeyUtxo).Result()
	if err == redis.Nil {
		logger.Log.Info("GetUtxoNftOffsetByTxIdAndIdx, but not found(can fail)", zap.String("utxid", txIdHex), zap.Int("vout", txIndex))
		return nil, nil
	}
	if err != nil {
		mtx.ReportError(mtx.ErrorRedis)
		logger.Log.Error("GetUtxoNftOffsetByTxIdAndIdx redis get outpoint strlen err", zap.Error(err), zap.String("utxid", txIdHex), zap.Int("vout", txIndex), zap.Error(err))
		return nil, err
	}
	if dataSize > int64(constant.MAX_STR_LEN_ALLOWED_TO_QUERY_FROM_RDB) {
		logger.Log.Warn("large utxo", zap.Int64("strLen", dataSize))
		logger.Log.Error("GetUtxoNftOffsetByTxIdAndIdx, datasize too large", zap.Error(err), zap.String("utxid", txIdHex), zap.Int("vout", txIndex), zap.Error(err))
		return nil, fmt.Errorf("datasize too large")
	}

	res, err := rdb.RdbRedisClient.Get(ctx, "u"+string(outpoint[:])).Result()
	if err == redis.Nil {
		logger.Log.Info("query utxo by api, but not found(can fail)",
			zap.String("utxid", txIdHex),
			zap.Int("vout", txIndex),
		)
		return nil, nil
	} else if err != nil {
		mtx.ReportError(mtx.ErrorRedis)
		logger.Log.Info("redis get outpoint err", zap.Error(err), zap.String("utxid", txIdHex),
			zap.Int("vout", txIndex))
		return nil, err
	}

	txout := model.NewTxoData(outpoint[:], []byte(res))
	for _, nftpoint := range txout.CreatePointOfNFTs {
		offsetList = append(offsetList, nftpoint.Offset)
		// Return only the first 10 entries.
		if len(offsetList) >= 10 {
			break
		}
	}

	return offsetList, nil
}
