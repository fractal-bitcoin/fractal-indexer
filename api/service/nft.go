package service

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"fractal-indexer/api/constant"
	"fractal-indexer/api/dao/clickhouse"
	"fractal-indexer/api/dao/rdb"
	mtx "fractal-indexer/api/lib/midware"
	"fractal-indexer/api/lib/utils"
	"fractal-indexer/api/logger"
	"fractal-indexer/api/model"
	"sort"

	redis "github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// fixme: np
// getInscriptionPointFromRedis gets NFT details from a list of NFT create_idx keys.
func getInscriptionPointFromRedis(inscriptionCreateIdxs []uint64) (nftsRsp []*model.InscriptionResp, err error) {
	logger.Log.Info("getInscriptionPointFromRedis", zap.Int("nCreateIdxs", len(inscriptionCreateIdxs)))
	nftsRsp = make([]*model.InscriptionResp, 0)
	pipe := rdb.RdbRedisClient.Pipeline()

	createIdxsCmd := make([]*redis.StringCmd, 0)
	for _, createIdx := range inscriptionCreateIdxs {
		createIdxsCmd = append(createIdxsCmd, pipe.Get(ctx, fmt.Sprintf("mp:np%x", createIdx)))
		createIdxsCmd = append(createIdxsCmd, pipe.Get(ctx, fmt.Sprintf("np%x", createIdx)))

		nftsRsp = append(nftsRsp, &model.InscriptionResp{
			CreateIdxKey: createIdx,
		})
	}
	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		mtx.ReportError(mtx.ErrorRedis)
		panic(err)
	}

	for idx, nft := range nftsRsp {
		// check mempool first
		data := createIdxsCmd[idx*2]
		res, err := data.Result()
		if err == redis.Nil {
			{
				// if mempool not exist, then load block
				data := createIdxsCmd[idx*2+1]
				res, err := data.Result()
				if err == redis.Nil {
					continue
				} else if err != nil {
					mtx.ReportError(mtx.ErrorRedis)
					panic(err)
				}
				nft.UtxoOutpoint = res[:36]
				nft.Offset = binary.LittleEndian.Uint64([]byte(res[36:]))
			}
			continue
		} else if err != nil {
			mtx.ReportError(mtx.ErrorRedis)
			panic(err)
		}
		nft.UtxoOutpoint = res[:36]
		nft.Offset = binary.LittleEndian.Uint64([]byte(res[36:]))
	}
	return nftsRsp, nil
}

// GetCreateIdxByInscriptionIdFromRedis gets create_idx keys by NFT ID.
func GetCreateIdxByInscriptionIdFromRedis(nftIdList []string) (inscriptionCreatePoints []model.NFTCreatePoint, err error) {
	logger.Log.Info("getCreateIdxByInscriptionIdFromRedis", zap.Int("n", len(nftIdList)))
	pipe := rdb.RdbRedisClient.Pipeline()

	createIdxsCmd := make([]*redis.StringCmd, 0)
	for _, nftId := range nftIdList {
		createIdxsCmd = append(createIdxsCmd, pipe.Get(ctx, nftId))
	}
	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		mtx.ReportError(mtx.ErrorRedis)
		logger.Log.Error("redis failed", zap.Error(err))
		return
	}

	for createIdxIdx, data := range createIdxsCmd {
		createIdx, err := data.Uint64()
		if err != nil {
			nftId := nftIdList[createIdxIdx]
			logger.Log.Info("nftid to createidx not found from utxo rdb", zap.String("inscription", nftId))
			inscriptionCreatePoints = append(inscriptionCreatePoints, model.NFTCreatePoint{})
			continue
		}
		point := model.NFTCreatePoint{
			Height:     uint32(createIdx >> constant.HEIGHT_MUTIPLY_NBIT),
			IdxInBlock: uint32(createIdx & constant.HEIGHT_MUTIPLY_MASK),
		}

		inscriptionCreatePoints = append(inscriptionCreatePoints, point)
	}

	return inscriptionCreatePoints, nil
}

