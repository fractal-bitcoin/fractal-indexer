package brc20

import (
	"encoding/hex"
	"errors"
	"fractal-indexer/api/lib/utils"
	"fractal-indexer/api/logger"
	"fractal-indexer/api/model"
	"sort"
	"strings"

	"github.com/unisat-wallet/libbrc20-indexer/conf"
	"github.com/unisat-wallet/libbrc20-indexer/constant"
	"github.com/unisat-wallet/libbrc20-indexer/decimal"
	brc20Model "github.com/unisat-wallet/libbrc20-indexer/model"
	swapModel "github.com/unisat-wallet/libbrc20-indexer/model"
	"github.com/unisat-wallet/libbrc20-indexer/uint128"
	brc20Utils "github.com/unisat-wallet/libbrc20-indexer/utils"
	"go.uber.org/zap"
)

// for api, holders
func GetBRC20TickerHolders(ticker string, start, size int) (total int, nftsRsp []*model.BRC20TickerHoldersInfo, err error) {
	logger.Log.Info("GetBRC20TickerHolders",
		zap.String("ticker", ticker),
		zap.Int("start", start),
		zap.Int("size", size))

	if model.GSwap == nil {
		return 0, nil, errors.New("brc20 not ready")
	}

	holdersBalance := model.GSwap.GetBRC20TokenUsersBalanceDataSortedCacheForAPI(ticker)
	total = len(holdersBalance)
	if start >= total {
		return total, make([]*model.BRC20TickerHoldersInfo, 0), nil
	}

	for idx, balance := range holdersBalance[start:] {
		if idx >= size {
			break
		}

		address, err := brc20Utils.GetAddressFromScript([]byte(balance.PkScript), conf.GlobalNetParams)
		if err != nil {
			address = hex.EncodeToString([]byte(balance.PkScript))
		}

		availableBalanceUnSafe := "0"
		if balance.AvailableBalance.Cmp(balance.AvailableBalanceSafe) > 0 {
			availableBalanceUnSafe = balance.AvailableBalance.Sub(balance.AvailableBalanceSafe).String()
		}

		nftsRsp = append(nftsRsp, &model.BRC20TickerHoldersInfo{
			Address:                address,
			OverallBalance:         balance.OverallBalance().String(),
			TransferableBalance:    balance.TransferableBalance.String(),
			AvailableBalance:       balance.AvailableBalance.String(),
			AvailableBalanceSafe:   balance.AvailableBalanceSafe.String(),
			AvailableBalanceUnSafe: availableBalanceUnSafe,
		})
	}

	return total, nftsRsp, nil
}

func GetBRC20Status(tickLenFilter int, ticker, completeType, sortBy string, start, size int) (total int, nftsRsp []*model.BRC20TickerStatusInfo, err error) {
	logger.Log.Info("GetBRC20Status",
		zap.Int("start", start),
		zap.Int("size", size))

	if model.GSwap == nil {
		return 0, nil, errors.New("brc20 not ready")
	}

	nftsRsp = make([]*model.BRC20TickerStatusInfo, 0)

	var statusInfoBaseData []*brc20Model.BRC20TokenInfo

	if completeType == "999" {
		if sortBy == "holders" {
			statusInfoBaseData = GlobalBRC20CacheStatusInfoOrderByHolders999
		} else if sortBy == "transactions" {
			statusInfoBaseData = GlobalBRC20CacheStatusInfoOrderByTransaction999
		} else if sortBy == "minted" {
			statusInfoBaseData = GlobalBRC20CacheStatusInfoOrderByMinted999
		} else if sortBy == "deploy" {
			statusInfoBaseData = GlobalBRC20CacheStatusInfoOrderByDeploy999
		}

	} else {
		if sortBy == "holders" {
			statusInfoBaseData = GlobalBRC20CacheStatusInfoOrderByHolders
		} else if sortBy == "transactions" {
			statusInfoBaseData = GlobalBRC20CacheStatusInfoOrderByTransaction
		} else if sortBy == "minted" {
			statusInfoBaseData = GlobalBRC20CacheStatusInfoOrderByMinted
		} else if sortBy == "deploy" {
			statusInfoBaseData = GlobalBRC20CacheStatusInfoOrderByDeploy
		}
	}
	idx := 0
	for _, info := range statusInfoBaseData {
		if tickLenFilter == 8 { // 4-tick only
			if info.SelfMint {
				continue
			}
		} else if tickLenFilter == 16 { // 5-tick only
			if !info.SelfMint {
				continue
			}
		}

		if ticker != "" {
			uniqueLowerTicker := strings.ToLower(info.Deploy.Data.BRC20Tick)
			if !strings.Contains(uniqueLowerTicker, ticker) {
				continue
			}
		}

		if completeType == "yes" {
			if uint64(info.Deploy.MintTimes) < info.Deploy.MaxMintTimes {
				continue
			}
			if info.Deploy.TotalMinted.Cmp(info.Deploy.Max) < 0 {
				continue
			}
		} else if completeType == "no" {
			if uint64(info.Deploy.MintTimes) >= info.Deploy.MaxMintTimes {
				if info.Deploy.TotalMinted.Cmp(info.Deploy.Max) == 0 {
					continue
				}
			}
		} else if completeType == "999" {
			// ok
		} else if completeType == "" {
			// ok
		} else {
			continue
		}

		total += 1

		if start > 0 && idx < start {
			idx += 1
			continue
		}

		if size > 0 && idx-start >= size {
			idx += 1
			continue
		}

		idx += 1

		uniqueLowerTicker := strings.ToLower(info.Deploy.Data.BRC20Tick)
		originalTicker := info.Deploy.Data.BRC20Tick

		var dConfirmedMinted1h uint128.Decimal
		var dConfirmedMinted24h uint128.Decimal
		if count, ok := GlobalBRC201HourTickerMintCount[originalTicker]; !ok || count == 0 {
			dConfirmedMinted1h = info.Deploy.ConfirmedMinted
		} else {
			if decimal.FromUint128(info.Deploy.Limit).Mul(decimal.NewDecimal(count, 0)).Uint128().Cmp(info.Deploy.ConfirmedMinted) > 0 {
				dConfirmedMinted1h = info.Deploy.ConfirmedMinted
			} else {
				dConfirmedMinted1h = info.Deploy.ConfirmedMinted.Sub(decimal.FromUint128(info.Deploy.Limit).Mul(decimal.NewDecimal(count, 0)).Uint128())
			}
		}
		if count, ok := GlobalBRC2024HourTickerMintCount[originalTicker]; !ok || count == 0 {
			dConfirmedMinted24h = info.Deploy.ConfirmedMinted
		} else {
			if decimal.FromUint128(info.Deploy.Limit).Mul(decimal.NewDecimal(count, 0)).Uint128().Cmp(info.Deploy.ConfirmedMinted) > 0 {
				dConfirmedMinted24h = info.Deploy.ConfirmedMinted
			} else {
				dConfirmedMinted24h = info.Deploy.ConfirmedMinted.Sub(decimal.FromUint128(info.Deploy.Limit).Mul(decimal.NewDecimal(count, 0)).Uint128())
			}
		}

		nftsRsp = append(nftsRsp, &model.BRC20TickerStatusInfo{
			Ticker:       originalTicker,
			SelfMint:     info.SelfMint,
			HoldersCount: GlobalBRC20Holders[uniqueLowerTicker],
			HistoryCount: len(info.History),

			InscriptionNumber: info.Deploy.InscriptionNumber,
			InscriptionId:     info.Deploy.GetInscriptionId(),

			Max:   info.Deploy.Max.String(),
			Limit: info.Deploy.Limit.String(),

			Minted:             info.Deploy.TotalMinted.String(),
			TotalMinted:        info.Deploy.TotalMinted.String(),
			ConfirmedMinted:    info.Deploy.ConfirmedMinted.String(),
			ConfirmedMinted1h:  dConfirmedMinted1h.String(),
			ConfirmedMinted24h: dConfirmedMinted24h.String(),

			MintTimes: info.Deploy.MintTimes,
			Decimal:   info.Deploy.Decimal,

			DeployHeight:    info.Deploy.Height,
			DeployBlockTime: info.Deploy.BlockTime,

			CompleteHeight:    info.Deploy.CompleteHeight,
			CompleteBlockTime: info.Deploy.CompleteBlockTime,

			InscriptionNumberStart: info.Deploy.InscriptionNumberStart,
			InscriptionNumberEnd:   info.Deploy.InscriptionNumberEnd,
		})
	}

	return total, nftsRsp, nil
}

