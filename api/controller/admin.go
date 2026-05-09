package controller

import (
	"encoding/hex"
	"fractal-query/logger"
	"fractal-query/model"
	"fractal-query/service/brc20"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func GetBRC20StatusAll(ctx *gin.Context) {
	logger.Log.Info("GetBRC20Status enter")

	complete := ""
	ticker := strings.ToLower(ctx.DefaultQuery("ticker", ""))
	tickerHex := strings.ToLower(ctx.DefaultQuery("ticker_hex", ""))
	if len(tickerHex) > 0 {
		tickerStr, err := hex.DecodeString(tickerHex)
		if err != nil {
			logger.Log.Info("ticker invalid", zap.String("ticker", tickerHex))
			ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "ticker invalid"})
			return
		}
		ticker = strings.ToLower(string(tickerStr))
	}

	sortby := ctx.DefaultQuery("sort", "holders")
	if sortby != "holders" && sortby != "deploy" && sortby != "minted" && sortby != "transactions" {
		logger.Log.Info("sortby invalid")
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "sortby invalid"})
		return
	}

	tickLenFilterString := ctx.DefaultQuery("tick_filter", "8")
	tickLenFilter, err := strconv.Atoi(tickLenFilterString)
	if err != nil || tickLenFilter < 8 || tickLenFilter > 24 {
		logger.Log.Info("filter invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "filter invalid"})
		return
	}

	total, nftsRsp, err := brc20.GetBRC20Status(tickLenFilter, ticker, complete, sortby, 0, 0)
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
			Start:  0,
			Detail: nftsRsp,
		},
	})
}
