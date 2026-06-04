package controller

import (
	"fractal-indexer/api/model"
	"fractal-indexer/api/service"
	"fractal-indexer/logger"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type responseCoreDataUpToHeight struct {
	Hash              string `json:"hash"`              // Hash of all following fields for fast comparison.
	HashDBData        string `json:"hashDBData"`        // Hash of data fetched from DB.
	HashMemoryData    string `json:"hashMemoryData"`    // Hash of in-memory program data.
	Height            uint64 `json:"height"`            // Specified height.
	TxCount           uint64 `json:"txCount"`           // Up to the specified height.
	TxInValues        uint64 `json:"txInValues"`        // Up to the specified height.
	TxOutValues       uint64 `json:"txOutValues"`       // Up to the specified height.
	NFTNewCount       uint64 `json:"nftNewCount"`       // Up to the specified height.
	NFTInCount        uint64 `json:"nftInCount"`        // Up to the specified height.
	NFTOutCount       uint64 `json:"nftOutCount"`       // Up to the specified height.
	NFTLostCount      uint64 `json:"nftLostCount"`      // Up to the specified height.
	BRC20HistoryCount uint64 `json:"brc20HistoryCount"` // Up to the specified height.
}

func GetCoreDataUpToHeight(ctx *gin.Context) {
	heightStr := ctx.Param("height")
	height, err := strconv.Atoi(heightStr)
	if err != nil {
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "invalid height", Data: nil})
		return
	}
	if height < 0 {
		logger.Log.Warn("get core data upto height, invalid height", zap.Int("height", height))
		ctx.JSON(http.StatusOK, model.Response{Code: -1, Msg: "invalid height", Data: nil})
		return
	}

	coreData := service.GetCoreDataUpToHeight(height)
	var resp responseCoreDataUpToHeight
	resp.Hash = coreData.Hash
	resp.Height = coreData.Height
	resp.TxCount = coreData.TxCount
	resp.TxInValues = coreData.TxInValue
	resp.TxOutValues = coreData.TxOutValue
	resp.NFTNewCount = coreData.NFTNewCount
	resp.NFTInCount = coreData.NFTInCount
	resp.NFTOutCount = coreData.NFTOutCount
	resp.NFTLostCount = coreData.NFTLostCount
	resp.BRC20HistoryCount = coreData.BRC20HistoryCount
	resp.HashDBData = coreData.HashDBData
	resp.HashMemoryData = coreData.HashMemoryData
	ctx.JSON(http.StatusOK, model.Response{Code: 0, Msg: "ok", Data: resp})
}

type heightInvalue struct {
	Height  int `json:"height"`
	Invalue int `json:"invalue"`
}

type responseHeightsInvalue []heightInvalue

func GetLatestBlocksHeightAndInvalue(c *gin.Context) {
	sizeStr := c.DefaultQuery("size", "20")
	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		logger.Log.Info("invalid size", zap.String("size", sizeStr))
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "invalid size", Data: nil})
		return
	}
	if size < 0 {
		logger.Log.Info("invalid size", zap.String("size", sizeStr))
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "invalid size", Data: nil})
		return
	}

	invalues, err := service.GetLatestBlocksHeightAndInvalue(uint32(size))
	if err != nil {
		logger.Log.Error("GetLatestBlocksHeightAndInvalue failed", zap.Error(err))
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "failed", Data: nil})
		return
	}

	resp := make(responseHeightsInvalue, 0, len(invalues))
	for _, invalue := range invalues {
		resp = append(resp, heightInvalue{Height: invalue.Height, Invalue: invalue.Invalue})
	}
	c.JSON(http.StatusOK, model.Response{Code: 0, Msg: "ok", Data: resp})
}