func GetBRC20List(tickLenFilter, start, size int) (total int, nftsRsp []string, err error) {
	logger.Log.Info("GetBRC20List",
		zap.Int("start", start),
		zap.Int("size", size),
		zap.Int("filter", tickLenFilter),
	)

	if model.GSwap == nil {
		return 0, nil, errors.New("brc20 not ready")
	}

	var statusInfo []*brc20Model.BRC20TokenInfo
	for _, info := range model.GSwap.InscriptionsTickerInfoMap {
		if tickLenFilter == 8 { // 4-tick only
			if info.SelfMint {
				continue
			}
		} else if tickLenFilter == 16 { // 5-tick only
			if !info.SelfMint {
				continue
			}
		}
		statusInfo = append(statusInfo, info)
	}

	total = len(statusInfo)
	if start >= total {
		return total, make([]string, 0), nil
	}

	// sort by deploy height
	sort.Slice(statusInfo, func(i, j int) bool {
		idxi := uint64(statusInfo[i].Deploy.Height)*4294967296 + uint64(statusInfo[i].Deploy.TxIdx)
		idxj := uint64(statusInfo[j].Deploy.Height)*4294967296 + uint64(statusInfo[j].Deploy.TxIdx)
		return idxi < idxj
	})

	for idx, info := range statusInfo[start:] {
		if size > 0 && idx >= size {
			break
		}
		nftsRsp = append(nftsRsp, info.Deploy.Data.BRC20Tick)
	}

	return total, nftsRsp, nil
}

func GetBRC20TickerInfo(ticker string) (nftRsp *model.BRC20TickerStatusInfo, err error) {
	logger.Log.Info("GetBRC20TickerInfo", zap.String("ticker", ticker))

	if model.GSwap == nil {
		return nil, errors.New("brc20 not ready")
	}

	tokenInfo, ok := model.GSwap.InscriptionsTickerInfoMap[ticker]
	if !ok {
		return nil, errors.New("ticker invalid")
	}

	uniqueLowerTicker := strings.ToLower(tokenInfo.Deploy.Data.BRC20Tick)
	originalTicker := tokenInfo.Deploy.Data.BRC20Tick

	creatorAddress, err := brc20Utils.GetAddressFromScript([]byte(tokenInfo.Deploy.PkScript), conf.GlobalNetParams)
	if err != nil {
		creatorAddress = hex.EncodeToString([]byte(tokenInfo.Deploy.PkScript))
	}

	var dConfirmedMinted1h uint128.Decimal
	var dConfirmedMinted24h uint128.Decimal
	if count, ok := GlobalBRC201HourTickerMintCount[originalTicker]; !ok || count == 0 {
		dConfirmedMinted1h = tokenInfo.Deploy.ConfirmedMinted
	} else {
		if decimal.FromUint128(tokenInfo.Deploy.Limit).Mul(decimal.NewDecimal(count, 0)).Uint128().Cmp(tokenInfo.Deploy.ConfirmedMinted) > 0 {
			dConfirmedMinted1h = tokenInfo.Deploy.ConfirmedMinted
		} else {
			dConfirmedMinted1h = tokenInfo.Deploy.ConfirmedMinted.Sub(decimal.FromUint128(tokenInfo.Deploy.Limit).Mul(decimal.NewDecimal(count, 0)).Uint128())
		}
	}
	if count, ok := GlobalBRC2024HourTickerMintCount[originalTicker]; !ok || count == 0 {
		dConfirmedMinted24h = tokenInfo.Deploy.ConfirmedMinted
	} else {
		if decimal.FromUint128(tokenInfo.Deploy.Limit).Mul(decimal.NewDecimal(count, 0)).Uint128().Cmp(tokenInfo.Deploy.ConfirmedMinted) > 0 {
			dConfirmedMinted24h = tokenInfo.Deploy.ConfirmedMinted
		} else {
			dConfirmedMinted24h = tokenInfo.Deploy.ConfirmedMinted.Sub(decimal.FromUint128(tokenInfo.Deploy.Limit).Mul(decimal.NewDecimal(count, 0)).Uint128())
		}
	}

	nftRsp = &model.BRC20TickerStatusInfo{
		Ticker:       originalTicker,
		SelfMint:     tokenInfo.SelfMint,
		HoldersCount: GlobalBRC20Holders[uniqueLowerTicker],
		HistoryCount: len(tokenInfo.History),

		InscriptionNumber: tokenInfo.Deploy.InscriptionNumber,
		InscriptionId:     tokenInfo.Deploy.GetInscriptionId(),

		Max:   tokenInfo.Deploy.Max.String(),
		Limit: tokenInfo.Deploy.Limit.String(),

		Minted:             tokenInfo.Deploy.TotalMinted.String(),
		TotalMinted:        tokenInfo.Deploy.TotalMinted.String(),
		ConfirmedMinted:    tokenInfo.Deploy.ConfirmedMinted.String(),
		ConfirmedMinted1h:  dConfirmedMinted1h.String(),
		ConfirmedMinted24h: dConfirmedMinted24h.String(),

		MintTimes: tokenInfo.Deploy.MintTimes,
		Decimal:   tokenInfo.Deploy.Decimal,

		CreatorAddress:  creatorAddress,
		TxIdHex:         utils.GetReversedStringHex(tokenInfo.Deploy.TxId),
		DeployHeight:    tokenInfo.Deploy.Height,
		DeployBlockTime: tokenInfo.Deploy.BlockTime,

		CompleteHeight:    tokenInfo.Deploy.CompleteHeight,
		CompleteBlockTime: tokenInfo.Deploy.CompleteBlockTime,

		InscriptionNumberStart: tokenInfo.Deploy.InscriptionNumberStart,
		InscriptionNumberEnd:   tokenInfo.Deploy.InscriptionNumberEnd,
	}

	return nftRsp, nil
}

