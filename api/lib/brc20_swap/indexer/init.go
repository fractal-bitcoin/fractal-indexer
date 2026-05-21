package indexer

import (
	"log"
	"sort"
	"strings"
	"sync"

	"fractal-indexer/api/lib/brc20_swap/constant"
	"fractal-indexer/api/lib/brc20_swap/model"
	"fractal-indexer/api/lib/brc20_swap/muhash"

	"github.com/wangjohn/quickselect"
)

var (
	mutex                = sync.Mutex{}
	CacheContentBodyPool sync.Pool
)

func init() {
	CacheContentBodyPool = sync.Pool{
		New: func() interface{} {
			var brc20 model.InscriptionBRC20Data
			return &brc20
		},
	}
}

type BRC20ModuleIndexer struct {
	BestHeight    uint32
	EnableHistory bool

	BaseHistoryCount uint32
	HistoryCount     uint32
	HistoryData      [][]byte

	DumpHistoryText    bool
	HistoryEventToDump chan *model.BRC20History

	// history height
	FirstHistoryByHeight map[uint32]uint32
	LastHistoryHeight    uint32
	FirstMempoolHistory  uint32

	TickerMintCountByHeight map[uint32]map[string]uint64

	// brc20 base
	UserAllHistory        map[string]*model.BRC20UserHistory
	OverlayUserAllHistory map[string]*model.BRC20UserHistory

	InscriptionsTickerInfoMap map[string]*model.BRC20TokenInfo

	// Base balance (shared via shallow copy in DeepCopy, never modified at runtime)
	UserTokensBalanceData map[string]map[string]*model.BRC20TokenBalance // [address][ticker]balance
	TokenUsersBalanceData map[string]map[string]*model.BRC20TokenBalance // [ticker][address]balance
	// L1 overlay: working layer for current block processing
	OverlayUserTokensBalanceData map[string]map[string]*model.BRC20TokenBalance // [address][ticker]balance
	OverlayTokenUsersBalanceData map[string]map[string]*model.BRC20TokenBalance // [ticker][address]balance
	// L2 overlay: accumulated from L1 merges, merges to base only at dump time
	PendingUserTokensBalanceData map[string]map[string]*model.BRC20TokenBalance // [address][ticker]balance
	PendingTokenUsersBalanceData map[string]map[string]*model.BRC20TokenBalance // [ticker][address]balance
	// balance MuHash3072
	BalanceMuHash *muhash.MuHash3072
	// [height] → MuHash finalize result at that height
	StateHashByHeight map[uint32][32]byte

	InscriptionsValidBRC20DataMap map[uint64]*model.InscriptionBRC20InfoResp

	TokenUsersBalanceDataSortedCache map[string][]*model.BRC20TokenBalance // [ticker][]balance cache

	// inner valid transfer
	InscriptionsValidTransferMap map[uint64]*model.InscriptionBRC20TickInfo
	// inner invalid transfer
	InscriptionsInvalidTransferMap map[uint64]*model.InscriptionBRC20TickInfo

	// module
	AllModulesWithdrawHistory []*model.BRC20ModuleHistory
	// all modules info
	ModulesInfoMap map[string]*model.BRC20ModuleSwapInfo

	// module of users [address]moduleid
	UsersModuleWithTokenMap map[string]string

	// module lp of users [address]moduleid
	UsersModuleWithLpTokenMap map[string]string

	// runtime for commit
	InscriptionsValidCommitMap   map[uint64]*model.InscriptionBRC20Data // inner valid commit by key
	InscriptionsInvalidCommitMap map[uint64]*model.InscriptionBRC20Data

	InscriptionsValidCommitMapById map[string]*model.InscriptionBRC20Data // inner valid commit by id

	// runtime for withdraw
	InscriptionsWithdrawMap map[uint64]*model.InscriptionBRC20SwapInfo // inner all ready to withdraw by key
}

