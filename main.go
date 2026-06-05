package main

import (
	"context"
	"flag"
	"fmt"
	queryapi "fractal-indexer/api"
	"fractal-indexer/constant"
	"fractal-indexer/lib/midware"
	"fractal-indexer/loader"
	"fractal-indexer/loader/clickhouse"
	"fractal-indexer/logger"
	"fractal-indexer/model"
	"fractal-indexer/parser"
	"fractal-indexer/rdb"
	"fractal-indexer/store"
	"fractal-indexer/task"
	"fractal-indexer/task/serial"
	"fractal-indexer/utils"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/pprof"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	ctx = context.Background()

	listen_address = os.Getenv("LISTEN")

	apiFlag = flag.Bool("api", false, "run query api server")

	reorgBlock int

	startBlockHeight uint32
	endBlockHeight   uint32
	syncLagBlocks    uint32
	batchBlkCount    uint32
	isFull           bool
	syncOnce         bool
	reorgTest        bool
)

func initIndexer() {
	var startBlockHeightVar uint
	var endBlockHeightVar uint
	var syncLagBlocksVar uint
	var batchBlkCountVar uint

	flag.BoolVar(&reorgTest, "reorg", false, "reorg 1~5 blocks random")
	flag.BoolVar(&syncOnce, "once", false, "sync 1 block then stop")
	flag.BoolVar(&isFull, "full", false, "start from genesis")
	flag.UintVar(&startBlockHeightVar, "start", 0, "start block height")
	flag.UintVar(&endBlockHeightVar, "end", 0, "end block height")
	flag.UintVar(&syncLagBlocksVar, "lag", 0, "number of latest blocks to keep unsynced")
	flag.UintVar(&syncLagBlocksVar, "span", 0, "deprecated alias for -lag")
	flag.UintVar(&batchBlkCountVar, "nblock", 256, "batch of blocks id fetch from rpc")

	flag.Parse()

	startBlockHeight = uint32(startBlockHeightVar)
	endBlockHeight = uint32(endBlockHeightVar)
	if endBlockHeight > 0 {
		syncLagBlocksVar = 0
	}
	syncLagBlocks = uint32(syncLagBlocksVar)
	batchBlkCount = uint32(batchBlkCountVar)

	if os.Getenv("ENABLE_WAL") == "true" || os.Getenv("ENABLE_WAL") == "1" {
		model.EnableWAL = true
	}

	viper.SetConfigFile("conf/chain.yaml")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			panic(fmt.Errorf("Fatal error config file: %s \n", err))
		} else {
			panic(fmt.Errorf("Fatal error config file: %s \n", err))
		}
	}

	model.SkipMissingUTXO = viper.GetBool("skip_missing_utxo")

	constant.ORDINALS_ACTIVATION_HEIGHT = viper.GetUint32("ordinals_activation_height")
	constant.JUBILEE_ACTIVATION_HEIGHT = viper.GetUint32("jubilee_activation_height")
	constant.REINSCRIPTION_ACTIVATION_HEIGHT = viper.GetUint32("reinscription_activation_height")
	constant.BRC20_SINGLE_STEP_TRANSFER_HEIGHT = viper.GetUint32("brc20_single_step_transfer_height")

	constant.CHAIN_TYPE = viper.GetString("chain_type")
	if constant.CHAIN_TYPE != constant.CHAIN_TYPE_BTC && constant.CHAIN_TYPE != constant.CHAIN_TYPE_FRACTAL {
		panic(fmt.Errorf("chain_type must be set, use <%v|%v>", constant.CHAIN_TYPE_BTC, constant.CHAIN_TYPE_FRACTAL))
	}

	rdb.RdbClient = rdb.Init("conf/kvdb.yaml") // redis balance/nftpointer/nftnumber
	clickhouse.Init()

	strReorgBlock := os.Getenv("MAX_REORG_BLOCKS")
	reorgBlock, _ = strconv.Atoi(strReorgBlock)
	if reorgBlock == 0 {
		reorgBlock = 100
	}
	loader.InitRpc()
	constant.InitNFTData()
}