// for api, all history
func GetBRC20AllHistoryByHeight(height, start, size int) (total int, nftsRsp []*model.BRC20TickerHistoryInfo, err error) {
	logger.Log.Info("GetBRC20AllHistoryByHeight",
		zap.Int("height", height),
		zap.Int("start", start),
		zap.Int("size", size))

	if model.GSwap == nil {
		return 0, nil, errors.New("brc20 not ready")
	}

	firstHistoryByHeight := model.GSwap.FirstHistoryByHeight
	firstHistory, ok := firstHistoryByHeight[uint32(height)]
	if !ok {
		return 0, make([]*model.BRC20TickerHistoryInfo, 0), nil
	}

	lastHistory, ok := firstHistoryByHeight[uint32(height+1)]
	if !ok {
		lastHistory = model.GSwap.HistoryCount
		if height != constant.MEMPOOL_HEIGHT && model.GSwap.FirstMempoolHistory > 0 {
			lastHistory = model.GSwap.FirstMempoolHistory
		}
	}

	total = int(lastHistory - firstHistory)
	if start >= total {
		return total, make([]*model.BRC20TickerHistoryInfo, 0), nil
	}

	end := start + size
	if end > total {
		end = total
	}
	for historyIdx := firstHistory + uint32(start); historyIdx < firstHistory+uint32(end); historyIdx++ {

		history, err := GetHistoryByIdx(model.GSwap, historyIdx)
		if err != nil {
			return total, make([]*model.BRC20TickerHistoryInfo, 0), err
		}

		if height != int(history.Height) {
			continue
		}

		addressFrom, err := brc20Utils.GetAddressFromScript([]byte(history.PkScriptFrom), conf.GlobalNetParams)
		if err != nil {
			addressFrom = hex.EncodeToString([]byte(history.PkScriptFrom))
		}

		addressTo, err := brc20Utils.GetAddressFromScript([]byte(history.PkScriptTo), conf.GlobalNetParams)
		if err != nil {
			addressTo = hex.EncodeToString([]byte(history.PkScriptTo))
		}

		blockhash := ""
		if int(history.Height) < len(model.GlobalBlocksHash) {
			blockhash = utils.GetReversedStringHex(model.GlobalBlocksHash[history.Height])
		}

		nftsRsp = append(nftsRsp, &model.BRC20TickerHistoryInfo{
			Ticker: history.Inscription.Data.BRC20Tick,
			Type:   constant.BRC20_HISTORY_TYPE_NAMES[history.Type],
			Valid:  history.Valid,

			InscriptionNumber: history.Inscription.InscriptionNumber,
			InscriptionId:     history.Inscription.GetInscriptionId(),
			TxIdHex:           utils.GetReversedStringHex(history.TxId),
			Vout:              history.Vout,
			Offset:            history.Offset,
			Idx:               history.Idx,

			AddressFrom:         addressFrom,
			AddressTo:           addressTo,
			Satoshi:             history.Satoshi,
			Fee:                 history.Fee,
			Amount:              history.Amount,
			OverallBalance:      history.OverallBalance,
			TransferableBalance: history.TransferableBalance,
			AvailableBalance:    history.AvailableBalance,

			Height:       history.Height,
			TxIdx:        history.TxIdx,
			BlockHashHex: blockhash,
			BlockTime:    history.BlockTime,

			History: historyIdx,
		})
	}

	return total, nftsRsp, nil
}

func IsValidHistoryTypeForBRC20AllHistoryByAddress(historyType string) bool {
	switch historyType {
	case constant.BRC20_HISTORY_TYPE_INSCRIBE_MINT, constant.BRC20_HISTORY_TYPE_INSCRIBE_DEPLOY, constant.BRC20_HISTORY_TYPE_INSCRIBE_TRANSFER,
		constant.BRC20_HISTORY_TYPE_SEND, constant.BRC20_HISTORY_TYPE_RECEIVE, constant.BRC20_HISTORY_TYPE_WITHDRAW:
		return true
	default:
		return false
	}
}

// for api, all history
func GetBRC20AllHistoryByAddress(pkScript []byte, historyType string, start, size int) (total int, nftsRsp []*model.BRC20TickerHistoryInfo, err error) {
	logger.Log.Info("GetBRC20AllHistoryByAddress",
		zap.Int("start", start),
		zap.Int("size", size),
		zap.String("type", historyType),
	)

	if model.GSwap == nil {
		return 0, nil, errors.New("brc20 not ready")
	}

	var tokenInfoHistory = []uint32{}
	userHistory := model.GSwap.GetBRC20HistoryByUserForAPI(string(pkScript))
	switch historyType {
	case constant.BRC20_HISTORY_TYPE_INSCRIBE_DEPLOY:
		tokenInfoHistory = userHistory.HistoryDeploy
	case constant.BRC20_HISTORY_TYPE_INSCRIBE_MINT:
		tokenInfoHistory = userHistory.HistoryMint
	case constant.BRC20_HISTORY_TYPE_INSCRIBE_TRANSFER:
		tokenInfoHistory = userHistory.HistoryInscribeTransfer
	case constant.BRC20_HISTORY_TYPE_SEND:
		tokenInfoHistory = userHistory.HistorySend
	case constant.BRC20_HISTORY_TYPE_RECEIVE:
		tokenInfoHistory = userHistory.HistoryReceive
	case constant.BRC20_HISTORY_TYPE_WITHDRAW:
		tokenInfoHistory = userHistory.HistoryWithdraw
	default:
		tokenInfoHistory = userHistory.History
	}
	total = len(tokenInfoHistory)
	if start >= total {
		return total, make([]*model.BRC20TickerHistoryInfo, 0), nil
	}

	count := 0
	for idx := total - start - 1; idx >= 0; idx-- {
		if count >= size {
			break
		}
		count += 1

		historyIdx := tokenInfoHistory[idx]
		history, err := GetHistoryByIdx(model.GSwap, historyIdx)
		if err != nil {
			return total, make([]*model.BRC20TickerHistoryInfo, 0), err
		}

		addressFrom, err := brc20Utils.GetAddressFromScript([]byte(history.PkScriptFrom), conf.GlobalNetParams)
		if err != nil {
			addressFrom = hex.EncodeToString([]byte(history.PkScriptFrom))
		}

		addressTo, err := brc20Utils.GetAddressFromScript([]byte(history.PkScriptTo), conf.GlobalNetParams)
		if err != nil {
			addressTo = hex.EncodeToString([]byte(history.PkScriptTo))
		}

		blockhash := ""
		if int(history.Height) < len(model.GlobalBlocksHash) {
			blockhash = utils.GetReversedStringHex(model.GlobalBlocksHash[history.Height])
		}

		nftsRsp = append(nftsRsp, &model.BRC20TickerHistoryInfo{
			Ticker: history.Inscription.Data.BRC20Tick,
			Type:   constant.BRC20_HISTORY_TYPE_NAMES[history.Type],
			Valid:  history.Valid,

			InscriptionNumber: history.Inscription.InscriptionNumber,
			InscriptionId:     history.Inscription.GetInscriptionId(),
			TxIdHex:           utils.GetReversedStringHex(history.TxId),
			Vout:              history.Vout,
			Offset:            history.Offset,
			Idx:               history.Idx,

			AddressFrom:         addressFrom,
			AddressTo:           addressTo,
			Satoshi:             history.Satoshi,
			Fee:                 history.Fee,
			Amount:              history.Amount,
			OverallBalance:      history.OverallBalance,
			TransferableBalance: history.TransferableBalance,
			AvailableBalance:    history.AvailableBalance,

			Height:       history.Height,
			TxIdx:        history.TxIdx,
			BlockHashHex: blockhash,
			BlockTime:    history.BlockTime,
		})
	}

	return total, nftsRsp, nil
}