func (g *BRC20ModuleIndexer) GetBRC20HistoryByUser(pkScript string) (userHistory *model.BRC20UserHistory) {
	if history, ok := g.OverlayUserAllHistory[pkScript]; ok {
		userHistory = history
		return userHistory
	}

	if history, ok := g.UserAllHistory[pkScript]; !ok {
		userHistory = &model.BRC20UserHistory{}
		g.OverlayUserAllHistory[pkScript] = userHistory
	} else {
		userHistory = &model.BRC20UserHistory{
			History:                 make([]uint32, len(history.History)),
			HistoryDeploy:           make([]uint32, len(history.HistoryDeploy)),
			HistoryMint:             make([]uint32, len(history.HistoryMint)),
			HistoryInscribeTransfer: make([]uint32, len(history.HistoryInscribeTransfer)),
			HistorySend:             make([]uint32, len(history.HistorySend)),
			HistoryReceive:          make([]uint32, len(history.HistoryReceive)),
			HistoryWithdraw:         make([]uint32, len(history.HistoryWithdraw)),
		}
		copy(userHistory.History, history.History)
		copy(userHistory.HistoryDeploy, history.HistoryDeploy)
		copy(userHistory.HistoryMint, history.HistoryMint)
		copy(userHistory.HistoryInscribeTransfer, history.HistoryInscribeTransfer)
		copy(userHistory.HistorySend, history.HistorySend)
		copy(userHistory.HistoryReceive, history.HistoryReceive)
		copy(userHistory.HistoryWithdraw, history.HistoryWithdraw)
		g.OverlayUserAllHistory[pkScript] = userHistory
	}
	return userHistory
}

func (g *BRC20ModuleIndexer) GetBRC20HistoryByUserForAPI(pkScript string) (userHistory *model.BRC20UserHistory) {
	// check overlay first
	if history, ok := g.OverlayUserAllHistory[pkScript]; ok {
		userHistory = history
		return userHistory
	}

	// check base
	if history, ok := g.UserAllHistory[pkScript]; !ok {
		userHistory = &model.BRC20UserHistory{}
	} else {
		userHistory = history
	}
	return userHistory
}

func (g *BRC20ModuleIndexer) GetBRC20TokenUsersBalanceDataSortedCacheForAPI(ticker string) (holdersBalance []*model.BRC20TokenBalance) {
	log.Println("GetBRC20TokenUsersBalanceDataSortedCacheForAPI", "ticker", ticker)

	mutex.Lock()
	defer mutex.Unlock()

	holdersBalance, ok := g.TokenUsersBalanceDataSortedCache[ticker]
	if ok {
		return holdersBalance
	}

	holdersBalance = g.GetTokenHoldersBalanceMapForAPI(ticker)
	if len(holdersBalance) == 0 {
		return make([]*model.BRC20TokenBalance, 0)
	}

	log.Printf("GetBRC20TokenUsersBalanceDataSortedCacheForAPI sort, ticker:%s", ticker)
	if len(holdersBalance) >= 500000 {
		quickselect.QuickSelect(model.BRC20TokenBalanceSlice(holdersBalance), 1000)
		sort.Slice(holdersBalance[:1000], func(i, j int) bool {
			result := holdersBalance[i].Balance - holdersBalance[j].Balance
			if result == 0 {
				return strings.Compare(holdersBalance[i].PkScript, holdersBalance[j].PkScript) > 0
			} else {
				return result > 0
			}
		})
	} else {
		sort.Slice(holdersBalance, func(i, j int) bool {
			result := holdersBalance[i].Balance - holdersBalance[j].Balance
			if result == 0 {
				return strings.Compare(holdersBalance[i].PkScript, holdersBalance[j].PkScript) > 0
			} else {
				return result > 0
			}
		})
	}
	log.Printf("GetBRC20TokenUsersBalanceDataSortedCacheForAPI sort finish, ticker:%s", ticker)

	g.TokenUsersBalanceDataSortedCache[ticker] = holdersBalance

	return holdersBalance
}

func (g *BRC20ModuleIndexer) UpdateHistoryHeightAndGetHistoryIndex(historyObj *model.BRC20History) uint32 {
	height := historyObj.Height
	history := g.HistoryCount
	// g.HistoryData = append(g.HistoryData, historyObj.Marshal())

	if g.DumpHistoryText {
		g.HistoryEventToDump <- historyObj
	}
	g.HistoryCount += 1

	if height == g.LastHistoryHeight {
		return history
	}

	if height == constant.MEMPOOL_HEIGHT {
		if g.FirstMempoolHistory == 0 {
			g.FirstMempoolHistory = history
		}
		return history
	}

	if g.LastHistoryHeight == 0 {
		g.FirstHistoryByHeight[height] = history
	} else {
		for h := g.LastHistoryHeight + 1; h <= height; h++ {
			g.FirstHistoryByHeight[h] = history
		}
	}
	g.LastHistoryHeight = height

	return history
}

