package indexer

import (
	"errors"
	"fmt"

	"fractal-indexer/api/lib/brc20_swap/conf"
	"fractal-indexer/api/lib/brc20_swap/model"
	"fractal-indexer/api/lib/brc20_swap/utils"
)

func (g *BRC20ModuleIndexer) ProcessCommitFunctionSendLp(moduleInfo *model.BRC20ModuleSwapInfo, f *model.SwapFunctionData) error {
	addressTo := f.Params[0]
	pkScriptTo, _ := utils.GetPkScriptByAddress(addressTo, conf.GlobalNetParams)

	token0, token1 := f.Params[1], f.Params[2]
	poolPair := GetLowerInnerPairNameByToken(token0, token1)
	if _, ok := moduleInfo.SwapPoolTotalBalanceDataMap[poolPair]; !ok {
		return errors.New("sendlp: pool invalid")
	}

	if _, ok := moduleInfo.LPTokenUsersBalanceMap[poolPair]; !ok {
		return errors.New("sendlp: lps balance map missing pair")
	}

	tokenAmtStr := f.Params[3]
	tokenLpAmt, _ := CheckAmountVerify(tokenAmtStr, 18)

	userbalanceFrom := moduleInfo.GetBRC20ModuleLPTokenBalanceByUser(f.PkScript, poolPair)
	// Check if the user's lp balance is sufficient.
	if userbalanceFrom.Balance.Cmp(tokenLpAmt) < 0 {
		return errors.New(fmt.Sprintf("sendlp: user's tokenLp balance insufficient, %s < %s", userbalanceFrom.Balance, tokenLpAmt))
	}

	// update from lp balance
	userbalanceFrom.Balance = userbalanceFrom.Balance.Sub(tokenLpAmt)

	// update to lp balance
	lpBalanceTo := moduleInfo.GetBRC20ModuleLPTokenBalanceByUser(string(pkScriptTo), poolPair)
	lpBalanceTo.Balance = lpBalanceTo.Balance.Add(tokenLpAmt)

	// log.Printf("pool sendlp [%s] lp: %s -> %s", poolPair, lpBalanceFrom, lpBalanceTo)
	return nil
}