// GetInscriptionBodyFromRedis gets NFT details from a list of NFT create_idx keys.
func GetInscriptionBodyFromRedis(inscriptionCreateIdxs []uint64) (nftsRsp []*model.InscriptionResp, err error) {
	nftsRsp = make([]*model.InscriptionResp, 0)
	if len(inscriptionCreateIdxs) == 0 {
		return nftsRsp, nil
	}

	logger.Log.Info("GetInscriptionBodyFromRedis", zap.Int("nCreateIdxs", len(inscriptionCreateIdxs)))
	pipe := rdb.RdbRedisClient.Pipeline()

	createIdxsCmd := make([]*redis.StringCmd, 0)
	for _, createIdx := range inscriptionCreateIdxs {
		createIdxsCmd = append(createIdxsCmd, pipe.Get(ctx, fmt.Sprintf("nb%x", createIdx)))
	}
	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		mtx.ReportError(mtx.ErrorRedis)
		panic(err)
	}

	for createIdxIdx, data := range createIdxsCmd {
		createIdx := inscriptionCreateIdxs[createIdxIdx]
		res, err := data.Result()
		if err == redis.Nil {
			logger.Log.Info("nft body not found", zap.Uint64("createIdx", createIdx))
			nftsRsp = append(nftsRsp, &model.InscriptionResp{})
			continue
		} else if err != nil {
			mtx.ReportError(mtx.ErrorRedis)
			panic(err)
		}

		nftData := []byte(res)

		var d model.NewInscriptionInfo
		d.LoadString(nftData)
		// fixme: loadstring
		d.CreatePoint.IdxInBlock = uint32(createIdx & constant.HEIGHT_MUTIPLY_MASK)

		nftsRsp = append(nftsRsp, &model.InscriptionResp{
			InscriptionNumber: int64(d.Number),
			InscriptionId:     fmt.Sprintf("%si%d", hex.EncodeToString(utils.ReverseBytes(d.TxId)), d.IdxInTx),

			IsStrip:            d.NFTData.IsStrip,
			HasPointer:         d.NFTData.HasPointer,
			HasParent:          d.NFTData.HasParent,
			HasDeligate:        d.NFTData.HasDeligate,
			HasMetaProtocal:    d.NFTData.HasMetaProtocal,
			HasMetadata:        d.NFTData.HasMetadata,
			HasContentEncoding: d.NFTData.HasContentEncoding,
			Pointer:            d.NFTData.Pointer,
			Parent:             d.NFTData.GetParentId(),
			Deligate:           d.NFTData.GetDeligateID(),
			MetaProtocol:       string(d.NFTData.MetaProtocol),
			Metadata:           string(d.NFTData.Metadata),
			ContentEncoding:    string(d.NFTData.ContentEncoding),
			ContentType:        string(d.NFTData.ContentType),
			ContentLength:      int(d.ContentLength),
			Height:             d.CreatePoint.Height,
			IdxInBlock:         d.CreatePoint.IdxInBlock,
			BlockTime:          int(d.BlockTime),
			InSatoshi:          int(d.InputsValue),
			OutSatoshi:         int(d.OutputsValue),
		})
	}

	return nftsRsp, nil
}

