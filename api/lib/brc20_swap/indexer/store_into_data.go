package indexer

import (
	"context"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"fractal-indexer/api/lib/brc20_swap/conf"
	"fractal-indexer/api/lib/brc20_swap/model"
	"fractal-indexer/api/lib/brc20_swap/muhash"
	"github.com/go-redis/redis/v8"
)

type BRC20ModuleIndexerStore struct {
	BestHeight    uint32
	EnableHistory bool

	HistoryCount uint32

	// history height
	FirstHistoryByHeight map[uint32]uint32
	LastHistoryHeight    uint32
	FirstMempoolHistory  uint32

	TickerMintCountByHeight map[uint32]map[string]uint64

	// brc20 base
	UserAllHistory map[string]*model.BRC20UserHistory

	// balance muhash state
	StateHashByHeight map[uint32][32]byte
	MuHashNumerator   []byte
	MuHashDenominator []byte

	// InscriptionsTickerInfoMap map[string]*model.BRC20TokenInfo
	// UserTokensBalanceData     map[string]map[string]*model.BRC20TokenBalance

	// valid data, store in another file
	// [ticker][]balance cache, no need store

	// inner valid transfer
	// InscriptionsValidTransferMap map[uint64]*model.InscriptionBRC20TickInfo
	// inner invalid transfer
	InscriptionsInvalidTransferMap map[uint64]*model.InscriptionBRC20TickInfo

	// module
	AllModulesWithdrawHistory []*model.BRC20ModuleHistory
	// all modules info
	ModulesInfoMap map[string]*model.BRC20ModuleSwapInfoStore

	// module of users [address]moduleid
	UsersModuleWithTokenMap map[string]string

	// module lp of users [address]moduleid
	UsersModuleWithLpTokenMap map[string]string

	// runtime for commit
	InscriptionsValidCommitMap   map[uint64]*model.InscriptionBRC20Data // inner valid commit by key
	InscriptionsInvalidCommitMap map[uint64]*model.InscriptionBRC20Data

	// inner valid commit by id, no need

	// runtime fro withdraw
	InscriptionsWithdrawMap map[uint64]*model.InscriptionBRC20SwapInfo
}

func GetHistoryDataRdbKey(idx int) string {
	return fmt.Sprintf("h%v", idx)
}

// base
func (g *BRC20ModuleIndexer) Load(fname string) {
	log.Printf("loading brc20 ...")
	gobFile, err := os.Open(fname)
	if err != nil {
		log.Printf("open brc20 file failed: %s", err)
		return
	}

	gob.Register(model.BRC20SwapHistoryWithdrawData{})
	gob.Register(model.BRC20SwapHistoryCommitData{})
	gobDec := gob.NewDecoder(gobFile)

	store := &BRC20ModuleIndexerStore{}
	if err := gobDec.Decode(&store); err != nil {
		log.Printf("load store failed: %s", err)
		return
	}

	g.LoadStore(store)

	log.Printf("load brc20 ok")
}

func (g *BRC20ModuleIndexer) Save(fname string) {
	log.Printf("saving brc20 ...")

	gobFile, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		log.Printf("open brc20 file failed: %s", err)
		return
	}
	defer gobFile.Close()

	gob.Register(model.BRC20SwapHistoryWithdrawData{})
	gob.Register(model.BRC20SwapHistoryCommitData{})
	enc := gob.NewEncoder(gobFile)
	if err := enc.Encode(g.GetStore()); err != nil {
		log.Printf("save store failed: %s", err)
		return
	}

	log.Printf("save brc20 ok")
}

// history
func (g *BRC20ModuleIndexer) LoadHistory(fname string) {
	log.Printf("loading brc20 history...")
	gobFile, err := os.Open(fname)
	if err != nil {
		log.Printf("open brc20 history file failed: %s", err)
		return
	}

	gobDec := gob.NewDecoder(gobFile)

	for idx := 0; ; idx++ {
		var h []byte
		if err := gobDec.Decode(&h); err != nil {
			log.Printf("load history data end: %s", err)
			break
		}
		g.HistoryData[idx] = h
	}
	log.Printf("load brc20 history ok: %d", len(g.HistoryData))
}

func (g *BRC20ModuleIndexer) SaveHistory(fname string) {
	log.Printf("saving brc20 history...")

	gobFile, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		log.Printf("open brc20 history file failed: %s", err)
		return
	}
	defer gobFile.Close()

	enc := gob.NewEncoder(gobFile)
	for _, h := range g.HistoryData {
		if err := enc.Encode(h); err != nil {
			log.Printf("save history data failed: %s", err)
			return
		}
	}
	log.Printf("save brc20 history ok")
}

