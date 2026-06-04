package model

import (
	"fmt"
	"strings"

	"fractal-indexer/api/lib/brc20_swap/decimal"
	"fractal-indexer/api/lib/brc20_swap/uint128"
)

// decode data
type InscriptionBRC20ModuleDeploySwapContent struct {
	Proto     string            `json:"p,omitempty"`
	Operation string            `json:"op,omitempty"`
	Name      string            `json:"name,omitempty"`
	Source    string            `json:"source,omitempty"`
	Init      map[string]string `json:"init,omitempty"`
}

// check state for API
type SwapFunctionResultCheckStateForUser struct {
	Address       string `json:"address,omitempty"`
	Tick          string `json:"tick,omitempty"`
	Balance       string `json:"balance,omitempty"`
	LockedBalance string `json:"lockedBalance,omitempty"`
}

type SwapFunctionResultCheckStateForPool struct {
	Pair           string `json:"pair,omitempty"`
	ReserveAmount0 string `json:"reserve0,omitempty"`
	ReserveAmount1 string `json:"reserve1,omitempty"`
	LPAmount       string `json:"lp,omitempty"`
}

type SwapFunctionResultCheckState struct {
	Users       []SwapFunctionResultCheckStateForUser `json:"users,omitempty"`
	Pools       []SwapFunctionResultCheckStateForPool `json:"pools,omitempty"`
	CommitId    string                                `json:"commit,omitempty"`
	FunctionIdx int                                   `json:"function,omitempty"`
}

// load events
type BRC20ModuleHistoryInfoEvent struct {
	Type  string `json:"type"` // inscribe-deploy/inscribe-mint/inscribe-transfer/transfer/send/receive
	Valid bool   `json:"valid"`

	TxIdHex           string `json:"txid"`
	Idx               uint32 `json:"idx"` // inscription index
	Vout              uint32 `json:"vout"`
	Offset            uint64 `json:"offset"`
	InscriptionNumber int64  `json:"inscriptionNumber"`
	InscriptionId     string `json:"inscriptionId"`

	ContentType string `json:"contentType"`
	ContentBody string `json:"contentBody"`

	AddressFrom string `json:"from"`
	AddressTo   string `json:"to"`
	Satoshi     uint64 `json:"satoshi"`

	Height       uint32 `json:"height"`
	TxIdx        uint32 `json:"txidx"` // txidx in block
	BlockHashHex string `json:"blockhash"`
	BlockTime    uint32 `json:"blocktime"`
}

// commit function data
type SwapFunctionData struct {
	Address   string   `json:"addr,omitempty"`
	Function  string   `json:"func,omitempty"`
	Params    []string `json:"params,omitempty"`
	Timestamp uint     `json:"ts,omitempty"`
	Signature string   `json:"sig,omitempty"`

	ID       string `json:"-"`
	PkScript string `json:"-"`
	Invalid  bool   `json:"-"`
}

type InscriptionBRC20ModuleSwapCommitContent struct {
	Proto     string `json:"p,omitempty"`
	Operation string `json:"op,omitempty"`
	Module    string `json:"module,omitempty"`

	Parent      string              `json:"parent,omitempty"`
	GasPrice    string              `json:"gas_price,omitempty"`
	FeeRateSwap string              `json:"swap_fee_rate,omitempty"`
	Data        []*SwapFunctionData `json:"data,omitempty"`
}

// module state
type BRC20ModuleSwapInfo struct {
	ID                string // module id
	Name              string // module name
	DeployerPkScript  string // deployer
	SequencerPkScript string // operator, sequencer
	GasToPkScript     string //
	LpFeePkScript     string //

	FeeRateSwap string
	GasTick     string

	FeeRateSwapMut string

	History []*BRC20ModuleHistory // history for deploy, deposit, commit, quit

	// runtime for commit
	CommitInvalidMap map[string]struct{} // All invalid create commits; never used

	CommitIdMap      map[string]struct{} // All valid create commits; cannot be replaced
	CommitIdChainMap map[string]struct{} // All connected commits; cannot be used as parents for subsequent commits again.

	// token holders in module
	// ticker of users in module [address][tick]balanceData
	UsersTokenBalanceDataMap map[string]map[string]*BRC20ModuleTokenBalance
	// token balance of address in module [tick][address]balanceData
	TokenUsersBalanceDataMap map[string]map[string]*BRC20ModuleTokenBalance

	// swap
	// lp token balance of address in module [pool][address]balance
	LPTokenUsersBalanceMap map[string]map[string]*BRC20ModuleLPTokenBalance
	// lp token of users in module [address][pool]balance
	UsersLPTokenBalanceMap map[string]map[string]*BRC20ModuleLPTokenBalance

	// swap total balance
	// total balance of pool in module [pool]balanceData
	SwapPoolTotalBalanceDataMap map[string]*BRC20ModulePoolTotalBalance
}

