package brc20

import (
	"fractal-indexer/api/model"
	"fractal-indexer/logger"
	"sort"
	"strings"
	"time"

	"fractal-indexer/api/lib/brc20_swap/constant"
	brc20Model "fractal-indexer/api/lib/brc20_swap/model"
	"go.uber.org/zap"
)

var (
	GlobalBRC20CacheStatusInfoOrderByDeploy      []*brc20Model.BRC20TokenInfo
	GlobalBRC20CacheStatusInfoOrderByMinted      []*brc20Model.BRC20TokenInfo
	GlobalBRC20CacheStatusInfoOrderByTransaction []*brc20Model.BRC20TokenInfo
	GlobalBRC20CacheStatusInfoOrderByHolders     []*brc20Model.BRC20TokenInfo

	GlobalBRC20CacheStatusInfoOrderByDeploy999      []*brc20Model.BRC20TokenInfo
	GlobalBRC20CacheStatusInfoOrderByMinted999      []*brc20Model.BRC20TokenInfo
	GlobalBRC20CacheStatusInfoOrderByTransaction999 []*brc20Model.BRC20TokenInfo
	GlobalBRC20CacheStatusInfoOrderByHolders999     []*brc20Model.BRC20TokenInfo

	GlobalBRC20HeatMap1Hour          []model.BRC20HeatMap
	GlobalBRC20HeatMap6Hour          []model.BRC20HeatMap
	GlobalBRC20HeatMap24Hour         []model.BRC20HeatMap
	GlobalBRC201HourTickerMintCount  = make(map[string]uint64)
	GlobalBRC206HourTickerMintCount  = make(map[string]uint64)
	GlobalBRC2024HourTickerMintCount = make(map[string]uint64)

	GlobalBRC20Holders = make(map[string]int)
)

func UpdateBRC20StatusTickerInfoCache() {
	var statusInfo []*brc20Model.BRC20TokenInfo

	var statusInfoOrderByDeploy []*brc20Model.BRC20TokenInfo
	var statusInfoOrderByMinted []*brc20Model.BRC20TokenInfo
	var statusInfoOrderByTransactions []*brc20Model.BRC20TokenInfo
	var statusInfoOrderByHolders []*brc20Model.BRC20TokenInfo

	// model.gswap should not be nil, or panic
	if model.GSwap == nil {
		logger.Log.Error("UpdateBRC20StatusTickerInfoCache but brc20 not ready")
		return
	}
	for _, info := range model.GSwap.InscriptionsTickerInfoMap {
		statusInfo = append(statusInfo, info)
	}

	// sort by deploy height
	sort.Slice(statusInfo, func(i, j int) bool {
		idxi := uint64(statusInfo[i].Deploy.Height)*4294967296 + uint64(statusInfo[i].Deploy.TxIdx)
		idxj := uint64(statusInfo[j].Deploy.Height)*4294967296 + uint64(statusInfo[j].Deploy.TxIdx)
		return idxi < idxj
	})

	for _, info := range statusInfo {
		statusInfoOrderByDeploy = append(statusInfoOrderByDeploy, info)
		statusInfoOrderByMinted = append(statusInfoOrderByMinted, info)
		statusInfoOrderByTransactions = append(statusInfoOrderByTransactions, info)
		statusInfoOrderByHolders = append(statusInfoOrderByHolders, info)
	}

	brc20Holders := make(map[string]int)
	for _, info := range model.GSwap.InscriptionsTickerInfoMap {
		lowerTicker := strings.ToLower(info.Ticker)
		brc20Holders[lowerTicker] = model.GSwap.GetTokenHoldersCountNotAccurateForAPI(lowerTicker, true)
	}
	GlobalBRC20Holders = brc20Holders

	// if sortBy == "holders" {
	sort.SliceStable(statusInfoOrderByHolders, func(i, j int) bool {
		holdersi := GlobalBRC20Holders[strings.ToLower(statusInfoOrderByHolders[i].Ticker)]
		holdersj := GlobalBRC20Holders[strings.ToLower(statusInfoOrderByHolders[j].Ticker)]
		return holdersi > holdersj
	})

	// } else if sortBy == "transactions" {
	sort.SliceStable(statusInfoOrderByTransactions, func(i, j int) bool {
		return len(statusInfoOrderByTransactions[i].History) > len(statusInfoOrderByTransactions[j].History)
	})

	// } else if sortBy == "minted" {
	// sort by progress
	sort.SliceStable(statusInfoOrderByMinted, func(i, j int) bool {
		ratei := uint64(0)
		ratej := uint64(0)
		maxi := statusInfoOrderByMinted[i].Deploy.MaxMintTimes
		maxj := statusInfoOrderByMinted[j].Deploy.MaxMintTimes

		ratei = uint64(statusInfoOrderByMinted[i].Deploy.MintTimes) * 1000 / maxi
		ratej = uint64(statusInfoOrderByMinted[j].Deploy.MintTimes) * 1000 / maxj

		return ratei > ratej
	})

	// sort by mintedtimes progress
	// sort.Slice(statusInfoOrderByMinted, func(i, j int) bool {
	// 	return statusInfoOrderByMinted[i].Deploy.MintTimes > statusInfoOrderByMinted[j].Deploy.MintTimes
	// })

	// update 999
	var statusInfoOrderByDeploy999 []*brc20Model.BRC20TokenInfo
	var statusInfoOrderByMinted999 []*brc20Model.BRC20TokenInfo
	var statusInfoOrderByTransactions999 []*brc20Model.BRC20TokenInfo
	var statusInfoOrderByHolders999 []*brc20Model.BRC20TokenInfo

	for _, info := range statusInfoOrderByDeploy {
		if uint64(info.Deploy.MintTimes) >= info.Deploy.MaxMintTimes*99/100 {
			if info.Deploy.TotalMinted.Cmp(info.Deploy.Max999) >= 0 {
				continue
			}
		}
		statusInfoOrderByDeploy999 = append(statusInfoOrderByDeploy999, info)
	}
	for _, info := range statusInfoOrderByMinted {
		if uint64(info.Deploy.MintTimes) >= info.Deploy.MaxMintTimes*99/100 {
			if info.Deploy.TotalMinted.Cmp(info.Deploy.Max999) >= 0 {
				continue
			}
		}
		statusInfoOrderByMinted999 = append(statusInfoOrderByMinted999, info)
	}
	for _, info := range statusInfoOrderByTransactions {
		if uint64(info.Deploy.MintTimes) >= info.Deploy.MaxMintTimes*99/100 {
			if info.Deploy.TotalMinted.Cmp(info.Deploy.Max999) >= 0 {
				continue
			}
		}
		statusInfoOrderByTransactions999 = append(statusInfoOrderByTransactions999, info)
	}
	for _, info := range statusInfoOrderByHolders {
		if uint64(info.Deploy.MintTimes) >= info.Deploy.MaxMintTimes*99/100 {
			if info.Deploy.TotalMinted.Cmp(info.Deploy.Max999) >= 0 {
				continue
			}
		}
		statusInfoOrderByHolders999 = append(statusInfoOrderByHolders999, info)
	}

	GlobalBRC20CacheStatusInfoOrderByDeploy = statusInfoOrderByDeploy
	GlobalBRC20CacheStatusInfoOrderByMinted = statusInfoOrderByMinted
	GlobalBRC20CacheStatusInfoOrderByTransaction = statusInfoOrderByTransactions
	GlobalBRC20CacheStatusInfoOrderByHolders = statusInfoOrderByHolders

	GlobalBRC20CacheStatusInfoOrderByDeploy999 = statusInfoOrderByDeploy999
	GlobalBRC20CacheStatusInfoOrderByMinted999 = statusInfoOrderByMinted999
	GlobalBRC20CacheStatusInfoOrderByTransaction999 = statusInfoOrderByTransactions999
	GlobalBRC20CacheStatusInfoOrderByHolders999 = statusInfoOrderByHolders999
}

