package controller

import (
	"fractal-indexer/api/lib/utils"
	"fractal-indexer/api/model"
	"fractal-indexer/api/service"
	serviceNFT "fractal-indexer/api/service/nft"
	"fractal-indexer/logger"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GetInscriptionsEventsByHeightRange
// @Summary Get inscription events within a height range
// @Tags Inscription
// @Produce  json
// @Param start query int true "Start Block Height" default(0)
// @Param end query int true "End Block Height" default(0)
// @Param cursor query int true "Start cursor" default(0)
// @Param size query int true "Number of records to return" default(16)
// @Success 200 {object} model.Response{data=model.InscriptionEventsResultsResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /inscriptions/events [get]
func GetInscriptionsEventsByHeightRange(ctx *gin.Context) {
	logger.Log.Info("GetInscriptionsEventsByHeightRange enter")
	GetInscriptionsEventsByHeightRangeCommon(ctx, false)
}

// GetInscriptionsBRC20EventsByHeightRange
// @Summary Get BRC-20 inscription events within a height range
// @Tags Inscription
// @Produce  json
// @Param start query int true "Start Block Height" default(0)
// @Param end query int true "End Block Height" default(0)
// @Param cursor query int true "Start cursor" default(0)
// @Param size query int true "Number of records to return" default(16)
// @Success 200 {object} model.Response{data=model.InscriptionEventsResultsResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /inscriptions/events [get]
func GetInscriptionsBRC20EventsByHeightRange(ctx *gin.Context) {
	logger.Log.Info("GetInscriptionsBRC20EventsByHeightRange enter")
	GetInscriptionsEventsByHeightRangeCommon(ctx, true)
}

func GetInscriptionsEventsByHeightRangeCommon(ctx *gin.Context, isBrc20 bool) {
	logger.Log.Info("GetInscriptionsEventsByHeightRange enter")

	// check height
	blkStartHeightString := ctx.DefaultQuery("start", "0")
	blkStartHeight, err := strconv.Atoi(blkStartHeightString)
	if err != nil || blkStartHeight <= 0 {
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

	// 2025-06-25: limit end to start + 1.
	if blkEndHeight != blkStartHeight+1 {
		logger.Log.Info("blk end height invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "blk end height must be start+1"})
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

	total, err := serviceNFT.GetInscriptionEventsCount(blkStartHeight, blkEndHeight, isBrc20)
	if err != nil {
		logger.Log.Info("get nft events count failed", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get nft events count failed"})
		return
	}
	if total == 0 {
		ctx.JSON(http.StatusOK, model.Response{
			Code: 0,
			Msg:  "ok",
			Data: &model.InscriptionEventsResultsResp{
				Total:  total,
				Cursor: cursor,
				Detail: make([]*model.InscriptionEventResp, 0),
			},
		})
		return
	}

	results, err := serviceNFT.GetInscriptionEventsByHeightRange(cursor, size, blkStartHeight, blkEndHeight, isBrc20)
	if err != nil {
		logger.Log.Info("get nft events failed", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get nft events failed"})
		return
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: &model.InscriptionEventsResultsResp{
			Total:  total,
			Cursor: cursor,
			Detail: results,
		},
	})
}

// GetInscriptionEventsByHeightRange
// @Summary Get events related to an inscription within a height range
// @Tags Inscription
// @Produce  json
// @Param inscriptionId path string true "InscriptionID" default("")
// @Param start query int true "Start Block Height" default(0)
// @Param end query int true "End Block Height" default(0)
// @Param cursor query int true "Start cursor" default(0)
// @Param size query int true "Number of records to return" default(16)
// @Success 200 {object} model.Response{data=model.InscriptionEventsResultsResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /inscriptions/events/{inscriptionId} [get]
func GetInscriptionEventsByHeightRange(ctx *gin.Context) {
	logger.Log.Info("GetInscriptionEventsByHeightRange enter")

	inscriptionId := ctx.Param("inscriptionId")
	if err := utils.VerifyInscriptionId(inscriptionId); err != nil {
		logger.Log.Info("inscriptionId invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: err.Error()})
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
	if err != nil || size <= 0 || cursor+size > MAX_INSCRIPTION_LIMIT {
		logger.Log.Info("size invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "size invalid"})
		return
	}

	inscriptionCreatePoints, err := service.GetCreateIdxByInscriptionIdFromRedis([]string{inscriptionId})
	if err != nil || len(inscriptionCreatePoints) != 1 || inscriptionCreatePoints[0].Height == 0 {
		logger.Log.Info("get nft index failed", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get nft index failed"})
		return
	}
	nftPoint := inscriptionCreatePoints[0]

	total, err := serviceNFT.GetInscriptionEventsCountByCreatePoint(nftPoint)
	if err != nil {
		logger.Log.Info("get nft events count failed", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get nft events count failed"})
		return
	}
	if total == 0 {
		ctx.JSON(http.StatusOK, model.Response{
			Code: 0,
			Msg:  "ok",
			Data: &model.InscriptionEventsResultsResp{
				Total:  total,
				Cursor: cursor,
				Detail: make([]*model.InscriptionEventResp, 0),
			},
		})
		return
	}

	sortString := ctx.DefaultQuery("sort", "asc")
	if sortString != "asc" && sortString != "desc" {
		logger.Log.Info("sort invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "sort invalid"})
		return
	}

	results, err := serviceNFT.GetInscriptionEventsByHeightRangeAndCreatePoint(cursor, size, sortString, nftPoint)
	if err != nil {
		logger.Log.Info("get nft events failed", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get nft events failed"})
		return
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: &model.InscriptionEventsResultsResp{
			Total:  total,
			Cursor: cursor,
			Detail: results,
		},
	})
}
