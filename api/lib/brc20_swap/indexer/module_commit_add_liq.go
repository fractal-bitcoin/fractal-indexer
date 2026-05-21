package indexer

import (
	"errors"
	"log"

	"fractal-indexer/api/lib/brc20_swap/constant"
	"fractal-indexer/api/lib/brc20_swap/decimal"
	"fractal-indexer/api/lib/brc20_swap/model"
)

func (g *BRC20ModuleIndexer) ProcessCommitFunctionAddLiquidity(moduleInfo *model.BRC20ModuleSwapInfo, f *model.SwapFunctionData) (err error) {
	token0, token1 := f.Params[0], f.Params[1]
	poolPair := GetLowerInnerPairNameByToken(token0, token1)

	pool, ok := moduleInfo.SwapPoolTotalBalanceDataMap[poolPair]
	if !ok {
		return errors.New("addLiq: pool invalid")
	}

	if _, ok := moduleInfo.LPTokenUsersBalanceMap[poolPair]; !ok {
		return errors.New("addLiq: users invalid")
	}

	// log.Printf("[%s] pool before addliq [%s] %s: %s, %s: %s, lp: %s", moduleInfo.ID, poolPair, pool.Tick[0], pool.TickBalance[0], pool.Tick[1], pool.TickBalance[1], pool.LpBalance)
	// log.Printf("pool addliq params: %v", f.Params)

	token0AmtStr := f.Params[2]
	token1AmtStr := f.Params[3]
	tokenLpAmtStr := f.Params[4]

	token0Amt, ok := g.CheckTickVerifyDecimal(token0, token0AmtStr)
	if !ok {
		return errors.New("addLiq: amt0 invalid")
	}

	token1Amt, ok := g.CheckTickVerifyDecimal(token1, token1AmtStr)
	if !ok {
		return errors.New("addLiq: amt1 invalid")
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

	var first bool = false
	var lpFee, lpForPool, lpForUser *decimal.Decimal
	if pool.TickBalance[0].Sign() == 0 && pool.TickBalance[1].Sign() == 0 {
		first = true
		lpForPool = token0Amt.Mul(token1Amt).Sqrt()
		if lpForPool.Cmp(decimal.NewDecimal(1000, 18)) < 0 {
			return errors.New("addLiq: lp less than 1000")
		}
		lpForUser = lpForPool.Sub(decimal.NewDecimal(1000, 18))

	} else {
		// Issuing additional LP, as a way of collecting service fees.
		feeRateSwapAmt, ok := CheckAmountVerify(moduleInfo.FeeRateSwapMut, 3)
		if !ok {
			log.Printf("pool addLiq FeeRateSwap invalid: %s", moduleInfo.FeeRateSwapMut)
			return errors.New("addLiq: feerate swap invalid")
		}
		poolLpBalance := pool.LpBalance
		if feeRateSwapAmt.Sign() > 0 {
			// lp = (poolLp * (rootK - rootKLast)) / (rootK * 5 + rootKLast)
			rootK := pool.TickBalance[token0Idx].Mul(pool.TickBalance[token1Idx]).Sqrt()

			lpFee = poolLpBalance.Mul(rootK.Sub(pool.LastRootK)).Div(
				rootK.Mul(decimal.NewDecimal(5, 0)).Add(pool.LastRootK))

			// log.Printf("pool addliq issue lp: %s", lpFee.String())
			if lpFee.Sign() > 0 {
				// pool lp update
				poolLpBalance = poolLpBalance.Add(lpFee)
			}
		}

		// Calculate the amount of liquidity tokens acquired
		token1AdjustAmt := pool.TickBalance[token1Idx].Mul(token0Amt).Div(pool.TickBalance[token0Idx])
		if token1Amt.Cmp(token1AdjustAmt) >= 0 {
			token1Amt = token1AdjustAmt
		} else {
			token0AdjustAmt := pool.TickBalance[token0Idx].Mul(token1Amt).Div(pool.TickBalance[token1Idx])
			token0Amt = token0AdjustAmt
		}

		lp0 := poolLpBalance.Mul(token0Amt).Div(pool.TickBalance[token0Idx])
		lp1 := poolLpBalance.Mul(token1Amt).Div(pool.TickBalance[token1Idx])
		if lp0.Cmp(lp1) > 0 {
			lpForPool = lp1
		} else {
			lpForPool = lp0
		}
		lpForUser = lpForPool
	}

	if lpForUser.Cmp(tokenLpAmt.Mul(decimal.NewDecimal(1000, 3).Sub(slippageAmt)).Div(decimal.NewDecimal(1000, 3))) < 0 {
		log.Printf("user[%s], lp: %s < expect: %s. * %s", f.Address, lpForUser, tokenLpAmt, tokenLpAmt.Sub(tokenLpAmt.Mul(slippageAmt)))
		return errors.New("addLiq: over slippage")
	}

	// User Balance Check
	token0Balance := moduleInfo.GetUserTokenBalance(token0, f.PkScript)
	token1Balance := moduleInfo.GetUserTokenBalance(token1, f.PkScript)

	token0AmtUint128 := token0Amt.Uint128()
	// fixme: Must use the confirmed amount
	if token0Balance.SwapAccountBalance.Cmp(token0AmtUint128) < 0 {
		log.Printf("token0[%s] user[%s], balance %s", token0, f.Address, token0Balance)
		return errors.New("addLiq: token0 balance insufficient")
	}
	token1AmtUint128 := token1Amt.Uint128()
	// fixme: Must use the confirmed amount
	if token1Balance.SwapAccountBalance.Cmp(token1AmtUint128) < 0 {
		log.Printf("token1[%s] user[%s], balance %s", token1, f.Address, token1Balance)
		return errors.New("addLiq: token1 balance insufficient")
	}

	// User Real-time Balance Update
	token0Balance.SwapAccountBalance = token0Balance.SwapAccountBalance.Sub(token0AmtUint128)
	token1Balance.SwapAccountBalance = token1Balance.SwapAccountBalance.Sub(token1AmtUint128)
	// User safety balance update
	token0Balance.SwapAccountBalanceSafe = token0Balance.SwapAccountBalanceSafe.Sub(token0AmtUint128)
	token1Balance.SwapAccountBalanceSafe = token1Balance.SwapAccountBalanceSafe.Sub(token1AmtUint128)

	// lp balance update
	// lp-user-balance
	lpbalance := moduleInfo.GetBRC20ModuleLPTokenBalanceByUser(f.PkScript, poolPair)
	lpbalance.Balance = lpbalance.Balance.Add(lpForUser)

	// zero address lp balance update
	if first {
		zerolpbalance := moduleInfo.GetBRC20ModuleLPTokenBalanceByUser(constant.ZERO_ADDRESS_PKSCRIPT, poolPair)
		zerolpbalance.Balance = zerolpbalance.Balance.Add(decimal.NewDecimal(1000, 18))
	}

	// Changes in pool balance
	pool.TickBalance[token0Idx] = pool.TickBalance[token0Idx].Add(token0Amt)
	pool.TickBalance[token1Idx] = pool.TickBalance[token1Idx].Add(token1Amt)
	pool.LpBalance = pool.LpBalance.Add(lpForPool)

	if lpFee.Sign() > 0 {
		// pool lp update
		pool.LpBalance = pool.LpBalance.Add(lpFee)

		// lpFee lp balance update
		lpFeelpbalance := moduleInfo.GetBRC20ModuleLPTokenBalanceByUser(moduleInfo.LpFeePkScript, poolPair)
		lpFeelpbalance.Balance = lpFeelpbalance.Balance.Add(lpFee)
	}

	// update lastRootK
	pool.LastRootK = pool.TickBalance[token0Idx].Mul(pool.TickBalance[token1Idx]).Sqrt()

	// log.Printf("[%s] pool after addliq [%s] %s: %s, %s: %s, lp: %s", moduleInfo.ID, poolPair, pool.Tick[0], pool.TickBalance[0], pool.Tick[1], pool.TickBalance[1], pool.LpBalance)
	return nil
}