func (g *BRC20ModuleIndexer) initBRC20() {
	g.EnableHistory = true
	g.BestHeight = 0

	g.BaseHistoryCount = 0
	g.HistoryCount = 0
	g.HistoryData = make([][]byte, 0)

	g.FirstHistoryByHeight = make(map[uint32]uint32, 0)
	g.LastHistoryHeight = 0
	g.FirstMempoolHistory = 0

	g.TickerMintCountByHeight = make(map[uint32]map[string]uint64)

	// user history
	g.UserAllHistory = make(map[string]*model.BRC20UserHistory, 0)
	g.OverlayUserAllHistory = make(map[string]*model.BRC20UserHistory, 0)

	// all ticker info
	g.InscriptionsTickerInfoMap = make(map[string]*model.BRC20TokenInfo, 0)

	g.UserTokensBalanceData = make(map[string]map[string]*model.BRC20TokenBalance, 0) // ticker of users
	g.TokenUsersBalanceData = make(map[string]map[string]*model.BRC20TokenBalance, 0) // ticker holders
	g.OverlayUserTokensBalanceData = make(map[string]map[string]*model.BRC20TokenBalance, 0)
	g.OverlayTokenUsersBalanceData = make(map[string]map[string]*model.BRC20TokenBalance, 0)
	g.PendingUserTokensBalanceData = make(map[string]map[string]*model.BRC20TokenBalance, 0)
	g.PendingTokenUsersBalanceData = make(map[string]map[string]*model.BRC20TokenBalance, 0)

	// ticker holders sorted cache
	g.TokenUsersBalanceDataSortedCache = make(map[string][]*model.BRC20TokenBalance, 0)

	// balance muhash
	g.BalanceMuHash = muhash.NewMuHash3072()
	g.StateHashByHeight = make(map[uint32][32]byte, 0)

	// valid brc20 inscriptions
	g.InscriptionsValidBRC20DataMap = make(map[uint64]*model.InscriptionBRC20InfoResp, 0)

	// inner valid transfer
	g.InscriptionsValidTransferMap = make(map[uint64]*model.InscriptionBRC20TickInfo, 0)
	// inner invalid transfer
	g.InscriptionsInvalidTransferMap = make(map[uint64]*model.InscriptionBRC20TickInfo, 0)
}

func (g *BRC20ModuleIndexer) initModule() {
	// all modules withdraw history
	g.AllModulesWithdrawHistory = make([]*model.BRC20ModuleHistory, 0)

	// all modules info
	g.ModulesInfoMap = make(map[string]*model.BRC20ModuleSwapInfo, 0)

	// module of users [address]moduleid
	g.UsersModuleWithTokenMap = make(map[string]string, 0)

	// swap
	// module of users [address]moduleid
	g.UsersModuleWithLpTokenMap = make(map[string]string, 0)

	// runtime for commit
	g.InscriptionsValidCommitMap = make(map[uint64]*model.InscriptionBRC20Data, 0) // inner valid commit
	g.InscriptionsInvalidCommitMap = make(map[uint64]*model.InscriptionBRC20Data, 0)

	g.InscriptionsValidCommitMapById = make(map[string]*model.InscriptionBRC20Data, 0) // inner valid commit

	// runtime for withdraw
	g.InscriptionsWithdrawMap = make(map[uint64]*model.InscriptionBRC20SwapInfo, 0)
}

func (g *BRC20ModuleIndexer) MergeHistoryOverlay() {
	for u, userHistory := range g.OverlayUserAllHistory {
		g.UserAllHistory[u] = userHistory
	}
}

