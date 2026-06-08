package service

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"fractal-indexer/api/constant"
	"fractal-indexer/api/dao/clickhouse"
	"fractal-indexer/api/model"
	"fractal-indexer/logger"
	"sync"

	"go.uber.org/zap"
)

type CoreDataUpToHeight struct {
	Hash   string // Hash of all following fields for fast comparison.
	Height uint64 // Specified height.

	// from db
	TxCount      uint64 // Up to the specified height.
	TxInValue    uint64 // Up to the specified height.
	TxOutValue   uint64 // Up to the specified height.
	NFTNewCount  uint64 // Up to the specified height.
	NFTInCount   uint64 // Up to the specified height.
	NFTOutCount  uint64 // Up to the specified height.
	NFTLostCount uint64 // Up to the specified height.
	HashDBData   string // Hash of data fetched from DB.

	// from memory
	BRC20HistoryCount uint64 // Up to the specified height.
	HashMemoryData    string // Hash of in-memory program data.
}

func GetCoreDataUpToHeight(height int) CoreDataUpToHeight {
	var coreData CoreDataUpToHeight
	coreData.Height = uint64(height)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		sqlGetCountsUpToHeight := fmt.Sprintf( // sum may overflow, but equal overflowed values still indicate the values are correct.
			`SELECT
				SUM(ntx) AS tx_count,
				SUM(invalue) AS invalues,
				SUM(outvalue) AS outvalues,
				SUM(nftnew) AS nftnew,
				SUM(nftin) AS nftin,
				SUM(nftout) AS nftout,
				SUM(nftlost) AS nftlost
			FROM
				blk_height
			WHERE
				height <= %d`,
			height,
		)
		type Counts struct {
			TotalTxCount  uint64
			TotalInValue  uint64
			TotalOutValue uint64
			TotalNFTNew   uint64
			TotalNFTIn    uint64
			TotalNFTOut   uint64
			TotalNFTLost  uint64
		}
		srf := func(rows *sql.Rows) (interface{}, error) {
			var ret Counts
			err := rows.Scan(&ret.TotalTxCount, &ret.TotalInValue, &ret.TotalOutValue, &ret.TotalNFTNew, &ret.TotalNFTIn, &ret.TotalNFTOut, &ret.TotalNFTLost)
			return ret, err
		}
		ret, err := clickhouse.ScanOne(sqlGetCountsUpToHeight, srf)
		if err != nil {
			logger.Log.Error("failed to get core data upto height", zap.Error(err), zap.Int("height", height))
			return
		}
		counts := ret.(Counts)
		coreData.TxCount = counts.TotalTxCount
		coreData.TxInValue = counts.TotalInValue
		coreData.TxOutValue = counts.TotalOutValue
		coreData.NFTNewCount = counts.TotalNFTNew
		coreData.NFTInCount = counts.TotalNFTIn
		coreData.NFTOutCount = counts.TotalNFTOut
		coreData.NFTLostCount = counts.TotalNFTLost
		formatted := fmt.Sprintf("%d%d%d%d%d%d%d%d",
			coreData.Height,
			coreData.TxCount,
			coreData.TxInValue,
			coreData.TxOutValue,
			coreData.NFTNewCount,
			coreData.NFTInCount,
			coreData.NFTOutCount,
			coreData.NFTLostCount,
		)
		hash := md5.Sum([]byte(formatted))
		coreData.HashDBData = hex.EncodeToString(hash[:])
	}()

	if model.GSwap != nil {
		lastHistory, ok := model.GSwap.FirstHistoryByHeight[uint32(height+1)]
		if !ok {
			lastHistory = model.GSwap.HistoryCount
			if height != constant.MEMPOOL_HEIGHT && model.GSwap.FirstMempoolHistory > 0 {
				lastHistory = model.GSwap.FirstMempoolHistory
			}
		}
		coreData.BRC20HistoryCount = uint64(lastHistory)
	}

	wg.Wait()
	formattedMemoryData := fmt.Sprintf("%d%d",
		coreData.Height,
		coreData.BRC20HistoryCount,
	)
	hashMemoryData := md5.Sum([]byte(formattedMemoryData))
	coreData.HashMemoryData = hex.EncodeToString(hashMemoryData[:])

	hash := md5.Sum(append([]byte(coreData.HashDBData), []byte(coreData.HashMemoryData)...))
	coreData.Hash = hex.EncodeToString(hash[:])

	return coreData
}

type HeightInvalue struct {
	Height  int
	Invalue int
}

const sqlGetLatestBlocksHeightAndInvalue = `
SELECT
	height, invalue
FROM (
	SELECT
		height, invalue
	FROM
		blk_height
	ORDER BY
		height DESC
	LIMIT ?
) AS latest
ORDER BY
	height ASC
`

func GetLatestBlocksHeightAndInvalue(size uint32) ([]HeightInvalue, error) {
	if size == 0 {
		return nil, nil
	}
	srf := func(rows *sql.Rows) (interface{}, error) {
		var ret HeightInvalue
		err := rows.Scan(&ret.Height, &ret.Invalue)
		return ret, err
	}
	rows, err := clickhouse.ScanAll(sqlGetLatestBlocksHeightAndInvalue, srf, size)
	if err != nil {
		logger.Log.Error("GetLatestBlocksHeightAndInvalue failed", zap.Error(err))
		return nil, err
	}
	ret := rows.([]HeightInvalue)
	return ret, nil
}

const sqlGetInvalueByHeightRange = `
SELECT
	height, invalue
FROM
	blk_height
WHERE
	height >= ? AND height < ?
ORDER BY
	height ASC
`

