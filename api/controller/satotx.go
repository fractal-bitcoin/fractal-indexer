package controller

import (
	"fractal-indexer/api/logger"
	"fractal-indexer/api/model"
	"fractal-indexer/api/service"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var (
	is_testnet = os.Getenv("TESTNET")
)

// FractalBitcoin
// @Summary Welcome message
// @Produce  json
// @Success 200 {object} model.Response{data=model.Welcome} "{"code": 0, "data": {}, "msg": "ok"}"
// @Router / [get]
func FractalBitcoin(ctx *gin.Context) {
	logger.Log.Info("FractalBitcoin enter")

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "Welcome to use FractalBitcoin API for Bitcoin ordinals & BRC-20!",
		Data: &model.Welcome{
			Contact: "",
			Job:     "",
			Github:  "https://github.com/fractal-bitcoin",
		},
	})
}

// GetBlockchainInfo gets the latest block position, sync status, and related information.
// @Summary Get the latest block position, sync status, and related information
// @Produce  json
// @Success 200 {object} model.Response{data=model.BlockchainInfoResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /blockchain/info [get]
func GetBlockchainInfo(ctx *gin.Context) {
	logger.Log.Info("GetBlockchainInfo enter")

	bestHeight, err := service.GetBestBlockHeight()
	if err != nil {
		logger.Log.Info("best block failed", zap.Error(err))
	}

	blk, err := service.GetBestBlockByHeight(bestHeight)
	if err != nil {
		logger.Log.Info("best block failed", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get best block failed"})
		return
	}

	mtp, err := service.GetBlockMedianTimePast(bestHeight)
	if err != nil {
		logger.Log.Info("block mtp failed", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get block mtp failed"})
		return
	}
	chain := "main"
	if is_testnet != "" {
		chain = "test"
	}
	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: &model.BlockchainInfoResp{
			Chain:         chain,
			Blocks:        bestHeight + 1,
			Headers:       bestHeight + 1,
			BestBlockHash: blk.BlockIdHex,
			PrevBlockHash: blk.PrevBlockIdHex,
			Difficulty:    "",
			MedianTime:    mtp,
			Chainwork:     "",
		},
	})
}
