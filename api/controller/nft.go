package controller

import (
	"fractal-query/lib/utils"
	"fractal-query/logger"
	"fractal-query/model"
	"fractal-query/service"
	events "fractal-query/service/nft"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const MAX_INSCRIPTION_LIMIT = 40960

// GetInscriptionsSummaryByHeightRange
// @Summary Get inscription summaries within a height range
// @Tags Inscription
// @Produce  json
// @Param start query int true "Start Block Height" default(0)
// @Param end query int true "End Block Height" default(0)
// @Param limit query int true "number of nft" default(10)
// @Success 200 {object} model.Response{data=model.InscriptionsSummaryResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /inscriptions-summary [get]
func GetInscriptionsSummaryByHeightRange(ctx *gin.Context) {
	logger.Log.Info("GetInscriptionsSummaryByHeightRange enter")

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

	limitString := ctx.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitString)
	if err != nil || limit < 2 || limit > 64 {
		logger.Log.Info("limit invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "limit invalid"})
		return
	}

	results, err := service.GetInscriptionsSummaryByHeightRange(blkStartHeight, blkEndHeight)
	if err != nil {
		logger.Log.Info("get nft summary failed", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get nft summary failed"})
		return
	}

	resp := &model.InscriptionsSummaryResp{Detail: make([]*model.InscriptionCountByContentTypeResp, 0)}
	countMap := make(map[string]int)
	for _, typeCount := range results {
		summaryType := service.GetSummaryTypeByContentType(typeCount.ContentType)
		if n, ok := countMap[summaryType]; ok {
			countMap[summaryType] = n + typeCount.Count
		} else {
			countMap[summaryType] = typeCount.Count
		}
	}

	summaryTypeList := []string{}
	summaryTypeCountList := []int{}
	if n, ok := countMap["image"]; ok {
		summaryTypeList = append(summaryTypeList, "image")
		summaryTypeCountList = append(summaryTypeCountList, n)
	}
	if n, ok := countMap["text"]; ok {
		summaryTypeList = append(summaryTypeList, "text")
		summaryTypeCountList = append(summaryTypeCountList, n)
	}
	if n, ok := countMap["video"]; ok {
		summaryTypeList = append(summaryTypeList, "video")
		summaryTypeCountList = append(summaryTypeCountList, n)
	}
	if n, ok := countMap["audio"]; ok {
		summaryTypeList = append(summaryTypeList, "audio")
		summaryTypeCountList = append(summaryTypeCountList, n)
	}
	if n, ok := countMap["others"]; ok {
		summaryTypeList = append(summaryTypeList, "others")
		summaryTypeCountList = append(summaryTypeCountList, n)
	}

	for idx, summaryType := range summaryTypeList {
		limitMul := 1
		if idx < 2 {
			limitMul = 6
		}
		cntRsp := &model.InscriptionCountByContentTypeResp{
			ContentType:  summaryType,
			Count:        summaryTypeCountList[idx],
			Inscriptions: make([]*model.InscriptionResp, 0),
		}
		resp.Total += summaryTypeCountList[idx]
		resp.Detail = append(resp.Detail, cntRsp)

		var inscriptionCreateIdxes []uint64
		var inscriptionContents []string
		if summaryType != "text" {
			inscriptionCreateIdxes, err = service.GetLatestNFTCreateIdxBySummaryTypeAndHeightRange(summaryType, blkStartHeight, blkEndHeight,
				limit*limitMul/2)
			if err != nil {
				logger.Log.Info("get nft summary createIdx failed", zap.String("contentType", summaryType), zap.Error(err))
				continue
				// ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get nft summary createIdx failed"})
				// return
			}
		} else {
			inscriptionCreateIdxes, inscriptionContents, err = service.GetLatestNFTCreateIdxAndContentBySummaryTypeAndHeightRange(blkStartHeight, blkEndHeight,
				limit*limitMul/2)
			if err != nil {
				logger.Log.Info("get nft summary createIdx failed", zap.String("contentType", summaryType), zap.Error(err))
				continue
				// ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get nft summary createIdx failed"})
				// return
			}
		}

		nftsRsp, err := service.GetInscriptionsByCreateIdxes(inscriptionCreateIdxes)
		if err != nil {
			logger.Log.Info("get nft detail failed", zap.Error(err))
			continue
			// ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get nft detail failed"})
			// return
		}

		if summaryType == "text" && len(inscriptionContents) == len(nftsRsp) {
			for idx, nft := range nftsRsp {
				nft.ContentBody = inscriptionContents[idx]
			}
		}
		cntRsp.Inscriptions = nftsRsp
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: resp,
	})
}

// GetInscriptionsByTagAndHeightRange
// @Summary Get inscriptions by type within a height range
// @Tags Inscription
// @Produce  json
// @Param start query int true "Start Block Height" default(0)
// @Param end query int true "End Block Height" default(0)
// @Param limit query int true "number of nft" default(10)
// @Param tag path string true "content type tag" default("image")
// @Success 200 {object} model.Response{data=[]model.InscriptionResp} "{"code": 0, "data": [{}], "msg": "ok"}"
// @Security BearerAuth
// @Router /inscriptions/{tag} [get]
func GetInscriptionsByTagAndHeightRange(ctx *gin.Context) {
	logger.Log.Info("GetInscriptionsByTagAndHeightRange enter")

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

	limitString := ctx.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitString)
	if err != nil || limit < 1 || limit > 128 {
		logger.Log.Info("limit invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "limit invalid"})
		return
	}

	summaryType := ctx.Param("tag")
	var summaryTypeMap map[string]bool = map[string]bool{
		"image":  true,
		"text":   true,
		"audio":  true,
		"video":  true,
		"others": true,
	}
	if ok := summaryTypeMap[summaryType]; !ok {
		logger.Log.Info("tag invalid")
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "tag invalid"})
		return
	}

	inscriptionCreateIdxes, err := service.GetLatestNFTCreateIdxBySummaryTypeAndHeightRange(summaryType, blkStartHeight, blkEndHeight, limit)
	if err != nil {
		logger.Log.Info("get nft summary createIdx failed", zap.String("contentType", summaryType), zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get nft summary createIdx failed"})
		return
	}
	nftsRsp, err := service.GetInscriptionsByCreateIdxes(inscriptionCreateIdxes)
	if err != nil {
		logger.Log.Error("get nft detail failed", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get nft detail failed"})
		return
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: nftsRsp,
	})
}