// validation
type InscriptionsValidBRC20DataStore struct {
	K uint64
	V *model.InscriptionBRC20InfoResp
}

func (g *BRC20ModuleIndexer) LoadValidation(fname string) {
	log.Printf("loading brc20 validation...")
	gobFile, err := os.Open(fname)
	if err != nil {
		log.Printf("open brc20 validation file failed: %s", err)
		return
	}

	gobDec := gob.NewDecoder(gobFile)

	if conf.PruneValidBRC20DataMap {
		log.Printf("load brc20 validation skipped (PruneValidBRC20DataMap=true)")
		gobFile.Close()
		return
	}

	for {
		var d InscriptionsValidBRC20DataStore
		if err := gobDec.Decode(&d); err != nil {
			log.Printf("load validation data end: %s", err)
			break
		}
		g.InscriptionsValidBRC20DataMap[d.K] = d.V
	}
	log.Printf("load brc20 validation ok: %d", len(g.InscriptionsValidBRC20DataMap))
}

func (g *BRC20ModuleIndexer) SaveValidation(fname string) {
	log.Printf("saving brc20 validation...")

	gobFile, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		log.Printf("open brc20 validation file failed: %s", err)
		return
	}
	defer gobFile.Close()

	enc := gob.NewEncoder(gobFile)
	for k, v := range g.InscriptionsValidBRC20DataMap {
		if err := enc.Encode(InscriptionsValidBRC20DataStore{K: k, V: v}); err != nil {
			log.Printf("save validation data failed: %s", err)
			return
		}
	}
	log.Printf("save brc20 validation ok")
}

// validation transfer
type InscriptionsValidTransferStore struct {
	K uint64
	V *model.InscriptionBRC20TickInfo
}

func (g *BRC20ModuleIndexer) LoadValidTransfer(fname string) {
	log.Printf("loading brc20 valid transfer...")
	gobFile, err := os.Open(fname)
	if err != nil {
		log.Printf("open brc20 valid transfer file failed: %s", err)
		return
	}

	gobDec := gob.NewDecoder(gobFile)

	for {
		var d InscriptionsValidTransferStore
		if err := gobDec.Decode(&d); err != nil {
			log.Printf("load valid transfer data end: %s", err)
			break
		}
		g.InscriptionsValidTransferMap[d.K] = d.V
	}
	log.Printf("load brc20 valid transfer ok: %d", len(g.InscriptionsValidTransferMap))
}

func (g *BRC20ModuleIndexer) SaveValidTransfer(fname string) {
	log.Printf("saving brc20 valid transfer...")

	gobFile, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		log.Printf("open brc20 valid transfer file failed: %s", err)
		return
	}
	defer gobFile.Close()

	enc := gob.NewEncoder(gobFile)
	for k, v := range g.InscriptionsValidTransferMap {
		if err := enc.Encode(InscriptionsValidTransferStore{K: k, V: v}); err != nil {
			log.Printf("save valid transfer data failed: %s", err)
			return
		}
	}
	log.Printf("save brc20 valid transfer ok")
}

// balance
func (g *BRC20ModuleIndexer) LoadBalance(fname string) {
	log.Printf("loading brc20 balance...")
	gobFile, err := os.Open(fname)
	if err != nil {
		log.Printf("open brc20 balance file failed: %s", err)
		return
	}

	gobDec := gob.NewDecoder(gobFile)

	count := 0
	for {
		var b model.BRC20TokenBalance
		if err := gobDec.Decode(&b); err != nil {
			log.Printf("load balance data end: %s", err)
			break
		}

		count++
		// user token balance
		userTokens, ok := g.UserTokensBalanceData[b.PkScript]
		if !ok {
			userTokens = make(map[string]*model.BRC20TokenBalance, 0)
			g.UserTokensBalanceData[b.PkScript] = userTokens
		}
		uniqueLowerTicker := strings.ToLower(b.Ticker)
		userTokens[uniqueLowerTicker] = &b

		// token balance
		tokenUsers, ok := g.TokenUsersBalanceData[uniqueLowerTicker]
		if !ok {
			tokenUsers = make(map[string]*model.BRC20TokenBalance, 0)
			g.TokenUsersBalanceData[uniqueLowerTicker] = tokenUsers
		}

		overallBalance := b.OverallBalance()
		b.Balance = overallBalance.Float64() // set float balance
		if overallBalance.Sign() > 0 {
			tokenUsers[b.PkScript] = &b
		}
	}
	log.Printf("load brc20 balance ok: %d, holders: %d", count, len(g.UserTokensBalanceData))
}