func syncBlock() {
	if !isFull {
		if ok := task.CheckAndRecover(); !ok {
			triggerStop()
			return
		}
	}

	blockchain, err := parser.NewBlockchain(reorgBlock) // Initialize the blockchain state.
	if err != nil {
		logger.Log.Error("init blockchain error", zap.Error(err))
		return
	}

	if isFull {
		startBlockHeight = 0                    // Start a full rescan from genesis.
		rdb.FlushdbInRedis()                    // Clear Redis.
		if ok := store.CreateAllSyncCk(); !ok { // Initialize sync tables.
			triggerStop()
			return
		}
		store.PrepareFullSyncCk()
	} else {
		// load latest blocks from ck
		if ok := blockchain.InitLatestBlockFromDB(); !ok {
			return
		}
	}

	// Scan blocks.
	for {
		if model.NeedStop.Load() { // Stop when shutdown was requested.
			break
		}

		if _, ok := blockchain.InitLatestBlockFromRPC(batchBlkCount); !ok { // Load the latest block headers.
			break
		}

		if syncLagBlocks > 0 {
			endBlockHeight = 0
		}

		if !isFull {
			// Append to the existing sync state.
			needRemove := false
			if startBlockHeight == 0 {
				// Read synced blocks from ClickHouse and choose the next sync range.
				commonHeight, orphanCount, newBlocks, ok := blockchain.GetBlockSyncCommonBlockHeight(endBlockHeight)
				if !ok {
					logger.Log.Error("reorg more than 100 blocks, or less blocks in /node/blocks/")
					time.Sleep(time.Second * 5)
					break
				}
				if orphanCount > 0 {
					needRemove = true
				}

				if syncLagBlocks >= newBlocks {
					waitUntilNewBlocksExceedLag(commonHeight, newBlocks)
					continue
				}

				// Lagged sync keeps the latest syncLagBlocks blocks unsynced.
				if syncLagBlocks > 0 {
					startBlockHeight = commonHeight + 1 // Start from the block after COMMON_HEIGHT.
					endBlockHeight = startBlockHeight + newBlocks - syncLagBlocks
				} else {
					startBlockHeight = commonHeight + 1 // Start from the block after COMMON_HEIGHT.
				}

			} else {
				// A manually specified sync position requires reorg cleanup.
				needRemove = true
			}

			if reorgTest {
				needRemove = true
				startBlockHeight -= (startBlockHeight % 5)
			}

			if needRemove {
				logger.Log.Info("need to reorg, continue.")

				if startBlockHeight == 0 {
					logger.Log.Info("reorg from 0, quit.")
					triggerStop()
					break
				}

				if ok := task.RemoveBlocksForReorg(startBlockHeight); !ok {
					logger.Log.Info("reorg failed, quit.")
					triggerStop()
					break
				}

				commonBlock := blockchain.BlocksOfChainByHeight[startBlockHeight-1]
				if _, err := rdb.RdbClient.HSet(ctx, constant.TASK_INFO_KEYNAME,
					constant.TASK_BLOCK, commonBlock.HashHex,
				).Result(); err != nil {
					logger.Log.Error("update reorg block failed", zap.Error(err))
					triggerStop()
					break
				}
				logger.Log.Info("reorg ok", zap.String("nowBlockId", commonBlock.HashHex))
			}

			if ok := store.CreatePartSyncCk(); !ok { // Initialize partial sync tables.
				triggerStop()
				break
			}
			store.PreparePartSyncCk()
		}

		nftStartNumber := serial.GetNFTCountBeforeHeight(constant.ORDINALS_INSCRIPTION_COUNTS_BY_HEIGHT, startBlockHeight)
		nftCursedStartNumber := serial.GetNFTCountBeforeHeight(constant.ORDINALS_INSCRIPTION_CURSED_COUNTS_BY_HEIGHT, startBlockHeight)

		logger.Log.Debug("start", zap.Uint32("height", startBlockHeight))
		// Scan blocks in [startBlockHeight, endBlockHeight).
		stageBlockID, stageBlockHeight, txCount := blockchain.ParseLongestChain(startBlockHeight, endBlockHeight, nftStartNumber, nftCursedStartNumber)
		if !model.SkipMissingUTXO && model.MissingUTXO {
			break
		}
		if model.NeedStop.Load() { // Stop when shutdown or an error requested it.
			break
		}

		// Process blocks in batches.
		logger.Log.Debug("range", zap.Uint32("start", startBlockHeight), zap.Uint32("end", stageBlockHeight+1), zap.Int("ntx", txCount))
		// Exit on an invalid stage height to avoid corrupting stored block progress.
		// Block read errors can leave stageBlockHeight at 0 and otherwise write height 0 to DB.
		if stageBlockHeight < startBlockHeight {
			logger.Log.Error("stageBlockHeight less than startBlockHeight, break",
				zap.Uint32("start", startBlockHeight),
				zap.Uint32("stage", stageBlockHeight),
			)
			break
		}

		{
			if ok := task.SubmitBlocks(isFull, stageBlockHeight); !ok {
				triggerStop()
				break
			}

			if len(stageBlockID) == 32 {
				if _, err := rdb.RdbClient.HSet(ctx, constant.TASK_INFO_KEYNAME,
					constant.TASK_BLOCK, utils.HashString(stageBlockID),
				).Result(); err != nil {
					logger.Log.Error("update best block failed", zap.Error(err))
					triggerStop()
					break
				}
			}
			isFull = false // Prepare for incremental sync.
			startBlockHeight = 0
			logger.Log.Debug("all updated")
		}

		// Stop when the requested scan range has finished.
		if syncLagBlocks == 0 && endBlockHeight > 0 && stageBlockHeight == endBlockHeight-1 {
			break
		}

		if syncOnce { // Stop after one sync pass.
			logger.Log.Info("sync once")
			break
		}
	}
	logger.Log.Info("stoped")
}