func GetInscriptionsByCreateIdxes(inscriptionCreateIdxes []uint64) (nftsRsp []*model.InscriptionResp, err error) {
	// Get inscription details from a list of inscription keys.
	nftsRsp, err = getInscriptionPointFromRedis(inscriptionCreateIdxes)
	if err != nil {
		return nil, err
	}

	var utxoOutpoints []string
	for _, nft := range nftsRsp {
		utxoOutpoints = append(utxoOutpoints, nft.UtxoOutpoint)
	}
	// Get UTXO details from a list of UTXO keys.
	txOutsRsp, err := getNonTokenUtxoFromRedis("get nft utxo by pointer(must found)", utxoOutpoints, false)
	if err != nil {
		return nil, err
	}

	nftBodyRsp, err := GetInscriptionBodyFromRedis(inscriptionCreateIdxes)
	if err != nil {
		logger.Log.Info("redis nftsNumber not found")
		return nil, err
	}

	for idx, nft := range nftsRsp {
		nft.InscriptionNumber = nftBodyRsp[idx].InscriptionNumber
		nft.InscriptionId = nftBodyRsp[idx].InscriptionId

		nft.IsStrip = nftBodyRsp[idx].IsStrip
		nft.HasPointer = nftBodyRsp[idx].HasPointer
		nft.HasParent = nftBodyRsp[idx].HasParent
		nft.HasDeligate = nftBodyRsp[idx].HasDeligate
		nft.HasMetaProtocal = nftBodyRsp[idx].HasMetaProtocal
		nft.HasMetadata = nftBodyRsp[idx].HasMetadata
		nft.HasContentEncoding = nftBodyRsp[idx].HasContentEncoding
		nft.Pointer = nftBodyRsp[idx].Pointer
		nft.Parent = nftBodyRsp[idx].Parent
		nft.Deligate = nftBodyRsp[idx].Deligate
		nft.MetaProtocol = nftBodyRsp[idx].MetaProtocol
		nft.Metadata = nftBodyRsp[idx].Metadata
		nft.ContentEncoding = nftBodyRsp[idx].ContentEncoding

		nft.ContentType = nftBodyRsp[idx].ContentType
		nft.ContentLength = nftBodyRsp[idx].ContentLength
		nft.Height = nftBodyRsp[idx].Height
		nft.IdxInBlock = nftBodyRsp[idx].IdxInBlock
		nft.BlockTime = nftBodyRsp[idx].BlockTime
		nft.InSatoshi = nftBodyRsp[idx].InSatoshi
		nft.OutSatoshi = nftBodyRsp[idx].OutSatoshi

		if len(nftsRsp) == len(txOutsRsp) && txOutsRsp[idx] != nil {
			nft.UTXO = txOutsRsp[idx]
			nft.Address = nft.UTXO.Address

			for _, nftpoint := range nft.UTXO.CreatePointOfNFTs {
				if nftpoint.Height != nft.Height {
					continue
				}
				if nftpoint.IdxInBlock != nft.IdxInBlock {
					continue
				}
				nft.UTXO.Inscriptions = append(nft.UTXO.Inscriptions, &model.NFTData{
					InscriptionNumber: nftBodyRsp[idx].InscriptionNumber,
					InscriptionId:     nftBodyRsp[idx].InscriptionId,
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
				break
			}
		}
	}

	return nftsRsp, nil
}

func GetSortInscriptionsByCreateIdxes(inscriptionCreateIdxes []uint64, cursor, size int, sortType string) (nftsRsp []*model.InscriptionResp, err error) {
	nftsRsp, err = GetInscriptionsByCreateIdxes(inscriptionCreateIdxes)
	if err != nil {
		return
	}
	if sortType == "asc" {
		sort.Slice(nftsRsp, func(i, j int) bool {
			return nftsRsp[i].InscriptionNumber < nftsRsp[j].InscriptionNumber
		})
	} else {
		sort.Slice(nftsRsp, func(i, j int) bool {
			return nftsRsp[i].InscriptionNumber > nftsRsp[j].InscriptionNumber
		})
	}

	if cursor > len(nftsRsp)-1 {
		return []*model.InscriptionResp{}, nil
	}
	nftsRsp = nftsRsp[cursor:]
	if len(nftsRsp) > size {
		nftsRsp = nftsRsp[:size]
	}

	return nftsRsp, nil
}

func GetNftCreatePointByNftNumber(strNftNumber string) (oCreatePoint *model.NFTCreatePoint, err error) {
	psql := fmt.Sprintf(`
		SELECT height, nftidx 
		FROM mv_nftnumber_to_height_nftidx
		WHERE nftnumber=?
		order by nftnumber desc
	`)

	nftsRet, err := clickhouse.ScanOne(psql, inscriptionResultSRF, strNftNumber)
	if err != nil {
		logger.Log.Error("query nft idx failed", zap.Error(err))
		return nil, err
	}
	if nftsRet == nil {
		return nil, errors.New("not exist")
	}
	oCreatePoint = nftsRet.(*model.NFTCreatePoint)
	return
}

func GetInscriptionsStatus() (inscriptionStatusResp *model.InscriptionStatusResp, err error) {
	count := int64(0)
	cursedCount := int64(0)
	result, err := rdb.RdbRedisClient.ZRevRangeWithScores(ctx, constant.ORDINALS_INSCRIPTION_COUNTS_BY_HEIGHT, 0, 0).Result()
	if err != nil && err != redis.Nil {
		logger.Log.Error("query nft counts failed", zap.Error(err))
		return nil, err
	}
	if len(result) != 0 {
		count = int64(result[0].Score)
	}

	resultCursed, err := rdb.RdbRedisClient.ZRevRangeWithScores(ctx, constant.ORDINALS_INSCRIPTION_CURSED_COUNTS_BY_HEIGHT, 0, 0).Result()
	if err != nil && err != redis.Nil {
		logger.Log.Error("query nft counts failed", zap.Error(err))
		return nil, err
	}
	if len(resultCursed) != 0 {
		cursedCount = int64(resultCursed[0].Score)
	}

	psql := `
		SELECT MAX(nftnumber), MIN(nftnumber) FROM mv_nftnumber_to_height_nftidx WHERE height = ?;
	`
	ret, err := clickhouse.ScanOne(psql, inscriptionsStatusResultSRF, constant.MEMPOOL_HEIGHT)
	if err != nil {
		logger.Log.Error("query inscriptions count failed", zap.Error(err))
		return nil, err
	}

	if ret == nil {
		return nil, errors.New("not exist")
	}

	rangeDo := ret.(*model.InscriptionNumberRangeDO)

	lastNumber := max(count-1, rangeDo.MaxNumber)

	lastCursedNumber := int64(0)
	if cursedCount > 0 {
		lastCursedNumber = -(cursedCount - 1)
	}
	if rangeDo.MinNumber < 0 {
		lastCursedNumber = min(lastCursedNumber, rangeDo.MinNumber)
	}

	// Cursed inscription numbers start from -1, so no +1 is needed.
	inscriptionStatusResp = &model.InscriptionStatusResp{
		Count:            lastNumber + 1 + (-lastCursedNumber),
		LastNumber:       lastNumber,
		LastCursedNumber: lastCursedNumber,
	}
	return inscriptionStatusResp, nil
}