func (g *BRC20ModuleIndexer) SaveBalance(fname string) {
	log.Printf("saving brc20 balance...")

	gobFile, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		log.Printf("open brc20 balance file failed: %s", err)
		return
	}
	defer gobFile.Close()

	enc := gob.NewEncoder(gobFile)

	// balance
	count := 0
	for _, userTokens := range g.UserTokensBalanceData {
		for _, balance := range userTokens {
			if err := enc.Encode(balance); err != nil {
				log.Printf("save balance data failed: %s", err)
				return
			}
			count++
		}
	}
	log.Printf("save brc20 balance ok: %d", count)
}

// ResetValidTickerInfoData need call after LoadTickerInfo + LoadValidation
func (g *BRC20ModuleIndexer) ResetValidTickerInfoData() {
	if conf.PruneValidBRC20DataMap {
		log.Printf("reset brc20 ticker valid data skipped (PruneValidBRC20DataMap=true)")
		return
	}
	log.Printf("reset brc20 ticker valid data...")
	for _, tinfo := range g.InscriptionsTickerInfoMap {
		g.InscriptionsValidBRC20DataMap[tinfo.Deploy.CreateIdxKey] = tinfo.Deploy.Data
	}
	log.Printf("reset brc20 ticker valid data ok: %d", len(g.InscriptionsTickerInfoMap))
}

// tickers-info
// validation
func (g *BRC20ModuleIndexer) LoadTickerInfo(fname string) {
	log.Printf("loading brc20 ticker...")
	gobFile, err := os.Open(fname)
	if err != nil {
		log.Printf("open brc20 ticker file failed: %s", err)
		return
	}

	gobDec := gob.NewDecoder(gobFile)

	for {
		var tinfo model.BRC20TokenInfo
		if err := gobDec.Decode(&tinfo); err != nil {
			log.Printf("load ticker data end: %s", err)
			break
		}
		uniqueLowerTicker := strings.ToLower(tinfo.Ticker)
		g.InscriptionsTickerInfoMap[uniqueLowerTicker] = &tinfo
	}
	log.Printf("load brc20 ticker ok: %d", len(g.InscriptionsTickerInfoMap))
}

func (g *BRC20ModuleIndexer) SaveTickerInfo(fname string) {
	log.Printf("saving brc20 ticker...")

	gobFile, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		log.Printf("open brc20 ticker file failed: %s", err)
		return
	}
	defer gobFile.Close()

	enc := gob.NewEncoder(gobFile)
	for _, tinfo := range g.InscriptionsTickerInfoMap {
		if err := enc.Encode(tinfo); err != nil {
			log.Printf("save ticker data failed: %s", err)
			return
		}
	}
	log.Printf("save brc20 ticker ok")
}