// MergeOverlayToPending merges L1 overlay into L2 pending overlay, updating
// MuHash incrementally. This is safe to call at any height because it never
// touches the shared base maps.
func (g *BRC20ModuleIndexer) MergeOverlayToPending() {
	// merge user tokens: L1 → L2
	for userPkScript, userTokensL1 := range g.OverlayUserTokensBalanceData {
		var userTokensL2 map[string]*model.BRC20TokenBalance
		if tokens, ok := g.PendingUserTokensBalanceData[userPkScript]; !ok {
			userTokensL2 = make(map[string]*model.BRC20TokenBalance, 0)
			g.PendingUserTokensBalanceData[userPkScript] = userTokensL2
		} else {
			userTokensL2 = tokens
		}
		for uniqueLowerTicker, balance := range userTokensL1 {
			userTokensL2[uniqueLowerTicker] = balance
		}
	}

	// merge token users: L1 → L2, with MuHash update
	for uniqueLowerTicker, tokenUsersL1 := range g.OverlayTokenUsersBalanceData {
		var tokenUsersL2 map[string]*model.BRC20TokenBalance
		if users, ok := g.PendingTokenUsersBalanceData[uniqueLowerTicker]; !ok {
			tokenUsersL2 = make(map[string]*model.BRC20TokenBalance, 0)
			g.PendingTokenUsersBalanceData[uniqueLowerTicker] = tokenUsersL2
		} else {
			tokenUsersL2 = users
		}

		for userPkScript, balance := range tokenUsersL1 {
			// find old balance for MuHash: check L2 first, then base
			if oldBalance, ok := tokenUsersL2[userPkScript]; ok {
				g.BalanceMuHash.RemoveBalanceHash(oldBalance)
			} else if baseUsers, ok := g.TokenUsersBalanceData[uniqueLowerTicker]; ok {
				if oldBalance, ok := baseUsers[userPkScript]; ok {
					g.BalanceMuHash.RemoveBalanceHash(oldBalance)
				}
			}
			g.BalanceMuHash.ApplyBalanceHash(balance)
			// store in L2 (keep zero entries to shadow base)
			tokenUsersL2[userPkScript] = balance
		}
	}

	// clear L1
	g.OverlayUserTokensBalanceData = make(map[string]map[string]*model.BRC20TokenBalance, 0)
	g.OverlayTokenUsersBalanceData = make(map[string]map[string]*model.BRC20TokenBalance, 0)

	// record state hash for current height
	if g.BestHeight > 0 {
		g.StateHashByHeight[g.BestHeight] = g.BalanceMuHash.Finalize()
	}
}

// MergeBalanceOverlay flushes L1 → L2 (with MuHash), then merges L2 → base.
// Only safe when base maps are not shared (i.e. during loading or dump).
func (g *BRC20ModuleIndexer) MergeBalanceOverlay() {
	// flush any remaining L1 into L2
	g.MergeOverlayToPending()

	// merge L2 user tokens → base
	for userPkScript, userTokensL2 := range g.PendingUserTokensBalanceData {
		var userTokens map[string]*model.BRC20TokenBalance
		if tokens, ok := g.UserTokensBalanceData[userPkScript]; !ok {
			userTokens = make(map[string]*model.BRC20TokenBalance, 0)
			g.UserTokensBalanceData[userPkScript] = userTokens
		} else {
			userTokens = tokens
		}
		for uniqueLowerTicker, balance := range userTokensL2 {
			userTokens[uniqueLowerTicker] = balance
		}
	}

	// merge L2 token users → base (no MuHash, already accounted for)
	for uniqueLowerTicker, tokenUsersL2 := range g.PendingTokenUsersBalanceData {
		var tokenUsers map[string]*model.BRC20TokenBalance
		if holders, ok := g.TokenUsersBalanceData[uniqueLowerTicker]; !ok {
			tokenUsers = make(map[string]*model.BRC20TokenBalance, 0)
			g.TokenUsersBalanceData[uniqueLowerTicker] = tokenUsers
		} else {
			tokenUsers = holders
		}
		for userPkScript, balance := range tokenUsersL2 {
			if balance.AvailableBalance.Sign() == 0 && balance.TransferableBalance.Sign() == 0 {
				delete(tokenUsers, userPkScript)
			} else {
				tokenUsers[userPkScript] = balance
			}
		}
	}

	// clear L2
	g.PendingUserTokensBalanceData = make(map[string]map[string]*model.BRC20TokenBalance, 0)
	g.PendingTokenUsersBalanceData = make(map[string]map[string]*model.BRC20TokenBalance, 0)
}

// RebuildMuHashFromBalances reconstructs the MuHash state from all non-zero
// balance entries in TokenUsersBalanceData. Used after loading from dump.
func (g *BRC20ModuleIndexer) RebuildMuHashFromBalances() {
	g.BalanceMuHash = muhash.NewMuHash3072()
	count := 0
	for _, tokenUsers := range g.TokenUsersBalanceData {
		for _, balance := range tokenUsers {
			g.BalanceMuHash.ApplyBalanceHash(balance)
			count++
		}
	}
	log.Printf("RebuildMuHashFromBalances: applied %d balance entries", count)
}

