package controller

import (
	"encoding/hex"
	"fmt"
	"fractal-indexer/api/constant"
	"fractal-indexer/api/lib/utils"
	"fractal-indexer/api/model"
	"fractal-indexer/api/service"
	"fractal-indexer/api/service/brc20"
	"fractal-indexer/logger"
	"net/http"
	"strconv"
	"strings"

	brc20Utils "github.com/unisat-wallet/libbrc20-indexer/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GetBRC20TickerHolders
// @Summary Get BRC-20 holder list by ticker, including address and balance information
// @Tags BRC20
// @Produce  json
// @Param ticker path string true "token ticker" default(ordi)
// @Param start query int true "start offset" default(0)
// @Param limit query int true "number of nft" default(10)
// @Success 200 {object} model.Response{data=model.BRC20TickerHoldersResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /brc20/{ticker}/holders [get]
func GetBRC20TickerHolders(ctx *gin.Context) {
	logger.Log.Info("GetBRC20TickerHolders enter")

	startString := ctx.DefaultQuery("start", "0")
	start, err := strconv.Atoi(startString)
	if err != nil || start < 0 {
		logger.Log.Info("start invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "start invalid"})
		return
	}

	limitString := ctx.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitString)
	if err != nil || limit < 1 || limit > 512 {
		logger.Log.Info("limit invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "limit invalid"})
		return
	}

	ticker := ctx.Param("ticker")
	uniqueLowerTicker, err := brc20Utils.GetValidUniqueLowerTickerTicker(ticker)
	if err != nil {
		logger.Log.Info("ticker invalid", zap.String("ticker", ticker))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "ticker invalid"})
		return
	}

	total, nftsRsp, err := brc20.GetBRC20TickerHolders(uniqueLowerTicker, start, limit)
	if err != nil {
		logger.Log.Info("get brc20 holders failed", zap.String("ticker", ticker), zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get brc20 holders failed"})
		return
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: &model.BRC20TickerHoldersResp{
			Height: len(model.GlobalBlocksHash) - 1,
			Total:  total,
			Start:  start,
			Detail: nftsRsp,
		},
	})
}

// GetBRC20Status
// @Summary Get BRC-20 token list, including holder count and total supply
// @Tags BRC20
// @Produce  json
// @Param ticker query string false "search brc20 ticker" default()
// @Param complete query string false "complete type(yes/no/999)" default()
// @Param sort query string false "sort by (holders/deploy/minted/transactions)" default(holders)
// @Param start query int true "start offset" default(0)
// @Param limit query int true "number of nft" default(10)
// @Param tick_filter query int true "mask of tick length" default(0x08)
// @Success 200 {object} model.Response{data=model.BRC20TickerStatusResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /brc20/status [get]
func GetBRC20Status(ctx *gin.Context) {
	logger.Log.Info("GetBRC20Status enter")

	startString := ctx.DefaultQuery("start", "0")
	start, err := strconv.Atoi(startString)
	if err != nil || start < 0 {
		logger.Log.Info("start invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "start invalid"})
		return
	}

	limitString := ctx.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitString)
	if err != nil || limit < 1 || limit > 512 {
		logger.Log.Info("limit invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "limit invalid"})
		return
	}

	complete := ctx.DefaultQuery("complete", "")
	ticker := ctx.DefaultQuery("ticker", "")
	tickerHex := strings.ToLower(ctx.DefaultQuery("ticker_hex", ""))
	if len(tickerHex) > 0 {
		tickerStr, err := hex.DecodeString(tickerHex)
		if err != nil {
			logger.Log.Info("ticker invalid", zap.String("ticker", tickerHex))
			ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "ticker invalid"})
			return
		}
		ticker = string(tickerStr)
	}

	sortby := ctx.DefaultQuery("sort", "holders")
	if sortby != "holders" && sortby != "deploy" && sortby != "minted" && sortby != "transactions" {
		logger.Log.Info("sortby invalid")
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "sortby invalid"})
		return
	}

	tickLenFilterString := ctx.DefaultQuery("tick_filter", "24")
	tickLenFilter, err := strconv.Atoi(tickLenFilterString)
	if err != nil || tickLenFilter < 8 || tickLenFilter > 24 {
		logger.Log.Info("filter invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "filter invalid"})
		return
	}

	total, nftsRsp, err := brc20.GetBRC20Status(tickLenFilter, strings.ToLower(ticker), complete, sortby, start, limit)
	if err != nil {
		logger.Log.Info("get brc20 status failed", zap.String("ticker", "all"), zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get brc20 status failed"})
		return
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: &model.BRC20TickerStatusResp{
			Height: len(model.GlobalBlocksHash) - 1,
			Total:  total,
			Start:  start,
			Detail: nftsRsp,
		},
	})
}

