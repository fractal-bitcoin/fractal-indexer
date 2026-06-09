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

func parseMetricHeightRange(c *gin.Context) (int, int, int, bool) {
	fromHeightStr := c.DefaultQuery("fromHeight", "0")
	toHeightStr := c.DefaultQuery("toHeight", "0")
	intervalStr := c.DefaultQuery("interval", "10")

	fromHeight, err := strconv.Atoi(fromHeightStr)
	if err != nil {
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "invalid fromHeight", Data: nil})
		return 0, 0, 0, false
	}
	toHeight, err := strconv.Atoi(toHeightStr)
	if err != nil {
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "invalid toHeight", Data: nil})
		return 0, 0, 0, false
	}
	interval, err := strconv.Atoi(intervalStr)
	if err != nil || !isBlockMetricIntervalAllowed(interval) {
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "invalid interval", Data: nil})
		return 0, 0, 0, false
	}
	if fromHeight < 0 || toHeight <= fromHeight || toHeight-fromHeight > maxBlockMetricRange {
		logger.Log.Info("invalid metric height range", zap.Int("fromHeight", fromHeight), zap.Int("toHeight", toHeight))
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "invalid fromHeight or toHeight", Data: nil})
		return 0, 0, 0, false
	}
	return fromHeight, toHeight, interval, true
}

func isBlockMetricIntervalAllowed(interval int) bool {
	switch interval {
	case 1, 10, 20, 50, 100, 500:
		return true
	default:
		return false
	}
}

func GetBlockMetricsByHeightRange(c *gin.Context) {
	fromHeight, toHeight, interval, ok := parseMetricHeightRange(c)
	if !ok {
		return
	}

	points, err := service.GetBlockMetricsByHeightRange(fromHeight, toHeight, interval)
	if err != nil {
		logger.Log.Error("GetBlockMetricsByHeightRange failed", zap.Error(err), zap.Int("fromHeight", fromHeight), zap.Int("toHeight", toHeight))
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "failed", Data: nil})
		return
	}
	c.JSON(http.StatusOK, model.Response{Code: 0, Msg: "ok", Data: points})
}

