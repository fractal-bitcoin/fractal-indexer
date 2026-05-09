package indexer

import (
	"strings"

	"github.com/unisat-wallet/libbrc20-indexer/model"
)

func (g *BRC20ModuleIndexer) GetUserTokenBalanceMapForAPI(userPkScript string) (userTokens map[string]*model.BRC20TokenBalance) {
	userTokens = make(map[string]*model.BRC20TokenBalance)
	if userTokensBase, ok := g.UserTokensBalanceData[userPkScript]; ok {
		for key, info := range userTokensBase {
			userTokens[key] = info
		}
	}
	if userTokensL2, ok := g.PendingUserTokensBalanceData[userPkScript]; ok {
		for key, info := range userTokensL2 {
			userTokens[key] = info
		}
	}
	if userTokensL1, ok := g.OverlayUserTokensBalanceData[userPkScript]; ok {
		for key, info := range userTokensL1 {
			userTokens[key] = info
		}
	}
	return userTokens
}

func (g *BRC20ModuleIndexer) GetTokenHoldersBalanceMapForAPI(ticker string) []*model.BRC20TokenBalance {
	uniqueLowerTicker := strings.ToLower(ticker)

	// collect addresses already seen in higher-priority layers
	seen := make(map[string]bool)

	holdersBalance := make([]*model.BRC20TokenBalance, 0,
		len(g.TokenUsersBalanceData[uniqueLowerTicker])+
			len(g.PendingTokenUsersBalanceData[uniqueLowerTicker])+
			len(g.OverlayTokenUsersBalanceData[uniqueLowerTicker]))

	// L1 overlay (highest priority)
	if tokenUsersL1, ok := g.OverlayTokenUsersBalanceData[uniqueLowerTicker]; ok {
		for addr, info := range tokenUsersL1 {
			seen[addr] = true
			if info.Balance > 0 {
				holdersBalance = append(holdersBalance, info)
			}
		}
	}
	// L2 pending overlay
	if tokenUsersL2, ok := g.PendingTokenUsersBalanceData[uniqueLowerTicker]; ok {
		for addr, info := range tokenUsersL2 {
			if seen[addr] {
				continue
			}
			seen[addr] = true
			if info.Balance > 0 {
				holdersBalance = append(holdersBalance, info)
			}
		}
	}
	// base
	if tokenUsersBase, ok := g.TokenUsersBalanceData[uniqueLowerTicker]; ok {
		for addr, info := range tokenUsersBase {
			if seen[addr] {
				continue
			}
			if info.Balance > 0 {
				holdersBalance = append(holdersBalance, info)
			}
		}
	}

	return holdersBalance
}

func (g *BRC20ModuleIndexer) GetTokenHoldersCountNotAccurateForAPI(uniqueLowerTicker string, exact bool) (total int) {
	total = len(g.TokenUsersBalanceData[uniqueLowerTicker])
	total += len(g.PendingTokenUsersBalanceData[uniqueLowerTicker])
	total += len(g.OverlayTokenUsersBalanceData[uniqueLowerTicker])
	if !exact {
		return total
	}

	tokenUsersBase := g.TokenUsersBalanceData[uniqueLowerTicker]
	tokenUsersL2 := g.PendingTokenUsersBalanceData[uniqueLowerTicker]

	// dedup L1 vs L2 and base
	for addr, newBalance := range g.OverlayTokenUsersBalanceData[uniqueLowerTicker] {
		if _, ok := tokenUsersL2[addr]; ok {
			total -= 1
		} else if _, ok := tokenUsersBase[addr]; ok {
			total -= 1
		}
		if newBalance.Balance == 0 {
			total -= 1
		}
	}
	// dedup L2 vs base
	for addr, newBalance := range tokenUsersL2 {
		if _, ok := tokenUsersBase[addr]; ok {
			total -= 1
		}
		if newBalance.Balance == 0 {
			total -= 1
		}
	}
	return
}

func (g *BRC20ModuleIndexer) GetUserTokenBalanceOverlayForAPI(ticker, userPkScript string) (tokenBalance *model.BRC20TokenBalance, ok bool) {
	uniqueLowerTicker := strings.ToLower(ticker)

	// get from L1 overlay first
	if userTokens, ok := g.OverlayUserTokensBalanceData[userPkScript]; ok {
		if tb, ok := userTokens[uniqueLowerTicker]; ok {
			return tb, true
		}
	}

	// get from L2 pending overlay
	if userTokens, ok := g.PendingUserTokensBalanceData[userPkScript]; ok {
		if tb, ok := userTokens[uniqueLowerTicker]; ok {
			return tb, true
		}
	}

	// get from base then
	if userTokens, ok := g.UserTokensBalanceData[userPkScript]; ok {
		if tb, ok := userTokens[uniqueLowerTicker]; ok {
			return tb, true
		}
	}
	return nil, false
}