func waitUntilNewBlocksExceedLag(commonHeight, newBlocks uint32) {
	logger.Log.Info("waiting new block...",
		zap.Uint32("lastHeight", commonHeight),
		zap.Uint32("nBlkNew", newBlocks),
		zap.Uint32("syncLag", syncLagBlocks),
	)

	// Wait until the actual RPC tip has more blocks after the common block
	// than the configured sync lag.
	for {
		if model.NeedStop.Load() {
			break
		}

		time.Sleep(time.Second * 5)
		latestBlockHeight := loader.GetBlockCountRPC()
		if latestBlockHeight > commonHeight && latestBlockHeight-commonHeight > syncLagBlocks {
			break
		}
	}
}

func main() {
	if isAPIArg(os.Args[1:]) {
		*apiFlag = true
		queryapi.Run()
		return
	}

	initIndexer()

	router := gin.New()
	router.Use(ginzap.Ginzap(logger.Log, time.RFC3339, true))
	router.Use(ginzap.RecoveryWithZap(logger.Log, true))
	router.Use(midware.Metrics())

	router.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithDecompressFn(gzip.DefaultDecompressHandle)))

	router.UseRawPath = true
	router.UnescapePathValues = true

	pprof.Register(router) // Profiling endpoints.
	midware.CreateMetricsEndpoint(router)

	logger.Log.Info("LISTEN:",
		zap.String("address", listen_address),
	)
	svr := &http.Server{
		Addr:    listen_address,
		Handler: router,
	}

	go func() {
		err := svr.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("ListenAndServe:",
				zap.Error(err),
			)
		}
	}()

	// Listen for shutdown signals.
	sigCtrl := make(chan os.Signal, 1)
	// Handle Ctrl+C and kill signals.
	signal.Notify(sigCtrl, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		for s := range sigCtrl {
			switch s {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				triggerStop()
			default:
				fmt.Println("other signal", s)
				logger.Log.Info("other signal", zap.String("sig", s.String()))
			}
		}
	}()

	// GC
	go func() {
		for {
			runtime.GC()
			time.Sleep(time.Second * 10)
		}
	}()

	syncBlock()
	logger.SyncLog()

	// shutdown metrics
	timeout := time.Duration(1) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := svr.Shutdown(ctx); err != nil {
		logger.Log.Fatal("Shutdown:",
			zap.Error(err),
		)
	}

	////////////////
	if model.NeedStop.Load() {
		os.Exit(1)
	}
}

func isAPIArg(args []string) bool {
	for _, arg := range args {
		if arg == "-api" || arg == "--api" {
			return true
		}
		if strings.HasPrefix(arg, "-api=") {
			v := strings.TrimPrefix(arg, "-api=")
			enabled, err := strconv.ParseBool(v)
			return err == nil && enabled
		}
		if strings.HasPrefix(arg, "--api=") {
			v := strings.TrimPrefix(arg, "--api=")
			enabled, err := strconv.ParseBool(v)
			return err == nil && enabled
		}
	}
	return false
}

func triggerStop() {
	logger.Log.Info("program exit...")
	model.NeedStop.Store(true)
}
