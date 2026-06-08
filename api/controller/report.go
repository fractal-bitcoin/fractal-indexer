package controller

import (
	"encoding/json"
	"fractal-indexer/api/model"
	"fractal-indexer/api/service"
	"fractal-indexer/logger"
	"html/template"
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

const maxBlockMetricRange = 10000

func parseMetricHeightRange(c *gin.Context) (int, int, bool) {
	fromHeightStr := c.DefaultQuery("fromHeight", "0")
	toHeightStr := c.DefaultQuery("toHeight", "0")

	fromHeight, err := strconv.Atoi(fromHeightStr)
	if err != nil {
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "invalid fromHeight", Data: nil})
		return 0, 0, false
	}
	toHeight, err := strconv.Atoi(toHeightStr)
	if err != nil {
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "invalid toHeight", Data: nil})
		return 0, 0, false
	}
	if fromHeight < 0 || toHeight <= fromHeight || toHeight-fromHeight > maxBlockMetricRange {
		logger.Log.Info("invalid metric height range", zap.Int("fromHeight", fromHeight), zap.Int("toHeight", toHeight))
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "invalid fromHeight or toHeight", Data: nil})
		return 0, 0, false
	}
	return fromHeight, toHeight, true
}

func GetBlockMetricsByHeightRange(c *gin.Context) {
	fromHeight, toHeight, ok := parseMetricHeightRange(c)
	if !ok {
		return
	}

	points, err := service.GetBlockMetricsByHeightRange(fromHeight, toHeight)
	if err != nil {
		logger.Log.Error("GetBlockMetricsByHeightRange failed", zap.Error(err), zap.Int("fromHeight", fromHeight), zap.Int("toHeight", toHeight))
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "failed", Data: nil})
		return
	}
	c.JSON(http.StatusOK, model.Response{Code: 0, Msg: "ok", Data: points})
}

func GetBlockMetricsDemo(c *gin.Context) {
	fromHeight, toHeight, ok := parseMetricHeightRange(c)
	if !ok {
		return
	}

	points, err := service.GetBlockMetricsByHeightRange(fromHeight, toHeight)
	if err != nil {
		logger.Log.Error("GetBlockMetricsDemo failed", zap.Error(err), zap.Int("fromHeight", fromHeight), zap.Int("toHeight", toHeight))
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	pointsJSON, err := json.Marshal(points)
	if err != nil {
		logger.Log.Error("marshal block metrics demo data failed", zap.Error(err))
		c.String(http.StatusInternalServerError, "failed")
		return
	}

	tmpl := template.Must(template.New("block-metrics-demo").Parse(blockMetricsDemoHTML))
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(c.Writer, gin.H{
		"FromHeight": fromHeight,
		"ToHeight":   toHeight,
		"PointsJSON": template.JS(pointsJSON),
	}); err != nil {
		logger.Log.Error("render block metrics demo failed", zap.Error(err))
	}
}

const blockMetricsDemoHTML = `<!doctype html>
<html>
<head>
	<meta charset="utf-8">
	<title>Block Metrics</title>
	<style>
		body { margin: 24px; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; color: #1f2933; }
		.toolbar { display: flex; gap: 8px; align-items: center; margin-bottom: 16px; }
		input { width: 120px; padding: 6px 8px; border: 1px solid #cbd5e1; border-radius: 4px; }
		button { padding: 7px 12px; border: 1px solid #334155; border-radius: 4px; background: #334155; color: #fff; cursor: pointer; }
		.legend { display: flex; gap: 14px; flex-wrap: wrap; margin: 12px 0; font-size: 13px; }
		.legend span::before { content: ""; display: inline-block; width: 10px; height: 10px; margin-right: 5px; background: var(--c); }
		canvas { width: 100%; max-width: 1200px; height: 520px; border: 1px solid #d9e2ec; border-radius: 6px; }
	</style>
</head>
<body>
	<h2>Block Metrics {{.FromHeight}} - {{.ToHeight}}</h2>
	<form class="toolbar" method="get">
		<label>from <input name="fromHeight" value="{{.FromHeight}}"></label>
		<label>to <input name="toHeight" value="{{.ToHeight}}"></label>
		<button type="submit">Load</button>
	</form>
	<div class="legend">
		<span style="--c:#2563eb">witness</span>
		<span style="--c:#dc2626">opreturn</span>
		<span style="--c:#16a34a">inscription</span>
		<span style="--c:#9333ea">runes runestone</span>
		<span style="--c:#ea580c">runes etching</span>
		<span style="--c:#0891b2">tacit</span>
		<span style="--c:#4b5563">alkanes</span>
	</div>
	<canvas id="chart" width="1200" height="520"></canvas>
	<script>
	const points = {{ .PointsJSON }};
	const series = [
		["witness", "Witness", "#2563eb"],
		["opreturn", "OpReturn", "#dc2626"],
		["inscription", "Inscription", "#16a34a"],
		["runesRunestone", "Runes Runestone", "#9333ea"],
		["runesEtching", "Runes Etching", "#ea580c"],
		["tacit", "Tacit", "#0891b2"],
		["alkanes", "Alkanes", "#4b5563"],
	];
	const canvas = document.getElementById("chart");
	const ctx = canvas.getContext("2d");
	const pad = { left: 58, right: 20, top: 20, bottom: 38 };
	const w = canvas.width, h = canvas.height;
	const innerW = w - pad.left - pad.right;
	const innerH = h - pad.top - pad.bottom;
	let maxY = 1;
	for (const p of points) for (const [key] of series) maxY = Math.max(maxY, p[key] || 0);
	ctx.clearRect(0, 0, w, h);
	ctx.strokeStyle = "#d9e2ec";
	ctx.lineWidth = 1;
	ctx.beginPath();
	ctx.moveTo(pad.left, pad.top);
	ctx.lineTo(pad.left, h - pad.bottom);
	ctx.lineTo(w - pad.right, h - pad.bottom);
	ctx.stroke();
	ctx.fillStyle = "#52606d";
	ctx.font = "12px sans-serif";
	for (let i = 0; i <= 4; i++) {
		const yVal = Math.round(maxY * i / 4);
		const y = h - pad.bottom - innerH * i / 4;
		ctx.fillText(String(yVal), 10, y + 4);
		ctx.strokeStyle = "#eef2f7";
		ctx.beginPath();
		ctx.moveTo(pad.left, y);
		ctx.lineTo(w - pad.right, y);
		ctx.stroke();
	}
	function xAt(i) { return pad.left + (points.length <= 1 ? 0 : innerW * i / (points.length - 1)); }
	function yAt(v) { return h - pad.bottom - innerH * v / maxY; }
	for (const [key, , color] of series) {
		ctx.strokeStyle = color;
		ctx.lineWidth = 2;
		ctx.beginPath();
		points.forEach((p, i) => {
			const x = xAt(i), y = yAt(p[key] || 0);
			if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
		});
		ctx.stroke();
	}
	if (points.length > 0) {
		ctx.fillStyle = "#52606d";
		ctx.fillText(String(points[0].height), pad.left, h - 12);
		ctx.fillText(String(points[points.length - 1].height), w - pad.right - 70, h - 12);
	}
	</script>
</body>
</html>`

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