func InitBRC20HeatMap() {
	logger.Log.Info("InitBRC20HeatMap start", zap.Int64("startTime", time.Now().UnixMilli()))
	if model.GSwap == nil {
		logger.Log.Error("InitBRC20HeatMap but brc20 not ready")
		return
	}

	map1HourTickerMintCount := make(map[string]uint64, 0)
	map6HourTickerMintCount := make(map[string]uint64, 0)
	map24HourTickerMintCount := make(map[string]uint64, 0)

	tickerMintCount := model.GSwap.TickerMintCountByHeight[uint32(constant.MEMPOOL_HEIGHT)]
	for ticker, count := range tickerMintCount {
		map24HourTickerMintCount[ticker] += count
		map6HourTickerMintCount[ticker] += count
		map1HourTickerMintCount[ticker] += count
	}

	bestHeight := len(model.GlobalBlocksHash) - 1
	for currentHeight := bestHeight; currentHeight >= bestHeight-2880; currentHeight-- {
		tickerMintCount := model.GSwap.TickerMintCountByHeight[uint32(currentHeight)]
		for ticker, count := range tickerMintCount {
			map24HourTickerMintCount[ticker] += count
			if currentHeight >= bestHeight-720 {
				map6HourTickerMintCount[ticker] += count
				if currentHeight >= bestHeight-120 {
					map1HourTickerMintCount[ticker] += count
				}
			}
		}
	}

	var temp1Hour []model.BRC20HeatMap
	for ticker, count := range map1HourTickerMintCount {
		temp1Hour = append(temp1Hour, model.BRC20HeatMap{
			Ticker: ticker,
			Count:  int64(count),
		})
	}

	var temp6Hour []model.BRC20HeatMap
	for ticker, count := range map6HourTickerMintCount {
		temp6Hour = append(temp6Hour, model.BRC20HeatMap{
			Ticker: ticker,
			Count:  int64(count),
		})
	}

	sort.SliceStable(temp1Hour, func(i, j int) bool {
		return temp1Hour[i].Count > temp1Hour[j].Count
	})
	GlobalBRC20HeatMap1Hour = temp1Hour
	GlobalBRC201HourTickerMintCount = map1HourTickerMintCount

	sort.SliceStable(temp6Hour, func(i, j int) bool {
		return temp6Hour[i].Count > temp6Hour[j].Count
	})
	GlobalBRC20HeatMap6Hour = temp6Hour
	GlobalBRC206HourTickerMintCount = map6HourTickerMintCount

	var temp24Hour []model.BRC20HeatMap
	for ticker, count := range map24HourTickerMintCount {
		temp24Hour = append(temp24Hour, model.BRC20HeatMap{
			Ticker: ticker,
			Count:  int64(count),
		})

		info := model.GSwap.InscriptionsTickerInfoMap[ticker]
		if info == nil {
			continue
		}

	}

	sort.SliceStable(temp24Hour, func(i, j int) bool {
		return temp24Hour[i].Count > temp24Hour[j].Count
	})
	GlobalBRC20HeatMap24Hour = temp24Hour
	GlobalBRC2024HourTickerMintCount = map24HourTickerMintCount

	logger.Log.Info("InitBRC20HeatMap done", zap.Int64("endTime", time.Now().UnixMilli()))
}