func (g *BRC20ModuleIndexer) GetUserTokenBalance(ticker, userPkScript string) (tokenBalance *model.BRC20TokenBalance) {
	uniqueLowerTicker := strings.ToLower(ticker)

	// get from L1 overlay first
	if userTokens, ok := g.OverlayUserTokensBalanceData[userPkScript]; ok {
		if tb, ok := userTokens[uniqueLowerTicker]; ok {
			return tb
		}
	}

	// get from L2 pending overlay
	if userTokens, ok := g.PendingUserTokensBalanceData[userPkScript]; ok {
		if tb, ok := userTokens[uniqueLowerTicker]; ok {
			tokenBalance = tb.DeepCopy()
		}
	}

	// get from base
	if tokenBalance == nil {
		if userTokens, ok := g.UserTokensBalanceData[userPkScript]; ok {
			if tb, ok := userTokens[uniqueLowerTicker]; ok {
				tokenBalance = tb.DeepCopy()
			}
		}
	}

	// not found anywhere
	if tokenBalance == nil {
		tokenBalance = &model.BRC20TokenBalance{Ticker: ticker, PkScript: userPkScript}
	}

	// set user's tokens to update (in L1)
	var userTokens map[string]*model.BRC20TokenBalance
	if tokens, ok := g.OverlayUserTokensBalanceData[userPkScript]; !ok {
		userTokens = make(map[string]*model.BRC20TokenBalance, 0)
		g.OverlayUserTokensBalanceData[userPkScript] = userTokens
	} else {
		userTokens = tokens
	}
	userTokens[uniqueLowerTicker] = tokenBalance

	// set token's users (in L1)
	tokenUsers, ok := g.OverlayTokenUsersBalanceData[uniqueLowerTicker]
	if !ok {
		tokenUsers = make(map[string]*model.BRC20TokenBalance, 0)
		g.OverlayTokenUsersBalanceData[uniqueLowerTicker] = tokenUsers
	}
	tokenUsers[userPkScript] = tokenBalance

	return tokenBalance
}