func (m *BRC20ModuleSwapInfo) GetBRC20ModuleLPTokenBalanceByUser(pkScript, poolPair string) (data *BRC20ModuleLPTokenBalance) {
	usersLpBalanceInPool := m.LPTokenUsersBalanceMap[poolPair]

	// lpFee lp balance update
	lpFeelpbalance, ok := usersLpBalanceInPool[pkScript]
	if !ok {
		lpFeelpbalance = &BRC20ModuleLPTokenBalance{}
		usersLpBalanceInPool[pkScript] = lpFeelpbalance

		// lpFee-lp-balance
		lpFeelpsBalance, ok := m.UsersLPTokenBalanceMap[pkScript]
		if !ok {
			lpFeelpsBalance = make(map[string]*BRC20ModuleLPTokenBalance, 0)
			m.UsersLPTokenBalanceMap[pkScript] = lpFeelpsBalance
		}
		lpFeelpsBalance[poolPair] = lpFeelpbalance
	}

	return lpFeelpbalance
}

func (m *BRC20ModuleSwapInfo) DeepCopy() (copy *BRC20ModuleSwapInfo) {
	copy = &BRC20ModuleSwapInfo{
		ID:                m.ID,
		Name:              m.Name,
		DeployerPkScript:  m.DeployerPkScript,  // deployer
		SequencerPkScript: m.SequencerPkScript, // Sequencer
		GasToPkScript:     m.GasToPkScript,
		LpFeePkScript:     m.LpFeePkScript,

		FeeRateSwap: m.FeeRateSwap,
		GasTick:     m.GasTick,

		History: make([]*BRC20ModuleHistory, 0),

		// runtime for commit
		CommitInvalidMap: make(map[string]struct{}, 0),
		CommitIdChainMap: make(map[string]struct{}, 0),
		CommitIdMap:      make(map[string]struct{}, 0),

		// runtime for holders
		// token holders in module
		// ticker of users in module [address][tick]balanceData
		UsersTokenBalanceDataMap: make(map[string]map[string]*BRC20ModuleTokenBalance, 0),

		// token balance of address in module [tick][address]balanceData
		TokenUsersBalanceDataMap: make(map[string]map[string]*BRC20ModuleTokenBalance, 0),

		// swap
		// lp token balance of address in module [pair][address]balance
		LPTokenUsersBalanceMap: make(map[string]map[string]*BRC20ModuleLPTokenBalance, 0),

		// lp token of users in module [address][pair]balance
		UsersLPTokenBalanceMap: make(map[string]map[string]*BRC20ModuleLPTokenBalance, 0),

		// swap total balance
		// total balance of pool in module [pair]balanceData
		SwapPoolTotalBalanceDataMap: make(map[string]*BRC20ModulePoolTotalBalance, 0),
	}

	for _, h := range m.History {
		copy.History = append(copy.History, h)
		// fix: more history
	}

	// invalid commit
	for k := range m.CommitInvalidMap {
		copy.CommitInvalidMap[k] = struct{}{}
	}

	for k := range m.CommitIdChainMap {
		copy.CommitIdChainMap[k] = struct{}{}
	}
	for k := range m.CommitIdMap {
		copy.CommitIdMap[k] = struct{}{}
	}

	// user/tick: balance
	for address, dataMap := range m.UsersTokenBalanceDataMap {
		dataMapCopy := make(map[string]*BRC20ModuleTokenBalance, 0)
		for tick, balance := range dataMap {
			dataMapCopy[tick] = balance.DeepCopy()
		}
		copy.UsersTokenBalanceDataMap[address] = dataMapCopy
	}
	// tick/user: balance
	for tick, dataMap := range m.TokenUsersBalanceDataMap {
		dataMapCopy := make(map[string]*BRC20ModuleTokenBalance, 0)
		for address := range dataMap {
			dataMapCopy[address] = copy.UsersTokenBalanceDataMap[address][tick]
		}
		copy.TokenUsersBalanceDataMap[tick] = dataMapCopy
	}

	// user/pair: lpbalance
	for address, dataMap := range m.UsersLPTokenBalanceMap {
		dataMapCopy := make(map[string]*BRC20ModuleLPTokenBalance, 0)
		for pair, balance := range dataMap {
			dataMapCopy[pair] = balance.DeepCopy()
		}
		copy.UsersLPTokenBalanceMap[address] = dataMapCopy
	}
	// pair/user: lpbalance
	for pair, dataMap := range m.LPTokenUsersBalanceMap {
		dataMapCopy := make(map[string]*BRC20ModuleLPTokenBalance, 0)
		for address := range dataMap {
			dataMapCopy[address] = copy.UsersLPTokenBalanceMap[address][pair]
		}
		copy.LPTokenUsersBalanceMap[pair] = dataMapCopy
	}

	// swap total balance
	for pair, balance := range m.SwapPoolTotalBalanceDataMap {
		copy.SwapPoolTotalBalanceDataMap[pair] = balance.DeepCopy()
	}

	return copy
}

