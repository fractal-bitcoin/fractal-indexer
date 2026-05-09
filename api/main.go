package api

import (
	"context"
	"fmt"
	"fractal-query/constant"
	"fractal-query/controller"
	"fractal-query/dao/clickhouse"
	"fractal-query/dao/rdb"
	_ "fractal-query/docs"
	"fractal-query/lib/midware"
	"fractal-query/logger"
	"fractal-query/model"
	"fractal-query/service"
	"fractal-query/service/brc20"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/viper"
	brc20swapIndexer "github.com/unisat-wallet/libbrc20-indexer/indexer"

	"github.com/btcsuite/btcd/chaincfg"
	cache "github.com/chenyahui/gin-cache"
	"github.com/chenyahui/gin-cache/persist"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/pprof"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/unisat-wallet/libbrc20-indexer/conf"
	"go.uber.org/zap"
)

var (
	// 0.0.0.0:8000
	listen_address         = os.Getenv("LISTEN")
	basePath               = os.Getenv("BASE_PATH")
	disableBRC20Process    = os.Getenv("DISABLE_BRC20_PROCESS")
	dumpBRC20Process       = os.Getenv("DUMP_BRC20_DATA")
	listenBeforeBrc20Ready = os.Getenv("LISTEN_BEFORE_BRC20_PROCESS")
	heightBRC20Process     = os.Getenv("BRC20_PROCESS_BEFORE_HEIGHT")
	debugBRC20             = os.Getenv("BRC20_DEBUG_LOG")
)

var (
	brc20SwapReady = false

	brc20Enable = true

	dumpBrc20 = false

	// for http server
	cacheTimeout time.Duration
	readTimeout  time.Duration
	writeTimeout time.Duration
	idleTimeout  time.Duration
)

func initConfig() {
	viper.SetConfigFile("conf/api/conf.yaml")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			panic(fmt.Errorf("Fatal error config file: %s \n", err))
		} else {
			panic(fmt.Errorf("Fatal error config file: %s \n", err))
		}
	}
	cacheTimeout = viper.GetDuration("cache_timeout")
	readTimeout = viper.GetDuration("read_timeout")
	writeTimeout = viper.GetDuration("write_timeout")
	idleTimeout = viper.GetDuration("idle_timeout")
	if cacheTimeout == 0 {
		cacheTimeout = 30 * time.Second
	}
	if readTimeout == 0 {
		readTimeout = 30 * time.Second
	}
	if writeTimeout == 0 {
		writeTimeout = 40 * time.Second
	}
	if idleTimeout == 0 {
		idleTimeout = 60 * time.Second
	}

	if os.Getenv("TESTNET") != "" {
		service.GlobalNetParams = &chaincfg.TestNet3Params
		conf.GlobalNetParams = &chaincfg.TestNet3Params
	}
	if ticks := os.Getenv("TICKS_ENABLED"); ticks != "" {
		conf.TICKS_ENABLED = ticks
	}
	if id := os.Getenv("MODULE_SWAP_SOURCE_INSCRIPTION_ID"); id != "" {
		conf.MODULE_SWAP_SOURCE_INSCRIPTION_ID = id
	}
	if id := os.Getenv("MODULE_SWAP_INSCRIPTION_ID"); id != "" {
		conf.MODULE_SWAP_INSCRIPTION_ID = id
	}
	if prune := os.Getenv("BRC20_PRUNE_MINT_HISTORY"); prune == "true" {
		conf.PruneBRC20MintHistory = true
	}
	if prune := os.Getenv("BRC20_PRUNE_HISTORY_LIST"); prune == "true" {
		conf.PruneHistoryList = true
		logger.Log.Info("BRC20_PRUNE_HISTORY_LIST enabled - history slices will not be populated")
	}
	if prune := os.Getenv("BRC20_PRUNE_VALID_DATA_MAP"); prune == "true" {
		conf.PruneValidBRC20DataMap = true
		logger.Log.Info("BRC20_PRUNE_VALID_DATA_MAP enabled - InscriptionsValidBRC20DataMap will not be populated")
	}

	if heightStr := os.Getenv("BRC20_SWAP_MANDATORY_COMMIT_BEFORE_HEIGHT"); heightStr != "" {
		if height, err := strconv.Atoi(heightStr); err == nil {
			conf.BRC20_SWAP_MANDATORY_COMMIT_BEFORE_HEIGHT = height
		}
	}

	if start := os.Getenv("BRC20_PIKA_REWRITE_HISTORY_START_INDEX"); start != "" {
		if startIndex, err := strconv.Atoi(start); err == nil {
			conf.PikaRewriteHistoryStartIndex = startIndex
		}
	}

	if heightStr := os.Getenv("BRC20_SINGLE_STEP_TRANSFER_HEIGHT"); heightStr != "" {
		if height, err := strconv.Atoi(heightStr); err == nil {
			conf.BRC20_SINGLE_STEP_TRANSFER_HEIGHT = height
		}
	}

	if heightStr := os.Getenv("BRC20_ACCEPT_VINDICATED_INSCRIPTION_HEIGHT"); heightStr != "" {
		if height, err := strconv.Atoi(heightStr); err == nil {
			conf.BRC20_ACCEPT_VINDICATED_INSCRIPTION_HEIGHT = height
		}
	}

	if heightStr := os.Getenv("BRC20_SINGLE_STEP_TRANSFER_NON_TRANSFERABLE_HEIGHT"); heightStr != "" {
		if height, err := strconv.Atoi(heightStr); err == nil {
			conf.BRC20_SINGLE_STEP_TRANSFER_NON_TRANSFERABLE_HEIGHT = height
		}
	}

	if heightStr := os.Getenv("BRC20_LOAD_AFTER_MEMPOOL_HEIGHT"); heightStr != "" {
		if height, err := strconv.Atoi(heightStr); err == nil {
			conf.BRC20_LOAD_AFTER_MEMPOOL_HEIGHT = height
		}
	}

	conf.DEBUG = debugBRC20 == "true"
}

