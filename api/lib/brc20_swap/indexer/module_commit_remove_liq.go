package indexer

import (
	"errors"
	"fmt"
	"log"

	"fractal-indexer/api/lib/brc20_swap/decimal"
	"fractal-indexer/api/lib/brc20_swap/model"
)

func (g *BRC20ModuleIndexer) ProcessCommitFunctionRemoveLiquidity(moduleInfo *model.BRC20ModuleSwapInfo, f *model.SwapFunctionData) (err error) {
	token0, token1 := f.Params[0], f.Params[1]
	poolPair := GetLowerInnerPairNameByToken(token0, token1)

	pool, ok := moduleInfo.SwapPoolTotalBalanceDataMap[poolPair]
	if !ok {
		return errors.New("removeLiq: pool invalid")
	}

	if _, ok := moduleInfo.LPTokenUsersBalanceMap[poolPair]; !ok {
		return errors.New("removeLiq: lps balance map missing pair")
	}

	// log.Printf("[%s] pool before removeliq [%s] %s: %s, %s: %s, lp: %s", moduleInfo.ID, poolPair, pool.Tick[0], pool.TickBalance[0], pool.Tick[1], pool.TickBalance[1], pool.LpBalance)
	// log.Printf("pool removeliq params: %v", f.Params)

	tokenLpAmtStr := f.Params[2]
	token0AmtStr := f.Params[3]
	token1AmtStr := f.Params[4]

	token0Amt, ok := g.CheckTickVerifyDecimal(token0, token0AmtStr)
	if !ok {
		return errors.New("removeLiq: amt0 invalid")
	}

	token1Amt, ok := g.CheckTickVerifyDecimal(token1, token1AmtStr)
	if !ok {
		return errors.New("removeLiq: amt1 invalid")
	}

	tokenLpAmt, _ := decimal.FromString(tokenLpAmtStr, 18)

	// LP Balance Slippage Check
	slippageAmtStr := f.Params[5]
	slippageAmt, _ := decimal.FromString(slippageAmtStr, 3)

	var token0Idx, token1Idx int
	if token0 == pool.Tick[0] {
		token0Idx = 0
		token1Idx = 1
	} else {
		token0Idx = 1
		token1Idx = 0
	}

	var lpFee *decimal.Decimal
	// Increase LP, as a method of collecting service fees.
	feeRateSwapAmt, ok := CheckAmountVerify(moduleInfo.FeeRateSwapMut, 3)
	if !ok {
		log.Printf("pool removeLiq FeeRateSwap invalid: %s", moduleInfo.FeeRateSwapMut)
		return errors.New("removeLiq: feerate swap invalid")
	}

	poolLpBalance := pool.LpBalance
	if feeRateSwapAmt.Sign() > 0 {
		// lp = (poolLp * (rootK - rootKLast)) / (rootK * 5 + rootKLast)
		rootK := pool.TickBalance[token0Idx].Mul(pool.TickBalance[token1Idx]).Sqrt()

		lpFee = poolLpBalance.Mul(rootK.Sub(pool.LastRootK)).Div(
			rootK.Mul(decimal.NewDecimal(5, 0)).Add(pool.LastRootK))
		if lpFee.Sign() > 0 {
			// pool lp update
			poolLpBalance = poolLpBalance.Add(lpFee)
		}
	}

	// Slippage Check
	amt0 := pool.TickBalance[token0Idx].Mul(tokenLpAmt).Div(poolLpBalance)
	if amt0.Cmp(token0Amt.Sub(token0Amt.Mul(slippageAmt))) < 0 {
		log.Printf("user[%s], token0: %s, expect: %s", f.Address, amt0, token0Amt)
		return errors.New("removeLiq: over slippage")
	}
	amt1 := pool.TickBalance[token1Idx].Mul(tokenLpAmt).Div(poolLpBalance)
	if amt1.Cmp(token1Amt.Sub(token1Amt.Mul(slippageAmt))) < 0 {
		log.Printf("user[%s], token1: %s, expect: %s", f.Address, amt1, token1Amt)
		return errors.New("removeLiq: over slippage")
	}

	// Changes in pool balance
	if poolLpBalance.Cmp(tokenLpAmt) < 0 {
		return errors.New(fmt.Sprintf("removeLiq: tokenLp balance insufficient, %s < %s", poolLpBalance, tokenLpAmt))
	}
	if pool.TickBalance[token0Idx].Cmp(amt0) < 0 {
		return errors.New(fmt.Sprintf("removeLiq: pool %s balance insufficient", pool.Tick[token1Idx]))
	}
	if pool.TickBalance[token1Idx].Cmp(amt1) < 0 {
		return errors.New(fmt.Sprintf("removeLiq: pool %s balance insufficient", pool.Tick[token1Idx]))
	}

	// Check whether the user's LP balance is consistent (consider storing only one copy)
	userbalance := moduleInfo.GetBRC20ModuleLPTokenBalanceByUser(f.PkScript, poolPair)
	// Check whether the balance of user LP is sufficient.
	if userbalance.Balance.Cmp(tokenLpAmt) < 0 {
		return errors.New(fmt.Sprintf("removeLiq: user's tokenLp balance insufficient, %s < %s", userbalance.Balance, tokenLpAmt))
	}

	// update lp balance
	userbalance.Balance = userbalance.Balance.Sub(tokenLpAmt)

	token0Balance := moduleInfo.GetUserTokenBalance(token0, f.PkScript)
	token1Balance := moduleInfo.GetUserTokenBalance(token1, f.PkScript)

	amt0Uint128 := amt0.Uint128()
	amt1Uint128 := amt1.Uint128()
	// Obtains user token balance
	token0Balance.SwapAccountBalance = token0Balance.SwapAccountBalance.Add(amt0Uint128)
	token1Balance.SwapAccountBalance = token1Balance.SwapAccountBalance.Add(amt1Uint128)
	token0Balance.SwapAccountBalanceSafe = token0Balance.SwapAccountBalanceSafe.Add(amt0Uint128)
	token1Balance.SwapAccountBalanceSafe = token1Balance.SwapAccountBalanceSafe.Add(amt1Uint128)

	pool.LpBalance = pool.LpBalance.Sub(tokenLpAmt) // fixme

	if lpFee.Sign() > 0 {
		pool.LpBalance = pool.LpBalance.Add(lpFee)
		// lpFee update
		lpFeelpbalance := moduleInfo.GetBRC20ModuleLPTokenBalanceByUser(moduleInfo.LpFeePkScript, poolPair)
		lpFeelpbalance.Balance = lpFeelpbalance.Balance.Add(lpFee)
	}

	// Deduct token balance in the pool
	pool.TickBalance[token0Idx] = pool.TickBalance[token0Idx].Sub(amt0)
	pool.TickBalance[token1Idx] = pool.TickBalance[token1Idx].Sub(amt1)

	// update lastRootK
	pool.LastRootK = pool.TickBalance[token0Idx].Mul(pool.TickBalance[token1Idx]).Sqrt()

	// log.Printf("[%s] pool after removeliq [%s] %s: %s, %s: %s, lp: %s", moduleInfo.ID, poolPair, pool.Tick[0], pool.TickBalance[0], pool.Tick[1], pool.TickBalance[1], pool.LpBalance)
	return nil
}