func (m *BRC20ModuleSwapInfo) CherryPick(pickUsersPkScript, pickTokensTick, pickPoolsPair map[string]bool) (copy *BRC20ModuleSwapInfo) {
	copy = &BRC20ModuleSwapInfo{
		ID:                m.ID,
		Name:              m.Name,
		DeployerPkScript:  m.DeployerPkScript,  // deployer
		SequencerPkScript: m.SequencerPkScript, // Sequencer
		GasToPkScript:     m.GasToPkScript,
		LpFeePkScript:     m.LpFeePkScript,

		FeeRateSwap: m.FeeRateSwap,
		GasTick:     m.GasTick,

		// runtime for commit
		CommitIdChainMap: make(map[string]struct{}, 0),
		CommitIdMap:      make(map[string]struct{}, 0),

		// runtime for holders
		// token holders in module
		// ticker of users in module [address][tick]balanceData
		UsersTokenBalanceDataMap: make(map[string]map[string]*BRC20ModuleTokenBalance, 0),

		// token balance of address in module [tick][address]balanceData
		TokenUsersBalanceDataMap: make(map[string]map[string]*BRC20ModuleTokenBalance, 0),

		// swap
		// lp token balance of address in module [pair][address]balance
		LPTokenUsersBalanceMap: make(map[string]map[string]*BRC20ModuleLPTokenBalance, 0),

		// lp token of users in module [address][pair]balance
		UsersLPTokenBalanceMap: make(map[string]map[string]*BRC20ModuleLPTokenBalance, 0),

		// swap total balance
		// total balance of pool in module [pair]balanceData
		SwapPoolTotalBalanceDataMap: make(map[string]*BRC20ModulePoolTotalBalance, 0),
	}

	for k := range m.CommitIdChainMap {
		copy.CommitIdChainMap[k] = struct{}{}
	}
	for k := range m.CommitIdMap {
		copy.CommitIdMap[k] = struct{}{}
	}

	////////////////
	// user/tick: balance
	for u := range pickUsersPkScript {
		userTokens, ok := m.UsersTokenBalanceDataMap[u]
		if !ok {
			continue
		}
		userTokensCopy := make(map[string]*BRC20ModuleTokenBalance, 0)
		for lowerTick := range pickTokensTick {
			balance, ok := userTokens[lowerTick]
			if !ok {
				continue
			}
			userTokensCopy[lowerTick] = balance.CherryPick()
		}
		copy.UsersTokenBalanceDataMap[u] = userTokensCopy
	}
	// tick/user: balance
	for lowerTick := range pickTokensTick {
		tokenUsers, ok := m.TokenUsersBalanceDataMap[lowerTick]
		if !ok {
			continue
		}
		tokenUsersCopy := make(map[string]*BRC20ModuleTokenBalance, 0)
		for u := range pickUsersPkScript {
			balance, ok := tokenUsers[lowerTick]
			if !ok {
				continue
			}
			tokenUsersCopy[u] = balance.CherryPick()
		}
		copy.TokenUsersBalanceDataMap[lowerTick] = tokenUsersCopy
	}
	////////////////
	// user/pair: lpbalance
	for u := range pickUsersPkScript {
		userTokens, ok := m.UsersLPTokenBalanceMap[u]
		if !ok {
			continue
		}
		userTokensCopy := make(map[string]*BRC20ModuleLPTokenBalance, 0)
		for pair := range pickPoolsPair {
			balance, ok := userTokens[pair]
			if !ok {
				continue
			}
			userTokensCopy[pair] = balance.DeepCopy()
		}
		copy.UsersLPTokenBalanceMap[u] = userTokensCopy
	}
	// pair/user: lpbalance
	for pair := range pickPoolsPair {
		tokenUsers, ok := m.LPTokenUsersBalanceMap[pair]
		if !ok {
			continue
		}
		tokenUsersCopy := make(map[string]*BRC20ModuleLPTokenBalance, 0)
		for u := range pickUsersPkScript {
			balance, ok := tokenUsers[u]
			if !ok {
				continue
			}
			tokenUsersCopy[u] = balance.DeepCopy()
		}
		copy.LPTokenUsersBalanceMap[pair] = tokenUsersCopy
	}
	////////////////
	// swap total balance
	for pair := range pickPoolsPair {
		balance, ok := m.SwapPoolTotalBalanceDataMap[pair]
		if !ok {
			continue
		}
		copy.SwapPoolTotalBalanceDataMap[pair] = balance.CherryPick()
	}

	// swap deposit/approve state balance
	// no need
	return copy
}