func (copyDup *BRC20ModuleIndexer) deepCopyBRC20Data(base *BRC20ModuleIndexer, withData bool) {
	// Base balance
	copyDup.UserTokensBalanceData = base.UserTokensBalanceData
	copyDup.TokenUsersBalanceData = base.TokenUsersBalanceData

	// MuHash deep copy
	copyDup.BalanceMuHash = base.BalanceMuHash.DeepCopy()
	copyDup.StateHashByHeight = make(map[uint32][32]byte, len(base.StateHashByHeight))
	for h, hash := range base.StateHashByHeight {
		copyDup.StateHashByHeight[h] = hash
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		log.Printf("deepCopyBRC20Data history start. total: %d", base.HistoryCount)
		// history
		copyDup.BestHeight = base.BestHeight
		copyDup.EnableHistory = base.EnableHistory
		copyDup.HistoryCount = base.HistoryCount
		copyDup.BaseHistoryCount = base.BaseHistoryCount

		copyDup.FirstHistoryByHeight = make(map[uint32]uint32, len(base.FirstHistoryByHeight))
		for height, history := range base.FirstHistoryByHeight {
			copyDup.FirstHistoryByHeight[height] = history
		}
		copyDup.LastHistoryHeight = base.LastHistoryHeight
		copyDup.FirstMempoolHistory = base.FirstMempoolHistory

		copyDup.HistoryData = make([][]byte, 0, len(base.HistoryData))
		for _, h := range base.HistoryData {
			copyDup.HistoryData = append(copyDup.HistoryData, h)
		}
		log.Printf("deepCopyBRC20Data history finish. total: %d", base.HistoryCount)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		log.Printf("deepCopyBRC20Data mint count start. total: %d", len(base.TickerMintCountByHeight))

		copyDup.TickerMintCountByHeight = make(map[uint32]map[string]uint64, len(base.TickerMintCountByHeight))
		for height, mTickerMintCount := range base.TickerMintCountByHeight {
			cpTickerMintCount := make(map[string]uint64, len(mTickerMintCount))
			for ticker, count := range mTickerMintCount {
				cpTickerMintCount[ticker] = count
			}
			copyDup.TickerMintCountByHeight[height] = cpTickerMintCount
		}

		log.Printf("deepCopyBRC20Data mint count finish. total: %d", len(base.TickerMintCountByHeight))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		log.Printf("deepCopyBRC20Data user history start. total: %d", len(base.UserAllHistory))
		copyDup.UserAllHistory = base.UserAllHistory
		log.Printf("deepCopyBRC20Data user history finish. total: %d", len(base.UserAllHistory))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		log.Printf("deepCopyBRC20Data tick info start. total: %d", len(base.InscriptionsTickerInfoMap))
		copyDup.InscriptionsTickerInfoMap = make(map[string]*model.BRC20TokenInfo, len(base.InscriptionsTickerInfoMap))
		for k, v := range base.InscriptionsTickerInfoMap {
			tinfo := &model.BRC20TokenInfo{
				Ticker:   v.Ticker,
				SelfMint: v.SelfMint,
				Deploy:   v.Deploy.DeepCopy(),
			}

			// history
			tinfo.History = make([]uint32, len(v.History))
			copy(tinfo.History, v.History)

			tinfo.HistoryDeploy = make([]uint32, len(v.HistoryDeploy))
			copy(tinfo.HistoryDeploy, v.HistoryDeploy)

			tinfo.HistoryMint = make([]uint32, len(v.HistoryMint))
			copy(tinfo.HistoryMint, v.HistoryMint)

			tinfo.HistoryInscribeTransfer = make([]uint32, len(v.HistoryInscribeTransfer))
			copy(tinfo.HistoryInscribeTransfer, v.HistoryInscribeTransfer)

			tinfo.HistoryTransfer = make([]uint32, len(v.HistoryTransfer))
			copy(tinfo.HistoryTransfer, v.HistoryTransfer)

			// set info
			copyDup.InscriptionsTickerInfoMap[k] = tinfo
		}
		log.Printf("deepCopyBRC20Data tick info finish. total: %d", len(base.InscriptionsTickerInfoMap))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		log.Printf("deepCopyBRC20Data user balance start. total: %d", len(base.OverlayUserTokensBalanceData))
		// deep copy L1 overlay
		for u, userTokens := range base.OverlayUserTokensBalanceData {
			userTokensCopy := make(map[string]*model.BRC20TokenBalance, len(userTokens))
			copyDup.OverlayUserTokensBalanceData[u] = userTokensCopy
			for uniqueLowerTicker, v := range userTokens {
				tb := v.DeepCopy()
				userTokensCopy[uniqueLowerTicker] = tb

				tokenUsers, ok := copyDup.OverlayTokenUsersBalanceData[uniqueLowerTicker]
				if !ok {
					tokenUsers = make(map[string]*model.BRC20TokenBalance, 0)
					copyDup.OverlayTokenUsersBalanceData[uniqueLowerTicker] = tokenUsers
				}
				tokenUsers[u] = tb
			}
		}
		// deep copy L2 pending overlay
		for u, userTokens := range base.PendingUserTokensBalanceData {
			userTokensCopy := make(map[string]*model.BRC20TokenBalance, len(userTokens))
			copyDup.PendingUserTokensBalanceData[u] = userTokensCopy
			for uniqueLowerTicker, v := range userTokens {
				tb := v.DeepCopy()
				userTokensCopy[uniqueLowerTicker] = tb

				tokenUsers, ok := copyDup.PendingTokenUsersBalanceData[uniqueLowerTicker]
				if !ok {
					tokenUsers = make(map[string]*model.BRC20TokenBalance, 0)
					copyDup.PendingTokenUsersBalanceData[uniqueLowerTicker] = tokenUsers
				}
				tokenUsers[u] = tb
			}
		}
		log.Printf("deepCopyBRC20Data user balance finish. total L1: %d, L2: %d", len(base.OverlayUserTokensBalanceData), len(base.PendingUserTokensBalanceData))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		if withData {
			log.Printf("deepCopyBRC20Data valid data start. total: %d", len(base.InscriptionsValidBRC20DataMap))
			for k, v := range base.InscriptionsValidBRC20DataMap {
				copyDup.InscriptionsValidBRC20DataMap[k] = v
			}
			log.Printf("deepCopyBRC20Data valid data finish. total: %d", len(base.InscriptionsValidBRC20DataMap))
		}

	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		log.Printf("deepCopyBRC20Data valid transfer start. total: %d", len(base.InscriptionsValidTransferMap))
		// transferInfo
		copyDup.InscriptionsValidTransferMap = make(map[uint64]*model.InscriptionBRC20TickInfo, len(base.InscriptionsValidTransferMap))
		for k, v := range base.InscriptionsValidTransferMap {
			copyDup.InscriptionsValidTransferMap[k] = v
		}

		// invalid transfer info
		copyDup.InscriptionsInvalidTransferMap = make(map[uint64]*model.InscriptionBRC20TickInfo, len(base.InscriptionsInvalidTransferMap))
		for k, v := range base.InscriptionsInvalidTransferMap {
			copyDup.InscriptionsInvalidTransferMap[k] = v
		}
		log.Printf("deepCopyBRC20Data valid transfer finish. total: %d", len(base.InscriptionsValidTransferMap))
	}()

	wg.Wait()
	log.Printf("deepCopyBRC20Data finish. total: %d", len(base.InscriptionsTickerInfoMap))
}