// for api, history
func GetBRC20TickerHistory(historyType, ticker string, height, start, size int) (total int, nftsRsp []*model.BRC20TickerHistoryInfo, err error) {
	logger.Log.Info("GetBRC20TickerHistory",
		zap.String("type", historyType),
		zap.String("ticker", ticker),
		zap.Int("height", height),
		zap.Int("start", start),
		zap.Int("size", size))

	if model.GSwap == nil {
		return 0, nil, errors.New("brc20 not ready")
	}

	tokenInfo, ok := model.GSwap.InscriptionsTickerInfoMap[ticker]
	if !ok {
		return 0, make([]*model.BRC20TickerHistoryInfo, 0), nil
	}

	var tokenInfoHistory []uint32 = tokenInfo.History
	if historyType == constant.BRC20_HISTORY_TYPE_INSCRIBE_MINT {
		tokenInfoHistory = tokenInfo.HistoryMint
	} else if historyType == constant.BRC20_HISTORY_TYPE_INSCRIBE_TRANSFER {
		tokenInfoHistory = tokenInfo.HistoryInscribeTransfer
	} else if historyType == constant.BRC20_HISTORY_TYPE_TRANSFER {
		tokenInfoHistory = tokenInfo.HistoryTransfer
	} else if historyType == constant.BRC20_HISTORY_MODULE_TYPE_WITHDRAW {
		tokenInfoHistory = tokenInfo.HistoryWithdraw
	}

	if height > 0 {
		firstHistoryByHeight := model.GSwap.FirstHistoryByHeight
		firstHistory := firstHistoryByHeight[uint32(height)]
		lastHistory := firstHistoryByHeight[uint32(height+1)]

		var tokenInfoHistoryHeight []uint32
		for idx := len(tokenInfoHistory) - 1; idx >= 0; idx-- {
			history := tokenInfoHistory[idx]

			if history < firstHistory {
				break
			}
			if lastHistory > 0 && history >= lastHistory {
				continue
			}
			tokenInfoHistoryHeight = append(tokenInfoHistoryHeight, history)
		}
		for i, j := 0, len(tokenInfoHistoryHeight)-1; i < j; i, j = i+1, j-1 {
			tokenInfoHistoryHeight[i], tokenInfoHistoryHeight[j] = tokenInfoHistoryHeight[j], tokenInfoHistoryHeight[i]
		}
		tokenInfoHistory = tokenInfoHistoryHeight
	}

	total = len(tokenInfoHistory)
	if start >= total {
		return total, make([]*model.BRC20TickerHistoryInfo, 0), nil
	}

	count := 0
	for idx := total - start - 1; idx >= 0; idx-- {
		if count >= size {
			break
		}
		count += 1

		historyIdx := tokenInfoHistory[idx]
		history, err := GetHistoryByIdx(model.GSwap, historyIdx)
		if err != nil {
			return total, make([]*model.BRC20TickerHistoryInfo, 0), err
		}

		addressFrom, err := brc20Utils.GetAddressFromScript([]byte(history.PkScriptFrom), conf.GlobalNetParams)
		if err != nil {
			addressFrom = hex.EncodeToString([]byte(history.PkScriptFrom))
		}

		addressTo, err := brc20Utils.GetAddressFromScript([]byte(history.PkScriptTo), conf.GlobalNetParams)
		if err != nil {
			addressTo = hex.EncodeToString([]byte(history.PkScriptTo))
		}

		blockhash := ""
		if int(history.Height) < len(model.GlobalBlocksHash) {
			blockhash = utils.GetReversedStringHex(model.GlobalBlocksHash[history.Height])
		}
		nftsRsp = append(nftsRsp, &model.BRC20TickerHistoryInfo{
			Ticker: tokenInfo.Deploy.Data.BRC20Tick,
			Type:   constant.BRC20_HISTORY_TYPE_NAMES[history.Type],
			Valid:  history.Valid,

			InscriptionNumber: history.Inscription.InscriptionNumber,
			InscriptionId:     history.Inscription.GetInscriptionId(),
			TxIdHex:           utils.GetReversedStringHex(history.TxId),
			Vout:              history.Vout,
			Offset:            history.Offset,
			Idx:               history.Idx,

			AddressFrom:         addressFrom,
			AddressTo:           addressTo,
			Satoshi:             history.Satoshi,
			Fee:                 history.Fee,
			Amount:              history.Amount,
			OverallBalance:      history.OverallBalance,
			TransferableBalance: history.TransferableBalance,
			AvailableBalance:    history.AvailableBalance,

			Height:       history.Height,
			TxIdx:        history.TxIdx,
			BlockHashHex: blockhash,
			BlockTime:    history.BlockTime,
		})
	}

	return total, nftsRsp, nil
}

