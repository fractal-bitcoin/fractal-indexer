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

const (
	maxBlockMetricPoints      = 10000
	randomBlockMetricMaxRange = 10000
)

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
	if fromHeight < 0 || toHeight <= fromHeight {
		logger.Log.Info("invalid metric height range", zap.Int("fromHeight", fromHeight), zap.Int("toHeight", toHeight))
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "invalid fromHeight or toHeight", Data: nil})
		return 0, 0, 0, false
	}
	pointCount := (toHeight - fromHeight + interval - 1) / interval
	if pointCount > maxBlockMetricPoints {
		logger.Log.Info("metric point count exceeds limit",
			zap.Int("fromHeight", fromHeight),
			zap.Int("toHeight", toHeight),
			zap.Int("interval", interval),
			zap.Int("pointCount", pointCount),
			zap.Int("maxPointCount", maxBlockMetricPoints))
		c.JSON(http.StatusOK, model.Response{Code: -1, Msg: "too many metric points", Data: nil})
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
	maxKnownHeight, err := service.GetBlockMetricsMaxKnownHeight()
	if err != nil {
		logger.Log.Error("GetBlockMetricsMaxKnownHeight failed", zap.Error(err))
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
		"FromHeight":             fromHeight,
		"ToHeight":               toHeight,
		"Interval":               interval,
		"MaxKnownHeight":         maxKnownHeight,
		"MaxBlockMetricPoints":   maxBlockMetricPoints,
		"RandomBlockMetricRange": randomBlockMetricMaxRange,
		"PointsJSON":             template.JS(pointsJSON),
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
		button:disabled { opacity: .65; cursor: wait; }
		.controls { display: flex; gap: 12px; flex-wrap: wrap; margin: 12px 0; font-size: 13px; }
		.controls label { display: inline-flex; align-items: center; gap: 5px; }
		.swatch { display: inline-block; width: 10px; height: 10px; background: var(--c); }
		.status { min-height: 18px; margin: -6px 0 10px; font-size: 13px; color: #52606d; }
		.status.error { color: #dc2626; }
		.summary { display: flex; gap: 18px; flex-wrap: wrap; margin: 10px 0 14px; font-size: 13px; color: #52606d; }
		canvas { width: 100%; max-width: 1200px; height: 520px; border: 1px solid #d9e2ec; border-radius: 6px; }
	</style>
</head>
<body>
	<h2 id="title">Block Metrics {{.FromHeight}} - {{.ToHeight}}</h2>
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
	<div id="status" class="status"></div>
	<div id="controls" class="controls">
		<label><input type="checkbox" data-key="txCount"><span class="swatch" style="--c:#111827"></span>total tx</label>
		<label><input type="checkbox" data-key="witness"><span class="swatch" style="--c:#2563eb"></span>witness</label>
		<label><input type="checkbox" data-key="opreturn" checked><span class="swatch" style="--c:#dc2626"></span>opreturn</label>
		<label><input type="checkbox" data-key="inscription" checked><span class="swatch" style="--c:#16a34a"></span>inscription</label>
		<label><input type="checkbox" data-key="runesRunestone" checked><span class="swatch" style="--c:#9333ea"></span>runestone envelope</label>
		<label><input type="checkbox" data-key="runesEtching" checked><span class="swatch" style="--c:#ea580c"></span>runes etching</label>
		<label><input type="checkbox" data-key="runesMint" checked><span class="swatch" style="--c:#f59e0b"></span>runes mint</label>
		<label><input type="checkbox" data-key="tacit" checked><span class="swatch" style="--c:#0891b2"></span>tacit</label>
		<label><input type="checkbox" data-key="alkanes" checked><span class="swatch" style="--c:#4b5563"></span>alkanes</label>
		<label><input type="checkbox" data-key="other" checked><span class="swatch" style="--c:#64748b"></span>other</label>
	</div>
	<div id="summary" class="summary"></div>
	<canvas id="chart" width="1200" height="520"></canvas>
	<script>
	let points = {{ .PointsJSON }};
	const protocolKeys = ["witness", "opreturn", "inscription", "runesRunestone", "runesEtching", "runesMint", "tacit", "alkanes"];
	const seriesConfig = {
		txCount: ["Total Tx", "#111827"],
		witness: ["Witness", "#2563eb"],
		opreturn: ["OpReturn", "#dc2626"],
		inscription: ["Inscription", "#16a34a"],
		runesRunestone: ["Runestone Envelope", "#9333ea"],
		runesEtching: ["Runes Etching", "#ea580c"],
		runesMint: ["Runes Mint", "#f59e0b"],
		tacit: ["Tacit", "#0891b2"],
		alkanes: ["Alkanes", "#4b5563"],
		other: ["Other", "#64748b"],
	};
	const canvas = document.getElementById("chart");
	const ctx = canvas.getContext("2d");
	const title = document.getElementById("title");
	const status = document.getElementById("status");
	const summary = document.getElementById("summary");
	const pad = { left: 58, right: 20, top: 20, bottom: 38 };
	const w = canvas.width, h = canvas.height;
	const innerW = w - pad.left - pad.right;
	const innerH = h - pad.top - pad.bottom;
	let hoverIndex = -1;
	const checks = Array.from(document.querySelectorAll("#controls input[type=checkbox]"));
	const form = document.querySelector(".toolbar");
	const fromInput = form.querySelector("input[name='fromHeight']");
	const toInput = form.querySelector("input[name='toHeight']");
	const intervalInput = form.querySelector("select[name='interval']");
	const loadButton = form.querySelector("button[type='submit']");
	const randomButton = document.getElementById("randomRange");
	const maxKnownHeight = Number({{.MaxKnownHeight}});
	const maxBlockMetricPoints = Number({{.MaxBlockMetricPoints}});
	const randomBlockMetricMaxRange = Number({{.RandomBlockMetricRange}});
	function parseInteger(value) {
		const text = String(value).trim();
		if (!/^-?\d+$/.test(text)) return null;
		const n = Number(text);
		return Number.isSafeInteger(n) ? n : null;
	}
	function readRange() {
		const fromHeight = parseInteger(fromInput.value);
		if (fromHeight === null) return { error: "invalid fromHeight" };
		const toHeight = parseInteger(toInput.value);
		if (toHeight === null) return { error: "invalid toHeight" };
		const interval = parseInteger(intervalInput.value);
		if (interval === null) return { error: "invalid interval" };
		if (fromHeight < 0 || toHeight <= fromHeight) {
			return { error: "invalid fromHeight or toHeight" };
		}
		const pointCount = Math.ceil((toHeight - fromHeight) / interval);
		if (pointCount > maxBlockMetricPoints) {
			return { error: "too many points: max " + maxBlockMetricPoints.toLocaleString() };
		}
		return { fromHeight, toHeight, interval };
	}
	function setStatus(message, isError) {
		status.textContent = message || "";
		status.className = isError ? "status error" : "status";
	}
	function setLoading(loading) {
		loadButton.disabled = loading;
		randomButton.disabled = loading;
		if (loading) setStatus("loading...", false);
	}
	function updateURL(range) {
		const url = new URL(window.location.href);
		url.searchParams.set("fromHeight", String(range.fromHeight));
		url.searchParams.set("toHeight", String(range.toHeight));
		url.searchParams.set("interval", String(range.interval));
		window.history.replaceState(null, "", url);
	}
	function updateTitle(range) {
		title.textContent = "Block Metrics " + range.fromHeight + " - " + range.toHeight;
	}
	async function loadMetrics() {
		const range = readRange();
		if (range.error) {
			setStatus(range.error, true);
			return;
		}
		setLoading(true);
		try {
			const url = new URL(window.location.href);
			url.pathname = url.pathname.replace(/\/block-metrics-demo$/, "/block-metrics-range");
			url.searchParams.set("fromHeight", String(range.fromHeight));
			url.searchParams.set("toHeight", String(range.toHeight));
			url.searchParams.set("interval", String(range.interval));
			const response = await fetch(url, { headers: { "Accept": "application/json" } });
			if (!response.ok) throw new Error("request failed: " + response.status);
			const body = await response.json();
			if (!body || body.code !== 0) throw new Error((body && body.msg) || "failed");
			points = Array.isArray(body.data) ? body.data : [];
			hoverIndex = -1;
			updateURL(range);
			updateTitle(range);
			draw();
			setStatus("", false);
		} catch (err) {
			setStatus(err.message || "failed", true);
		} finally {
			setLoading(false);
		}
	}
	function randomizeRange() {
		const upperBound = Math.max(0, maxKnownHeight);
		if (upperBound <= 0) {
			setStatus("no metrics data", true);
			return;
		}
		if (upperBound <= randomBlockMetricMaxRange) {
			fromInput.value = "0";
			toInput.value = String(upperBound);
			loadMetrics();
			return;
		}
		const maxSpan = Math.min(randomBlockMetricMaxRange, upperBound);
		const span = 1000 + Math.floor(Math.random() * (maxSpan - 999));
		const maxStart = upperBound - span;
		const start = Math.floor(Math.random() * (maxStart + 1));
		fromInput.value = String(start);
		toInput.value = String(start + span);
		loadMetrics();
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
	function xAt(i) {
		return pad.left + (points.length <= 1 ? 0 : innerW * i / (points.length - 1));
	}
	function yAt(v, maxY) {
		return h - pad.bottom - innerH * v / maxY;
	}
	function formatBlockTime(seconds) {
		if (!seconds) return "";
		const ms = Number(seconds) * 1000;
		const d = new Date(ms);
		const yyyy = d.getUTCFullYear();
		const mm = String(d.getUTCMonth() + 1).padStart(2, "0");
		const dd = String(d.getUTCDate()).padStart(2, "0");
		const hh = String(d.getUTCHours()).padStart(2, "0");
		const mi = String(d.getUTCMinutes()).padStart(2, "0");
		const ss = String(d.getUTCSeconds()).padStart(2, "0");
		return yyyy + "-" + mm + "-" + dd + " " + hh + ":" + mi + ":" + ss + " UTC";
	}
	function formatTimeRange(p) {
		const from = formatBlockTime(p.fromTime);
		const to = formatBlockTime(p.toTime);
		if (!from && !to) return "unknown";
		if (!to || from === to) return from;
		if (!from) return to;
		return from + " - " + to;
	}
	function wrapTooltipText(lines, maxWidth) {
		return lines.flatMap((line) => {
			const parts = String(line).split(" ");
			const wrapped = [];
			let current = "";
			for (const part of parts) {
				const next = current ? current + " " + part : part;
				if (current && ctx.measureText(next).width > maxWidth) {
					wrapped.push(current);
					current = part;
				} else {
					current = next;
				}
			}
			if (current) wrapped.push(current);
			return wrapped;
		});
	}
	function renderHover(active, maxY) {
		if (hoverIndex < 0 || hoverIndex >= points.length) return;
		const p = points[hoverIndex];
		const x = xAt(hoverIndex);
		ctx.save();
		ctx.strokeStyle = "#0f172a";
		ctx.lineWidth = 1;
		ctx.setLineDash([4, 4]);
		ctx.beginPath();
		ctx.moveTo(x, pad.top);
		ctx.lineTo(x, h - pad.bottom);
		ctx.stroke();
		ctx.setLineDash([]);

		for (const key of active) {
			const [, color] = seriesConfig[key];
			const y = yAt(valueOf(p, key), maxY);
			ctx.fillStyle = color;
			ctx.beginPath();
			ctx.arc(x, y, 3, 0, Math.PI * 2);
			ctx.fill();
		}

		ctx.font = "12px sans-serif";
		const lines = [
			"height: " + p.height.toLocaleString() + " - " + (p.toHeight - 1).toLocaleString(),
			"time: " + formatTimeRange(p),
			...active.map((key) => seriesConfig[key][0] + ": " + valueOf(p, key).toLocaleString()),
		];
		const textLines = wrapTooltipText(lines, 220);
		const lineHeight = 18;
		const boxW = Math.min(260, Math.max(...textLines.map((line) => ctx.measureText(line).width)) + 24);
		const boxH = textLines.length * lineHeight + 16;
		let boxX = x + 12;
		if (boxX + boxW > w - 8) boxX = x - boxW - 12;
		if (boxX < 8) boxX = 8;
		let boxY = pad.top + 8;
		if (boxY + boxH > h - pad.bottom - 8) boxY = h - pad.bottom - boxH - 8;
		ctx.fillStyle = "rgba(255, 255, 255, 0.96)";
		ctx.strokeStyle = "#cbd5e1";
		ctx.lineWidth = 1;
		ctx.beginPath();
		ctx.roundRect(boxX, boxY, boxW, boxH, 6);
		ctx.fill();
		ctx.stroke();
		textLines.forEach((line, i) => {
			ctx.fillStyle = i < 2 ? "#334155" : "#0f172a";
			ctx.fillText(line, boxX + 12, boxY + 20 + i * lineHeight);
		});
		ctx.restore();
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
		for (const key of active) {
			const [label, color] = seriesConfig[key];
			ctx.strokeStyle = color;
			ctx.lineWidth = key === "txCount" || key === "other" ? 2.5 : 2;
			ctx.beginPath();
			points.forEach((p, i) => {
				const x = xAt(i), y = yAt(valueOf(p, key), maxY);
				if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
			});
			ctx.stroke();
		}
		if (points.length > 0) {
			ctx.fillStyle = "#52606d";
			ctx.fillText(String(points[0].height), pad.left, h - 12);
			ctx.fillText(String(points[points.length - 1].toHeight), w - pad.right - 70, h - 12);
		}
		renderHover(active, maxY);
		renderSummary();
	}
	function updateHover(event) {
		if (points.length === 0) return;
		const rect = canvas.getBoundingClientRect();
		const scaleX = canvas.width / rect.width;
		const scaleY = canvas.height / rect.height;
		const x = (event.clientX - rect.left) * scaleX;
		const y = (event.clientY - rect.top) * scaleY;
		if (x < pad.left || x > w - pad.right || y < pad.top || y > h - pad.bottom) {
			if (hoverIndex !== -1) {
				hoverIndex = -1;
				draw();
			}
			return;
		}
		const nextIndex = points.length <= 1 ? 0 : Math.round((x - pad.left) * (points.length - 1) / innerW);
		if (nextIndex !== hoverIndex) {
			hoverIndex = Math.max(0, Math.min(points.length - 1, nextIndex));
			draw();
		}
	}
	function clearHover() {
		if (hoverIndex === -1) return;
		hoverIndex = -1;
		draw();
	}
	checks.forEach((el) => el.addEventListener("change", draw));
	canvas.addEventListener("mousemove", updateHover);
	canvas.addEventListener("mouseleave", clearHover);
	form.addEventListener("submit", (event) => {
		event.preventDefault();
		loadMetrics();
	});
	randomButton.addEventListener("click", randomizeRange);
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