func (copyDup *BRC20ModuleIndexer) cherryPickBRC20Data(base *BRC20ModuleIndexer, pickUsersPkScript, pickTokensTick map[string]bool) {
	// log.Printf("cherryPickBRC20Data start")
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		for lowerTick := range pickTokensTick {
			v, ok := base.InscriptionsTickerInfoMap[lowerTick]
			if !ok {
				continue
			}

			tinfo := &model.BRC20TokenInfo{
				Ticker:   v.Ticker,
				SelfMint: v.SelfMint,
				Deploy:   v.Deploy.DeepCopy(),
			}
			copyDup.InscriptionsTickerInfoMap[lowerTick] = tinfo
		}

	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		// use base
		copyDup.UserTokensBalanceData = base.UserTokensBalanceData
		copyDup.TokenUsersBalanceData = base.TokenUsersBalanceData

		// pick L1 overlay
		for u := range pickUsersPkScript {
			userTokens, ok := base.OverlayUserTokensBalanceData[u]
			if !ok {
				continue
			}
			userTokensCopy := make(map[string]*model.BRC20TokenBalance, 0)
			for lowerTick := range pickTokensTick {
				balance, ok := userTokens[lowerTick]
				if !ok {
					continue
				}
				userTokensCopy[lowerTick] = balance.DeepCopy()
			}
			copyDup.OverlayUserTokensBalanceData[u] = userTokensCopy
		}

		for u, userTokens := range copyDup.OverlayUserTokensBalanceData {
			for uniqueLowerTicker, balance := range userTokens {
				tokenUsers, ok := copyDup.OverlayTokenUsersBalanceData[uniqueLowerTicker]
				if !ok {
					tokenUsers = make(map[string]*model.BRC20TokenBalance, 0)
					copyDup.OverlayTokenUsersBalanceData[uniqueLowerTicker] = tokenUsers
				}
				tokenUsers[u] = balance
			}
		}

		// pick L2 pending overlay
		for u := range pickUsersPkScript {
			userTokens, ok := base.PendingUserTokensBalanceData[u]
			if !ok {
				continue
			}
			userTokensCopy := make(map[string]*model.BRC20TokenBalance, 0)
			for lowerTick := range pickTokensTick {
				balance, ok := userTokens[lowerTick]
				if !ok {
					continue
				}
				userTokensCopy[lowerTick] = balance.DeepCopy()
			}
			copyDup.PendingUserTokensBalanceData[u] = userTokensCopy
		}

		for u, userTokens := range copyDup.PendingUserTokensBalanceData {
			for uniqueLowerTicker, balance := range userTokens {
				tokenUsers, ok := copyDup.PendingTokenUsersBalanceData[uniqueLowerTicker]
				if !ok {
					tokenUsers = make(map[string]*model.BRC20TokenBalance, 0)
					copyDup.PendingTokenUsersBalanceData[uniqueLowerTicker] = tokenUsers
				}
				tokenUsers[u] = balance
			}
		}
	}()

	wg.Wait()

	// log.Printf("cherryPickBRC20Data finish. total: %d", len(copyDup.InscriptionsTickerInfoMap))
}