// GetBRC20List
// @Summary Get BRC-20 token list
// @Tags BRC20
// @Produce  json
// @Param start query int true "start offset" default(0)
// @Param limit query int true "number of nft" default(10)
// @Param tick_filter query int true "mask of tick length" default(0x08)
// @Success 200 {object} model.Response{data=model.BRC20TickerListResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /brc20/list [get]
func GetBRC20List(ctx *gin.Context) {
	logger.Log.Info("GetBRC20List enter")

	startString := ctx.DefaultQuery("start", "0")
	start, err := strconv.Atoi(startString)
	if err != nil || start < 0 {
		logger.Log.Info("start invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "start invalid"})
		return
	}

	limitString := ctx.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitString)
	if err != nil || limit < 1 || limit > 512 {
		logger.Log.Info("limit invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "limit invalid"})
		return
	}

	tickLenFilterString := ctx.DefaultQuery("tick_filter", "8")
	tickLenFilter, err := strconv.Atoi(tickLenFilterString)
	if err != nil || tickLenFilter < 8 || tickLenFilter > 24 {
		logger.Log.Info("filter invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "filter invalid"})
		return
	}

	total, nftsRsp, err := brc20.GetBRC20List(tickLenFilter, start, limit)
	if err != nil {
		logger.Log.Info("get brc20 status failed", zap.String("ticker", "all"), zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get brc20 status failed"})
		return
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: &model.BRC20TickerListResp{
			Height: len(model.GlobalBlocksHash) - 1,
			Total:  total,
			Start:  start,
			Detail: nftsRsp,
		},
	})
}

// GetBRC20BestHeight
// @Summary Get the latest BRC-20 data block height
// @Tags BRC20
// @Produce  json
// @Success 200 {object} model.Response{data=model.BRC20TickerBestHeightResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /brc20/bestheight [get]
func GetBRC20BestHeight(ctx *gin.Context) {
	logger.Log.Info("GetBRC20BestHeight enter")

	if model.GSwap == nil {
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "brc20 not ready"})
		return
	}

	blockid := ""
	blocktime := 0
	last := len(model.GlobalBlocksHash) - 1
	if last >= 0 {
		blockid = utils.GetReversedStringHex(model.GlobalBlocksHash[last])
		blocktime = int(model.GlobalBlocksTime[last])
	}
	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: &model.BRC20TickerBestHeightResp{
			Height:     len(model.GlobalBlocksHash) - 1,
			BlockIdHex: blockid,
			BlockTime:  blocktime,
			Total:      len(model.GSwap.InscriptionsTickerInfoMap),
		},
	})
}

func GetBRC20StateHash(ctx *gin.Context) {
	logger.Log.Info("GetBRC20StateHash enter")

	if model.GSwap == nil {
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "brc20 not ready"})
		return
	}

	startHeightStr := ctx.DefaultQuery("start", "0")
	endHeightStr := ctx.DefaultQuery("end", "0")

	startHeight, err := strconv.Atoi(startHeightStr)
	if err != nil || startHeight < 0 {
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "start height invalid"})
		return
	}
	endHeight, err := strconv.Atoi(endHeightStr)
	if err != nil || endHeight < 0 {
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "end height invalid"})
		return
	}
	if endHeight > 0 && endHeight < startHeight {
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "end must be >= start"})
		return
	}
	if endHeight > 0 && endHeight-startHeight > 1000 {
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "end - start must be <= 1000"})
		return
	}

	stateHashMap := model.GSwap.StateHashByHeight
	emptyResp := model.Response{
		Code: 0,
		Msg:  "ok",
		Data: &model.BRC20StateHashResp{
			Height: len(model.GlobalBlocksHash) - 1,
			Total:  0,
			Detail: []*model.BRC20StateHashEntry{},
		},
	}
	if stateHashMap == nil || len(stateHashMap) == 0 {
		ctx.JSON(http.StatusOK, emptyResp)
		return
	}

	bestHeight := len(model.GlobalBlocksHash) - 1
	if bestHeight < 0 {
		ctx.JSON(http.StatusOK, emptyResp)
		return
	}

	// normalize range:
	// - start=0,end=0 => [bestHeight-999, bestHeight]
	// - end=0 => [start, start+999]
	// - start=0 => [end-999, end]
	if startHeight == 0 && endHeight == 0 {
		endHeight = bestHeight
		startHeight = bestHeight - 999
	} else if endHeight == 0 {
		endHeight = startHeight + 999
	} else if startHeight == 0 {
		startHeight = endHeight - 999
	}
	if startHeight < 0 {
		startHeight = 0
	}
	if endHeight < startHeight {
		ctx.JSON(http.StatusOK, emptyResp)
		return
	}

	// one pass over stateHashMap:
	// 1) latest known hash at or before startHeight
	// 2) exact hash updates inside [startHeight, endHeight]
	prevKnownHeight := -1
	var prevKnownHash [32]byte
	inRangeHash := make(map[int][32]byte, endHeight-startHeight+1)
	for h, hash := range stateHashMap {
		hi := int(h)
		if hi <= startHeight && hi > prevKnownHeight {
			prevKnownHeight = hi
			prevKnownHash = hash
		}
		if hi >= startHeight && hi <= endHeight {
			inRangeHash[hi] = hash
		}
	}

	blocksHash := model.GlobalBlocksHash
	detail := make([]*model.BRC20StateHashEntry, 0, endHeight-startHeight+1)
	currentHash := prevKnownHash
	hasCurrent := prevKnownHeight >= 0
	for h := startHeight; h <= endHeight; h++ {
		if hash, ok := inRangeHash[h]; ok {
			currentHash = hash
			hasCurrent = true
		}
		if !hasCurrent {
			continue
		}

		blockHash := ""
		if h < len(blocksHash) {
			blockHash = utils.GetReversedStringHex(blocksHash[h])
		}
		detail = append(detail, &model.BRC20StateHashEntry{
			Height:    h,
			BlockHash: blockHash,
			StateHash: fmt.Sprintf("%x", currentHash),
		})
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: &model.BRC20StateHashResp{
			Height: len(model.GlobalBlocksHash) - 1,
			Total:  len(detail),
			Detail: detail,
		},
	})
}