// GetInscriptionContent
// @Summary Get download content by Inscription ID
// @Tags Inscription
// @Produce  json
// @Param inscriptionId path string true "InscriptionID" default("")
// @Success 200 {file} file
// @Security BearerAuth
// @Router /inscription/content/{inscriptionId} [get]
func GetInscriptionContent(ctx *gin.Context) {
	logger.Log.Info("GetInscriptionContent enter")

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
	nftPoint := inscriptionCreatePoints[0]
	contentType, contentBody, err := service.GetNFTContentByCreateIdx(int(nftPoint.Height), int(nftPoint.IdxInBlock))
	if err != nil {
		logger.Log.Info("get nft detail failed", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get nft detail failed"})
		return
	}
	ctx.Data(http.StatusOK, contentType, []byte(contentBody))
}

// GetInscriptionInfo
// @Summary Get details by Inscription ID
// @Tags Inscription
// @Produce  json
// @Param inscriptionId path string true "InscriptionID" default("")
// @Success 200 {object} model.Response{data=model.InscriptionResp} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /inscription/info/{inscriptionId} [get]
func GetInscriptionInfo(ctx *gin.Context) {
	logger.Log.Info("GetInscriptionInfo enter")

	var inscriptionCreateIdxUint64 uint64
	inscriptionId := ctx.Param("inscriptionId")
	if err := utils.VerifyInscriptionId(inscriptionId); err == nil {
		inscriptionCreatePoints, err := service.GetCreateIdxByInscriptionIdFromRedis([]string{inscriptionId})
		if err != nil || len(inscriptionCreatePoints) != 1 || inscriptionCreatePoints[0].Height == 0 {
			logger.Log.Info("get nft index failed", zap.Error(err))
			ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get nft index failed"})
			return
		}
		inscriptionCreateIdxUint64 = inscriptionCreatePoints[0].GetCreateIdxKey()
	} else if err2 := utils.VerifyInscriptionNumber(inscriptionId); err2 == nil {
		oCreatePoint, err := service.GetNftCreatePointByNftNumber(inscriptionId)
		if err != nil {
			logger.Log.Info("get nft index failed", zap.Error(err))
			ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get nft index failed"})
			return
		}
		inscriptionCreateIdxUint64 = oCreatePoint.GetCreateIdxKey()
	} else {
		logger.Log.Info("inscriptionId invalid", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: err.Error()})
		return
	}

	nftRsp, err := service.GetInscriptionsByCreateIdxes([]uint64{inscriptionCreateIdxUint64})
	if err != nil || len(nftRsp) != 1 {
		logger.Log.Info("get nft detail failed", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get nft detail failed"})
		return
	}

	if model.GSwap != nil {
		nftRsp[0].BRC20BestHeight = uint32(len(model.GlobalBlocksHash) - 1)
		validTransferInfo, ok := model.GSwapBase.InscriptionsValidBRC20DataMap[inscriptionCreateIdxUint64]
		if ok {
			nftRsp[0].BRC20 = validTransferInfo
		} else {
			validTransferInfo, ok := model.GSwap.InscriptionsValidBRC20DataMap[inscriptionCreateIdxUint64]
			if ok {
				nftRsp[0].BRC20 = validTransferInfo
			}
		}
	}
	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: nftRsp[0],
	})

}

