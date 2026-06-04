package indexer

import (
	"errors"
	"fmt"

	"fractal-indexer/api/lib/brc20_swap/model"
)

func (g *BRC20ModuleIndexer) ProcessCommitFunctionUnlockLp(moduleInfo *model.BRC20ModuleSwapInfo, f *model.SwapFunctionData) error {
	token0, token1 := f.Params[0], f.Params[1]
	poolPair := GetLowerInnerPairNameByToken(token0, token1)
	if _, ok := moduleInfo.SwapPoolTotalBalanceDataMap[poolPair]; !ok {
		return errors.New("unlocklp: pool invalid")
	}

	if _, ok := moduleInfo.LPTokenUsersBalanceMap[poolPair]; !ok {
		return errors.New("unlocklp: lps balance map missing pair")
	}

	tokenAmtStr := f.Params[2]
	tokenLpAmt, _ := CheckAmountVerify(tokenAmtStr, 18)

	userbalanceFrom := moduleInfo.GetBRC20ModuleLPTokenBalanceByUser(f.PkScript, poolPair)
	// Check if the user's lp balance is sufficient.
	if userbalanceFrom.LockedBalance.Cmp(tokenLpAmt) < 0 {
		return errors.New(fmt.Sprintf("unlocklp: user's tokenLp balance insufficient, %s < %s", userbalanceFrom.LockedBalance, tokenLpAmt))
	}

	// update from lp balance
	userbalanceFrom.Balance = userbalanceFrom.Balance.Add(tokenLpAmt)
	userbalanceFrom.LockedBalance = userbalanceFrom.LockedBalance.Sub(tokenLpAmt)

	// log.Printf("pool unlocklp [%s] lp: %s -> %s", poolPair, userbalanceFrom.Balance, userbalanceFrom.LockedBalance)
	return nil
}