// for api, history
func GetBRC20TickerHistoryByTxID(historyType, ticker string, txId []byte, height, start, size int) (total int, nftsRsp []*model.BRC20TickerHistoryInfo, err error) {
	logger.Log.Info("GetBRC20TickerHistoryByTxID",
		zap.String("type", historyType),
		zap.String("ticker", ticker),
		zap.Int("height", height),
		zap.Int("start", start),
		zap.Int("size", size))

	if height <= 0 {
		logger.Log.Info("GetBRC20TickerHistoryByTxID get height failed")
		return 0, make([]*model.BRC20TickerHistoryInfo, 0), nil
	}

	if model.GSwap == nil {
		return 0, nil, errors.New("brc20 not ready")
	}

	tokenInfo, ok := model.GSwap.InscriptionsTickerInfoMap[ticker]
	if !ok {
		return 0, make([]*model.BRC20TickerHistoryInfo, 0), nil
	}

	txIdStr := string(txId)
	foundTxid := ""

	var tokenInfoHistory []uint32 = tokenInfo.History
	if historyType == constant.BRC20_HISTORY_TYPE_INSCRIBE_MINT {
		tokenInfoHistory = tokenInfo.HistoryMint
	} else if historyType == constant.BRC20_HISTORY_TYPE_INSCRIBE_TRANSFER {
		tokenInfoHistory = tokenInfo.HistoryInscribeTransfer
	} else if historyType == constant.BRC20_HISTORY_TYPE_TRANSFER {
		tokenInfoHistory = tokenInfo.HistoryTransfer
	}

	firstHistoryByHeight := model.GSwap.FirstHistoryByHeight
	firstHistory := firstHistoryByHeight[uint32(height)]
	lastHistory := firstHistoryByHeight[uint32(height+1)]

	var tokenInfoHistoryHeight []uint32
	for idx := len(tokenInfoHistory) - 1; idx >= 0; idx-- {
		historyIdx := tokenInfoHistory[idx]

		if historyIdx < firstHistory {
			break
		}
		if lastHistory > 0 && historyIdx >= lastHistory {
			continue
		}

		history, err := GetHistoryByIdx(model.GSwap, historyIdx)
		if err != nil {
			return total, make([]*model.BRC20TickerHistoryInfo, 0), err
		}

		if foundTxid != "" && foundTxid != history.TxId {
			break
		}
		if txIdStr != history.TxId {
			continue
		}
		foundTxid = txIdStr

		tokenInfoHistoryHeight = append(tokenInfoHistoryHeight, historyIdx)
	}

	for i, j := 0, len(tokenInfoHistoryHeight)-1; i < j; i, j = i+1, j-1 {
		tokenInfoHistoryHeight[i], tokenInfoHistoryHeight[j] = tokenInfoHistoryHeight[j], tokenInfoHistoryHeight[i]
	}
	tokenInfoHistory = tokenInfoHistoryHeight

	total = len(tokenInfoHistory)
	if start >= total {
		return total, make([]*model.BRC20TickerHistoryInfo, 0), nil
	}

	count := 0
	for idx := total - start - 1; idx >= 0; idx-- {
		if count >= size {
			break
		}
		count += 1

		historyIdx := tokenInfoHistory[idx]
		history, err := GetHistoryByIdx(model.GSwap, historyIdx)
		if err != nil {
			return total, make([]*model.BRC20TickerHistoryInfo, 0), err
		}

		addressFrom, err := brc20Utils.GetAddressFromScript([]byte(history.PkScriptFrom), conf.GlobalNetParams)
		if err != nil {
			addressFrom = hex.EncodeToString([]byte(history.PkScriptFrom))
		}

		addressTo, err := brc20Utils.GetAddressFromScript([]byte(history.PkScriptTo), conf.GlobalNetParams)
		if err != nil {
			addressTo = hex.EncodeToString([]byte(history.PkScriptTo))
		}

		blockhash := ""
		if int(history.Height) < len(model.GlobalBlocksHash) {
			blockhash = utils.GetReversedStringHex(model.GlobalBlocksHash[history.Height])
		}
		nftsRsp = append(nftsRsp, &model.BRC20TickerHistoryInfo{
			Ticker: tokenInfo.Deploy.Data.BRC20Tick,
			Type:   constant.BRC20_HISTORY_TYPE_NAMES[history.Type],
			Valid:  history.Valid,

			InscriptionNumber: history.Inscription.InscriptionNumber,
			InscriptionId:     history.Inscription.GetInscriptionId(),
			TxIdHex:           utils.GetReversedStringHex(history.TxId),
			Vout:              history.Vout,
			Offset:            history.Offset,
			Idx:               history.Idx,

			AddressFrom:         addressFrom,
			AddressTo:           addressTo,
			Satoshi:             history.Satoshi,
			Fee:                 history.Fee,
			Amount:              history.Amount,
			OverallBalance:      history.OverallBalance,
			TransferableBalance: history.TransferableBalance,
			AvailableBalance:    history.AvailableBalance,

			Height:       history.Height,
			TxIdx:        history.TxIdx,
			BlockHashHex: blockhash,
			BlockTime:    history.BlockTime,
		})
	}

	return total, nftsRsp, nil
}

func GetBRC20SummaryByAddress(tickLenFilter int, ticker, address string, pkScript []byte, start, size int, excludeZero bool, withSwap bool) (total int, nftsRsp []*model.BRC20TokenSummaryInfo, err error) {
	logger.Log.Info("GetBRC20SummaryByAddress",
		zap.Int("start", start),
		zap.Int("size", size),
		zap.Bool("excludeZero", excludeZero),
		zap.Bool("withSwap", withSwap),
	)

	var tokenBalance []*brc20Model.BRC20TokenBalance
	mapTokenBalance := make(map[string]struct{})

	if model.GSwap == nil {
		return 0, nil, errors.New("brc20 not ready")
	}

	userTokens := model.GSwap.GetUserTokenBalanceMapForAPI(string(pkScript))
	if len(userTokens) == 0 && !withSwap {
		return 0, make([]*model.BRC20TokenSummaryInfo, 0), nil
	}

	// 5-tick deploy
	for {
		if model.Global5bDeployInscriptionsAddressSummaryMap == nil {
			break
		}
		if tickLenFilter == 8 { // 4-tick only
			break
		}
		user5bTokens, ok := model.Global5bDeployInscriptionsAddressSummaryMap[address]
		if !ok {
			break
		}
		for _, inscription := range user5bTokens.DeployInscriptions {
			uniqueLowerTicker := strings.ToLower(inscription.Data.BRC20Tick)
			if _, ok := userTokens[uniqueLowerTicker]; ok {
				continue
			}

			if ticker != "" && !strings.Contains(uniqueLowerTicker, ticker) {
				continue
			}

			tokenBalance = append(tokenBalance, &brc20Model.BRC20TokenBalance{
				Ticker:   inscription.Data.BRC20Tick,
				PkScript: string(pkScript),
			})
			mapTokenBalance[uniqueLowerTicker] = struct{}{}
		}
		break
	}

	// normal
	for _, info := range userTokens {
		uniqueLowerTicker := strings.ToLower(info.Ticker)
		tokenInfo, ok := model.GSwap.InscriptionsTickerInfoMap[uniqueLowerTicker]
		if !ok {
			continue
		}

		if tickLenFilter == 8 { // 4-tick only
			if tokenInfo.SelfMint {
				continue
			}
		} else if tickLenFilter == 16 { // 5-tick only
			if !tokenInfo.SelfMint {
				continue
			}
		}

		if excludeZero && info.OverallBalance().Sign() <= 0 {
			continue
		}

		if ticker != "" && !strings.Contains(uniqueLowerTicker, ticker) {
			continue
		}

		tokenBalance = append(tokenBalance, info)
		mapTokenBalance[uniqueLowerTicker] = struct{}{}
	}

	mapSwapTokenBalance := make(map[string]*brc20Model.BRC20ModuleTokenBalance)
	if withSwap {
		if moduleSwapInfo, ok := model.GSwap.ModulesInfoMap[conf.MODULE_SWAP_INSCRIPTION_ID]; ok {
			for _, balance := range moduleSwapInfo.UsersTokenBalanceDataMap[string(pkScript)] {
				uniqueLowerTicker := strings.ToLower(balance.Tick)
				if _, ok := mapTokenBalance[uniqueLowerTicker]; ok {
					mapSwapTokenBalance[uniqueLowerTicker] = balance
					continue
				}

				tokenInfo, ok := model.GSwap.InscriptionsTickerInfoMap[uniqueLowerTicker]
				if !ok {
					continue
				}

				if tickLenFilter == 8 { // 4-tick only
					if tokenInfo.SelfMint {
						continue
					}
				} else if tickLenFilter == 16 { // 5-tick only
					if !tokenInfo.SelfMint {
						continue
					}
				}

				if excludeZero && balance.SwapAccountBalance.Sign() <= 0 {
					continue
				}

				if ticker != "" && !strings.Contains(uniqueLowerTicker, ticker) {
					continue
				}

				tokenBalance = append(tokenBalance, &brc20Model.BRC20TokenBalance{
					Ticker:   balance.Tick,
					PkScript: string(pkScript),
				})
				mapSwapTokenBalance[uniqueLowerTicker] = balance
			}
		}
	}

	sort.Slice(tokenBalance, func(i, j int) bool {
		return strings.Compare(tokenBalance[i].Ticker, tokenBalance[j].Ticker) > 0
	})

	sort.SliceStable(tokenBalance, func(i, j int) bool {
		return tokenBalance[i].OverallBalance().CmpAlign(tokenBalance[j].OverallBalance()) > 0
	})

	total = len(tokenBalance)
	if start >= total {
		return total, make([]*model.BRC20TokenSummaryInfo, 0), nil
	}

	for idx, balance := range tokenBalance[start:] {
		if idx >= size {
			break
		}

		uniqueLowerTicker := strings.ToLower(balance.Ticker)
		tokenInfo, ok := model.GSwap.InscriptionsTickerInfoMap[uniqueLowerTicker]
		if !ok {
			continue
		}

		availableBalanceUnSafe := "0"
		if balance.AvailableBalance.Cmp(balance.AvailableBalanceSafe) > 0 {
			availableBalanceUnSafe = balance.AvailableBalance.Sub(balance.AvailableBalanceSafe).String()
		}

		var swapBalance *model.BRC20TickerSwapBalance
		if withSwap {
			swapBalance = &model.BRC20TickerSwapBalance{
				SwapAccountBalanceSafe:   "0",
				ModuleAccountBalanceSafe: "0",
				SwapAccountBalance:       "0",
				AvailableBalanceSafe:     "0",
				AvailableBalance:         "0",
			}
			swapTokenBalance, ok := mapSwapTokenBalance[uniqueLowerTicker]
			if ok {
				swapBalance = &model.BRC20TickerSwapBalance{
					SwapAccountBalanceSafe:   swapTokenBalance.SwapAccountBalanceSafe.String(),
					ModuleAccountBalanceSafe: swapTokenBalance.ModuleAccountBalanceSafe.String(),
					SwapAccountBalance:       swapTokenBalance.SwapAccountBalance.String(),
					AvailableBalanceSafe:     swapTokenBalance.AvailableBalanceSafe.String(),
					AvailableBalance:         swapTokenBalance.AvailableBalance.String(),
				}
			}
		}

		nftsRsp = append(nftsRsp, &model.BRC20TokenSummaryInfo{
			Ticker:                 balance.Ticker,
			OverallBalance:         balance.OverallBalance().String(),
			TransferableBalance:    balance.TransferableBalance.String(),
			AvailableBalance:       balance.AvailableBalance.String(),
			AvailableBalanceSafe:   balance.AvailableBalanceSafe.String(),
			AvailableBalanceUnSafe: availableBalanceUnSafe,
			Decimal:                int(tokenInfo.Deploy.Decimal),
			SelfMint:               tokenInfo.SelfMint,
			SwapBalance:            swapBalance,
		})
	}

	return total, nftsRsp, nil
}