func GetBlocksHeightAndInvalueRange(c *gin.Context) {
	fromHeightStr := c.DefaultQuery("fromHeight", "0")
	toHeightStr := c.DefaultQuery("toHeight", "0")

	fromHeight, err := strconv.Atoi(fromHeightStr)
	if err != nil {
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "invalid fromHeight", Data: nil})
		return
	}
	toHeight, err := strconv.Atoi(toHeightStr)
	if err != nil {
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "invalid toHeight", Data: nil})
		return
	}
	if fromHeight < 0 || toHeight < 0 || toHeight <= fromHeight {
		logger.Log.Info("invalid fromHeight or toHeight", zap.Int("fromHeight", fromHeight), zap.Int("toHeight", toHeight))
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "invalid fromHeight or toHeight", Data: nil})
		return
	}

	invalues, err := service.GetInvalueByHeightRange(fromHeight, toHeight)
	if err != nil {
		logger.Log.Error("GetInvalueByHeightRange failed", zap.Error(err), zap.Int("fromHeight", fromHeight), zap.Int("toHeight", toHeight))
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "failed", Data: nil})
		return
	}

	resp := make(responseHeightsInvalue, 0, len(invalues))
	for _, invalue := range invalues {
		resp = append(resp, heightInvalue{Height: invalue.Height, Invalue: invalue.Invalue})
	}
	c.JSON(http.StatusOK, model.Response{Code: 0, Msg: "ok", Data: resp})
}

type heightNFTIn struct {
	Height int `json:"height"`
	NFTIn  int `json:"nftin"`
}

type responseHeightNFTIns []heightNFTIn

func GetLatestBlocksHeightAndNFTIn(c *gin.Context) {
	sizeStr := c.DefaultQuery("size", "20")
	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		logger.Log.Info("invalid size", zap.String("size", sizeStr))
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "invalid size", Data: nil})
		return
	}
	if size < 0 {
		logger.Log.Info("invalid size", zap.String("size", sizeStr))
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "invalid size", Data: nil})
		return
	}

	nftins, err := service.GetLatestBlocksHeightAndNFTIn(uint32(size))
	if err != nil {
		logger.Log.Error("GetLatestBlocksHeightAndNFTIn failed", zap.Error(err))
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "failed", Data: nil})
		return
	}

	resp := make(responseHeightNFTIns, 0, len(nftins))
	for _, nftin := range nftins {
		resp = append(resp, heightNFTIn{Height: nftin.Height, NFTIn: nftin.NFTIn})
	}
	c.JSON(http.StatusOK, model.Response{Code: 0, Msg: "ok", Data: resp})
}

func GetBlocksHeightAndNFTInRange(c *gin.Context) {
	fromHeightStr := c.DefaultQuery("fromHeight", "0")
	toHeightStr := c.DefaultQuery("toHeight", "0")

	fromHeight, err := strconv.Atoi(fromHeightStr)
	if err != nil {
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "invalid fromHeight", Data: nil})
		return
	}
	toHeight, err := strconv.Atoi(toHeightStr)
	if err != nil {
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "invalid toHeight", Data: nil})
		return
	}
	if fromHeight < 0 || toHeight < 0 || toHeight <= fromHeight {
		logger.Log.Info("invalid fromHeight or toHeight", zap.Int("fromHeight", fromHeight), zap.Int("toHeight", toHeight))
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "invalid fromHeight or toHeight", Data: nil})
		return
	}

	inValues, err := service.GetNFTInByHeightRange(fromHeight, toHeight)
	if err != nil {
		logger.Log.Error("GetNFTInByHeightRange failed", zap.Error(err), zap.Int("fromHeight", fromHeight), zap.Int("toHeight", toHeight))
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "failed", Data: nil})
		return
	}

	resp := make(responseHeightNFTIns, 0, len(inValues))
	for _, inValue := range inValues {
		resp = append(resp, heightNFTIn{Height: inValue.Height, NFTIn: inValue.NFTIn})
	}
	c.JSON(http.StatusOK, model.Response{Code: 0, Msg: "ok", Data: resp})
}
