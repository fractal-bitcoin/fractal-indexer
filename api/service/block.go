package service

import (
	"database/sql"
	"errors"
	"fmt"
	"fractal-query/constant"
	"fractal-query/dao/clickhouse"
	"fractal-query/dao/rdb"
	"fractal-query/lib/blkparser"
	"fractal-query/logger"
	"fractal-query/model"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

const (
	SQL_FIELEDS_BEST_BLOCK = "height, version, blkid, previd, '0000000000000000000000000000000000000000000000000000000000000000', merkle, ntx, invalue, outvalue, coinbase_out, blocktime, bits, blocksize"
	SQL_FIELEDS_BLOCK      = "height, version, blkid, previd, next_blk.blkid, merkle, ntx, invalue, outvalue, coinbase_out, blocktime, bits, blocksize"
)

func blockResultSRF(rows *sql.Rows) (interface{}, error) {
	var ret model.BlockDO
	err := rows.Scan(&ret.Height, &ret.Version, &ret.BlockId, &ret.PrevBlockId, &ret.NextBlockId, &ret.MerkleRoot, &ret.TxCount, &ret.InSatoshi, &ret.OutSatoshi, &ret.CoinbaseOut, &ret.BlockTime, &ret.Bits, &ret.BlockSize)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func blockFeeRateResultSRF(rows *sql.Rows) (interface{}, error) {
	var ret model.BlockFeeRateDO
	err := rows.Scan(&ret.Height, &ret.BlockTime, &ret.FeeRate)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func GetBlocksByHeightRange(blkStartHeight, blkEndHeight int) (blksRsp []*model.BlockInfoResp, err error) {
	psql := fmt.Sprintf(`
SELECT %s FROM blk_height
LEFT JOIN (
    SELECT blkid, previd FROM blk_height
    WHERE height > %d AND height <= %d
    LIMIT %d
) AS next_blk
ON blk_height.blkid = next_blk.previd
WHERE height >= %d AND height < %d ORDER BY height ASC
LIMIT %d
`,
		SQL_FIELEDS_BLOCK, blkStartHeight, blkEndHeight, blkEndHeight-blkStartHeight,
		blkStartHeight, blkEndHeight, blkEndHeight-blkStartHeight)

	blksRet, err := clickhouse.ScanAll(psql, blockResultSRF)
	if err != nil {
		logger.Log.Error("query blk failed", zap.Error(err))
		return nil, err
	}
	if blksRet == nil {
		return nil, errors.New("not exist")
	}
	blocks := blksRet.([]*model.BlockDO)
	for _, block := range blocks {
		blksRsp = append(blksRsp, &model.BlockInfoResp{
			Height:         int(block.Height),
			Version:        int(block.Version),
			AuxPow:         block.Version&constant.AUXPOW_VERSON_MASK > 0,
			BlockIdHex:     blkparser.HashString(block.BlockId),
			PrevBlockIdHex: blkparser.HashString(block.PrevBlockId),
			NextBlockIdHex: blkparser.HashString(block.NextBlockId),
			MerkleRootHex:  blkparser.HashString(block.MerkleRoot),
			TxCount:        int(block.TxCount),
			InSatoshi:      int(block.InSatoshi),
			OutSatoshi:     int(block.OutSatoshi),
			CoinbaseOut:    int(block.CoinbaseOut),

			BlockTime: int(block.BlockTime),
			Bits:      int(block.Bits),
			BlockSize: int(block.BlockSize),
		})
	}
	return

}

func GetBlockByHeight(blkHeight int) (blk *model.BlockInfoResp, err error) {
	psql := fmt.Sprintf(`
SELECT %s FROM blk_height
LEFT JOIN (
    SELECT blkid, previd FROM blk_height
    WHERE height = %d+1
    LIMIT 1
) AS next_blk
ON blk_height.blkid = next_blk.previd
WHERE height = %d ORDER BY height ASC
LIMIT 1`, SQL_FIELEDS_BLOCK, blkHeight, blkHeight)
	return GetBlockBySql(psql)
}

func GetBlockById(blkidHex string) (blk *model.BlockInfoResp, err error) {
	psql := fmt.Sprintf(`SELECT %s FROM blk
LEFT JOIN (
    SELECT blkid, previd FROM blk_height
    WHERE height IN (
       SELECT toUInt32(height+1) FROM blk
       WHERE blkid = unhex('%s')
       LIMIT 1
    )
) AS next_blk
ON blk.blkid = next_blk.previd
WHERE blkid = unhex('%s')
LIMIT 1`, SQL_FIELEDS_BLOCK, blkidHex, blkidHex)
	return GetBlockBySql(psql)
}

func GetBestBlockByHeight(blkHeight int) (blk *model.BlockInfoResp, err error) {
	psql := fmt.Sprintf("SELECT %s FROM blk_height WHERE height = %d LIMIT 1", SQL_FIELEDS_BEST_BLOCK, blkHeight)
	return GetBlockBySql(psql)
}

func GetBlockBySql(psql string) (blk *model.BlockInfoResp, err error) {
	blkRet, err := clickhouse.ScanOne(psql, blockResultSRF)
	if err != nil {
		logger.Log.Info("query blk failed", zap.Error(err))
		return nil, err
	}
	if blkRet == nil {
		return nil, errors.New("not exist")
	}
	block := blkRet.(*model.BlockDO)
	blk = &model.BlockInfoResp{
		Height:         int(block.Height),
		Version:        int(block.Version),
		AuxPow:         block.Version&constant.AUXPOW_VERSON_MASK > 0,
		BlockIdHex:     blkparser.HashString(block.BlockId),
		PrevBlockIdHex: blkparser.HashString(block.PrevBlockId),
		NextBlockIdHex: blkparser.HashString(block.NextBlockId),
		MerkleRootHex:  blkparser.HashString(block.MerkleRoot),
		TxCount:        int(block.TxCount),
		InSatoshi:      int(block.InSatoshi),
		OutSatoshi:     int(block.OutSatoshi),
		CoinbaseOut:    int(block.CoinbaseOut),

		BlockTime: int(block.BlockTime),
		Bits:      int(block.Bits),
		BlockSize: int(block.BlockSize),
	}
	return
}

func mempoolResultSRF(rows *sql.Rows) (interface{}, error) {
	var ret int
	err := rows.Scan(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func GetBlockMedianTimePast(height int) (mtp int, err error) {
	psql := fmt.Sprintf(`
SELECT toUInt32(quantileExact(blocktime)) FROM (
    SELECT blocktime FROM blk_height WHERE height > %d AND height <= %d
)
`, height-11, height)

	blkRet, err := clickhouse.ScanOne(psql, mempoolResultSRF)
	if err != nil {
		logger.Log.Info("query mtp failed", zap.Error(err))
		return 0, err
	}
	if blkRet == nil {
		return 0, errors.New("not exist")
	}
	mtp = blkRet.(int)

	return mtp, nil
}

func GetBestBlockHeight() (height int, err error) {
	// get decimal from f info
	height, err = rdb.RdbRedisClient.HGet(ctx, "info", "block_height").Int()
	if err == redis.Nil {
		height = 0
		logger.Log.Info("GetBestBlockHeight, but info missing")
	} else if err != nil {
		logger.Log.Info("GetBestBlockHeight, but redis failed", zap.Error(err))
		return
	}

	return height, nil
}

func GetBestBlockId() (blockId string, err error) {
	// get decimal from f info
	blockId, err = rdb.RdbRedisClient.HGet(ctx, "info", "block").Result()
	if err == redis.Nil {
		blockId = ""
		logger.Log.Info("GetBestBlockId, but info missing")
	} else if err != nil {
		logger.Log.Info("GetBestBlockId, but redis failed", zap.Error(err))
		return
	}

	return blockId, nil
}

func GetBlocksFeeByHeightRange(count int) (blksFeeRateRsp []*model.BlockFeeRateResp, err error) {
	psql := fmt.Sprintf(`
SELECT height, blocktime, (invalue - outvalue)/(blocksize-witsize+witsize/4) FROM blk_height
ORDER BY height DESC
LIMIT %d
`,
		count)

	blksRet, err := clickhouse.ScanAll(psql, blockFeeRateResultSRF)
	if err != nil {
		logger.Log.Info("query blk failed", zap.Error(err))
		return nil, err
	}
	if blksRet == nil {
		return nil, errors.New("not exist")
	}
	blocks := blksRet.([]*model.BlockFeeRateDO)
	for _, block := range blocks {
		blksFeeRateRsp = append(blksFeeRateRsp, &model.BlockFeeRateResp{
			Height:    int(block.Height),
			BlockTime: int(block.BlockTime),
			FeeRate:   block.FeeRate,
		})
	}
	return
}

func blockHashResultSRF(rows *sql.Rows) (interface{}, error) {
	var ret model.BlockDO
	err := rows.Scan(&ret.BlockId, &ret.BlockTime)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func GetAllBlocksHashAndTime() (blockHashs []string, blocksTime []uint32, err error) {
	psql := fmt.Sprintf(`
SELECT blkid, blocktime FROM blk_height
ORDER BY height
`)

	blksRet, err := clickhouse.ScanAll(psql, blockHashResultSRF)
	if err != nil {
		logger.Log.Info("query blk failed", zap.Error(err))
		return nil, nil, err
	}
	if blksRet == nil {
		return nil, nil, errors.New("not exist")
	}
	blocks := blksRet.([]*model.BlockDO)
	for _, block := range blocks {
		blockHashs = append(blockHashs, string(block.BlockId))
		blocksTime = append(blocksTime, block.BlockTime)
	}
	return
}

func GetLastBlocksHashAndTime() (blockHashs []string, blocksTime []uint32, err error) {
	psql := fmt.Sprintf(`
SELECT blkid, blocktime FROM blk_height
ORDER BY height DESC
LIMIT 1
`)

	blksRet, err := clickhouse.ScanAll(psql, blockHashResultSRF)
	if err != nil {
		logger.Log.Info("query blk failed", zap.Error(err))
		return nil, nil, err
	}
	if blksRet == nil {
		return nil, nil, errors.New("not exist")
	}
	blocks := blksRet.([]*model.BlockDO)
	for _, block := range blocks {
		blockHashs = append(blockHashs, string(block.BlockId))
		blocksTime = append(blocksTime, block.BlockTime)
	}
	return
}