func (g *BRC20ModuleIndexer) GetStore() (store *BRC20ModuleIndexerStore) {
	store = &BRC20ModuleIndexerStore{
		BestHeight:    g.BestHeight,
		EnableHistory: g.EnableHistory,

		HistoryCount: g.HistoryCount,

		FirstHistoryByHeight: g.FirstHistoryByHeight,
		LastHistoryHeight:    g.LastHistoryHeight,
		FirstMempoolHistory:  g.FirstMempoolHistory,

		TickerMintCountByHeight: g.TickerMintCountByHeight,

		// brc20 base
		UserAllHistory: g.UserAllHistory,

		// balance muhash state
		StateHashByHeight: g.StateHashByHeight,
		MuHashNumerator:   g.BalanceMuHash.NumeratorBytes(),
		MuHashDenominator: g.BalanceMuHash.DenominatorBytes(),

		// InscriptionsTickerInfoMap: g.InscriptionsTickerInfoMap,
		// UserTokensBalanceData:     g.UserTokensBalanceData,

		// inner valid transfer
		// InscriptionsValidTransferMap: g.InscriptionsValidTransferMap,
		// inner invalid transfer
		InscriptionsInvalidTransferMap: g.InscriptionsInvalidTransferMap,

		// module
		AllModulesWithdrawHistory: g.AllModulesWithdrawHistory,
		// all modules info
		// module of users [address]moduleid
		UsersModuleWithTokenMap: g.UsersModuleWithTokenMap,

		// module lp of users [address]moduleid
		UsersModuleWithLpTokenMap: g.UsersModuleWithLpTokenMap,

		// runtime for commit
		InscriptionsValidCommitMap:   g.InscriptionsValidCommitMap,
		InscriptionsInvalidCommitMap: g.InscriptionsInvalidCommitMap,

		// runtime for withdraw
		InscriptionsWithdrawMap: g.InscriptionsWithdrawMap,
	}

	store.ModulesInfoMap = make(map[string]*model.BRC20ModuleSwapInfoStore)
	for module, info := range g.ModulesInfoMap {
		infoStore := &model.BRC20ModuleSwapInfoStore{
			ID:                info.ID,
			Name:              info.Name,
			DeployerPkScript:  info.DeployerPkScript,
			SequencerPkScript: info.SequencerPkScript,
			GasToPkScript:     info.GasToPkScript,
			LpFeePkScript:     info.LpFeePkScript,

			FeeRateSwap: info.FeeRateSwap,
			GasTick:     info.GasTick,

			History: info.History, // fixme

			// runtime for commit
			CommitInvalidMap: info.CommitInvalidMap,
			CommitIdMap:      info.CommitIdMap,
			CommitIdChainMap: info.CommitIdChainMap,

			// token holders in module
			// ticker of users in module [address][tick]balanceData
			UsersTokenBalanceDataMap: info.UsersTokenBalanceDataMap,

			// swap
			// lp token balance of address in module [pool][address]balance
			LPTokenUsersBalanceMap: info.LPTokenUsersBalanceMap,

			// swap total balance
			// total balance of pool in module [pool]balanceData
			SwapPoolTotalBalanceDataMap: info.SwapPoolTotalBalanceDataMap,
		}

		store.ModulesInfoMap[module] = infoStore
	}

	return store

}

func (g *BRC20ModuleIndexer) LoadStore(store *BRC20ModuleIndexerStore) {
	g.BestHeight = store.BestHeight
	g.EnableHistory = store.EnableHistory

	g.HistoryCount = store.HistoryCount
	g.BaseHistoryCount = store.HistoryCount
	// g.HistoryData = make([][]byte, store.HistoryCount)

	g.FirstHistoryByHeight = store.FirstHistoryByHeight
	g.LastHistoryHeight = store.LastHistoryHeight
	g.FirstMempoolHistory = store.FirstMempoolHistory

	g.TickerMintCountByHeight = store.TickerMintCountByHeight

	// brc20 base
	g.UserAllHistory = store.UserAllHistory

	// balance muhash state
	if len(store.MuHashNumerator) > 0 && len(store.MuHashDenominator) > 0 {
		g.BalanceMuHash = muhash.NewMuHash3072FromBytes(store.MuHashNumerator, store.MuHashDenominator)
	}
	if store.StateHashByHeight != nil {
		g.StateHashByHeight = store.StateHashByHeight
	}

	// g.InscriptionsTickerInfoMap = store.InscriptionsTickerInfoMap

	// inner valid transfer
	// g.InscriptionsValidTransferMap = store.InscriptionsValidTransferMap
	// inner invalid transfer
	g.InscriptionsInvalidTransferMap = store.InscriptionsInvalidTransferMap

	// module
	g.AllModulesWithdrawHistory = store.AllModulesWithdrawHistory
	// all modules info
	// module of users [address]moduleid
	g.UsersModuleWithTokenMap = store.UsersModuleWithTokenMap

	// module lp of users [address]moduleid
	g.UsersModuleWithLpTokenMap = store.UsersModuleWithLpTokenMap

	// runtime for commit
	g.InscriptionsValidCommitMap = store.InscriptionsValidCommitMap
	g.InscriptionsInvalidCommitMap = store.InscriptionsInvalidCommitMap

	// InscriptionsValidCommitMapById
	for _, v := range g.InscriptionsValidCommitMap {
		g.InscriptionsValidCommitMapById[v.GetInscriptionId()] = v
	}

	// runtime for withdraw
	g.InscriptionsWithdrawMap = store.InscriptionsWithdrawMap

	for module, infoStore := range store.ModulesInfoMap {
		info := &model.BRC20ModuleSwapInfo{
			ID:                infoStore.ID,
			Name:              infoStore.Name,
			DeployerPkScript:  infoStore.DeployerPkScript,
			SequencerPkScript: infoStore.SequencerPkScript,
			GasToPkScript:     infoStore.GasToPkScript,
			LpFeePkScript:     infoStore.LpFeePkScript,

			FeeRateSwap: infoStore.FeeRateSwap,
			GasTick:     infoStore.GasTick,

			History: infoStore.History,

			// runtime for commit
			CommitInvalidMap: infoStore.CommitInvalidMap,
			CommitIdMap:      infoStore.CommitIdMap,
			CommitIdChainMap: infoStore.CommitIdChainMap,

			// token holders in module
			// ticker of users in module [address][tick]balanceData
			UsersTokenBalanceDataMap: infoStore.UsersTokenBalanceDataMap,
			TokenUsersBalanceDataMap: make(map[string]map[string]*model.BRC20ModuleTokenBalance, 0),

			// swap
			// lp token balance of address in module [pool][address]balance
			LPTokenUsersBalanceMap: infoStore.LPTokenUsersBalanceMap,
			UsersLPTokenBalanceMap: make(map[string]map[string]*model.BRC20ModuleLPTokenBalance, 0),

			// swap total balance
			// total balance of pool in module [pool]balanceData
			SwapPoolTotalBalanceDataMap: infoStore.SwapPoolTotalBalanceDataMap,
		}

		// tick/user: balance
		for address, dataMap := range info.UsersTokenBalanceDataMap {
			for uniqueLowerTicker, tokenBalance := range dataMap {
				tokenUsers, ok := info.TokenUsersBalanceDataMap[uniqueLowerTicker]
				if !ok {
					tokenUsers = make(map[string]*model.BRC20ModuleTokenBalance, 0)
					info.TokenUsersBalanceDataMap[uniqueLowerTicker] = tokenUsers
				}
				tokenUsers[address] = tokenBalance
			}
		}

		// pair/user: lpbalance
		for pair, dataMap := range info.LPTokenUsersBalanceMap {
			for address, lpBalance := range dataMap {
				userTokens, ok := info.UsersLPTokenBalanceMap[address]
				if !ok {
					userTokens = make(map[string]*model.BRC20ModuleLPTokenBalance, 0)
					info.UsersLPTokenBalanceMap[address] = userTokens
				}
				userTokens[pair] = lpBalance
			}
		}

		g.ModulesInfoMap[module] = info
	}
}