func GetInvalueByHeightRange(fromHeight, toHeight int) ([]HeightInvalue, error) {
	if toHeight <= fromHeight {
		return nil, nil
	}
	if fromHeight < 0 {
		return nil, nil
	}
	srf := func(rows *sql.Rows) (interface{}, error) {
		var ret HeightInvalue
		err := rows.Scan(&ret.Height, &ret.Invalue)
		return ret, err
	}
	rows, err := clickhouse.ScanAll(sqlGetInvalueByHeightRange, srf, fromHeight, toHeight)
	if err != nil {
		logger.Log.Error("GetInvalueByHeightRange failed", zap.Error(err))
		return nil, err
	}
	ret := rows.([]HeightInvalue)
	return ret, nil
}

type HeightNFTIn struct {
	Height int
	NFTIn  int
}

const sqlGetLatestBlocksHeightAndNFTIn = `
SELECT
	height, nftin
FROM (
	SELECT
		height, nftin
	FROM
		blk_height
	ORDER BY
		height DESC
	LIMIT ?
) AS latest
ORDER BY
	height ASC
`

func GetLatestBlocksHeightAndNFTIn(size uint32) ([]HeightNFTIn, error) {
	if size == 0 {
		return nil, nil
	}
	srf := func(rows *sql.Rows) (interface{}, error) {
		var ret HeightNFTIn
		err := rows.Scan(&ret.Height, &ret.NFTIn)
		return ret, err
	}
	rows, err := clickhouse.ScanAll(sqlGetLatestBlocksHeightAndNFTIn, srf, size)
	if err != nil {
		logger.Log.Error("GetLatestBlocksHeightAndNFTIn failed", zap.Error(err))
		return nil, err
	}
	ret := rows.([]HeightNFTIn)
	return ret, nil
}

type BlockMetricTrendPoint struct {
	Height         int `json:"height"`
	Witness        int `json:"witness"`
	OpReturn       int `json:"opreturn"`
	Inscription    int `json:"inscription"`
	RunesRunestone int `json:"runesRunestone"`
	RunesEtching   int `json:"runesEtching"`
	Tacit          int `json:"tacit"`
	Alkanes        int `json:"alkanes"`
}

const (
	blockMetricTxWithWitness = iota + 1
	blockMetricTxWithOpReturn
	blockMetricTxWithInscription
	blockMetricTxWithRunesRunestone
	blockMetricTxWithRunesEtching
	blockMetricTxWithTacit
	blockMetricTxWithAlkanes
)

type blockMetricRow struct {
	Height int
	Metric int
	Value  int
}

const sqlGetBlockMetricsByHeightRange = `
SELECT
	height, metric, sum(value) AS value
FROM
	blkmetric_height
WHERE
	height >= ? AND height < ?
GROUP BY
	height, metric
ORDER BY
	height ASC, metric ASC
`

func GetBlockMetricsByHeightRange(fromHeight, toHeight int) ([]BlockMetricTrendPoint, error) {
	if toHeight <= fromHeight || fromHeight < 0 {
		return nil, nil
	}

	srf := func(rows *sql.Rows) (interface{}, error) {
		var ret blockMetricRow
		err := rows.Scan(&ret.Height, &ret.Metric, &ret.Value)
		return ret, err
	}
	rows, err := clickhouse.ScanAll(sqlGetBlockMetricsByHeightRange, srf, fromHeight, toHeight)
	if err != nil {
		logger.Log.Error("GetBlockMetricsByHeightRange failed", zap.Error(err), zap.Int("fromHeight", fromHeight), zap.Int("toHeight", toHeight))
		return nil, err
	}

	points := make([]BlockMetricTrendPoint, toHeight-fromHeight)
	for i := range points {
		points[i].Height = fromHeight + i
	}
	if rows == nil {
		return points, nil
	}
	for _, row := range rows.([]blockMetricRow) {
		idx := row.Height - fromHeight
		if idx < 0 || idx >= len(points) {
			continue
		}
		switch row.Metric {
		case blockMetricTxWithWitness:
			points[idx].Witness = row.Value
		case blockMetricTxWithOpReturn:
			points[idx].OpReturn = row.Value
		case blockMetricTxWithInscription:
			points[idx].Inscription = row.Value
		case blockMetricTxWithRunesRunestone:
			points[idx].RunesRunestone = row.Value
		case blockMetricTxWithRunesEtching:
			points[idx].RunesEtching = row.Value
		case blockMetricTxWithTacit:
			points[idx].Tacit = row.Value
		case blockMetricTxWithAlkanes:
			points[idx].Alkanes = row.Value
		}
	}
	return points, nil
}

const sqlGetNFTInByHeightRange = `
SELECT
	height, nftin
FROM
	blk_height
WHERE
	height >= ? AND height < ?
ORDER BY
	height ASC
`

func GetNFTInByHeightRange(fromHeight, toHeight int) ([]HeightNFTIn, error) {
	if toHeight <= fromHeight {
		return nil, nil
	}
	if fromHeight < 0 {
		return nil, nil
	}
	srf := func(rows *sql.Rows) (interface{}, error) {
		var ret HeightNFTIn
		err := rows.Scan(&ret.Height, &ret.NFTIn)
		return ret, err
	}
	rows, err := clickhouse.ScanAll(sqlGetNFTInByHeightRange, srf, fromHeight, toHeight)
	if err != nil {
		logger.Log.Error("GetNFTInByHeightRange failed", zap.Error(err))
		return nil, err
	}
	ret := rows.([]HeightNFTIn)
	return ret, nil
}