func GetBRC20SummaryByAddressAndHeight(tickLenFilter int, pkScript []byte, height, start, size int) (total int, nftsRsp []*model.BRC20TokenSummaryInfo, err error) {
	logger.Log.Info("GetBRC20SummaryByAddressAndHeight",
		zap.Int("start", start),
		zap.Int("size", size))

	if model.GSwap == nil {
		return 0, nil, errors.New("brc20 not ready")
	}

	var tokenBalance []*swapModel.BRC20TokenBalance

	userTokens := model.GSwap.GetUserTokenBalanceMapForAPI(string(pkScript))
	if len(userTokens) == 0 {
		return 0, make([]*model.BRC20TokenSummaryInfo, 0), nil
	}

	for _, info := range userTokens {
		uniqueLowerTicker := strings.ToLower(info.Ticker)
		tokenInfo, ok := model.GSwap.InscriptionsTickerInfoMap[uniqueLowerTicker]
		if !ok {
			continue
		}

		if tickLenFilter == 8 { // 4-tick only
			if tokenInfo.SelfMint {
				continue
			}
		} else if tickLenFilter == 16 { // 5-tick only
			if !tokenInfo.SelfMint {
				continue
			}
		}

		tokenBalance = append(tokenBalance, info)
	}

	sort.Slice(tokenBalance, func(i, j int) bool {
		return strings.Compare(tokenBalance[i].Ticker, tokenBalance[j].Ticker) > 0
	})

	total = len(tokenBalance)
	if start >= total {
		return total, make([]*model.BRC20TokenSummaryInfo, 0), nil
	}

	firstHistoryByHeight := model.GSwap.FirstHistoryByHeight
	lastHistory, ok := firstHistoryByHeight[uint32(height+1)]
	if !ok {
		lastHistory = model.GSwap.HistoryCount
		if height != constant.MEMPOOL_HEIGHT && model.GSwap.FirstMempoolHistory > 0 {
			lastHistory = model.GSwap.FirstMempoolHistory
		}
	}

	for idx, balance := range tokenBalance[start:] {
		if idx >= size {
			break
		}

		uniqueLowerTicker := strings.ToLower(balance.Ticker)
		tokenInfo, ok := model.GSwap.InscriptionsTickerInfoMap[uniqueLowerTicker]
		if !ok {
			continue
		}

		var lastHistoryPtr *swapModel.BRC20History
		historyIdx, ok := brc20Utils.FindMaxLessThan(lastHistory, balance.History)
		if ok {
			history, err := GetHistoryByIdx(model.GSwap, historyIdx)
			if err == nil {
				lastHistoryPtr = history
			}
		}

		info := &model.BRC20TokenSummaryInfo{
			Ticker:              balance.Ticker,
			OverallBalance:      "0",
			TransferableBalance: "0",
			AvailableBalance:    "0",
			Decimal:             int(tokenInfo.Deploy.Decimal),
			SelfMint:            tokenInfo.SelfMint,
		}

		if lastHistoryPtr != nil {
			info.TransferableBalance = lastHistoryPtr.TransferableBalance
			info.AvailableBalance = lastHistoryPtr.AvailableBalance
			info.OverallBalance = lastHistoryPtr.OverallBalance
		}
		nftsRsp = append(nftsRsp, info)
	}

	return total, nftsRsp, nil
}