func (g *BRC20ModuleIndexer) StoreHistoryIntoPika(client redis.UniversalClient) {
	log.Printf("saving brc20 history...")
	log.Printf("StoreHistoryIntoPika: BaseHistoryCount:%v, len(g.HistoryData):%v, HistoryCount:%v",
		g.BaseHistoryCount, len(g.HistoryData), g.HistoryCount)
	startIdx := int(g.BaseHistoryCount)
	if g.BaseHistoryCount+uint32(len(g.HistoryData)) != g.HistoryCount {
		panic("StoreHistoryIntoPika error")
	}
	sliceLen := 100000
	for idx := 0; idx < (len(g.HistoryData)-1)/sliceLen+1; idx++ {
		ctx := context.Background()
		pikaPipe := client.Pipeline()
		n := 0
		for _, historyBuf := range g.HistoryData[idx*sliceLen:] {
			if n == sliceLen {
				break
			}
			pikaPipe.Set(ctx, GetHistoryDataRdbKey(startIdx+idx*sliceLen+n), historyBuf, 0)
			n++
		}
		if _, err := pikaPipe.Exec(ctx); err != nil && err != redis.Nil {
			log.Printf("save brc20 history failed: %s", err)
			return
		}
	}

	g.HistoryData = [][]byte{}
	g.BaseHistoryCount = g.HistoryCount
	log.Printf("StoreHistoryIntoPika: HistoryData done")
}

func (g *BRC20ModuleIndexer) SaveBestHeight(fname string, height int) {
	// Open the file, creating it if it does not exist and truncating it if it does.
	file, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		log.Printf("failed to create file: %v", err)
		return
	}
	defer file.Close()

	_, err = fmt.Fprintf(file, "height:%d", height)
	if err != nil {
		log.Printf("failed to write height to file: %v", err)
		return
	}
	log.Printf("SaveBestHeight done, height:%v", g.BestHeight)
}

func (g *BRC20ModuleIndexer) DumpBrc20Data() {
	log.Printf("DumpBrc20Data start...")
	if err := os.MkdirAll("./data/dump", 0755); err != nil {
		log.Printf("create dump failed, err:%v", err)
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
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		g.SaveHistory("./data/dump/brc20.history.gob")
	}()

	wg.Wait()
	log.Printf("DumpBrc20Data done, height:%v", g.BestHeight)
}
