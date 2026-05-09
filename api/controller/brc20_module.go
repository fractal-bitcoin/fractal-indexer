package controller

import (
	"encoding/hex"
	"encoding/json"
	"fractal-query/lib/midware"
	"fractal-query/lib/utils"
	"fractal-query/logger"
	"fractal-query/model"
	"fractal-query/service"
	"fractal-query/service/brc20"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	swapModel "github.com/unisat-wallet/libbrc20-indexer/model"
	"go.uber.org/zap"
)

// GetBRC20ModuleHistory
// @Summary Get BRC-20 module transaction history by module, including address, balance, and mint information
// @Tags BRC20
// @Produce  json
// @Param type query string false "history type(inscribe-deploy/inscribe-mint/inscribe-transfer/transfer/send/receive)" default()
// @Param module path string true "module id" default("")
// @Param start query int false "Start Block Height" default(0)
// @Param end query int false "End Block Height" default(0)
// @Param cursor query int false "Start cursor" default(0)
// @Param size query int false "Number of records to return" default(16)
// @Success 200 {object} model.Response{data=model.BRC20ModuleHistoryResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /brc20-module/{module}/history [get]
func GetBRC20ModuleHistory(ctx *gin.Context) {
	logger.Log.Info("GetBRC20ModuleHistory enter")

	// check height
	blkStartHeightString := ctx.DefaultQuery("start", "0")
	blkStartHeight, err := strconv.Atoi(blkStartHeightString)
	if err != nil || blkStartHeight < 0 {
		logger.Log.Info("blk start height invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "blk start height invalid"})
		return
	}
	blkEndHeightString := ctx.DefaultQuery("end", "0")
	blkEndHeight, err := strconv.Atoi(blkEndHeightString)
	if err != nil || blkEndHeight < 0 {
		logger.Log.Info("blk end height invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "blk end height invalid"})
		return
	}

	if blkEndHeight != 0 && blkEndHeight <= blkStartHeight {
		logger.Log.Info("blk end height invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "blk end height invalid"})
		return
	}

	// get cursor/size
	cursorString := ctx.DefaultQuery("cursor", "0")
	cursor, err := strconv.Atoi(cursorString)
	if err != nil || cursor < 0 {
		logger.Log.Info("cursor invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "cursor invalid"})
		return
	}
	sizeString := ctx.DefaultQuery("size", "16")
	size, err := strconv.Atoi(sizeString)
	if err != nil || size <= 0 || size > MAX_INSCRIPTION_LIMIT {
		logger.Log.Info("size invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "size invalid"})
		return
	}

	module := strings.ToLower(ctx.Param("module"))
	if err := utils.VerifyInscriptionId(module); err != nil {
		logger.Log.Info("moduleId invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: err.Error()})
		return
	}

	historyType := ctx.DefaultQuery("type", "")

	total, nftsRsp, err := brc20.GetBRC20ModuleHistory(historyType, module, blkStartHeight, blkEndHeight, cursor, size)
	if err != nil {
		logger.Log.Info("get brc20 history failed", zap.String("module", module), zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get brc20 history failed"})
		return
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: &model.BRC20ModuleHistoryResp{
			Height: len(model.GlobalBlocksHash) - 1,
			Total:  total,
			Cursor: cursor,
			Detail: nftsRsp,
		},
	})
}

// GetBRC20ModuleWithdrawHistory
// @Summary Get BRC-20 module withdraw history by module, including address, balance, and mint information
// @Tags BRC20
// @Produce  json
// @Param start query int false "Start Block Height" default(0)
// @Param end query int false "End Block Height" default(0)
// @Param cursor query int false "Start cursor" default(0)
// @Param size query int false "Number of records to return" default(16)
// @Success 200 {object} model.Response{data=model.BRC20ModuleHistoryResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /brc20-module/withdraw-history [get]
func GetBRC20ModuleWithdrawHistory(ctx *gin.Context) {
	logger.Log.Info("GetBRC20ModuleWithdrawHistory enter")

	// check height
	blkStartHeightString := ctx.DefaultQuery("start", "0")
	blkStartHeight, err := strconv.Atoi(blkStartHeightString)
	if err != nil || blkStartHeight < 0 {
		logger.Log.Info("blk start height invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "blk start height invalid"})
		return
	}
	blkEndHeightString := ctx.DefaultQuery("end", "0")
	blkEndHeight, err := strconv.Atoi(blkEndHeightString)
	if err != nil || blkEndHeight < 0 {
		logger.Log.Info("blk end height invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "blk end height invalid"})
		return
	}

	if blkEndHeight != 0 && blkEndHeight <= blkStartHeight {
		logger.Log.Info("blk end height invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "blk end height invalid"})
		return
	}

	// get cursor/size
	cursorString := ctx.DefaultQuery("cursor", "0")
	cursor, err := strconv.Atoi(cursorString)
	if err != nil || cursor < 0 {
		logger.Log.Info("cursor invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "cursor invalid"})
		return
	}
	sizeString := ctx.DefaultQuery("size", "16")
	size, err := strconv.Atoi(sizeString)
	if err != nil || size <= 0 || size > MAX_INSCRIPTION_LIMIT {
		logger.Log.Info("size invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "size invalid"})
		return
	}

	total, nftsRsp, err := brc20.GetBRC20ModuleWithdrawHistory(blkStartHeight, blkEndHeight, cursor, size)
	if err != nil {
		logger.Log.Info("get brc20 withdraw history failed", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get brc20 withdraw history failed"})
		return
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: &model.BRC20ModuleHistoryResp{
			Height: len(model.GlobalBlocksHash) - 1,
			Total:  total,
			Cursor: cursor,
			Detail: nftsRsp,
		},
	})
}

// GetBRC20ModuleWithdrawHistoryByBlockId
// @Summary Get BRC-20 module withdraw history by module, including address, balance, and mint information
// @Tags BRC20
// @Produce  json
// @Param blkid path string true "BlockId" default(0000000082b5015589a3fdf2d4baff403e6f0be035a5d9742c1cae6295464449)
// @Success 200 {object} model.Response{data=model.BRC20ModuleHistoryResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /brc20-module/withdraw-history-by-block-id/{blkid} [get]
func GetBRC20ModuleWithdrawHistoryByBlockId(ctx *gin.Context) {
	logger.Log.Info("GetBRC20ModuleWithdrawHistoryByBlockId enter")

	blkIdHex := ctx.Param("blkid")
	// check
	blkIdReverse, err := hex.DecodeString(blkIdHex)
	if err != nil {
		logger.Log.Info("blkid invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "blkid invalid"})
		return
	}
	blkId := utils.ReverseBytes(blkIdReverse)

	block, err := service.GetBlockById(hex.EncodeToString(blkId))
	if err != nil {
		logger.Log.Info("get block id failed", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get block failed"})
		return
	}

	total, nftsRsp, err := brc20.GetBRC20ModuleWithdrawHistory(block.Height, block.Height+1, 0, 10000000)
	if err != nil {
		logger.Log.Info("get brc20 withdraw history failed", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get brc20 withdraw history failed"})
		return
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: &model.BRC20ModuleHistoryResp{
			Height: len(model.GlobalBlocksHash) - 1,
			Total:  total,
			Cursor: 0,
			Detail: nftsRsp,
		},
	})
}

// GetBRC20ModuleInscriptionInfo
// @Summary Get details by Inscription ID and return the inscription validity status in the swap module
// @Tags Inscription
// @Produce  json
// @Param inscriptionId path string true "InscriptionID" default("")
// @Success 200 {object} model.Response{data=model.InscriptionResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /brc20-module/inscription/info/{inscriptionId} [get]
func GetBRC20ModuleInscriptionInfo(ctx *gin.Context) {
	logger.Log.Info("GetBRC20ModuleInscriptionInfo enter")

	if model.GSwap == nil {
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "brc20 not ready"})
		return
	}

	inscriptionId := ctx.Param("inscriptionId")
	if err := utils.VerifyInscriptionId(inscriptionId); err != nil {
		logger.Log.Info("inscriptionId invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: err.Error()})
		return
	}

	inscriptionCreatePoints, err := service.GetCreateIdxByInscriptionIdFromRedis([]string{inscriptionId})
	if err != nil || len(inscriptionCreatePoints) != 1 || inscriptionCreatePoints[0].Height == 0 {
		logger.Log.Info("get nft index failed", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get nft index failed"})
		return
	}
	inscriptionCreateIdx := inscriptionCreatePoints[0].GetCreateIdxKey()

	nftRsp, err := service.GetInscriptionsByCreateIdxes([]uint64{inscriptionCreateIdx})
	if err != nil || len(nftRsp) != 1 {
		logger.Log.Info("get nft detail failed", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get nft detail failed"})
		return
	}

	info := nftRsp[0]
	infoRsp := &model.ModuleInscriptionInfoResp{
		UTXO:    info.UTXO,
		Address: info.Address,

		Offset:            info.Offset,
		InscriptionIndex:  info.InscriptionIndex,
		InscriptionNumber: info.InscriptionNumber,
		InscriptionId:     info.InscriptionId,

		ContentType:   info.ContentType,
		ContentLength: info.ContentLength,
		ContentBody:   info.ContentBody,
		Height:        info.Height,
		BlockTime:     info.BlockTime,
		InSatoshi:     info.InSatoshi,
		OutSatoshi:    info.OutSatoshi,
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: infoRsp,
	})

}

// BRC20ModuleVerifySwapCommitContent
// @Summary Validate commit inscription validity
// @Tags Swap
// @Produce  json
// @Param body body model.BRC20ModuleVerifySwapCommitReq true "commit and results"
// @Success 200 {object} model.Response{data=model.BRC20ModuleVerifySwapCommitResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /brc20-module/verify-commit [post]
func BRC20ModuleVerifySwapCommitContent(ctx *gin.Context) {
	logger.Log.Info("BRC20ModuleVerifySwapCommitContent enter")

	// check body
	req := model.BRC20ModuleVerifySwapCommitReq{}
	if err := ctx.BindJSON(&req); err != nil {
		logger.Log.Info("Bind json failed", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "json error"})
		return
	}

	var module string
	for _, commitStr := range req.CommitsStr {
		var commit *swapModel.InscriptionBRC20ModuleSwapCommitContent
		if err := json.Unmarshal([]byte(commitStr), &commit); err != nil {
			logger.Log.Info("unmarshal commit json failed", zap.Error(err))
			ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "commit json error"})
			return
		}
		module = commit.Module
		req.CommitsObj = append(req.CommitsObj, commit)
	}

	resp, err := brc20.BRC20ModuleVerifySwapCommitContent(module, req.CommitsStr, req.CommitsObj, req.LastResults)
	if err != nil {
		logger.Log.Info("verify swap commit results failed", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "verify swap commit results failed"})
		return
	}

	midware.ServiceMetrics.Observe("query", "swap-verify", 0)
	if resp.Valid {
		midware.ServiceMetrics.Observe("valid", "swap-verify", 0)
	} else {
		midware.ServiceMetrics.Observe("invalid", "swap-verify", 0)
	}
	if resp.Critical {
		midware.ServiceMetrics.Observe("critical", "swap-verify", 0)
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: resp,
	})
}

// GetBRC20ModuleTickerInfoByAddress
// @Summary Get BRC-20 module tokens by address, including available balance, transferable balance, transferable inscription count, and the first few inscriptions
// @Tags BRC20Module
// @Produce  json
// @Param address path string true "Address" default(17SkEw2md5avVNyYgj6RiXuQKNwkXaxFyQ)
// @Param ticker path string true "token ticker" default(ordi)
// @Success 200 {object} model.Response{data=model.BRC20ModuleTickerStatusInfoOfAddressResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /brc20-module/{module}/address/{address}/brc20/{ticker}/info [get]
func GetBRC20ModuleTickerInfoByAddress(ctx *gin.Context) {
	logger.Log.Info("GetBRC20ModuleTickerInfoByAddress enter")

	address := ctx.Param("address")
	// check
	pk, err := utils.GetPkScriptByAddress(address)
	if err != nil {
		logger.Log.Info("address invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "address invalid"})
		return
	}

	ticker := ctx.Param("ticker")
	if len(ticker) == 8 {
		tickerStr, err := hex.DecodeString(ticker)
		if err != nil {
			logger.Log.Info("ticker invalid", zap.String("ticker", ticker))
			ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "ticker invalid"})
			return
		}
		ticker = string(tickerStr)
	}
	if len(ticker) != 4 {
		logger.Log.Info("ticker invalid", zap.String("ticker", ticker))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "ticker invalid"})
		return
	}

	module := strings.ToLower(ctx.Param("module"))
	if err := utils.VerifyInscriptionId(module); err != nil {
		logger.Log.Info("moduleId invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: err.Error()})
		return
	}

	nftRsp, err := brc20.GetBRC20ModuleTickerInfoByAddress(pk, module, strings.ToLower(ticker))
	if err != nil {
		logger.Log.Info("get brc20 module info by address failed", zap.String("ticker", ticker), zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get brc20 module info failed"})
		return
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: nftRsp,
	})
}