func GetBRC20TickerHistoryByAddress(pkScript []byte, historyType, uniqueLowerTicker string, start, size, fromHeight int) (total int, nftsRsp []*model.BRC20TickerHistoryInfo, err error) {
	logger.Log.Info("GetBRC20TickerHistoryByAddress",
		zap.String("ticker", uniqueLowerTicker),
		zap.String("type", historyType),
		zap.Int("start", start),
		zap.Int("size", size),
		zap.Int("fromHeight", fromHeight))

	if model.GSwap == nil {
		return 0, nil, errors.New("brc20 not ready")
	}

	tokenInfo, ok := model.GSwap.GetUserTokenBalanceOverlayForAPI(uniqueLowerTicker, string(pkScript))
	if !ok {
		return 0, make([]*model.BRC20TickerHistoryInfo, 0), nil
	}

	var tokenInfoHistory []uint32 = tokenInfo.History
	if historyType == constant.BRC20_HISTORY_TYPE_INSCRIBE_MINT {
		tokenInfoHistory = tokenInfo.HistoryMint
	} else if historyType == constant.BRC20_HISTORY_TYPE_INSCRIBE_TRANSFER {
		tokenInfoHistory = tokenInfo.HistoryInscribeTransfer
	} else if historyType == constant.BRC20_HISTORY_TYPE_SEND {
		tokenInfoHistory = tokenInfo.HistorySend
	} else if historyType == constant.BRC20_HISTORY_TYPE_RECEIVE {
		tokenInfoHistory = tokenInfo.HistoryReceive
	} else if historyType == constant.BRC20_HISTORY_MODULE_TYPE_WITHDRAW {
		tokenInfoHistory = tokenInfo.HistoryWithdraw
	} else if historyType == constant.BRC20_HISTORY_TYPE_INSCRIBE_DEPLOY {
		tokenInfoHistory = tokenInfo.HistoryDeploy
	}

	firstHistoryByHeight := model.GSwap.FirstHistoryByHeight
	firstHistory := firstHistoryByHeight[uint32(fromHeight)]
	for idx, historyIdx := range tokenInfoHistory {
		if historyIdx >= firstHistory {
			tokenInfoHistory = tokenInfoHistory[idx:]
			break
		}
	}

	total = len(tokenInfoHistory)
	if start >= total {
		return total, make([]*model.BRC20TickerHistoryInfo, 0), nil
	}

	count := 0
	for idx := len(tokenInfoHistory) - start - 1; idx >= 0; idx-- {
		if count >= size {
			break
		}
		count += 1

		historyIdx := tokenInfoHistory[idx]
		history, err := GetHistoryByIdx(model.GSwap, historyIdx)
		if err != nil {
			return total, make([]*model.BRC20TickerHistoryInfo, 0), err
		}

		addressFrom, err := brc20Utils.GetAddressFromScript([]byte(history.PkScriptFrom), conf.GlobalNetParams)
		if err != nil {
			addressFrom = hex.EncodeToString([]byte(history.PkScriptFrom))
		}

		addressTo, err := brc20Utils.GetAddressFromScript([]byte(history.PkScriptTo), conf.GlobalNetParams)
		if err != nil {
			addressTo = hex.EncodeToString([]byte(history.PkScriptTo))
		}

		blockhash := ""
		if int(history.Height) < len(model.GlobalBlocksHash) {
			blockhash = utils.GetReversedStringHex(model.GlobalBlocksHash[history.Height])
		}
		nftsRsp = append(nftsRsp, &model.BRC20TickerHistoryInfo{
			Ticker: tokenInfo.Ticker,
			Type:   constant.BRC20_HISTORY_TYPE_NAMES[history.Type],
			Valid:  history.Valid,

			InscriptionNumber: history.Inscription.InscriptionNumber,
			InscriptionId:     history.Inscription.GetInscriptionId(),
			TxIdHex:           utils.GetReversedStringHex(history.TxId),
			Vout:              history.Vout,
			Offset:            history.Offset,
			Idx:               history.Idx,

			AddressFrom:         addressFrom,
			AddressTo:           addressTo,
			Satoshi:             history.Satoshi,
			Fee:                 history.Fee,
			Amount:              history.Amount,
			OverallBalance:      history.OverallBalance,
			TransferableBalance: history.TransferableBalance,
			AvailableBalance:    history.AvailableBalance,

			Height:       history.Height,
			TxIdx:        history.TxIdx,
			BlockHashHex: blockhash,
			BlockTime:    history.BlockTime,
		})
	}

	return total, nftsRsp, nil
}

func GetBRC20TickerInfoByAddress(pkScript []byte, ticker string, withSwap bool) (nftRsp *model.BRC20TickerStatusInfoOfAddressResp, err error) {
	logger.Log.Info("GetBRC20TickerInfoByAddress",
		zap.String("ticker", ticker),
	)

	if model.GSwap == nil {
		return nil, errors.New("brc20 not ready")
	}

	validTransferKey := make([]uint64, 0)
	validTransferResp := make([]*brc20Model.InscriptionBRC20TickInfoResp, 0)
	historyInscriptions := make([]*brc20Model.InscriptionBRC20TickInfoResp, 0)
	nftRsp = &model.BRC20TickerStatusInfoOfAddressResp{
		Ticker:              ticker,
		SelfMint:            false,
		OverallBalance:      "0",
		TransferableBalance: "0",
		AvailableBalance:    "0",

		AvailableBalanceSafe:   "0",
		AvailableBalanceUnSafe: "0",

		TransferableCount:        0,
		TransferableInscriptions: validTransferResp,

		HistoryCount:        0,
		HistoryInscriptions: historyInscriptions,
	}

	if withSwap {
		nftRsp.SwapBalance = &model.BRC20TickerSwapBalance{
			SwapAccountBalanceSafe:   "0",
			ModuleAccountBalanceSafe: "0",
			SwapAccountBalance:       "0",
			AvailableBalanceSafe:     "0",
			AvailableBalance:         "0",
		}
		if moduleSwapInfo, ok := model.GSwap.ModulesInfoMap[conf.MODULE_SWAP_INSCRIPTION_ID]; ok {
			swapTokenBalance := moduleSwapInfo.GetUserTokenBalance(strings.ToLower(ticker), string(pkScript))
			nftRsp.SwapBalance = &model.BRC20TickerSwapBalance{
				SwapAccountBalanceSafe:   swapTokenBalance.SwapAccountBalanceSafe.String(),
				ModuleAccountBalanceSafe: swapTokenBalance.ModuleAccountBalanceSafe.String(),
				SwapAccountBalance:       swapTokenBalance.SwapAccountBalance.String(),
				AvailableBalanceSafe:     swapTokenBalance.AvailableBalanceSafe.String(),
				AvailableBalance:         swapTokenBalance.AvailableBalance.String(),
			}
		}
	}

	tokenInfo, ok := model.GSwap.GetUserTokenBalanceOverlayForAPI(ticker, string(pkScript))
	if !ok {
		return nftRsp, nil
	}

	for key := range tokenInfo.ValidTransferMap {
		validTransferKey = append(validTransferKey, key)
	}
	// sort by deploy height
	sort.Slice(validTransferKey, func(i, j int) bool {
		idxi := validTransferKey[i]
		idxj := validTransferKey[j]
		return idxi > idxj
	})
	for idx, key := range validTransferKey {
		if idx > 7 {
			break
		}
		tr, ok := model.GSwap.InscriptionsValidTransferMap[key]
		if !ok {
			break
		}
		validTransferResp = append(validTransferResp, &brc20Model.InscriptionBRC20TickInfoResp{
			Height:            tr.Height,
			Data:              tr.Data,
			InscriptionNumber: tr.InscriptionNumber,
			InscriptionId:     tr.GetInscriptionId(),
			Satoshi:           tr.Satoshi,
		})
	}

	var tokenInfoHistory []uint32 = tokenInfo.History

	// 2025-06-26 ignore history data
	// count := 0
	// for idx := len(tokenInfoHistory) - 1; idx >= 0; idx-- {
	// 	historyIdx := tokenInfoHistory[idx]
	// 	history, err := GetHistoryByIdx(model.GSwap, historyIdx)
	// 	if err != nil {
	// 		break
	// 	}

	// 	if !history.Valid {
	// 		continue
	// 	}
	// 	if constant.BRC20_HISTORY_TYPE_N_INSCRIBE_TRANSFER == history.Type ||
	// 		constant.BRC20_HISTORY_TYPE_N_SEND == history.Type {
	// 		continue
	// 	}
	// 	if count >= 7 {
	// 		break
	// 	}
	// 	count += 1

	// 	historyInscriptions = append(historyInscriptions, &history.Inscription)
	// }

	uniqueLowerTicker := strings.ToLower(tokenInfo.Ticker)
	tokenInfo2, ok := model.GSwap.InscriptionsTickerInfoMap[uniqueLowerTicker]
	if !ok {
		return nftRsp, nil
	}

	nftRsp.Ticker = tokenInfo.Ticker
	nftRsp.SelfMint = tokenInfo2.SelfMint
	nftRsp.OverallBalance = tokenInfo.OverallBalance().String()
	nftRsp.TransferableBalance = tokenInfo.TransferableBalance.String()
	nftRsp.AvailableBalance = tokenInfo.AvailableBalance.String()
	nftRsp.TransferableCount = len(tokenInfo.ValidTransferMap)
	nftRsp.TransferableInscriptions = validTransferResp
	nftRsp.HistoryCount = len(tokenInfoHistory)
	nftRsp.HistoryInscriptions = historyInscriptions
	nftRsp.AvailableBalanceSafe = tokenInfo.AvailableBalanceSafe.String()

	if tokenInfo.AvailableBalance.Cmp(tokenInfo.AvailableBalanceSafe) > 0 {
		nftRsp.AvailableBalanceUnSafe = tokenInfo.AvailableBalance.Sub(tokenInfo.AvailableBalanceSafe).String()
	} else {
		nftRsp.AvailableBalanceUnSafe = "0"
	}

	return nftRsp, nil
}