func GetBlockMetricsDemo(c *gin.Context) {
	fromHeight, toHeight, interval, ok := parseMetricHeightRange(c)
	if !ok {
		return
	}

	points, err := service.GetBlockMetricsByHeightRange(fromHeight, toHeight, interval)
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
		"Interval":   interval,
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
		input, select { width: 120px; padding: 6px 8px; border: 1px solid #cbd5e1; border-radius: 4px; }
		select { width: 90px; }
		input[type="checkbox"] { width: auto; }
		button { padding: 7px 12px; border: 1px solid #334155; border-radius: 4px; background: #334155; color: #fff; cursor: pointer; }
		.controls { display: flex; gap: 12px; flex-wrap: wrap; margin: 12px 0; font-size: 13px; }
		.controls label { display: inline-flex; align-items: center; gap: 5px; }
		.swatch { display: inline-block; width: 10px; height: 10px; background: var(--c); }
		.summary { display: flex; gap: 18px; flex-wrap: wrap; margin: 10px 0 14px; font-size: 13px; color: #52606d; }
		canvas { width: 100%; max-width: 1200px; height: 520px; border: 1px solid #d9e2ec; border-radius: 6px; }
	</style>
</head>
<body>
	<h2>Block Metrics {{.FromHeight}} - {{.ToHeight}}</h2>
	<form class="toolbar" method="get">
		<label>from <input name="fromHeight" value="{{.FromHeight}}"></label>
		<label>to <input name="toHeight" value="{{.ToHeight}}"></label>
		<label>interval
			<select name="interval">
				<option value="10" {{if eq .Interval 10}}selected{{end}}>10</option>
				<option value="20" {{if eq .Interval 20}}selected{{end}}>20</option>
				<option value="50" {{if eq .Interval 50}}selected{{end}}>50</option>
				<option value="100" {{if eq .Interval 100}}selected{{end}}>100</option>
				<option value="500" {{if eq .Interval 500}}selected{{end}}>500</option>
			</select>
		</label>
		<button type="submit">Load</button>
		<button type="button" id="randomRange">Random</button>
	</form>
	<div id="controls" class="controls">
		<label><input type="checkbox" data-key="txCount"><span class="swatch" style="--c:#111827"></span>total tx</label>
		<label><input type="checkbox" data-key="witness"><span class="swatch" style="--c:#2563eb"></span>witness</label>
		<label><input type="checkbox" data-key="opreturn" checked><span class="swatch" style="--c:#dc2626"></span>opreturn</label>
		<label><input type="checkbox" data-key="inscription" checked><span class="swatch" style="--c:#16a34a"></span>inscription</label>
		<label><input type="checkbox" data-key="runesRunestone" checked><span class="swatch" style="--c:#9333ea"></span>runes runestone</label>
		<label><input type="checkbox" data-key="runesEtching" checked><span class="swatch" style="--c:#ea580c"></span>runes etching</label>
		<label><input type="checkbox" data-key="tacit" checked><span class="swatch" style="--c:#0891b2"></span>tacit</label>
		<label><input type="checkbox" data-key="alkanes" checked><span class="swatch" style="--c:#4b5563"></span>alkanes</label>
		<label><input type="checkbox" data-key="other" checked><span class="swatch" style="--c:#64748b"></span>other</label>
	</div>
	<div id="summary" class="summary"></div>
	<canvas id="chart" width="1200" height="520"></canvas>
	<script>
	const points = {{ .PointsJSON }};
	const protocolKeys = ["witness", "opreturn", "inscription", "runesRunestone", "runesEtching", "tacit", "alkanes"];
	const seriesConfig = {
		txCount: ["Total Tx", "#111827"],
		witness: ["Witness", "#2563eb"],
		opreturn: ["OpReturn", "#dc2626"],
		inscription: ["Inscription", "#16a34a"],
		runesRunestone: ["Runes Runestone", "#9333ea"],
		runesEtching: ["Runes Etching", "#ea580c"],
		tacit: ["Tacit", "#0891b2"],
		alkanes: ["Alkanes", "#4b5563"],
		other: ["Other", "#64748b"],
	};
	const canvas = document.getElementById("chart");
	const ctx = canvas.getContext("2d");
	const summary = document.getElementById("summary");
	const pad = { left: 58, right: 20, top: 20, bottom: 38 };
	const w = canvas.width, h = canvas.height;
	const innerW = w - pad.left - pad.right;
	const innerH = h - pad.top - pad.bottom;
	const checks = Array.from(document.querySelectorAll("#controls input[type=checkbox]"));
	const form = document.querySelector(".toolbar");
	const fromInput = form.querySelector("input[name='fromHeight']");
	const toInput = form.querySelector("input[name='toHeight']");
	const intervalSelect = form.querySelector("select[name='interval']");
	const maxKnownHeight = points.length > 0 ? points[points.length - 1].toHeight : Number(toInput.value || 0);
	function intervalForSpan(span) {
		if (span <= 200) return 10;
		if (span <= 400) return 20;
		if (span <= 700) return 50;
		if (span <= 1000) return 100;
		return 500;
	}
	function randomizeRange() {
		const currentFrom = Math.max(0, Number(fromInput.value || 0));
		const currentTo = Math.max(currentFrom + 1, Number(toInput.value || currentFrom + 1000));
		const upperBound = Math.max(maxKnownHeight, currentTo, currentFrom + 1000);
		const span = 10 + Math.floor(Math.random() * 991);
		const maxStart = Math.max(0, upperBound - span);
		const start = Math.floor(Math.random() * (maxStart + 1));
		fromInput.value = String(start);
		toInput.value = String(start + span);
		intervalSelect.value = String(intervalForSpan(span));
		form.submit();
	}
	function selectedProtocolKeys() {
		return protocolKeys.filter((key) => document.querySelector("input[data-key='" + key + "']").checked);
	}
	function selectedSeries() {
		return checks.filter((el) => el.checked).map((el) => el.dataset.key);
	}
	function selectedProtocolSum(p) {
		return selectedProtocolKeys().reduce((sum, key) => sum + (p[key] || 0), 0);
	}
	function otherValue(p) {
		return Math.max(0, (p.txCount || 0) - selectedProtocolSum(p));
	}
	function valueOf(p, key) {
		return key === "other" ? otherValue(p) : (p[key] || 0);
	}
	function renderSummary() {
		const totalTx = points.reduce((sum, p) => sum + (p.txCount || 0), 0);
		const selected = points.reduce((sum, p) => sum + selectedProtocolSum(p), 0);
		const other = points.reduce((sum, p) => sum + otherValue(p), 0);
		const pct = totalTx > 0 ? (other * 100 / totalTx).toFixed(2) : "0.00";
		summary.innerHTML = [
			"total tx: " + totalTx.toLocaleString(),
			"selected protocol tx: " + selected.toLocaleString(),
			"other: " + other.toLocaleString() + " (" + pct + "%)"
		].map((text) => "<span>" + text + "</span>").join("");
	}
	function draw() {
		const active = selectedSeries();
		let maxY = 1;
		for (const p of points) for (const key of active) maxY = Math.max(maxY, valueOf(p, key));
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
		for (const key of active) {
			const [label, color] = seriesConfig[key];
			ctx.strokeStyle = color;
			ctx.lineWidth = key === "txCount" || key === "other" ? 2.5 : 2;
			ctx.beginPath();
			points.forEach((p, i) => {
				const x = xAt(i), y = yAt(valueOf(p, key));
				if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
			});
			ctx.stroke();
		}
		if (points.length > 0) {
			ctx.fillStyle = "#52606d";
			ctx.fillText(String(points[0].height), pad.left, h - 12);
			ctx.fillText(String(points[points.length - 1].toHeight), w - pad.right - 70, h - 12);
		}
		renderSummary();
	}
	checks.forEach((el) => el.addEventListener("change", draw));
	document.getElementById("randomRange").addEventListener("click", randomizeRange);
	draw();
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