func (moduleInfo *BRC20ModuleSwapInfo) GetUserTokenBalance(ticker, userPkScript string) (tokenBalance *BRC20ModuleTokenBalance) {
	uniqueLowerTicker := strings.ToLower(ticker)
	// get user's tokens to update
	var usersTokens map[string]*BRC20ModuleTokenBalance
	if tokens, ok := moduleInfo.UsersTokenBalanceDataMap[userPkScript]; !ok {
		usersTokens = make(map[string]*BRC20ModuleTokenBalance, 0)
		moduleInfo.UsersTokenBalanceDataMap[userPkScript] = usersTokens
	} else {
		usersTokens = tokens
	}
	// get tokenBalance to update
	if tb, ok := usersTokens[uniqueLowerTicker]; !ok {
		tokenBalance = &BRC20ModuleTokenBalance{Tick: ticker, PkScript: userPkScript}
		usersTokens[uniqueLowerTicker] = tokenBalance
	} else {
		tokenBalance = tb
		return tokenBalance
	}

	// set token's users
	tokenUsers, ok := moduleInfo.TokenUsersBalanceDataMap[uniqueLowerTicker]
	if !ok {
		tokenUsers = make(map[string]*BRC20ModuleTokenBalance, 0)
		moduleInfo.TokenUsersBalanceDataMap[uniqueLowerTicker] = tokenUsers
	}
	tokenUsers[userPkScript] = tokenBalance
	return tokenBalance
}

type BRC20ModuleLPTokenBalance struct {
	Balance       *decimal.Decimal
	LockedBalance *decimal.Decimal
}

func (bal *BRC20ModuleLPTokenBalance) OverallBalance() *decimal.Decimal {
	return bal.Balance.Add(bal.LockedBalance)
}

func (in *BRC20ModuleLPTokenBalance) DeepCopy() *BRC20ModuleLPTokenBalance {
	tb := &BRC20ModuleLPTokenBalance{
		Balance:       in.Balance,
		LockedBalance: in.LockedBalance,
	}
	return tb
}

