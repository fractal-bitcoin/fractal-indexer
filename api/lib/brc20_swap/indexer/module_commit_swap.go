package indexer

import (
	"errors"
	"fmt"
	"log"

	"github.com/unisat-wallet/libbrc20-indexer/decimal"
	"github.com/unisat-wallet/libbrc20-indexer/model"
)

// ProcessCommitFunctionSwap
// exactIn:
//
//	amountInWithFee = amountIn * 997
//	amountOut = (amountInWithFee * reserveOut)/(reverseIn * 1000 + amountInWithFee)
//
// exactOut:
//
//	amountIn = (reserveIn * amountOut * 1000)/((reserveOut - amountOut) * 997) + 1
func (g *BRC20ModuleIndexer) ProcessCommitFunctionSwap(moduleInfo *model.BRC20ModuleSwapInfo, f *model.SwapFunctionData) (err error) {
	token0, token1 := f.Params[0], f.Params[1]
	poolPair := GetLowerInnerPairNameByToken(token0, token1)

	pool, ok := moduleInfo.SwapPoolTotalBalanceDataMap[poolPair]
	if !ok {
		return errors.New("swap: pool invalid")
	}

	if token0 != pool.Tick[0] && token0 != pool.Tick[1] {
		return errors.New("swap: token invalid")
	}
	if token1 != pool.Tick[0] && token1 != pool.Tick[1] {
		return errors.New("swap: token invalid")
	}

	// log.Printf("[%s] pool before swap [%s] %s: %s, %s: %s, lp: %s", moduleInfo.ID, poolPair, pool.Tick[0], pool.TickBalance[0], pool.Tick[1], pool.TickBalance[1], pool.LpBalance)
	// log.Printf("pool swap params: %v", f.Params)

	var tokenIn, tokenInAmtStr, tokenOut, tokenOutAmtStr string
	derection := f.Params[4]
	if derection == "exactIn" {
		tokenIn = f.Params[2]
		tokenInAmtStr = f.Params[3]

		if tokenIn == token0 {
			tokenOut = token1
		} else {
			tokenOut = token0
		}
		tokenOutAmtStr = f.Params[5]
	} else if derection == "exactOut" {
		tokenOut = f.Params[2]
		tokenOutAmtStr = f.Params[3]

		if tokenOut == token0 {
			tokenIn = token1
		} else {
			tokenIn = token0
		}
		tokenInAmtStr = f.Params[5]
	}

	tokenInAmt, ok := g.CheckTickVerifyDecimal(tokenIn, tokenInAmtStr)
	if !ok {
		return errors.New("swap: tokenIn amount invalid")
	}
	tokenOutAmt, ok := g.CheckTickVerifyDecimal(tokenOut, tokenOutAmtStr)
	if !ok {
		return errors.New("swap: tokenIn amount invalid")
	}

	// Confirm the token order id of the pool
	var tokenInIdx, tokenOutIdx int
	if tokenIn == pool.Tick[0] {
		tokenInIdx = 0
		tokenOutIdx = 1
	} else {
		tokenInIdx = 1
		tokenOutIdx = 0
	}

	// Please note that should use integer calculations here.
	// support exactIn
	// Slippage check
	// exactIn:  1/(1+slippage) * quoteAmount
	// exactOut:  (1+slippage) * quoteAmount
	slippageAmtStr := f.Params[6]
	slippageAmt, _ := decimal.FromString(slippageAmtStr, 3)

	feeRateSwapAmt, ok := CheckAmountVerify(moduleInfo.FeeRateSwapMut, 3)
	if !ok {
		log.Printf("pool swap FeeRateSwap invalid: %s", moduleInfo.FeeRateSwapMut)
		return errors.New("swap: feerate swap invalid")
	}

	var amountIn, amountOut *decimal.Decimal
	if derection == "exactIn" {
		if feeRateSwapAmt.Sign() > 0 {
			// with fee
			amountInWithFee := tokenInAmt.Mul(decimal.NewDecimal(1000, 3).Sub(feeRateSwapAmt))
			amountOut = pool.TickBalance[tokenOutIdx].Mul(amountInWithFee).Div(
				pool.TickBalance[tokenInIdx].Mul(decimal.NewDecimal(1000, 3)).Add(amountInWithFee))
		} else {
			amountOut = pool.TickBalance[tokenOutIdx].Mul(tokenInAmt).Div(
				pool.TickBalance[tokenInIdx].Add(tokenInAmt))
		}

		amountOutMin := tokenOutAmt.Mul(decimal.NewDecimal(1000, 3)).Div(decimal.NewDecimal(1000, 3).Add(slippageAmt))
		if amountOut.Cmp(amountOutMin) < 0 {
			log.Printf("user[%s], amountOut: %s < expect: %s", f.Address, amountOut, amountOutMin)
			return errors.New("swap: slippage error")
		}
		amountIn = tokenInAmt

	} else if derection == "exactOut" {
		if feeRateSwapAmt.Sign() > 0 {
			// with fee
			amountIn = pool.TickBalance[tokenInIdx].Mul(tokenOutAmt.Mul(decimal.NewDecimal(1000, 3))).Div(
				pool.TickBalance[tokenOutIdx].Sub(tokenOutAmt).Mul(decimal.NewDecimal(1000, 3).Sub(feeRateSwapAmt))).Add(
				decimal.NewDecimal(1, tokenInAmt.Precision))
		} else {
			amountIn = pool.TickBalance[tokenInIdx].Mul(tokenOutAmt).Div(
				pool.TickBalance[tokenOutIdx].Sub(tokenOutAmt)).Add(
				decimal.NewDecimal(1, tokenInAmt.Precision))
		}
		amountInMax := tokenInAmt.Mul(decimal.NewDecimal(1000, 3).Add(slippageAmt))
		if amountInMax.Cmp(amountIn) < 0 {
			log.Printf("user[%s], amountIn: %s > expect: %s", f.Address, amountIn, amountInMax)
			return errors.New("swap: slippage error")
		}
		amountOut = tokenOutAmt
	}

	// Check the balance range, prepare to update.
	if pool.TickBalance[tokenOutIdx].Cmp(amountOut) < 0 {
		return errors.New("swap: pool tokenOut balance insufficient")
	}

	tokenInBalance := moduleInfo.GetUserTokenBalance(tokenIn, f.PkScript)
	tokenOutBalance := moduleInfo.GetUserTokenBalance(tokenOut, f.PkScript)

	amountInUint128 := amountIn.Uint128()
	amountOutUint128 := amountOut.Uint128()

	if tokenInBalance.SwapAccountBalance.Cmp(amountInUint128) < 0 {
		return errors.New(fmt.Sprintf("swap[%s]: user tokenIn balance insufficient: %s < %s",
			f.ID,
			tokenInBalance.SwapAccountBalance, tokenInAmt))
	}

	// update balance
	// swap sub
	pool.TickBalance[tokenOutIdx] = pool.TickBalance[tokenOutIdx].Sub(amountOut)
	tokenInBalance.SwapAccountBalance = tokenInBalance.SwapAccountBalance.Sub(amountInUint128)
	tokenInBalance.SwapAccountBalanceSafe = tokenInBalance.SwapAccountBalanceSafe.Sub(amountInUint128)
	// swap add
	pool.TickBalance[tokenInIdx] = pool.TickBalance[tokenInIdx].Add(amountIn)
	tokenOutBalance.SwapAccountBalance = tokenOutBalance.SwapAccountBalance.Add(amountOutUint128)
	tokenOutBalance.SwapAccountBalanceSafe = tokenOutBalance.SwapAccountBalanceSafe.Add(amountOutUint128)

	// log.Printf("[%s] pool after swap [%s] %s: %s, %s: %s, lp: %s", moduleInfo.ID, poolPair, pool.Tick[0], pool.TickBalance[0], pool.Tick[1], pool.TickBalance[1], pool.LpBalance)
	return nil
}