// GetBRC20TickerInfo
// @Summary Get BRC-20 token information, including holder count and total supply
// @Tags BRC20
// @Produce  json
// @Param ticker path string true "token ticker" default(ordi)
// @Success 200 {object} model.Response{data=model.BRC20TickerStatusInfo} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /brc20/{ticker}/info [get]
func GetBRC20TickerInfo(ctx *gin.Context) {
	logger.Log.Info("GetBRC20TickerInfo enter")

	if model.GSwap == nil {
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "brc20 not ready"})
		return
	}

	ticker := ctx.Param("ticker")
	uniqueLowerTicker, err := brc20Utils.GetValidUniqueLowerTickerTicker(ticker)
	if err != nil {
		logger.Log.Info("ticker invalid", zap.String("ticker", ticker))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "ticker invalid"})
		return
	}

	nftRsp, err := brc20.GetBRC20TickerInfo(uniqueLowerTicker)
	if err != nil {
		logger.Log.Info("get brc20 status failed", zap.String("ticker", "all"), zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get brc20 status failed"})
		return
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: nftRsp,
	})
}

// GetBRC20TickersInfo
// @Summary Bulk get BRC-20 token information, including holder count and total supply
// @Tags BRC20
// @Produce  json
// @Param body body []string true "token tickers"
// @Success 200 {object} model.Response{data=[]model.BRC20TickerStatusInfo} "{"code": 0, "data": [{}], "msg": "ok"}"
// @Security BearerAuth
// @Router /brc20/tickers-info [get]
func GetBRC20TickersInfo(ctx *gin.Context) {
	logger.Log.Info("GetBRC20TickersInfo enter")

	if model.GSwap == nil {
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "brc20 not ready"})
		return
	}

	req := []string{}
	if err := ctx.BindJSON(&req); err != nil {
		logger.Log.Info("Bind json failed", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "json error"})
		return
	}

	tickers := []string{}

	for _, ticker := range req {
		uniqueLowerTicker, err := brc20Utils.GetValidUniqueLowerTickerTicker(ticker)
		if err != nil {
			logger.Log.Info("ticker invalid", zap.String("ticker", ticker))
			ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "ticker invalid"})
			return
		}

		tickers = append(tickers, uniqueLowerTicker)
	}
	if len(tickers) == 0 {
		ctx.JSON(http.StatusOK, model.Response{
			Code: 0,
			Msg:  "ok",
			Data: []string{},
		})
	}

	nftRsps := []*model.BRC20TickerStatusInfo{}
	for _, ticker := range tickers {
		nftRsp, err := brc20.GetBRC20TickerInfo(strings.ToLower(ticker))
		if err != nil {
			logger.Log.Info("get brc20 status failed", zap.String("ticker", "all"), zap.Error(err))
			ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get brc20 status failed: " + ticker})
			return
		}

		nftRsps = append(nftRsps, nftRsp)
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: nftRsps,
	})
}