// state of address for each tick, (balance and history)
type BRC20ModuleTokenBalance struct {
	Tick     string
	PkScript string

	// confirmed safe
	SwapAccountBalanceSafe   uint128.Decimal
	ModuleAccountBalanceSafe uint128.Decimal

	// with unconfirmed balance
	SwapAccountBalance    uint128.Decimal
	AvailableBalanceSafe  uint128.Decimal
	AvailableBalance      uint128.Decimal
	ReadyToWithdrawAmount uint128.Decimal

	ReadyToWithdrawMap map[uint64]*InscriptionBRC20Data // ready to use, but inscription may invalid(depends on available b)

	History []*BRC20ModuleHistory
}

func (b *BRC20ModuleTokenBalance) String() string {
	return fmt.Sprintf("%s", b.SwapAccountBalance.String())
}

func (in *BRC20ModuleTokenBalance) DeepCopy() *BRC20ModuleTokenBalance {
	tb := &BRC20ModuleTokenBalance{
		Tick:     in.Tick,
		PkScript: in.PkScript,

		SwapAccountBalanceSafe:   in.SwapAccountBalanceSafe,
		ModuleAccountBalanceSafe: in.ModuleAccountBalanceSafe,

		SwapAccountBalance: in.SwapAccountBalance,

		AvailableBalanceSafe: in.AvailableBalanceSafe,
		AvailableBalance:     in.AvailableBalance,

		ReadyToWithdrawAmount: in.ReadyToWithdrawAmount,

		ReadyToWithdrawMap: make(map[uint64]*InscriptionBRC20Data, len(in.ReadyToWithdrawMap)),
	}

	for k, v := range in.ReadyToWithdrawMap {
		data := *v
		tb.ReadyToWithdrawMap[k] = &data
	}

	for _, h := range in.History {
		tb.History = append(tb.History, h)
		// fix: more history
	}
	// tb.History = make([]BRC20History, len(in.History))
	// copy(tb.History, in.History)
	return tb
}

func (in *BRC20ModuleTokenBalance) CherryPick() *BRC20ModuleTokenBalance {
	tb := &BRC20ModuleTokenBalance{
		Tick:     in.Tick,
		PkScript: in.PkScript,

		SwapAccountBalanceSafe:   in.SwapAccountBalanceSafe,
		ModuleAccountBalanceSafe: in.ModuleAccountBalanceSafe,

		SwapAccountBalance: in.SwapAccountBalance,

		AvailableBalanceSafe: in.AvailableBalanceSafe,
		AvailableBalance:     in.AvailableBalance,

		ReadyToWithdrawAmount: in.ReadyToWithdrawAmount,
	}
	return tb
}

// state of address for each tick, (balance and history)
type BRC20ModulePoolTotalBalance struct {
	Tick        [2]string
	TickBalance [2]*decimal.Decimal
	LpBalance   *decimal.Decimal
	LastRootK   *decimal.Decimal

	// history
	History []*BRC20ModuleHistory
}

func (in *BRC20ModulePoolTotalBalance) DeepCopy() *BRC20ModulePoolTotalBalance {
	tb := &BRC20ModulePoolTotalBalance{
		Tick:        in.Tick,
		TickBalance: in.TickBalance,

		LpBalance: in.LpBalance,
		LastRootK: in.LastRootK,
	}

	for _, h := range in.History {
		tb.History = append(tb.History, h)
		// fix: more history
	}
	return tb
}

func (in *BRC20ModulePoolTotalBalance) CherryPick() *BRC20ModulePoolTotalBalance {
	tb := &BRC20ModulePoolTotalBalance{
		Tick:        in.Tick,
		TickBalance: in.TickBalance,

		LpBalance: in.LpBalance,
		LastRootK: in.LastRootK,
	}
	return tb
}

type InscriptionBRC20SwapInfo struct {
	Module string
	Tick   string
	Amount uint128.Decimal
	Data   *InscriptionBRC20Data
}

// history inscription info
type InscriptionBRC20SwapInfoResp struct {
	ContentBody       []byte `json:"content"`
	InscriptionNumber int64  `json:"inscriptionNumber"`
	InscriptionId     string `json:"inscriptionId"`

	Height        uint32 `json:"-"`
	Confirmations int    `json:"confirmations"`
}
