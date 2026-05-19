package controller

import (
	"encoding/hex"
	"fractal-indexer/api/lib/utils"
	"fractal-indexer/api/logger"
	"fractal-indexer/api/model"
	"fractal-indexer/api/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const MAX_UTXO_LIMIT = 10240
const MAX_UTXO_RESP_LIMIT = 500
const MAX_AVAILABLE_UTXO_LIMIT = 5000

// GetUtxoByTxIdAndIdx
// @Summary Get specified UTXO information by txid and index
// @Tags UTXO
// @Produce  json
// @Param txid path string true "TxId" default(f4184fc596403b9d638783cf57adfe4c75c605f6356fbc91338530e9831e9e16)
// @Param index path int true "output index" default(0)
// @Success 200 {object} model.Response{data=model.TxStandardOutResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /utxo/{txid}/{index} [get]
func GetUtxoByTxIdAndIdx(ctx *gin.Context) {
	logger.Log.Info("GetUtxoByTxIdAndIdx enter")

	// check tx
	txIdHex := ctx.Param("txid")
	txIdReverse, err := hex.DecodeString(txIdHex)
	if err != nil || len(txIdReverse) != 32 {
		logger.Log.Info("txid invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "txid invalid"})
		return
	}
	txId := utils.ReverseBytes(txIdReverse)

	// check index
	txIndexString := ctx.Param("index")
	txIndex, err := strconv.Atoi(txIndexString)
	if err != nil || txIndex < 0 {
		logger.Log.Info("txindex invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "txindex invalid"})
		return
	}

	result, err := service.GetUtxoByTxIdAndIdx(ctx, txId, txIdHex, txIndex)
	if err != nil {
		logger.Log.Info("get utxo failed", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get utxo failed"})
		return
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: result,
	})
}

// GetUtxoNftOffsetByTxIdAndIdx
// @Summary Get inscription offset list in the specified UTXO by txid and index
// @Tags UTXO
// @Produce  json
// @Param txid path string true "TxId" default(f4184fc596403b9d638783cf57adfe4c75c605f6356fbc91338530e9831e9e16)
// @Param index path int true "output index" default(0)
// @Success 200 {object} model.Response{data=model.TxStandardOutResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /utxo/{txid}/{index}/nft-offset [get]
func GetUtxoNftOffsetByTxIdAndIdx(ctx *gin.Context) {
	logger.Log.Info("GetUtxoNftOffsetByTxIdAndIdx enter")

	// check tx
	txIdHex := ctx.Param("txid")
	txIdReverse, err := hex.DecodeString(txIdHex)
	if err != nil || len(txIdReverse) != 32 {
		logger.Log.Info("txid invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "txid invalid"})
		return
	}
	txId := utils.ReverseBytes(txIdReverse)

	// check index
	txIndexString := ctx.Param("index")
	txIndex, err := strconv.Atoi(txIndexString)
	if err != nil || txIndex < 0 {
		logger.Log.Info("txindex invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "txindex invalid"})
		return
	}

	result, err := service.GetUtxoNftOffsetByTxIdAndIdx(txId, txIdHex, txIndex)
	if err != nil {
		logger.Log.Info("get utxo failed", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get utxo failed"})
		return
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: result,
	})
}