func GetBRC20TickerTransferableInscriptionsByAddress(pkScript []byte, ticker string, start, size int, isInvalid bool) (total int, nftsRsp []*brc20Model.InscriptionBRC20TickInfoResp, err error) {
	logger.Log.Info("GetBRC20TickerTransferableInscriptionsByAddress",
		zap.String("ticker", ticker),
		zap.Int("start", start),
		zap.Int("size", size))

	if model.GSwap == nil {
		return 0, nil, errors.New("brc20 not ready")
	}

	tokenInfo, ok := model.GSwap.GetUserTokenBalanceOverlayForAPI(ticker, string(pkScript))
	if !ok {
		return 0, make([]*brc20Model.InscriptionBRC20TickInfoResp, 0), nil
	}

	var transferInscriptionsKey []uint64

	if isInvalid {
		// for _, tr := range tokenInfo.InvalidTransferList {
		// 	transferInscriptions = append(transferInscriptions, tr)
		// }
	} else {
		for key := range tokenInfo.ValidTransferMap {
			transferInscriptionsKey = append(transferInscriptionsKey, key)
		}
	}
	// sort by deploy height
	sort.Slice(transferInscriptionsKey, func(i, j int) bool {
		idxi := transferInscriptionsKey[i]
		idxj := transferInscriptionsKey[j]
		return idxi > idxj
	})

	total = len(transferInscriptionsKey)
	if start >= total {
		return total, make([]*brc20Model.InscriptionBRC20TickInfoResp, 0), nil
	}

	for idx, key := range transferInscriptionsKey[start:] {
		if idx >= size {
			break
		}
		inscription, ok := model.GSwap.InscriptionsValidTransferMap[key]
		if !ok {
			break
		}

		nftsRsp = append(nftsRsp, &brc20Model.InscriptionBRC20TickInfoResp{
			Height:            inscription.Height,
			Data:              inscription.Data,
			InscriptionNumber: inscription.InscriptionNumber,
			InscriptionId:     inscription.GetInscriptionId(),
			Satoshi:           inscription.Satoshi,
		})
	}
	return total, nftsRsp, nil
}

func isNumericAndFourChar(s string) bool {
	if len(s) != 4 {
		return false
	}

	for _, char := range s {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}

func GetBRC20Ticker4dTransferableInscriptionsByAddress(pkScript []byte, start, size int) (total int, nftsRsp []*brc20Model.InscriptionBRC20TickInfoResp, err error) {
	logger.Log.Info("GetBRC20Ticker4dTransferableInscriptionsByAddress",
		zap.Int("start", start),
		zap.Int("size", size))

	if model.GSwap == nil {
		return 0, nil, errors.New("brc20 not ready")
	}

	userTokens := model.GSwap.GetUserTokenBalanceMapForAPI(string(pkScript))
	if len(userTokens) == 0 {
		return 0, make([]*brc20Model.InscriptionBRC20TickInfoResp, 0), nil
	}

	var transferInscriptionsKey []uint64

	for ticker, tokenInfo := range userTokens {
		if !isNumericAndFourChar(ticker) {
			continue
		}

		for key := range tokenInfo.ValidTransferMap {
			transferInscriptionsKey = append(transferInscriptionsKey, key)
		}
	}

	// sort by deploy height
	sort.Slice(transferInscriptionsKey, func(i, j int) bool {
		idxi := transferInscriptionsKey[i]
		idxj := transferInscriptionsKey[j]
		return idxi > idxj
	})

	total = len(transferInscriptionsKey)
	if start >= total {
		return total, make([]*brc20Model.InscriptionBRC20TickInfoResp, 0), nil
	}

	for idx, key := range transferInscriptionsKey[start:] {
		if idx >= size {
			break
		}
		inscription, ok := model.GSwap.InscriptionsValidTransferMap[key]
		if !ok {
			break
		}
		nftsRsp = append(nftsRsp, &brc20Model.InscriptionBRC20TickInfoResp{
			Height:            inscription.Height,
			Data:              inscription.Data,
			InscriptionNumber: inscription.InscriptionNumber,
			InscriptionId:     inscription.GetInscriptionId(),
			Satoshi:           inscription.Satoshi,
		})
	}
	return total, nftsRsp, nil
}

func GetBRC20Ticker5bDeployInscriptionsByAddress(address string, start, size int) (total int, nftsRsp []*model.DeployInscriptionsSummary, err error) {
	logger.Log.Info("GetBRC20Ticker5bDeployInscriptionsByAddress",
		zap.Int("start", start),
		zap.Int("size", size))

	if model.GSwap == nil || model.Global5bDeployInscriptionsAddressSummaryMap == nil {
		return 0, nil, errors.New("brc20 not ready")
	}

	userTokens, ok := model.Global5bDeployInscriptionsAddressSummaryMap[address]
	if !ok {
		return 0, make([]*model.DeployInscriptionsSummary, 0), nil
	}
	total = len(userTokens.DeployInscriptions)
	if start >= total {
		return total, make([]*model.DeployInscriptionsSummary, 0), nil
	}

	for idx, inscription := range userTokens.DeployInscriptions[start:] {
		if idx >= size {
			break
		}
		nftsRsp = append(nftsRsp, inscription)
	}

	return total, nftsRsp, nil
}