func KeepJsonContentType() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		c.Next()
	}
}

func SetSwagTitle(title string) func(*ginSwagger.Config) {
	return func(c *ginSwagger.Config) {
		c.Title = title
	}
}

// @title Fractal Query Spec
// @version 2.0
// @description API definition for Fractal Query APIs

// @contact.name fractal-indexer
// @contact.url https://github.com/fractal-bitcoin/fractal-indexer
// @contact.email contact@unisat.io

// @license.name MIT License
// @license.url https://opensource.org/licenses/MIT

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func Run() {
	logger.Init()
	constant.InitEnv()
	initConfig()
	clickhouse.Init()
	rdb.InitClients()

	router := gin.New()
	router.Use(ginzap.Ginzap(logger.Log, time.RFC3339, true))
	router.Use(ginzap.RecoveryWithZap(logger.Log, true))
	router.Use(midware.Metrics())

	router.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithDecompressFn(gzip.DefaultDecompressHandle)))

	router.UseRawPath = true
	router.UnescapePathValues = true

	store := persist.NewMemoryStore(3 * time.Second)
	ops := cache.WithSingleFlightForgetTimeout(cacheTimeout)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
		ginSwagger.URL(basePath+"/swagger/doc.json"),
		SetSwagTitle("FractalBitcoin")))

	pprof.Register(router) // Performance.
	midware.CreateMetricsEndpoint(router)

	router.GET("/", controller.FractalBitcoin)

	mainAPI := router.Group("/")
	{
		mainAPI.GET("/blockchain/info", cache.CacheByRequestURI(store, 5*time.Second, ops), controller.GetBlockchainInfo)
		mainAPI.GET("/blocks", controller.GetBlocksByHeightRange)
		mainAPI.GET("/block/id/:blkid", controller.GetBlockById)

		// utxo
		mainAPI.GET("/utxo/:txid/:index", controller.GetUtxoByTxIdAndIdx) // fixme: need check atomicals
		mainAPI.GET("/utxo-nft-offset/:txid/:index", controller.GetUtxoNftOffsetByTxIdAndIdx)
	}

	heightAPI := router.Group("/height/:height")
	{
		// irrelevant
		heightAPI.GET("/block", controller.GetBlockByHeight)
	}

	// for search
	searchAPI := router.Group("/")
	{
		searchAPI.GET("/inscriptions/:tag", controller.GetInscriptionsByTagAndHeightRange)
		searchAPI.GET("/inscriptions-summary",
			cache.CacheByRequestURI(store, 10*time.Second, ops), controller.GetInscriptionsSummaryByHeightRange)
		searchAPI.GET("/inscription/content/:inscriptionId", controller.GetInscriptionContent)
		searchAPI.GET("/inscription/info/:inscriptionId", controller.GetInscriptionInfo)

		searchAPI.GET("/inscriptions/status", cache.CacheByRequestURI(store, 2*time.Second, ops), controller.GetInscriptionsStatus)

		searchAPI.POST("/inscriptions/info", controller.GetInscriptionInfoBatch)
		searchAPI.POST("/inscriptions/height", controller.QueryInscriptionHeightByNFTIdList)
	}

	// for brc20
	brc20API := router.Group("/")
	{
		// brc20
		brc20API.GET("/brc20/bestheight",
			cache.CacheByRequestURI(store, 2*time.Second, ops), controller.GetBRC20BestHeight)
		brc20API.GET("/brc20/statehash",
			cache.CacheByRequestURI(store, 2*time.Second, ops), controller.GetBRC20StateHash)
		brc20API.GET("/brc20/list",
			cache.CacheByRequestURI(store, 2*time.Second, ops), controller.GetBRC20List)

		brc20API.GET("/brc20/status",
			cache.CacheByRequestURI(store, 2*time.Second, ops), controller.GetBRC20Status)
		brc20API.GET("/brc20/heatmap",
			cache.CacheByRequestURI(store, 2*time.Second, ops), controller.GetBRC20HeatMap)

		brc20API.GET("/brc20/history-by-height/:height",
			cache.CacheByRequestURI(store, 2*time.Second, ops), controller.GetBRC20AllHistoryByHeight)
		brc20API.GET("/address/:address/brc20/history",
			cache.CacheByRequestURI(store, 2*time.Second, ops), controller.GetBRC20AllHistoryByAddress)
		brc20API.GET("/brc20/:ticker/history",
			cache.CacheByRequestURI(store, 2*time.Second, ops), controller.GetBRC20TickerHistory)
		brc20API.GET("/address/:address/brc20/:ticker/history",
			cache.CacheByRequestURI(store, 2*time.Second, ops), controller.GetBRC20TickerHistoryByAddress)

		brc20API.GET("/brc20/:ticker/tx/:txid/history",
			cache.CacheByRequestURI(store, 2*time.Second, ops), controller.GetBRC20TickerHistoryByTxID)

		brc20API.POST("/brc20/tickers-info",
			cache.CacheByRequestURI(store, 2*time.Second, ops), controller.GetBRC20TickersInfo)

		brc20API.GET("/brc20/:ticker/info",
			cache.CacheByRequestURI(store, 2*time.Second, ops), controller.GetBRC20TickerInfo)
		brc20API.GET("/brc20/:ticker/holders",
			cache.CacheByRequestURI(store, 2*time.Second, ops), controller.GetBRC20TickerHolders)

		brc20API.GET("/address/:address/brc20/summary",
			cache.CacheByRequestURI(store, 2*time.Second, ops), controller.GetBRC20SummaryByAddress)
		brc20API.GET("/address/:address/brc20/summary-by-height/:height",
			cache.CacheByRequestURI(store, 2*time.Second, ops), controller.GetBRC20SummaryByAddressAndHeight)

		brc20API.GET("/address/:address/brc20/:ticker/info", controller.GetBRC20TickerInfoByAddress)
		brc20API.GET("/address/:address/brc20/:ticker/transferable-inscriptions", controller.GetBRC20TickerTransferableInscriptionsByAddress)

		brc20API.GET("/address/:address/brc20/ticker4d-transferable-inscriptions", controller.GetBRC20Ticker4dTransferableInscriptionsByAddress)

		brc20API.GET("/address/:address/brc20/ticker5b-deploy-inscriptions", controller.GetBRC20Ticker5bDeployInscriptionsByAddress)

		// brc20 swap module
		brc20API.GET("/brc20-module/:module/history",
			cache.CacheByRequestURI(store, 2*time.Second, ops), controller.GetBRC20ModuleHistory)
		brc20API.GET("/brc20-module/withdraw-history",
			cache.CacheByRequestURI(store, 2*time.Second, ops), controller.GetBRC20ModuleWithdrawHistory)

		brc20API.GET("/brc20-module/withdraw-history-by-block-id/:blkid",
			cache.CacheByRequestURI(store, 2*time.Second, ops), controller.GetBRC20ModuleWithdrawHistoryByBlockId)

		brc20API.GET("/brc20-module/:module/address/:address/brc20/:ticker/info", controller.GetBRC20ModuleTickerInfoByAddress)

		brc20API.GET("/brc20-module/inscription/info/:inscriptionId", controller.GetBRC20ModuleInscriptionInfo)

		brc20API.POST("/brc20-module/verify-commit", controller.BRC20ModuleVerifySwapCommitContent)
	}

	// for event
	eventAPI := router.Group("/")
	{
		// nft events
		eventAPI.GET("/inscription/events",
			cache.CacheByRequestURI(store, 10*time.Second, ops), controller.GetInscriptionsEventsByHeightRange)

		// nft events
		eventAPI.GET("/inscription/events/:inscriptionId",
			cache.CacheByRequestURI(store, 10*time.Second, ops), controller.GetInscriptionEventsByHeightRange)

		eventAPI.GET("/inscription/brc20-events",
			cache.CacheByRequestURI(store, 10*time.Second, ops), controller.GetInscriptionsBRC20EventsByHeightRange)

		eventAPI.GET("/inscription/brc20-swap-events",
			cache.CacheByRequestURI(store, 10*time.Second, ops), controller.GetInscriptionsBRC20EventsByHeightRange)
	}

	// for admin
	adminAPI := router.Group("/admin")
	{
		adminAPI.GET("/brc20/status", controller.GetBRC20StatusAll)

		adminAPI.POST("/flag/:key/:value", SetAdminFlag)

		adminAPI.POST("/dump/brc20-data", DumpBrc20Data)
	}

	// Report index data for monitoring and data correctness verification.
	reportAPI := router.Group("/report")
	{
		reportAPI.GET("/core-data-upto/:height", controller.GetCoreDataUpToHeight)

		reportAPI.GET("/blocks-height-invalue", controller.GetLatestBlocksHeightAndInvalue)
		reportAPI.GET("/blocks-height-nftin", controller.GetLatestBlocksHeightAndNFTIn)
		reportAPI.GET("/blocks-height-invalue-range", controller.GetBlocksHeightAndInvalueRange)
		reportAPI.GET("/blocks-height-nftin-range", controller.GetBlocksHeightAndNFTInRange)
	}

	// brc20 swap
	go func() {
		midware.ServiceMetrics.Inc("brc20_swap", "task")

		endHeightConf, _ := strconv.Atoi(heightBRC20Process)
		heightSpan := conf.BRC20_MODULE_SAFE_CONFIRMATION + 20

		if disableBRC20Process == "true" {
			brc20SwapReady = true
			return
		} else if disableBRC20Process == "once" {
			midware.ServiceMetrics.Inc("brc20_swap", "task")
			brc20.ProcessUpdateLatestBRC20SwapInit(dumpBRC20Process)
			startHeight := int(model.GSwapBase.BestHeight + 1)
			brc20.ProcessUpdateLatestBRC20SwapOnce(dumpBRC20Process, startHeight, endHeightConf, endHeightConf+heightSpan)
			brc20.UpdateBRC20StatusTickerInfoCache()
			brc20.InitBRC20HeatMap()
			midware.ServiceMetrics.Dec("brc20_swap", "task")
			brc20SwapReady = true
		} else {
			latestHeight, err := service.GetBestBlockHeight()
			if err != nil {
				logger.Log.Info("get blk height failed",
					zap.Error(err),
				)
				latestHeight = 0
			}

			// init
			loading := true
			brc20.ProcessUpdateLatestBRC20SwapInit(dumpBRC20Process)
			startHeight := int(model.GSwapBase.BestHeight + 1)
			// loop process
			for {
				if !brc20Enable {
					time.Sleep(time.Second)
					continue
				}
				midware.ServiceMetrics.Inc("brc20_swap", "task")

				if endHeightConf > 0 { // for deepcopy test
					if startHeight+10000 < endHeightConf {
						latestHeight = startHeight + 10000
					} else {
						latestHeight = endHeightConf
					}
				}

				if latestHeight > startHeight+heightSpan+120 { // 1 hour
					endHeight := latestHeight - heightSpan
					logger.Log.Info("brc20 roll",
						zap.Int("start", startHeight),
						zap.Int("end", endHeight),
					)
					// update GSwapBase
					g := &brc20swapIndexer.BRC20ModuleIndexer{}
					if loading {
						g = model.GSwapBase
					} else {
						g = model.GSwapBase.DeepCopy(true)
					}
					brc20Datas := make(chan interface{}, 32)
					go func() {
						brc20.GetLatestBRC20SwapCreateIdxAndHeightRange(startHeight, endHeight, brc20Datas)
						close(brc20Datas)
					}()
					g.ProcessUpdateLatestBRC20Loop(brc20Datas, endHeight, latestHeight, &rdb.RdbBrc20StateClient)
					model.GSwapBase = g
					if loading {
						model.GSwapBase.MergeBalanceOverlay()
					}
					model.GSwapBase.MergeHistoryOverlay()
					startHeight = endHeight

				} else {
					if endHeightConf > 0 { // for dump
						brc20.ProcessUpdateLatestBRC20SwapOnce(dumpBRC20Process, startHeight, endHeightConf, endHeightConf+heightSpan)
						brc20.UpdateBRC20StatusTickerInfoCache()
						brc20.InitBRC20HeatMap()
						midware.ServiceMetrics.Dec("brc20_swap", "task")
						brc20SwapReady = true
						break
					} else {
						latestHeight = brc20.ProcessUpdateLatestBRC20Swap(startHeight, constant.MEMPOOL_HEIGHT+conf.BRC20_LOAD_AFTER_MEMPOOL_HEIGHT)
						if latestHeight > 0 {
							brc20.UpdateBRC20StatusTickerInfoCache()
							brc20.InitBRC20HeatMap()
						}
					}
				}
				midware.ServiceMetrics.Dec("brc20_swap", "task")
				if !loading {
					brc20SwapReady = true
				}
				loading = false

				// once, start by api
				if dumpBrc20 {
					model.GSwapBase.MergeBalanceOverlay()
					model.GSwapBase.MergeHistoryOverlay()
					model.GSwapBase.DumpBrc20Data()
					model.GSwapBase.SaveBestHeight("./data/dump/info", int(model.GSwapBase.BestHeight))
					dumpBrc20 = false
				}
				time.Sleep(time.Second)
			}
		}
	}()

	// 5d ticker holder
	go func() {
		midware.ServiceMetrics.Inc("brc20_5d_holder", "task")

		if disableBRC20Process == "true" {
			return
		} else {
			for {
				if !brc20SwapReady || !brc20Enable {
					time.Sleep(time.Second * 10)
					continue
				}
				midware.ServiceMetrics.Inc("brc20_5d_holder", "task") // Track the running API request.
				service.ProcessUpdateLatest5dTickerHoldersSummary()
				midware.ServiceMetrics.Dec("brc20_5d_holder", "task")

				if disableBRC20Process == "once" {
					return
				}

				if dumpBRC20Process != "" {
					return
				}
				time.Sleep(time.Second * 10)
			}
		}
	}()

	svr := &http.Server{
		Addr:         listen_address,
		Handler:      router,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}
	go func() {
		for {
			if listenBeforeBrc20Ready != "true" && !brc20SwapReady {
				time.Sleep(time.Second * 2)
				continue
			}
			break
		}

		logger.Log.Info("LISTEN:",
			zap.String("address", listen_address),
		)
		logger.Log.Info("start to listen and serve...")
		err := svr.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("ListenAndServe:",
				zap.Error(err),
			)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	timeout := time.Duration(1) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := svr.Shutdown(ctx); err != nil {
		logger.Log.Fatal("Shutdown:",
			zap.Error(err),
		)

	}
}

func SetAdminFlag(ctx *gin.Context) {
	logger.Log.Info("SetAdminFlag enter")

	key := ctx.Param("key")
	value := ctx.Param("value")

	if key == "brc20" {
		brc20Enable = (value == "true")
	}
	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
	})
}

func DumpBrc20Data(ctx *gin.Context) {
	var dumpBrc20Params struct {
		Dump string `json:"dump"`
	}
	err := ctx.ShouldBindJSON(&dumpBrc20Params)
	if err != nil {
		ctx.JSON(http.StatusOK, model.Response{
			Code: -1,
			Msg:  "params invalid",
		})
		return
	}
	dumpBrc20 = true
	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
	})
}

func byteCountBinary(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