func (copyDup *BRC20ModuleIndexer) deepCopyModuleData(base *BRC20ModuleIndexer) {
	// all history
	copyDup.AllModulesWithdrawHistory = make([]*model.BRC20ModuleHistory, 0, len(base.AllModulesWithdrawHistory))
	for _, h := range base.AllModulesWithdrawHistory {
		copyDup.AllModulesWithdrawHistory = append(copyDup.AllModulesWithdrawHistory, h)
	}

	copyDup.ModulesInfoMap = make(map[string]*model.BRC20ModuleSwapInfo, len(base.ModulesInfoMap))
	for module, info := range base.ModulesInfoMap {
		copyDup.ModulesInfoMap[module] = info.DeepCopy()
	}

	// module of users
	copyDup.UsersModuleWithTokenMap = make(map[string]string, len(base.UsersModuleWithTokenMap))
	for k, v := range base.UsersModuleWithTokenMap {
		copyDup.UsersModuleWithTokenMap[k] = v
	}

	// module lp of users
	copyDup.UsersModuleWithLpTokenMap = make(map[string]string, len(base.UsersModuleWithLpTokenMap))
	for k, v := range base.UsersModuleWithLpTokenMap {
		copyDup.UsersModuleWithLpTokenMap[k] = v
	}

	// commitInfo
	copyDup.InscriptionsValidCommitMap = make(map[uint64]*model.InscriptionBRC20Data, len(base.InscriptionsValidCommitMap))
	for k, v := range base.InscriptionsValidCommitMap {
		copyDup.InscriptionsValidCommitMap[k] = v
	}

	copyDup.InscriptionsInvalidCommitMap = make(map[uint64]*model.InscriptionBRC20Data, len(base.InscriptionsInvalidCommitMap))
	for k, v := range base.InscriptionsInvalidCommitMap {
		copyDup.InscriptionsInvalidCommitMap[k] = v
	}

	copyDup.InscriptionsValidCommitMapById = make(map[string]*model.InscriptionBRC20Data, len(base.InscriptionsValidCommitMapById))
	for k, v := range base.InscriptionsValidCommitMapById {
		copyDup.InscriptionsValidCommitMapById[k] = v
	}

	// withdraw
	copyDup.InscriptionsWithdrawMap = make(map[uint64]*model.InscriptionBRC20SwapInfo, len(base.InscriptionsWithdrawMap))
	for k, v := range base.InscriptionsWithdrawMap {
		copyDup.InscriptionsWithdrawMap[k] = v
	}

	log.Printf("deepCopyModuleData finish. total: %d", len(base.ModulesInfoMap))
}

func (copyDup *BRC20ModuleIndexer) cherryPickModuleData(base *BRC20ModuleIndexer, module string, pickUsersPkScript, pickTokensTick, pickPoolsPair map[string]bool) {
	// log.Printf("cherryPickModuleData start")
	info, ok := base.ModulesInfoMap[module]
	if ok {
		// log.Printf("cherryPickModuleData. token_holder: %d, lp_holder: %d, pool: %d",
		// 	len(info.UsersTokenBalanceDataMap),
		// 	len(info.UsersLPTokenBalanceMap),
		// 	len(info.SwapPoolTotalBalanceDataMap),
		// )
		copyDup.ModulesInfoMap[module] = info.CherryPick(pickUsersPkScript, pickTokensTick, pickPoolsPair)
	}

	// Data required for verification
	for k, v := range base.InscriptionsValidCommitMapById {
		copyDup.InscriptionsValidCommitMapById[k] = v
	}
	// log.Printf("cherryPickModuleData finish. total: %d", len(base.ModulesInfoMap))
}

func (base *BRC20ModuleIndexer) DeepCopy(withData bool) (copyDup *BRC20ModuleIndexer) {
	copyDup = &BRC20ModuleIndexer{}
	copyDup.Init()

	copyDup.deepCopyBRC20Data(base, withData)
	copyDup.deepCopyModuleData(base)
	return copyDup
}

func (base *BRC20ModuleIndexer) CherryPick(module string, pickUsersPkScript, pickTokensTick, pickPoolsPair map[string]bool) (copyDup *BRC20ModuleIndexer) {
	copyDup = &BRC20ModuleIndexer{}
	copyDup.Init()

	moduleInfo, ok := base.ModulesInfoMap[module]
	if ok {
		lowerTick := strings.ToLower(moduleInfo.GasTick)
		pickTokensTick[lowerTick] = true
	}
	copyDup.cherryPickBRC20Data(base, pickUsersPkScript, pickTokensTick)
	copyDup.cherryPickModuleData(base, module, pickUsersPkScript, pickTokensTick, pickPoolsPair)
	return copyDup
}