// GetBRC20AllHistoryByHeight
// @History Get BRC-20 transaction history by block height, including address, balance, and mint information
// @Tags BRC20
// @Produce  json
// @Param height path int true "Block Height" default(0)
// @Param start query int false "start offset" default(0)
// @Param limit query int false "size of result" default(10)
// @Success 200 {object} model.Response{data=model.BRC20TickerHistoryResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /brc20/history-by-height/{height} [get]
func GetBRC20AllHistoryByHeight(ctx *gin.Context) {
	logger.Log.Info("GetBRC20AllHistoryByHeight enter")

	heightString := ctx.Param("height")
	height, err := strconv.Atoi(heightString)
	if err != nil || height < 0 {
		logger.Log.Info("height invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "height invalid"})
		return
	}

	startString := ctx.DefaultQuery("start", "0")
	start, err := strconv.Atoi(startString)
	if err != nil || start < 0 {
		logger.Log.Info("start invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "start invalid"})
		return
	}

	limitString := ctx.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitString)
	if err != nil || limit < 1 || limit > 10240 {
		logger.Log.Info("limit invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "limit invalid"})
		return
	}

	total, nftsRsp, err := brc20.GetBRC20AllHistoryByHeight(height, start, limit)
	if err != nil {
		logger.Log.Info("get brc20 history failed", zap.Int("height", height), zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get brc20 history by height failed"})
		return
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: &model.BRC20TickerHistoryResp{
			Height: len(model.GlobalBlocksHash) - 1,
			Total:  total,
			Start:  start,
			Detail: nftsRsp,
		},
	})
}

// GetBRC20AllHistoryByAddress
// @History Get BRC-20 transaction history by address, including address, balance, and mint information
// @Tags BRC20
// @Produce  json
// @Param address path string true "Address" default(17SkEw2md5avVNyYgj6RiXuQKNwkXaxFyQ)
// @Param start query int true "start offset" default(0)
// @Param limit query int true "number of nft" default(10)
// @Success 200 {object} model.Response{data=model.BRC20TickerHistoryResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /address/{address}/brc20/history [get]
func GetBRC20AllHistoryByAddress(ctx *gin.Context) {
	logger.Log.Info("GetBRC20AllHistoryByAddress enter")

	startString := ctx.DefaultQuery("start", "0")
	start, err := strconv.Atoi(startString)
	if err != nil || start < 0 {
		logger.Log.Info("start invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "start invalid"})
		return
	}

	limitString := ctx.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitString)
	if err != nil || limit < 1 || limit > 10240 {
		logger.Log.Info("limit invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "limit invalid"})
		return
	}

	address := ctx.Param("address")
	// check
	pk, err := utils.GetPkScriptByAddress(address)
	if err != nil {
		logger.Log.Info("address invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "address invalid"})
		return
	}

	historyType := ctx.DefaultQuery("type", "")
	if historyType != "" && !brc20.IsValidHistoryTypeForBRC20AllHistoryByAddress(historyType) {
		logger.Log.Info("type is invalid", zap.String("type", historyType))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "type invalid"})
		return
	}

	total, nftsRsp, err := brc20.GetBRC20AllHistoryByAddress(pk, historyType, start, limit)
	if err != nil {
		logger.Log.Info("get brc20 history failed", zap.String("address", address), zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get brc20 history failed"})
		return
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: &model.BRC20TickerHistoryResp{
			Height: len(model.GlobalBlocksHash) - 1,
			Total:  total,
			Start:  start,
			Detail: nftsRsp,
		},
	})
}

// GetBRC20TickerHistory
// @History Get BRC-20 transaction history by ticker, including address, balance, and mint information
// @Tags BRC20
// @Produce  json
// @Param type query string false "history type(inscribe-deploy/inscribe-mint/inscribe-transfer/transfer/send/receive)" default()
// @Param ticker path string true "token ticker" default(ordi)
// @Param height query int false "start offset" default(0)
// @Param start query int false "start offset" default(0)
// @Param limit query int false "number of nft" default(10)
// @Success 200 {object} model.Response{data=model.BRC20TickerHistoryResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /brc20/{ticker}/history [get]
func GetBRC20TickerHistory(ctx *gin.Context) {
	logger.Log.Info("GetBRC20TickerHistory enter")

	heightString := ctx.DefaultQuery("height", "0")
	height, err := strconv.Atoi(heightString)
	if err != nil || height < 0 {
		logger.Log.Info("height invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "height invalid"})
		return
	}

	startString := ctx.DefaultQuery("start", "0")
	start, err := strconv.Atoi(startString)
	if err != nil || start < 0 {
		logger.Log.Info("start invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "start invalid"})
		return
	}

	limitString := ctx.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitString)
	if err != nil || limit < 1 || limit > 10240 {
		logger.Log.Info("limit invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "limit invalid"})
		return
	}

	ticker := ctx.Param("ticker")
	uniqueLowerTicker, err := brc20Utils.GetValidUniqueLowerTickerTicker(ticker)
	if err != nil {
		logger.Log.Info("ticker invalid", zap.String("ticker", ticker))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "ticker invalid"})
		return
	}

	historyType := ctx.DefaultQuery("type", "")

	total, nftsRsp, err := brc20.GetBRC20TickerHistory(historyType, uniqueLowerTicker, height, start, limit)
	if err != nil {
		logger.Log.Info("get brc20 history failed", zap.String("ticker", ticker), zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get brc20 history failed"})
		return
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: &model.BRC20TickerHistoryResp{
			Height: len(model.GlobalBlocksHash) - 1,
			Total:  total,
			Start:  start,
			Detail: nftsRsp,
		},
	})
}

// GetBRC20TickerHistoryByTxID
// @History Get BRC-20 transaction history by ticker and txid, including address, balance, and mint information
// @Tags BRC20
// @Produce  json
// @Param type query string false "history type(inscribe-deploy/inscribe-mint/inscribe-transfer/transfer/send/receive)" default()
// @Param ticker path string true "token ticker" default(ordi)
// @Param txid path string true "TxId" default(999e1c837c76a1b7fbb7e57baf87b309960f5ffefbf2a9b95dd890602272f644)
// @Param height query int false "start offset" default(0)
// @Param start query int false "start offset" default(0)
// @Param limit query int false "number of nft" default(10)
// @Success 200 {object} model.Response{data=model.BRC20TickerHistoryResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /brc20/{ticker}/tx/{txid}/history [get]
func GetBRC20TickerHistoryByTxID(ctx *gin.Context) {
	logger.Log.Info("GetBRC20TickerHistoryByTxID enter")

	txIdHex := ctx.Param("txid")
	// check
	txIdReverse, err := hex.DecodeString(txIdHex)
	if err != nil || len(txIdReverse) != 32 {
		logger.Log.Info("txid invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "txid invalid"})
		return
	}
	txId := utils.ReverseBytes(txIdReverse)

	heightString := ctx.DefaultQuery("height", "0")
	height, err := strconv.Atoi(heightString)
	if err != nil || height < 0 {
		logger.Log.Info("height invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "height invalid"})
		return
	}

	startString := ctx.DefaultQuery("start", "0")
	start, err := strconv.Atoi(startString)
	if err != nil || start < 0 {
		logger.Log.Info("start invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "start invalid"})
		return
	}

	limitString := ctx.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitString)
	if err != nil || limit < 1 || limit > 512 {
		logger.Log.Info("limit invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "limit invalid"})
		return
	}

	ticker := ctx.Param("ticker")
	uniqueLowerTicker, err := brc20Utils.GetValidUniqueLowerTickerTicker(ticker)
	if err != nil {
		logger.Log.Info("ticker invalid", zap.String("ticker", ticker))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "ticker invalid"})
		return
	}

	historyType := ctx.DefaultQuery("type", "")

	total, nftsRsp, err := brc20.GetBRC20TickerHistoryByTxID(historyType, uniqueLowerTicker, txId, height, start, limit)
	if err != nil {
		logger.Log.Info("get brc20 history failed", zap.String("ticker", ticker), zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get brc20 history failed"})
		return
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: &model.BRC20TickerHistoryResp{
			Height: len(model.GlobalBlocksHash) - 1,
			Total:  total,
			Start:  start,
			Detail: nftsRsp,
		},
	})
}

// GetBRC20SummaryByAddress
// @Summary Get BRC-20 holdings by address, including ticker and balance information
// @Tags BRC20
// @Produce  json
// @Param address path string true "Address" default(17SkEw2md5avVNyYgj6RiXuQKNwkXaxFyQ)
// @Param start query int true "start offset" default(0)
// @Param limit query int true "number of nft" default(10)
// @Param tick_filter query int true "mask of tick length" default(0x08)
// @Success 200 {object} model.Response{data=model.BRC20TokenSummaryResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /address/{address}/brc20/summary [get]
func GetBRC20SummaryByAddress(ctx *gin.Context) {
	logger.Log.Info("GetBRC20SummaryByAddress enter")

	startString := ctx.DefaultQuery("start", "0")
	start, err := strconv.Atoi(startString)
	if err != nil || start < 0 {
		logger.Log.Info("start invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "start invalid"})
		return
	}

	limitString := ctx.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitString)
	if err != nil || limit < 1 || limit > 10240 {
		logger.Log.Info("limit invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "limit invalid"})
		return
	}

	excludeZeroStr := ctx.DefaultQuery("exclude_zero", "")
	excludeZero := excludeZeroStr == "true"

	address := ctx.Param("address")
	// check
	pk, err := utils.GetPkScriptByAddress(address)
	if err != nil {
		logger.Log.Info("address invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "address invalid"})
		return
	}

	tickLenFilterString := ctx.DefaultQuery("tick_filter", "8")
	tickLenFilter, err := strconv.Atoi(tickLenFilterString)
	if err != nil || tickLenFilter < 8 || tickLenFilter > 24 {
		logger.Log.Info("filter invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "filter invalid"})
		return
	}

	ticker := ctx.Query("ticker")
	ticker = strings.ToLower(ticker)

	withSwapStr := ctx.DefaultQuery("with_swap", "")
	withSwap := withSwapStr == "true"

	total, nftsRsp, err := brc20.GetBRC20SummaryByAddress(tickLenFilter, ticker, address, pk, start, limit, excludeZero, withSwap)
	if err != nil {
		logger.Log.Info("get brc20 summary failed", zap.String("address", address), zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get brc20 summary failed"})
		return
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: &model.BRC20TokenSummaryResp{
			Height: len(model.GlobalBlocksHash) - 1,
			Total:  total,
			Start:  start,
			Detail: nftsRsp,
		},
	})
}

// GetBRC20SummaryByAddressAndHeight
// @Summary Get BRC-20 holdings by address, including ticker and balance information
// @Tags BRC20
// @Produce  json
// @Param address path string true "Address" default(17SkEw2md5avVNyYgj6RiXuQKNwkXaxFyQ)
// @Param height path int true "Block Height" default(0)
// @Param start query int true "start offset" default(0)
// @Param limit query int true "number of nft" default(10)
// @Param tick_filter query int true "mask of tick length" default(0x08)
// @Success 200 {object} model.Response{data=model.BRC20TokenSummaryResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /address/{address}/brc20/summary-by-height/{height} [get]
func GetBRC20SummaryByAddressAndHeight(ctx *gin.Context) {
	logger.Log.Info("GetBRC20SummaryByAddressAndHeight enter")

	heightString := ctx.Param("height")
	height, err := strconv.Atoi(heightString)
	if err != nil || height < 0 {
		logger.Log.Info("height invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "height invalid"})
		return
	}

	startString := ctx.DefaultQuery("start", "0")
	start, err := strconv.Atoi(startString)
	if err != nil || start < 0 {
		logger.Log.Info("start invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "start invalid"})
		return
	}

	limitString := ctx.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitString)
	if err != nil || limit < 1 || limit > 10240 {
		logger.Log.Info("limit invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "limit invalid"})
		return
	}

	address := ctx.Param("address")
	// check
	pk, err := utils.GetPkScriptByAddress(address)
	if err != nil {
		logger.Log.Info("address invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "address invalid"})
		return
	}

	tickLenFilterString := ctx.DefaultQuery("tick_filter", "8")
	tickLenFilter, err := strconv.Atoi(tickLenFilterString)
	if err != nil || tickLenFilter < 8 || tickLenFilter > 24 {
		logger.Log.Info("filter invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "filter invalid"})
		return
	}

	total, nftsRsp, err := brc20.GetBRC20SummaryByAddressAndHeight(tickLenFilter, pk, height, start, limit)
	if err != nil {
		logger.Log.Info("get brc20 summary by height failed", zap.String("address", address), zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get brc20 summary by height failed"})
		return
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: &model.BRC20TokenSummaryResp{
			Height: len(model.GlobalBlocksHash) - 1,
			Total:  total,
			Start:  start,
			Detail: nftsRsp,
		},
	})
}

// GetBRC20TickerHistoryByAddress
// @History Get BRC-20 history by address, including ticker and balance information
// @Tags BRC20
// @Produce  json
// @Param type query string false "history type(inscribe-deploy/inscribe-mint/inscribe-transfer/transfer/send/receive)" default()
// @Param address path string true "Address" default(17SkEw2md5avVNyYgj6RiXuQKNwkXaxFyQ)
// @Param ticker path string true "token ticker" default(ordi)
// @Param start query int true "start offset" default(0)
// @Param limit query int true "number of nft" default(10)
// @Success 200 {object} model.Response{data=model.BRC20TickerHistoryResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /address/{address}/brc20/{ticker}/history [get]
func GetBRC20TickerHistoryByAddress(ctx *gin.Context) {
	logger.Log.Info("GetBRC20TickerHistoryByAddress enter")

	startString := ctx.DefaultQuery("start", "0")
	start, err := strconv.Atoi(startString)
	if err != nil || start < 0 {
		logger.Log.Info("start invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "start invalid"})
		return
	}

	limitString := ctx.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitString)
	if err != nil || limit < 1 || limit > 10240 {
		logger.Log.Info("limit invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "limit invalid"})
		return
	}

	address := ctx.Param("address")
	// check
	pk, err := utils.GetPkScriptByAddress(address)
	if err != nil {
		logger.Log.Info("address invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "address invalid"})
		return
	}

	ticker := ctx.Param("ticker")
	uniqueLowerTicker, err := brc20Utils.GetValidUniqueLowerTickerTicker(ticker)
	if err != nil {
		logger.Log.Info("ticker invalid", zap.String("ticker", ticker))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "ticker invalid"})
		return
	}

	historyType := ctx.DefaultQuery("type", "")

	fromHeightStr := ctx.DefaultPostForm("fromHeight", "0")
	fromHeight, err := strconv.Atoi(fromHeightStr)
	if err != nil {
		logger.Log.Info("fromHeight invalid", zap.Error(err), zap.String("fromHeightStr", fromHeightStr))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "fromHeight invalid"})
		return
	}

	total, nftsRsp, err := brc20.GetBRC20TickerHistoryByAddress(pk, historyType, uniqueLowerTicker, start, limit, fromHeight)
	if err != nil {
		logger.Log.Info("get brc20 summary failed", zap.String("address", address), zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get brc20 summary failed"})
		return
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: &model.BRC20TickerHistoryResp{
			Height: len(model.GlobalBlocksHash) - 1,
			Total:  total,
			Start:  start,
			Detail: nftsRsp,
		},
	})
}

// GetBRC20TickerInfoByAddress
// @Summary Get BRC-20 token balance by address, including available balance, transferable balance, transferable inscription count, and the first few inscriptions
// @Tags BRC20
// @Produce  json
// @Param address path string true "Address" default(17SkEw2md5avVNyYgj6RiXuQKNwkXaxFyQ)
// @Param ticker path string true "token ticker" default(ordi)
// @Success 200 {object} model.Response{data=model.BRC20TickerStatusInfoOfAddressResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /address/{address}/brc20/{ticker}/info [get]
func GetBRC20TickerInfoByAddress(ctx *gin.Context) {
	logger.Log.Info("GetBRC20TickerInfoByAddress enter")

	address := ctx.Param("address")
	// check
	pk, err := utils.GetPkScriptByAddress(address)
	if err != nil {
		logger.Log.Info("address invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "address invalid"})
		return
	}

	ticker := ctx.Param("ticker")
	uniqueLowerTicker, err := brc20Utils.GetValidUniqueLowerTickerTicker(ticker)
	if err != nil {
		logger.Log.Info("ticker invalid", zap.String("ticker", ticker))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "ticker invalid"})
		return
	}

	withSwapStr := ctx.DefaultQuery("with_swap", "")
	withSwap := withSwapStr == "true"

	nftRsp, err := brc20.GetBRC20TickerInfoByAddress(pk, uniqueLowerTicker, withSwap)
	if err != nil {
		logger.Log.Info("get brc20 info by address failed", zap.String("ticker", ticker), zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get brc20 info failed"})
		return
	}

	bestHeight, err := service.GetBestBlockHeight()
	if err != nil {
		logger.Log.Info("best block failed", zap.Error(err))
	}
	for _, resp := range nftRsp.TransferableInscriptions {
		if bestHeight > 0 {
			if resp.Height == constant.MEMPOOL_HEIGHT {
				resp.Confirmations = 0
			} else {
				resp.Confirmations = bestHeight - int(resp.Height) + 1
			}
		}
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: nftRsp,
	})
}

// GetBRC20TickerTransferableInscriptionsByAddress
// @Summary Get BRC-20 inscription list by address
// @Tags BRC20
// @Produce  json
// @Param address path string true "Address" default(17SkEw2md5avVNyYgj6RiXuQKNwkXaxFyQ)
// @Param ticker path string true "token ticker" default(ordi)
// @Param start query int false "start offset" default(0)
// @Param limit query int false "number of nft" default(10)
// @Param invalid query string false "number of nft" default(false)
// @Success 200 {object} model.Response{data=model.BRC20TickerInscriptionsResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /address/{address}/brc20/{ticker}/transferable-inscriptions [get]
func GetBRC20TickerTransferableInscriptionsByAddress(ctx *gin.Context) {
	logger.Log.Info("GetBRC20TickerTransferableInscriptionsByAddress enter")

	startString := ctx.DefaultQuery("start", "0")
	start, err := strconv.Atoi(startString)
	if err != nil || start < 0 {
		logger.Log.Info("start invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "start invalid"})
		return
	}

	limitString := ctx.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitString)
	if err != nil || limit < 1 || limit > 512 {
		logger.Log.Info("limit invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "limit invalid"})
		return
	}

	address := ctx.Param("address")
	// check
	pk, err := utils.GetPkScriptByAddress(address)
	if err != nil {
		logger.Log.Info("address invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "address invalid"})
		return
	}

	ticker := ctx.Param("ticker")
	uniqueLowerTicker, err := brc20Utils.GetValidUniqueLowerTickerTicker(ticker)
	if err != nil {
		logger.Log.Info("ticker invalid", zap.String("ticker", ticker))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "ticker invalid"})
		return
	}

	invalid := ctx.DefaultQuery("invalid", "false")
	total, nftsRsp, err := brc20.GetBRC20TickerTransferableInscriptionsByAddress(pk, uniqueLowerTicker, start, limit, invalid == "true")
	if err != nil {
		logger.Log.Info("get brc20 summary failed", zap.String("address", address), zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get brc20 summary failed"})
		return
	}

	bestHeight, err := service.GetBestBlockHeight()
	if err != nil {
		logger.Log.Info("best block failed", zap.Error(err))
	}
	for idx, resp := range nftsRsp {
		if bestHeight > 0 {
			if resp.Height == constant.MEMPOOL_HEIGHT {
				nftsRsp[idx].Confirmations = 0
			} else {
				nftsRsp[idx].Confirmations = bestHeight - int(resp.Height) + 1
			}
		}

	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: &model.BRC20TickerInscriptionsResp{
			Height: len(model.GlobalBlocksHash) - 1,
			Total:  total,
			Start:  start,
			Detail: nftsRsp,
		},
	})
}

// GetBRC20Ticker4dTransferableInscriptionsByAddress
// @Summary Get BRC-20 inscription list by address
// @Tags BRC20
// @Produce  json
// @Param address path string true "Address" default(17SkEw2md5avVNyYgj6RiXuQKNwkXaxFyQ)
// @Param start query int false "start offset" default(0)
// @Param limit query int false "number of nft" default(10)
// @Success 200 {object} model.Response{data=model.BRC20TickerInscriptionsResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /address/{address}/brc20/ticker4d-transferable-inscriptions [get]
func GetBRC20Ticker4dTransferableInscriptionsByAddress(ctx *gin.Context) {
	logger.Log.Info("GetBRC20Ticker4dTransferableInscriptionsByAddress enter")

	startString := ctx.DefaultQuery("start", "0")
	start, err := strconv.Atoi(startString)
	if err != nil || start < 0 {
		logger.Log.Info("start invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "start invalid"})
		return
	}

	limitString := ctx.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitString)
	if err != nil || limit < 1 || limit > 512 {
		logger.Log.Info("limit invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "limit invalid"})
		return
	}

	address := ctx.Param("address")
	// check
	pk, err := utils.GetPkScriptByAddress(address)
	if err != nil {
		logger.Log.Info("address invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "address invalid"})
		return
	}

	total, nftsRsp, err := brc20.GetBRC20Ticker4dTransferableInscriptionsByAddress(pk, start, limit)
	if err != nil {
		logger.Log.Info("get brc20 summary failed", zap.String("address", address), zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get brc20 summary failed"})
		return
	}

	bestHeight, err := service.GetBestBlockHeight()
	if err != nil {
		logger.Log.Info("best block failed", zap.Error(err))
	}
	for idx, resp := range nftsRsp {
		if bestHeight > 0 {
			if resp.Height == constant.MEMPOOL_HEIGHT {
				nftsRsp[idx].Confirmations = 0
			} else {
				nftsRsp[idx].Confirmations = bestHeight - int(resp.Height) + 1
			}
		}
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: &model.BRC20TickerInscriptionsResp{
			Height: len(model.GlobalBlocksHash) - 1,
			Total:  total,
			Start:  start,
			Detail: nftsRsp,
		},
	})
}

// GetBRC20Ticker5bDeployInscriptionsByAddress
// @Summary Get BRC-20 5b Deploy inscription list by address
// @Tags BRC20
// @Produce  json
// @Param address path string true "Address" default(17SkEw2md5avVNyYgj6RiXuQKNwkXaxFyQ)
// @Param start query int false "start offset" default(0)
// @Param limit query int false "number of nft" default(10)
// @Success 200 {object} model.Response{data=model.BRC20TickerSummaryResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /address/{address}/brc20/ticker5b-deploy-inscriptions [get]
func GetBRC20Ticker5bDeployInscriptionsByAddress(ctx *gin.Context) {
	logger.Log.Info("GetBRC20Ticker5bDeployInscriptionsByAddress enter")

	startString := ctx.DefaultQuery("start", "0")
	start, err := strconv.Atoi(startString)
	if err != nil || start < 0 {
		logger.Log.Info("start invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "start invalid"})
		return
	}

	limitString := ctx.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitString)
	if err != nil || limit < 1 || limit > 512 {
		logger.Log.Info("limit invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "limit invalid"})
		return
	}

	address := ctx.Param("address")
	// check
	if _, err := utils.GetPkScriptByAddress(address); err != nil {
		logger.Log.Info("address invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "address invalid"})
		return
	}

	total, nftsRsp, err := brc20.GetBRC20Ticker5bDeployInscriptionsByAddress(address, start, limit)
	if err != nil {
		logger.Log.Info("get brc20 summary failed", zap.String("address", address), zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get brc20 summary failed"})
		return
	}

	bestHeight, err := service.GetBestBlockHeight()
	if err != nil {
		logger.Log.Info("best block failed", zap.Error(err))
	}
	for idx, resp := range nftsRsp {
		if bestHeight > 0 {
			if resp.Height == constant.MEMPOOL_HEIGHT {
				nftsRsp[idx].Confirmations = 0
			} else {
				nftsRsp[idx].Confirmations = bestHeight - int(resp.Height) + 1
			}
		}
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: &model.BRC20TickerSummaryResp{
			Height: len(model.GlobalBlocksHash) - 1,
			Total:  total,
			Start:  start,
			Detail: nftsRsp,
		},
	})
}

// GetBRC20HeatMap
// @Summary Get the BRC-20 mint heatmap
// @Tags BRC20
// @Produce  json
// @Param start query int false "start offset" default(0)
// @Param limit query int false "number of nft" default(10)
// @Success 200 {object} model.Response{data=model.BRC20TickerSummaryResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /brc20/heatMap [get]
func GetBRC20HeatMap(ctx *gin.Context) {
	logger.Log.Info("GetBRC20HeatMap enter")

	startString := ctx.DefaultQuery("start", "0")
	start, err := strconv.Atoi(startString)
	if err != nil || start < 0 {
		logger.Log.Info("start invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "start invalid"})
		return
	}

	limitString := ctx.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitString)
	if err != nil || limit < 1 || limit > 512 {
		logger.Log.Info("limit invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "limit invalid"})
		return
	}

	durationString := ctx.DefaultQuery("duration", "1h")
	var filter, result = []model.BRC20HeatMap{}, []model.BRC20HeatMap{}
	switch durationString {
	case "1h":
		filter = brc20.GlobalBRC20HeatMap1Hour
	case "6h":
		filter = brc20.GlobalBRC20HeatMap6Hour
	case "24h":
		filter = brc20.GlobalBRC20HeatMap24Hour
	default:
		logger.Log.Error("duration invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "duration invalid"})
		return
	}

	ticker := ctx.Query("ticker")
	if ticker != "" {
		ticker = strings.ToLower(ticker)
		filterWithTicker := []model.BRC20HeatMap{}
		for _, oBRC20HeatMap := range filter {
			uniqueLowerTicker := strings.ToLower(oBRC20HeatMap.Ticker)
			if strings.Contains(uniqueLowerTicker, ticker) {
				filterWithTicker = append(filterWithTicker, oBRC20HeatMap)
			}
		}
		filter = filterWithTicker
	}

	complete := ctx.DefaultQuery("complete", "all")
	switch complete {
	case "all":
		break
	case "yes":
		filterComplete := []model.BRC20HeatMap{}
		for _, oBRC20HeatMap := range filter {
			uniqueLowerTicker := strings.ToLower(oBRC20HeatMap.Ticker)
			info := model.GSwap.InscriptionsTickerInfoMap[uniqueLowerTicker]
			if info == nil {
				continue
			}
			if uint64(info.Deploy.MintTimes) >= info.Deploy.MaxMintTimes {
				if info.Deploy.TotalMinted.Cmp(info.Deploy.Max) >= 0 {
					filterComplete = append(filterComplete, oBRC20HeatMap)
				}
			}
		}
		filter = filterComplete
	case "no":
		filterNoComplete := []model.BRC20HeatMap{}
		for _, oBRC20HeatMap := range filter {
			uniqueLowerTicker := strings.ToLower(oBRC20HeatMap.Ticker)
			info := model.GSwap.InscriptionsTickerInfoMap[uniqueLowerTicker]
			if info == nil {
				continue
			}
			if uint64(info.Deploy.MintTimes) < info.Deploy.MaxMintTimes || info.Deploy.TotalMinted.Cmp(info.Deploy.Max) < 0 {
				filterNoComplete = append(filterNoComplete, oBRC20HeatMap)
			}
		}
		filter = filterNoComplete
	default:
		logger.Log.Error("complete invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "complete invalid"})
		return
	}

	mapMintBRC20 := make(map[string]struct{})
	for _, oBRC20HeatMap := range filter {
		if oBRC20HeatMap.Ticker == "" {
			continue
		}
		mapMintBRC20[oBRC20HeatMap.Ticker] = struct{}{}
	}

	total := len(filter)
	idx := 0
	if start >= len(filter) {
		idx = start - len(filter)
		filter = []model.BRC20HeatMap{}
	} else {
		if start+limit >= len(filter) {
			filter = filter[start:]
		} else {
			filter = filter[start : start+limit]
		}
		limit = limit - len(filter)
	}

	for _, oBRC20TokenInfo := range brc20.GlobalBRC20CacheStatusInfoOrderByHolders {
		if _, ok := mapMintBRC20[oBRC20TokenInfo.Ticker]; ok {
			continue
		}
		if oBRC20TokenInfo.Ticker == "" {
			continue
		}
		uniqueLowerTicker := strings.ToLower(oBRC20TokenInfo.Ticker)
		if ticker != "" {
			if !strings.Contains(uniqueLowerTicker, ticker) {
				continue
			}
		}
		switch complete {
		case "yes":
			info := model.GSwap.InscriptionsTickerInfoMap[uniqueLowerTicker]
			if info == nil {
				continue
			}
			if uint64(info.Deploy.MintTimes) < info.Deploy.MaxMintTimes || info.Deploy.TotalMinted.Cmp(info.Deploy.Max) < 0 {
				continue
			}
		case "no":
			info := model.GSwap.InscriptionsTickerInfoMap[uniqueLowerTicker]
			if info == nil {
				continue
			}
			if uint64(info.Deploy.MintTimes) >= info.Deploy.MaxMintTimes && info.Deploy.TotalMinted.Cmp(info.Deploy.Max) >= 0 {
				continue
			}
		}

		total += 1
		if idx > 0 {
			idx--
			continue
		}
		if limit <= 0 {
			continue
		}
		filter = append(filter, model.BRC20HeatMap{Ticker: oBRC20TokenInfo.Ticker})
		limit--
	}

	for _, oBRC20HeatMap := range filter {
		nftRsp, err := brc20.GetBRC20TickerInfo(strings.ToLower(oBRC20HeatMap.Ticker))
		if err != nil {
			logger.Log.Error("GetBRC20HeatMap", zap.String("ticker", oBRC20HeatMap.Ticker), zap.Error(err))
			ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "GetBRC20HeatMap failed: "})
			return
		}
		result = append(result, model.BRC20HeatMap{Ticker: oBRC20HeatMap.Ticker, Count: oBRC20HeatMap.Count, StatusInfo: *nftRsp})
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: &model.BRC20HeatMapResp{
			Height: len(model.GlobalBlocksHash) - 1,
			Total:  total,
			Start:  start,
			Detail: result,
		},
	})
}
