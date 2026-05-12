package events

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"fractal-query/constant"
	"fractal-query/dao/clickhouse"
	"fractal-query/dao/rdb"
	scriptDecoder "fractal-query/lib/blkparser/script"
	"fractal-query/lib/utils"
	"fractal-query/logger"
	"fractal-query/model"
	"fractal-query/service"
	"strings"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

func latestEventInscriptionCountResultSRF(rows *sql.Rows) (interface{}, error) {
	var ret int
	err := rows.Scan(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func GetInscriptionEventsCount(blkStartHeight, blkEndHeight int, isBrc20 bool) (count int, err error) {
	whereExpr := fmt.Sprintf("height >= %d", blkStartHeight)
	if blkEndHeight != 0 {
		whereExpr = fmt.Sprintf("height >= %d AND height < %d", blkStartHeight, blkEndHeight)
	}

	if isBrc20 {
		whereExpr += " AND nfttype=3"
	}

	psql := fmt.Sprintf(`SELECT count(1) FROM blkevent_height WHERE %s`, whereExpr)
	blkRet, err := clickhouse.ScanOne(psql, latestEventInscriptionCountResultSRF)
	if err != nil {
		logger.Log.Error("query event count failed", zap.Error(err))
		return 0, err
	}
	if blkRet == nil {
		return 0, nil
	}
	count = blkRet.(int)
	return count, nil
}

// by nft point
func GetInscriptionEventsCountByCreatePoint(nftPoint model.NFTCreatePoint) (count int, err error) {
	psql := fmt.Sprintf(sqlCountInscriptionEvents, nftPoint.IdxInBlock, nftPoint.Height, nftPoint.Height, nftPoint.IdxInBlock, nftPoint.Height)
	logger.Log.Info("query event count sql", zap.String("sql", psql))
	blkRet, err := clickhouse.ScanOne(psql, latestEventInscriptionCountResultSRF)
	if err != nil {
		logger.Log.Error("query event count failed", zap.Error(err))
		return 0, err
	}
	if blkRet == nil {
		return 0, nil
	}
	count = blkRet.(int)
	return count, nil
}

func inscriptionEventResultSRF(rows *sql.Rows) (interface{}, error) {
	var event model.InscriptionEventData
	var ret model.NFTCreatePoint
	var contentCode uint8
	var contentBody string
	err := rows.Scan(&ret.Height, &event.TxIdx, &event.Height, &ret.IdxInBlock, &event.InscriptionNumber, &event.Sequence, // position
		&event.TxId, &event.Idx, // inscriptionId
		&event.Vout,
		&event.Offset,
		&event.ContentType, &contentCode, &contentBody, // content
		&event.Satoshi, &event.PkScript, // owner, and value
		&event.PkScriptFrom, &event.InputIdx,
		&event.BlockTime,  // block time
		&event.InSatoshi,  //
		&event.OutSatoshi, //
	)
	if err != nil {
		return nil, err
	}

	event.ContentBody = decodeNFTEventContent(contentCode, contentBody)

	// sending transfer-function
	if event.Height != 0 {
		event.IsTransfer = true
		event.Height, ret.Height = ret.Height, event.Height
	} else {
		event.Height = ret.Height
	}
	event.CreateIdxKey = ret.GetCreateIdxKey()

	return &event, nil
}

func GetInscriptionEventsByHeightRange(cursor, size, blkStartHeight, blkEndHeight int, isBrc20 bool) (nftsRsp []*model.InscriptionEventResp, err error) {
	whereExpr := fmt.Sprintf("height >= %d", blkStartHeight)
	if blkEndHeight != 0 {
		whereExpr = fmt.Sprintf("height >= %d AND height < %d", blkStartHeight, blkEndHeight)
	}

	if isBrc20 {
		whereExpr += " AND nfttype=3"
	}

	psql := fmt.Sprintf(`
	SELECT height, txidx, nftheight, nftidx, nftnumber, sequence, txid, idx, vout, offset, content_type, content_code, content, satoshi, script_pk, script_pk_from, input_idx, blocktime, invalue, outvalue FROM blkevent_height
	WHERE %s
	ORDER BY height, eventidx
	LIMIT %d, %d
`, whereExpr, cursor, size)

	nftsRet, err := clickhouse.ScanAll(psql, inscriptionEventResultSRF)
	if err != nil {
		logger.Log.Error("query event content failed", zap.Error(err))
		return nil, err
	}

	nftsRsp = make([]*model.InscriptionEventResp, 0)
	if nftsRet == nil {
		return nftsRsp, nil
	}

	inscriptionCreateIdxes := []uint64{}

	for _, nft := range nftsRet.([]*model.InscriptionEventData) {

		inscriptionCreateIdxes = append(inscriptionCreateIdxes, nft.CreateIdxKey)

		scriptType := scriptDecoder.GetLockingScriptType([]byte(nft.PkScript))
		addressData := scriptDecoder.ExtractPkScriptForTxo([]byte(nft.PkScript), scriptType)

		scriptTypeFrom := scriptDecoder.GetLockingScriptType([]byte(nft.PkScriptFrom))
		addressFromData := scriptDecoder.ExtractPkScriptForTxo([]byte(nft.PkScriptFrom), scriptTypeFrom)
		nftsRsp = append(nftsRsp, &model.InscriptionEventResp{
			IsTransfer: nft.IsTransfer,
			TxId:       utils.GetReversedStringHex(nft.TxId),
			Idx:        nft.Idx,
			Vout:       nft.Vout,
			Offset:     nft.Offset,
			Sequence:   nft.Sequence,

			InscriptionNumber: nft.InscriptionNumber,
			InscriptionId:     fmt.Sprintf("%si%d", utils.GetReversedStringHex(nft.TxId), nft.Idx),
			Address:           utils.EncodeAddressByCodeType([]byte(nft.PkScript), addressData.CodeType),
			Satoshi:           int(nft.Satoshi),
			PkScriptHex:       hex.EncodeToString([]byte(nft.PkScript)),
			PkScriptFromHex:   hex.EncodeToString([]byte(nft.PkScriptFrom)),
			AddressFrom:       utils.EncodeAddressByCodeType([]byte(nft.PkScriptFrom), addressFromData.CodeType),
			InputIdx:          nft.InputIdx,

			ContentType: nft.ContentType,
			ContentBody: nft.ContentBody,
			BlockTime:   int(nft.BlockTime),
			InSatoshi:   int(nft.InSatoshi),
			OutSatoshi:  int(nft.OutSatoshi),

			Height: int(nft.Height),
			TxIdx:  int(nft.TxIdx),
		})
	}

	nftBodyRsp, err := service.GetInscriptionBodyFromRedis(inscriptionCreateIdxes)
	if err != nil {
		logger.Log.Info("redis number/id not found")
		return nil, err
	}

	for idx, nft := range nftsRsp {
		nft.InscriptionNumber = nftBodyRsp[idx].InscriptionNumber
		nft.InscriptionId = nftBodyRsp[idx].InscriptionId
	}
	return
}

func GetInscriptionEventsByHeightRangeAndCreatePoint(cursor, size int, sortString string, nftPoint model.NFTCreatePoint) (nftsRsp []*model.InscriptionEventResp, err error) {
	psql := fmt.Sprintf(sqlSelectInscriptionEvents, nftPoint.IdxInBlock, nftPoint.Height, nftPoint.Height, nftPoint.IdxInBlock, sortString, nftPoint.Height, sortString, sortString, cursor, size)
	logger.Log.Info("query event content sql", zap.String("sql", psql))

	nftsRet, err := clickhouse.ScanAll(psql, inscriptionEventResultSRF)
	if err != nil {
		logger.Log.Error("query event content failed", zap.Error(err))
		return nil, err
	}

	nftsRsp = make([]*model.InscriptionEventResp, 0)
	if nftsRet == nil {
		return nftsRsp, nil
	}

	for _, nft := range nftsRet.([]*model.InscriptionEventData) {
		scriptType := scriptDecoder.GetLockingScriptType([]byte(nft.PkScript))
		addressData := scriptDecoder.ExtractPkScriptForTxo([]byte(nft.PkScript), scriptType)

		scriptTypeFrom := scriptDecoder.GetLockingScriptType([]byte(nft.PkScriptFrom))
		addressFromData := scriptDecoder.ExtractPkScriptForTxo([]byte(nft.PkScriptFrom), scriptTypeFrom)
		nftsRsp = append(nftsRsp, &model.InscriptionEventResp{
			IsTransfer: nft.IsTransfer,
			TxId:       utils.GetReversedStringHex(nft.TxId),
			Idx:        nft.Idx,
			Vout:       nft.Vout,
			Offset:     nft.Offset,
			Sequence:   nft.Sequence,

			InscriptionNumber: nft.InscriptionNumber,
			InscriptionId:     fmt.Sprintf("%si%d", utils.GetReversedStringHex(nft.TxId), nft.Idx),
			Address:           utils.EncodeAddressByCodeType([]byte(nft.PkScript), addressData.CodeType),
			Satoshi:           int(nft.Satoshi),
			PkScriptHex:       hex.EncodeToString([]byte(nft.PkScript)),
			PkScriptFromHex:   hex.EncodeToString([]byte(nft.PkScriptFrom)),
			AddressFrom:       utils.EncodeAddressByCodeType([]byte(nft.PkScriptFrom), addressFromData.CodeType),
			InputIdx:          nft.InputIdx,

			ContentType: nft.ContentType,
			ContentBody: nft.ContentBody,
			BlockTime:   int(nft.BlockTime),
			InSatoshi:   int(nft.InSatoshi),
			OutSatoshi:  int(nft.OutSatoshi),

			Height: int(nft.Height),
			TxIdx:  int(nft.TxIdx),
		})
	}

	inscriptionCreateIdxes := []uint64{nftPoint.GetCreateIdxKey()}
	nftBodyRsp, err := service.GetInscriptionBodyFromRedis(inscriptionCreateIdxes)
	if err != nil {
		logger.Log.Info("redis number/id not found")
		return nil, err
	}

	for _, nft := range nftsRsp {
		nft.InscriptionNumber = nftBodyRsp[0].InscriptionNumber
		nft.InscriptionId = nftBodyRsp[0].InscriptionId
	}
	return
}

// Query the block height by nftid; return -1 if the NFT does not exist.
func GetNFTBlockHeightByNFTId(ctx context.Context, nftIds ...string) (map[string]int, error) {
	if len(nftIds) == 0 {
		return nil, nil
	}
	pipe := rdb.RdbRedisClient.Pipeline()
	cmds := make([]*redis.StringCmd, 0, len(nftIds))
	for _, nftId := range nftIds {
		cmds = append(cmds, pipe.Get(ctx, nftId))
	}
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		logger.Log.Error("GetNFTBlockHeightByNFTId failed to pipe.Exec", zap.Error(err), zap.String("nftIds", strings.Join(nftIds, ",")))
		return nil, err
	}

	nftHeight := make(map[string]int, len(nftIds))
	for idx, cmd := range cmds {
		createIdx, err := cmd.Uint64()
		if err == redis.Nil {
			nftHeight[nftIds[idx]] = -1 // not found
			continue
		}
		if err != nil {
			logger.Log.Error("GetNFTBlockHeightByNFTId failed", zap.String("inscription", nftIds[idx]), zap.Error(err))
			return nil, err
		}
		nftHeight[nftIds[idx]] = int(createIdx >> constant.HEIGHT_MUTIPLY_NBIT)
	}
	return nftHeight, nil
}
