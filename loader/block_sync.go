package loader

import (
	"context"
	"errors"
	"fmt"
	"fractal-indexer/constant"
	"fractal-indexer/loader/clickhouse"
	"fractal-indexer/logger"
	"fractal-indexer/model"
	"fractal-indexer/rdb"

	redis "github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

func GetLatestBlockFromDB() (blkRsp *model.BlockDO, err error) {
	psql := "SELECT height, blkid FROM blk_height ORDER BY height DESC LIMIT 1"

	// Use v2 driver's direct query
	rows, err := clickhouse.Query(context.Background(), psql)
	if err != nil {
		logger.Log.Info("query blk failed", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var ret model.BlockDO
		var blkidStr string
		// ClickHouse v2 driver returns FixedString as string (raw binary data)
		if err := rows.Scan(&ret.Height, &blkidStr); err != nil {
			return nil, err
		}
		// Store as raw bytes (32 bytes binary data)
		ret.BlockId = []byte(blkidStr)
		return &ret, nil
	}

	return nil, errors.New("not exist")
}

func GetBlockListFromDB(startHeight, endHeight uint32) (blkRsp []*model.BlockDO, err error) {
	psql := fmt.Sprintf("SELECT height, blkid FROM blk_height WHERE height >= %d AND height < %d ORDER BY height", startHeight, endHeight)

	// Use v2 driver's direct query
	rows, err := clickhouse.Query(context.Background(), psql)
	if err != nil {
		logger.Log.Info("query blk failed", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var results []*model.BlockDO
	for rows.Next() {
		var ret model.BlockDO
		var blkidStr string
		// ClickHouse v2 driver returns FixedString as string (raw binary data)
		if err := rows.Scan(&ret.Height, &blkidStr); err != nil {
			return nil, err
		}
		// Store as raw bytes (32 bytes binary data)
		ret.BlockId = []byte(blkidStr)
		results = append(results, &ret)
	}

	if len(results) == 0 {
		return nil, errors.New("not exist")
	}

	return results, nil
}

func GetBlockInfoFromDB(height uint32) (blkRsp *model.BlockDO, err error) {
	psql := fmt.Sprintf("SELECT height, blkid FROM blk_height WHERE height = %d", height)

	// Use v2 driver's direct query
	rows, err := clickhouse.Query(context.Background(), psql)
	if err != nil {
		logger.Log.Info("query blk failed", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var ret model.BlockDO
		var blkidStr string
		// ClickHouse v2 driver returns FixedString as string (raw binary data)
		if err := rows.Scan(&ret.Height, &blkidStr); err != nil {
			return nil, err
		}
		// Store as raw bytes (32 bytes binary data)
		ret.BlockId = []byte(blkidStr)
		return &ret, nil
	}

	return nil, errors.New("not exist")
}

func GetIntFromDB(psql string) (result int, err error) {
	// Use v2 driver's direct query
	rows, err := clickhouse.Query(context.Background(), psql)
	if err != nil {
		logger.Log.Info("query blk failed", zap.Error(err))
		return 0, err
	}
	defer rows.Close()

	if rows.Next() {
		var ret int
		if err := rows.Scan(&ret); err != nil {
			return 0, err
		}
		return ret, nil
	}

	return 0, errors.New("not exist")
}

func GetBestBlockHeightFromRedis() (height int, err error) {
	// get decimal from f info
	ctx := context.Background()
	height, err = rdb.RdbClient.HGet(ctx, constant.TASK_INFO_KEYNAME, constant.TASK_BLOCK_HEIGHT).Int()
	if err == redis.Nil {
		height = 0
		logger.Log.Info("GetBestBlockHeightFromRedis, but info missing")
	} else if err != nil {
		logger.Log.Info("GetBestBlockHeightFromRedis, but redis failed", zap.Error(err))
		return
	}

	return height, nil
}

func GetBestBlockIdFromRedis() (blockId string, err error) {
	// get decimal from f info
	ctx := context.Background()
	blockId, err = rdb.RdbClient.HGet(ctx, constant.TASK_INFO_KEYNAME, constant.TASK_BLOCK).Result()
	if err == redis.Nil {
		blockId = ""
		logger.Log.Info("GetBestBlockIdFromRedis, but info missing")
	} else if err != nil {
		logger.Log.Info("GetBestBlockIdFromRedis, but redis failed", zap.Error(err))
		return
	}

	return blockId, nil
}

// GetSpentUTXOFromRevert returns spent UTXOs from the revert table (op=0).
// Returns map[outpoint_key]marshaled_TxoData.
func GetSpentUTXOFromRevert(startHeight uint32) (map[string][]byte, error) {
	psql := fmt.Sprintf(`SELECT outpoint, utxo_data FROM blkrevert_height WHERE op = 0 AND height >= %d`, startHeight)

	rows, err := clickhouse.Query(context.Background(), psql)
	if err != nil {
		logger.Log.Error("query revert spent failed", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]byte)
	for rows.Next() {
		var outpoint, utxoData string
		if err := rows.Scan(&outpoint, &utxoData); err != nil {
			return nil, err
		}
		result[outpoint] = []byte(utxoData)
	}
	return result, nil
}

// GetNewUTXOKeysFromRevert returns new UTXO outpoint keys from the revert table (op=1).
func GetNewUTXOKeysFromRevert(startHeight uint32) (map[string]struct{}, error) {
	psql := fmt.Sprintf(`SELECT outpoint FROM blkrevert_height WHERE op = 1 AND height >= %d`, startHeight)

	rows, err := clickhouse.Query(context.Background(), psql)
	if err != nil {
		logger.Log.Error("query revert new failed", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]struct{})
	for rows.Next() {
		var outpoint string
		if err := rows.Scan(&outpoint); err != nil {
			return nil, err
		}
		result[outpoint] = struct{}{}
	}
	return result, nil
}

// GetInscriptionIdsAfterBlockHeight returns inscription IDs from the revert table (op=2).
func GetInscriptionIdsAfterBlockHeight(startHeight uint32) ([]string, error) {
	psql := fmt.Sprintf("SELECT outpoint FROM blkrevert_height WHERE op = 2 AND height >= %d", startHeight)

	rows, err := clickhouse.Query(context.Background(), psql)
	if err != nil {
		logger.Log.Error("query revert nftid failed", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var inscriptionIds []string
	for rows.Next() {
		var binId string
		if err := rows.Scan(&binId); err != nil {
			return nil, err
		}
		inscriptionIds = append(inscriptionIds, "i"+binId)
	}
	return inscriptionIds, nil
}
