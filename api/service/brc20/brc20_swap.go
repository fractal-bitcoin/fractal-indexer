package brc20

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"fractal-indexer/api/dao/clickhouse"
	"fractal-indexer/api/dao/rdb"
	"fractal-indexer/api/lib/utils"
	"fractal-indexer/api/model"
	"fractal-indexer/api/service"
	indexerConstant "fractal-indexer/constant"
	mtx "fractal-indexer/lib/midware"
	"fractal-indexer/logger"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/btcsuite/btcd/txscript"
	"github.com/go-redis/redis/v8"
	"github.com/unisat-wallet/libbrc20-indexer/conf"
	"github.com/unisat-wallet/libbrc20-indexer/constant"
	brc20swapIndexer "github.com/unisat-wallet/libbrc20-indexer/indexer"
	brc20swapLoader "github.com/unisat-wallet/libbrc20-indexer/loader"
	brc20swapModel "github.com/unisat-wallet/libbrc20-indexer/model"
	swapModel "github.com/unisat-wallet/libbrc20-indexer/model"
	"go.uber.org/zap"
)

func latestBRC20SwapInscriptionNumberResultSRF(rows *sql.Rows) (interface{}, error) {
	var ret int
	err := rows.Scan(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func getLatestBRC20SwapInscriptionCount() (number int, err error) {
	psql := fmt.Sprintf("SELECT count(1) FROM blkevent_height WHERE (nfttype = 3 OR nfttype = 67) AND height = %d", constant.MEMPOOL_HEIGHT)
	blkRet, err := clickhouse.ScanOne(psql, latestBRC20SwapInscriptionNumberResultSRF)
	if err != nil {
		logger.Log.Info("query brc20 number failed", zap.Error(err))
		return 0, err
	}
	if blkRet == nil {
		return 0, errors.New("not exist")
	}
	number = blkRet.(int)
	return number, nil
}

func inscriptionBRC20SwapResultSRF(rows *sql.Rows) (interface{}, error) {
	brc20 := brc20swapIndexer.CacheContentBodyPool.Get().(*brc20swapModel.InscriptionBRC20Data)

	var ret model.NFTCreatePoint
	var contentCode uint8
	var contentBody []byte
	err := rows.Scan(&ret.Height, &brc20.TxIdx, &brc20.Height, &ret.IdxInBlock, &brc20.InscriptionNumber, // position
		&brc20.TxId, &brc20.Idx, // inscriptionId
		&brc20.Vout,
		&brc20.Offset,
		&brc20.Parent,              // parent
		&contentCode, &contentBody, // content
		&brc20.Satoshi, &brc20.PkScript, &brc20.TapScriptPk, &brc20.Fee, // owner, and value, fee
		&brc20.BlockTime, // block time
		&brc20.Sequence,
		&brc20.NFTType,
	)
	if err != nil {
		return nil, err
	}
	if decoded, ok := indexerConstant.GetNFTContentByCode(contentCode); ok {
		brc20.ContentBody = []byte(decoded)
	} else {
		brc20.ContentBody = contentBody
	}

	if len(brc20.TapScriptPk) == 35 &&
		brc20.TapScriptPk[0] == 32 &&
		brc20.TapScriptPk[33] == txscript.OP_CHECKSIGVERIFY &&
		brc20.TapScriptPk[34] >= txscript.OP_1 &&
		brc20.TapScriptPk[34] <= txscript.OP_8 {
		brc20.AddressType = brc20.TapScriptPk[34]
	}
	// sending transfer-function
	if brc20.Height != 0 {
		brc20.IsTransfer = true
		brc20.Height, ret.Height = ret.Height, brc20.Height
	} else {
		brc20.Height = ret.Height
	}
	brc20.CreateIdxKey = ret.GetCreateIdxKey()

	return brc20, nil
}

func getLatestBRC20SwapCreateIdxAndHeightRange(blkStartHeight, blkEndHeight int, results chan interface{}) (err error) {
	whereExpr := fmt.Sprintf("height >= %d AND height < %d", blkStartHeight, blkEndHeight)

	psql := fmt.Sprintf(`
	SELECT height, txidx, nftheight, nftidx, nftnumber, txid, idx, vout, offset, parent, content_code, content, satoshi, script_pk, tapscript_pk, invalue-outvalue, blocktime, sequence, nfttype FROM blkevent_height
	WHERE %s AND (nfttype = 3 OR
	             (height >= %d AND nfttype = 67) OR
             (height >= %d AND nfttype = 67 AND height < %d AND (nftflag = 0 OR bitAnd(nftflag, 0x40) = 0x40 OR substr(tapscript_pk, 35, 1) > unhex('00'))))
ORDER BY height, eventidx
`, whereExpr,
		conf.BRC20_ACCEPT_VINDICATED_INSCRIPTION_HEIGHT,
		conf.BRC20_SINGLE_STEP_TRANSFER_HEIGHT,
		conf.BRC20_ACCEPT_VINDICATED_INSCRIPTION_HEIGHT)

	logger.Log.Info("GetLatestBRC20SwapCreateIdxAndHeightRange",
		zap.String("sql", whereExpr))

	if err := clickhouse.ScanAllAsync(psql, inscriptionBRC20SwapResultSRF, results); err != nil {
		mtx.ReportError(mtx.ErrorClickhouse)
		logger.Log.Error("GetLatestBRC20SwapCreateIdxAndHeightRange failed to scan all", zap.Error(err))
		return err
	}
	return nil
}

func GetLatestBRC20SwapCreateIdxAndHeightRange(startHeight, endHeight int, results chan interface{}) (err error) {
	if endHeight >= constant.MEMPOOL_HEIGHT {
		if err := getLatestBRC20SwapCreateIdxAndHeightRange(startHeight, endHeight, results); err != nil {
			logger.Log.Info("getLatestBRC20SwapCreateIdxAndHeightRange failed",
				zap.Int("start", startHeight),
				zap.Int("end", endHeight),
				zap.Error(err),
			)
			return err
		}
		return nil
	}

	for h := startHeight; h < endHeight; h += 1000 {
		sHeight := h
		eHeight := h + 1000
		if eHeight > endHeight {
			eHeight = endHeight
		}

		if err := getLatestBRC20SwapCreateIdxAndHeightRange(sHeight, eHeight, results); err != nil {
			logger.Log.Info("getLatestBRC20SwapCreateIdxAndHeightRange failed",
				zap.Int("start", sHeight),
				zap.Int("end", eHeight),
				zap.Error(err),
			)
			return err
		}
	}
	return nil
}

var globalBRC20SwapInscriptionCount int = 0
var globalLatestBRC20SwapBlockID string

func ProcessUpdateLatestBRC20Swap(startHeight, endHeight int) (latestHeight int) {
	logger.Log.Info("start ProcessUpdateLatestBRC20Swap")
	var blocksHash []string
	var blocksTime []uint32

	{
		lastBlocksHash, _, err := service.GetLastBlocksHashAndTime()
		if err != nil || len(lastBlocksHash) == 0 {
			logger.Log.Info("GetLastBlocksHashAndTime error",
				zap.Int("blocksHash", 0))
			return
		}

		blockId := lastBlocksHash[0]

		var numberBrc20 int
		if globalLatestBRC20SwapBlockID == blockId {
			// check update
			numberBrc20, err = getLatestBRC20SwapInscriptionCount()
			if err != nil {
				return
			}
			if numberBrc20 == globalBRC20SwapInscriptionCount {
				logger.Log.Info("ProcessUpdateLatestBRC20Swap finish",
					zap.Int("latest", globalBRC20SwapInscriptionCount),
					zap.Int("current", numberBrc20))
				return
			}
		}

		blocksHash, blocksTime, err = service.GetAllBlocksHashAndTime()
		if err != nil || len(blocksHash) == 0 {
			logger.Log.Info("ProcessUpdateLatestBRC20Swap error",
				zap.Int("blocksHash", 0))
			return
		}
		if blocksHash[len(blocksHash)-1] != blockId {
			logger.Log.Info("ProcessUpdateLatestBRC20Swap latest blockid not match",
				zap.String("blockid", utils.GetReversedStringHex(blockId)))
			return
		}

		if globalLatestBRC20SwapBlockID == blockId {
			globalBRC20SwapInscriptionCount = numberBrc20
		} else {
			globalLatestBRC20SwapBlockID = blockId
			globalBRC20SwapInscriptionCount = 0
		}

		latestHeight = len(blocksHash) - 1
	}

	start := time.Now() // Start timer

	brc20Datas := make(chan interface{}, 32)
	go func() {
		GetLatestBRC20SwapCreateIdxAndHeightRange(startHeight, endHeight, brc20Datas)
		close(brc20Datas)
	}()

	latencyGetEvents := time.Now().Sub(start)

	g := &brc20swapIndexer.BRC20ModuleIndexer{}
	if model.GSwapBase != nil {
		g = model.GSwapBase.DeepCopy(false)
	}
	latencyDeepCopy := time.Now().Sub(start)

	g.ProcessUpdateLatestBRC20Loop(brc20Datas, endHeight, latestHeight, &rdb.RdbBrc20StateClient)
	latencyProcess := time.Now().Sub(start)

	model.GlobalBlocksHash = blocksHash
	model.GlobalBlocksTime = blocksTime
	model.GSwap = g

	logger.Log.Info("ProcessUpdateLatestBRC20Swap finish",
		// zap.Int("nEvents", len(brc20Datas)),
		zap.Int("latency_fetch_events", int(latencyGetEvents.Seconds())),
		zap.Int("latency_deep_copy", int(latencyDeepCopy.Seconds())),
		zap.Int("latency_process", int(latencyProcess.Seconds())),
	)

	return latestHeight
}

func ProcessUpdateLatestBRC20SwapOnce(dump string, startHeight, endHeight, latestHeight int) {
	logger.Log.Info("start ProcessUpdateLatestBRC20SwapOnce")

	start := time.Now() // Start timer

	brc20Datas := make(chan interface{}, 32)
	go func() {
		GetLatestBRC20SwapCreateIdxAndHeightRange(startHeight, endHeight, brc20Datas)
		close(brc20Datas)
	}()

	latencyGetEvents := time.Now().Sub(start)

	if strings.Contains(dump, "input") {
		brc20swapLoader.DumpBRC20InputData("./data/brc20swap.input.txt", brc20Datas, true)
		return
	}

	latencyDump := time.Now().Sub(start)

	g := &brc20swapIndexer.BRC20ModuleIndexer{}
	if model.GSwapBase != nil {
		if strings.Contains(dump, "deepcopy") {
			g = model.GSwapBase.DeepCopy(true)
		} else {
			g = model.GSwapBase
		}
		model.GSwapBase = nil
	}
	latencyDeepCopy := time.Now().Sub(start)

	var wgEventDone sync.WaitGroup

	if strings.Contains(dump, "historyevent") {
		g.DumpHistoryText = true
		g.HistoryEventToDump = make(chan *brc20swapModel.BRC20History, 32)
		wgEventDone.Add(1)
		go func() {
			defer wgEventDone.Done()
			brc20swapLoader.DumpBRC20HistoryEventData("./data/brc20.event.txt", g.HistoryEventToDump)
		}()
	}

	g.ProcessUpdateLatestBRC20Loop(brc20Datas, endHeight, latestHeight, &rdb.RdbBrc20StateClient)
	g.MergeBalanceOverlay()
	g.MergeHistoryOverlay()
	latencyProcess := time.Now().Sub(start)

	// fixme: without set blockhash/blocktime
	// model.GlobalBlocksHash
	// model.GlobalBlocksTime
	model.GSwap = g

	logger.Log.Info("ProcessUpdateLatestBRC20SwapOnce finish",
		// zap.Int("nEvents", len(brc20Datas)),
		zap.Int("latency_fetch_events", int(latencyGetEvents.Seconds())),
		zap.Int("latency_dump", int(latencyDump.Seconds())),
		zap.Int("latency_deep_copy", int(latencyDeepCopy.Seconds())),
		zap.Int("latency_process", int(latencyProcess.Seconds())),
	)

	if strings.Contains(dump, "output") {
		brc20swapLoader.DumpTickerInfoMap("./data/brc20swap.output.txt",
			rdb.RdbBrc20StateClient,
			g.BaseHistoryCount,
			g.HistoryData,
			g.InscriptionsTickerInfoMap,
			g.UserTokensBalanceData,
			g.TokenUsersBalanceData,
		)
	}
	if strings.Contains(dump, "statehash") {
		dumpStateHash(g, "./data/brc20swap.statehash.txt")
	}
	if strings.Contains(dump, "swap") {
		brc20swapLoader.DumpModuleInfoMap("./data/brc20module.output.txt",
			g.ModulesInfoMap,
		)
	}
	if strings.Contains(dump, "gob") {
		if err := os.MkdirAll("./data/dump", 0755); err != nil {
			logger.Log.Error("create dump failed", zap.Error(err))
			return
		}
		// store
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			g.Save("./data/dump/brc20.gob")
			g.SaveTickerInfo("./data/dump/brc20.ticker.gob")
			g.SaveBalance("./data/dump/brc20.balance.gob")
			g.SaveValidation("./data/dump/brc20.valid.gob")
			g.SaveValidTransfer("./data/dump/brc20.transfer.gob")
			g.SaveBestHeight("./data/dump/info", len(model.GlobalBlocksHash)-1)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			g.SaveHistory("./data/dump/brc20.history.gob")
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			if strings.Contains(dump, "pika") {
				StoreHistoryIntoPika(g, endHeight)
			}
		}()
		wg.Wait()
	}
	wgEventDone.Wait()
}

// ProcessUpdateLatestBRC20SwapInit initializes data before the previous 100 blocks.
func ProcessUpdateLatestBRC20SwapInit(dump string) {
	base := &brc20swapIndexer.BRC20ModuleIndexer{}
	base.Init()
	base.Load("./data/base/brc20.gob")
	base.LoadTickerInfo("./data/base/brc20.ticker.gob")
	base.LoadBalance("./data/base/brc20.balance.gob")
	// If MuHash state was not restored from store (old dump without MuHash data),
	// rebuild from all loaded balances. NewMuHash3072() has numerator=1 (1 byte),
	// while a real state would be ~384 bytes.
	if len(base.BalanceMuHash.NumeratorBytes()) <= 1 {
		base.RebuildMuHashFromBalances()
	}
	base.LoadValidation("./data/base/brc20.valid.gob")
	base.LoadValidTransfer("./data/base/brc20.transfer.gob")
	if dump != "" {
		base.LoadHistory("./data/base/brc20.history.gob")
	} else {
		if base.HistoryCount > 0 {
			if _, err := GetHistoryByIdx(base, base.HistoryCount-1); err != nil {
				// logger.Log.Panic("need dump history into pika...")
			}
		}
	}
	base.ResetValidTickerInfoData()
	model.GSwapBase = base
}

func StoreHistoryIntoPika(g *brc20swapIndexer.BRC20ModuleIndexer, endHeight int) {
	startIdx := conf.PikaRewriteHistoryStartIndex
	if startIdx < 0 {
		startIdx = int(g.BaseHistoryCount)
	}
	logger.Log.Info("saving brc20 history...",
		zap.Int("start", startIdx))

	sliceLen := 10000
	for idx := startIdx / sliceLen; idx < (len(g.HistoryData)-1)/sliceLen+1; idx++ {
		ctx := context.Background()
		pikaPipe := rdb.RdbBrc20StateClient.Pipeline()
		n := 0
		for _, historyBuf := range g.HistoryData[idx*sliceLen:] {
			if n == sliceLen {
				break
			}
			pikaPipe.Set(ctx, fmt.Sprintf("h%d", idx*sliceLen+n), historyBuf, 0)
			n++
		}
		if _, err := pikaPipe.Exec(ctx); err != nil && err != redis.Nil {
			logger.Log.Error("pika store brc20 histoyr exec failed", zap.Error(err))
			return
		}
	}
	ctx := context.Background()
	rdb.RdbBrc20StateClient.HMSet(ctx, "info", "height", endHeight, "total", g.HistoryCount)
	logger.Log.Info("save brc20 history ok")
}

func GetHistoryByIdx(g *brc20swapIndexer.BRC20ModuleIndexer, historyIdx uint32) (history *swapModel.BRC20History, err error) {
	if historyIdx >= g.BaseHistoryCount && historyIdx-g.BaseHistoryCount < uint32(len(g.HistoryData)) {
		buf := g.HistoryData[historyIdx-g.BaseHistoryCount]
		history = &swapModel.BRC20History{}
		history.Unmarshal(buf)
		return history, nil
	}

	ctx := context.Background()
	res, err := rdb.RdbBrc20StateClient.Get(ctx, fmt.Sprintf("h%d", historyIdx)).Result()
	if err == redis.Nil {
		logger.Log.Info("query brc20 history by api, but not found(can fail)",
			zap.Uint32("idx", historyIdx),
		)
		return nil, errors.New("history not exist")
	} else if err != nil {
		return nil, errors.New("db failed")
	}

	history = &swapModel.BRC20History{}
	history.Unmarshal([]byte(res))
	return history, nil
}

func dumpStateHash(g *brc20swapIndexer.BRC20ModuleIndexer, path string) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		logger.Log.Error("dumpStateHash open failed", zap.Error(err))
		return
	}
	defer f.Close()

	heights := make([]uint32, 0, len(g.StateHashByHeight))
	for h := range g.StateHashByHeight {
		heights = append(heights, h)
	}
	sort.Slice(heights, func(i, j int) bool { return heights[i] < heights[j] })

	blocksHash := model.GlobalBlocksHash
	for _, height := range heights {
		stateHash := g.StateHashByHeight[height]
		blockHash := ""
		if int(height) < len(blocksHash) {
			blockHash = utils.GetReversedStringHex(blocksHash[height])
		}
		fmt.Fprintf(f, "%d %s %x\n", height, blockHash, stateHash)
	}
	logger.Log.Info("dumpStateHash done", zap.Int("entries", len(heights)), zap.String("path", path))
}