// GetInscriptionInfoBatch
// @Summary Get details by a list of Inscription IDs
// @Tags Inscription
// @Produce  json
// @Param body body model.GetInscriptionInfoBatchRequest true "inscription list"
// @Success 200 {object} model.Response{data=[]model.InscriptionResp} "{"code": 0, "data": [{}], "msg": "ok"}"
// @Security BearerAuth
// @Router /inscriptions/info [post]
func GetInscriptionInfoBatch(ctx *gin.Context) {
	logger.Log.Info("GetInscriptionInfoBatch enter")

	if model.GSwap == nil {
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "brc20 not ready"})
		return
	}

	// check body
	req := model.GetInscriptionInfoBatchRequest{}
	if err := ctx.BindJSON(&req); err != nil {
		logger.Log.Info("Bind json failed", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "json error"})
		return
	}

	if len(req.InscriptionIds) == 0 {
		ctx.JSON(http.StatusOK, model.Response{
			Code: 0,
			Msg:  "ok",
			Data: make([]*model.InscriptionResp, 0),
		})
		return
	}

	for _, inscriptionId := range req.InscriptionIds {
		if err := utils.VerifyInscriptionId(inscriptionId); err != nil {
			logger.Log.Info("inscriptionId invalid", zap.Error(err))
			ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: err.Error()})
			return
		}
	}
	inscriptionCreatePoints, err := service.GetCreateIdxByInscriptionIdFromRedis(req.InscriptionIds)
	if err != nil || len(inscriptionCreatePoints) != len(req.InscriptionIds) {
		logger.Log.Info("get nft index failed", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get nft index failed"})
		return
	}

	inscriptionCreateIdxes := []uint64{}
	for _, point := range inscriptionCreatePoints {
		// if inscriptionCreatePoints[0].Height == 0
		inscriptionCreateIdxes = append(inscriptionCreateIdxes, point.GetCreateIdxKey())
	}

	// fixme: no need utxo
	nftsRsp, err := service.GetInscriptionsByCreateIdxes(inscriptionCreateIdxes)
	if err != nil || len(nftsRsp) != len(inscriptionCreateIdxes) {
		logger.Log.Info("get nft detail failed", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "get nft detail failed"})
		return
	}

	for idx, nft := range nftsRsp {
		inscriptionCreate := inscriptionCreateIdxes[idx]

		validTransferInfo, ok := model.GSwapBase.InscriptionsValidBRC20DataMap[inscriptionCreate]
		if ok {
			nft.BRC20 = validTransferInfo
		} else {
			validTransferInfo, ok := model.GSwap.InscriptionsValidBRC20DataMap[inscriptionCreate]
			if ok {
				nft.BRC20 = validTransferInfo
			}
		}
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: nftsRsp,
	})

}

type queryInscriptionHeightByNFTIdRequest struct {
	NFTIDList []string `json:"nftIdList"`
}

// QueryInscriptionHeightByNFTIdList
// @Summary Query NFT heights by NFT ID list
// @Tags Inscription
// @Produce  json
// @Param body body queryInscriptionHeightByNFTIdRequest true "nftIdList"
// @Success 200 {object} model.Response{data=map[string]int} "{"code": 0, "data": {}, "msg": "ok"}"
// @Security BearerAuth
// @Router /inscriptions/height [post]
func QueryInscriptionHeightByNFTIdList(ctx *gin.Context) {
	logger.Log.Info("QueryInscriptionHeightByNFTIdList enter")

	req := queryInscriptionHeightByNFTIdRequest{}
	if err := ctx.BindJSON(&req); err != nil {
		logger.Log.Error("failed to parse request arguments", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "faild to parse request arguments"})
		return
	}
	if len(req.NFTIDList) > 1000 {
		logger.Log.Warn("nft id list is too large", zap.Int("length of nft id list", len(req.NFTIDList)))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "nft id list is too large, keep it less than 1000"})
		return
	}

	nftHeight, err := events.GetNFTBlockHeightByNFTId(ctx, req.NFTIDList...)
	if err != nil {
		logger.Log.Error("failed to get nft height", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "failed to get nft height"})
		return
	}

	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: nftHeight,
	})
}

// GetInscriptionsStatus
// @Summary Query inscription status
// @Tags Inscription
// @Produce  json
// @Success 200 {object} model.Response{data=int} "{"code": 0, "data": 0, "msg": "ok"}"
// @Security BearerAuth
// @Router /inscriptions/status [get]
func GetInscriptionsStatus(ctx *gin.Context) {
	logger.Log.Info("GetInscriptionsStatus enter")

	inscriptionStatusResp, err := service.GetInscriptionsStatus()
	if err != nil {
		logger.Log.Error("failed to get inscriptions status", zap.Error(err))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "failed to get inscriptions status"})
		return
	}
	ctx.JSON(http.StatusOK, model.Response{Code: 0, Msg: "ok", Data: inscriptionStatusResp})
}
