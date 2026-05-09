package indexer

import (
	"errors"
	"fmt"

	"github.com/unisat-wallet/libbrc20-indexer/model"
)

func (g *BRC20ModuleIndexer) ProcessCommitFunctionLockLp(moduleInfo *model.BRC20ModuleSwapInfo, f *model.SwapFunctionData) error {
	token0, token1 := f.Params[0], f.Params[1]
	poolPair := GetLowerInnerPairNameByToken(token0, token1)
	if _, ok := moduleInfo.SwapPoolTotalBalanceDataMap[poolPair]; !ok {
		return errors.New("locklp: pool invalid")
	}

	if _, ok := moduleInfo.LPTokenUsersBalanceMap[poolPair]; !ok {
		return errors.New("locklp: lps balance map missing pair")
	}

	tokenAmtStr := f.Params[2]
	tokenLpAmt, _ := CheckAmountVerify(tokenAmtStr, 18)

	userbalanceFrom := moduleInfo.GetBRC20ModuleLPTokenBalanceByUser(f.PkScript, poolPair)
	// Check if the user's lp balance is sufficient.
	if userbalanceFrom.Balance.Cmp(tokenLpAmt) < 0 {
		return errors.New(fmt.Sprintf("locklp: user's tokenLp balance insufficient, %s < %s", userbalanceFrom.Balance, tokenLpAmt))
	}

	// update from lp balance
	userbalanceFrom.Balance = userbalanceFrom.Balance.Sub(tokenLpAmt)
	userbalanceFrom.LockedBalance = userbalanceFrom.LockedBalance.Add(tokenLpAmt)

	// log.Printf("pool locklp [%s] lp: %s -> %s", poolPair, userbalanceFrom.Balance, userbalanceFrom.LockedBalance)
	return nil
}
