package indexer

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"fractal-indexer/api/lib/brc20_swap/conf"
	"fractal-indexer/api/lib/brc20_swap/constant"
	"fractal-indexer/api/lib/brc20_swap/model"
	"fractal-indexer/api/lib/brc20_swap/utils"
)

var GResultsExternal []*model.SwapFunctionResultCheckState

func (g *BRC20ModuleIndexer) BRC20ResultsPreVerify(moduleInfo *model.BRC20ModuleSwapInfo, result *model.SwapFunctionResultCheckState) (err error) {
	// check user amt format
	for idxUser, user := range result.Users {
		userPkScript := constant.ZERO_ADDRESS_PKSCRIPT
		// format check
		if user.Address != "0" {
			if pk, err := utils.GetPkScriptByAddress(user.Address, conf.GlobalNetParams); err != nil {
				return errors.New(fmt.Sprintf("result users[%d] addr(%s) invalid", idxUser, user.Address))
			} else {
				userPkScript = string(pk)
			}
		}
		if strings.Index(user.Tick, "/") == -1 {
			tokenAmt, ok := g.CheckTickVerify(user.Tick, user.Balance)
			if !ok {
				return errors.New(fmt.Sprintf("result users[%d] balance invalid", idxUser))
			}

			// balance check
			tokenBalance := moduleInfo.GetUserTokenBalance(user.Tick, userPkScript)
			if tokenBalance.SwapAccountBalanceSafe.Cmp(tokenAmt) != 0 {
				return errors.New(fmt.Sprintf("result users[%d] %s amount not match (%s != %s)",
					idxUser, user.Tick,
					tokenAmt.String(),
					tokenBalance.SwapAccountBalanceSafe.String(),
				))
			}

		} else {
			token0, token1, err := utils.DecodeTokensFromSwapPair(user.Tick)
			if err != nil {
				return errors.New(fmt.Sprintf("result users[%d] tick invalid", idxUser))
			}

			if _, ok := g.CheckTickVerify(token0, ""); !ok {
				return errors.New(fmt.Sprintf("result users[%d] tick/0 invalid", idxUser))
			}

			if _, ok := g.CheckTickVerify(token1, ""); !ok {
				return errors.New(fmt.Sprintf("result users[%d] tick/1 invalid", idxUser))
			}
			lpAmt, ok := CheckAmountVerify(user.Balance, 18)
			if !ok {
				return errors.New(fmt.Sprintf("result users[%d] Lp Amount invalid", idxUser))
			}
			if user.LockedBalance == "" {
				user.LockedBalance = "0"
			}
			lpAmtLocked, ok := CheckAmountVerify(user.LockedBalance, 18)
			if !ok {
				return errors.New(fmt.Sprintf("result users[%d] Locked Lp Amount invalid", idxUser))
			}

			// balance check
			poolPair := GetLowerInnerPairNameByToken(token0, token1)
			if _, ok := moduleInfo.LPTokenUsersBalanceMap[poolPair]; !ok {
				return errors.New(fmt.Sprintf("result users[%d] Pair invalid", idxUser))
			}

			lpBalance := moduleInfo.GetBRC20ModuleLPTokenBalanceByUser(userPkScript, poolPair)
			if lpBalance.Balance.Cmp(lpAmt) != 0 {
				return errors.New(fmt.Sprintf("result users[%d] %s lp balance not match", idxUser, poolPair))
			}
			if lpBalance.LockedBalance.Cmp(lpAmtLocked) != 0 {
				return errors.New(fmt.Sprintf("result users[%d] %s lp balance not match", idxUser, poolPair))
			}
		}
	}

	// check pool amt format
	for idxPool, poolResult := range result.Pools {
		// format check
		token0, token1, err := utils.DecodeTokensFromSwapPair(poolResult.Pair)
		if err != nil {
			return errors.New(fmt.Sprintf("result pools[%d] Pair invalid", idxPool))
		}
		token0Amt, ok := g.CheckTickVerifyDecimal(token0, poolResult.ReserveAmount0)
		if !ok {
			return errors.New(fmt.Sprintf("result pools[%d] Amount0 invalid", idxPool))
		}

		token1Amt, ok := g.CheckTickVerifyDecimal(token1, poolResult.ReserveAmount1)
		if !ok {
			return errors.New(fmt.Sprintf("result pools[%d] Amount1 invalid", idxPool))
		}

		lpAmt, ok := CheckAmountVerify(poolResult.LPAmount, 18)
		if !ok {
			return errors.New(fmt.Sprintf("result pools[%d] Lp Amount invalid", idxPool))
		}

		// balance check
		poolPair := GetLowerInnerPairNameByToken(token0, token1)
		pool, ok := moduleInfo.SwapPoolTotalBalanceDataMap[poolPair]
		if !ok {
			return errors.New(fmt.Sprintf("result pools[%d] missing pair[%s]", idxPool, poolPair))
		}
		// Determine the token order id of the pool
		var token0Idx, token1Idx int
		if token0 == pool.Tick[0] {
			token0Idx = 0
			token1Idx = 1
		} else {
			token0Idx = 1
			token1Idx = 0
		}

		if token0Amt.Cmp(pool.TickBalance[token0Idx]) != 0 {
			return errors.New(fmt.Sprintf("result pool[%d] %s balance not match", idxPool, pool.Tick[token0Idx]))
		}

		if token1Amt.Cmp(pool.TickBalance[token1Idx]) != 0 {
			return errors.New(fmt.Sprintf("result pool[%d] %s balance not match", idxPool, pool.Tick[token1Idx]))
		}

		lpAmt.Precision = 18
		if lpAmt.Cmp(pool.LpBalance) != 0 {
			return errors.New(fmt.Sprintf("result pool[%d] %s lpbalance not match", idxPool, poolPair))
		}
	}

	return nil
}

// ProcessInscribeCommit Created a commit, but it has not yet taken effect.
func (g *BRC20ModuleIndexer) ProcessInscribeCommitPreVerify(body *model.InscriptionBRC20ModuleSwapCommitContent) (index int, err error) {
	if body.Module != strings.ToLower(body.Module) {
		return -1, errors.New("module id invalid")
	}

	// check module exist
	moduleInfo, ok := g.ModulesInfoMap[body.Module]
	if !ok {
		return -1, errors.New("module invalid")
	}

	// check gasPrice
	if _, ok := g.CheckTickVerify(moduleInfo.GasTick, body.GasPrice); !ok {
		log.Printf("ProcessInscribeCommit commit gas err: %s", body.GasPrice)
		return -1, errors.New("gas price invalid")
	}

	// common content
	content := fmt.Sprintf("module: %s\n", moduleInfo.ID)
	if body.Parent != "" {
		content += fmt.Sprintf("parent: %s\n", body.Parent)
	}
	if body.GasPrice != "" {
		content += fmt.Sprintf("gas_price: %s\n", body.GasPrice)
	}

	// for previous id
	functionsByAddressMap := make(map[string][]string)
	for idx, f := range body.Data {
		if pkScript, err := utils.GetPkScriptByAddress(f.Address, conf.GlobalNetParams); err != nil {
			return idx, errors.New("addr invalid")
		} else {
			f.PkScript = string(pkScript)
		}

		// log.Printf("ProcessInscribeCommitPreVerify func[%d] %s(%s)", idx, f.Function, strings.Join(f.Params, ", "))

		// get prevouse function id by user
		previous := functionsByAddressMap[f.Address]
		if id, ok := CheckFunctionSigVerify(content, f, previous); !ok {
			return idx, errors.New(fmt.Sprintf("function[%d]%s sig invalid", idx, id))
		} else {
			// update previous id list
			f.ID = id
			previous = append(previous, id)
			functionsByAddressMap[f.Address] = previous
		}

		// function process
		if f.Function == constant.BRC20_SWAP_FUNCTION_DEPLOY_POOL {
			if len(f.Params) != 2 {
				return idx, errors.New("func: deploy params invalid")
			}
			token0 := f.Params[0]
			token1 := f.Params[1]
			if token0 == token1 {
				return idx, errors.New("func: deploy same tokens")
			}
			if _, err := utils.GetValidUniqueLowerTickerTicker(token0); err != nil {
				return idx, errors.New("func: deploy token0 invalid")
			}
			if _, err := utils.GetValidUniqueLowerTickerTicker(token1); err != nil {
				return idx, errors.New("func: deploy token0 invalid")
			}
			// Check for duplicate pairs when the Commit inscription effect is applied.

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_ADD_LIQ {
			if len(f.Params) != 6 {
				return idx, errors.New("func: addLiq params invalid")
			}

			token0, token1 := f.Params[0], f.Params[1]
			// don't check ticker amount
			// token0AmtStr := f.Params[2]
			// token1AmtStr := f.Params[3]
			tokenLpAmtStr := f.Params[4]
			slippage := f.Params[5]

			if token0 == token1 {
				return idx, errors.New("func: addLiq same tokens")
			}
			if _, err := utils.GetValidUniqueLowerTickerTicker(token0); err != nil {
				return idx, errors.New("func: addLiq token0 invalid")
			}
			if _, err := utils.GetValidUniqueLowerTickerTicker(token1); err != nil {
				return idx, errors.New("func: addLiq token0 invalid")
			}

			if _, ok := CheckAmountVerify(tokenLpAmtStr, 18); !ok {
				return idx, errors.New("func: addLiq amtLp invalid")
			}

			if _, ok := CheckAmountVerify(slippage, 3); !ok {
				return idx, errors.New("func: addLiq slippage invalid")
			}

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_REMOVE_LIQ {
			if len(f.Params) != 6 {
				return idx, errors.New("func: removeLiq params invalid")
			}

			token0, token1 := f.Params[0], f.Params[1]
			tokenLpAmtStr := f.Params[2]
			// token0AmtStr := f.Params[3]
			// token1AmtStr := f.Params[4]
			slippage := f.Params[5]

			if token0 == token1 {
				return idx, errors.New("func: removeLiq same tokens")
			}
			if _, err := utils.GetValidUniqueLowerTickerTicker(token0); err != nil {
				return idx, errors.New("func: removeLiq token0 invalid")
			}
			if _, err := utils.GetValidUniqueLowerTickerTicker(token1); err != nil {
				return idx, errors.New("func: removeLiq token0 invalid")
			}

			if _, ok := CheckAmountVerify(tokenLpAmtStr, 18); !ok {
				return idx, errors.New("func: removeLiq amtLp invalid")
			}

			if _, ok := CheckAmountVerify(slippage, 3); !ok {
				return idx, errors.New("func: removeLiq slippage invalid")
			}

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_SWAP {
			if len(f.Params) != 7 {
				return idx, errors.New("func: swap params invalid")
			}

			token0, token1 := f.Params[0], f.Params[1]
			if token0 == token1 {
				return idx, errors.New("func: swap same tokens")
			}

			if _, err := utils.GetValidUniqueLowerTickerTicker(token0); err != nil {
				return idx, errors.New("func: swap token0 invalid")
			}
			if _, err := utils.GetValidUniqueLowerTickerTicker(token1); err != nil {
				return idx, errors.New("func: swap token0 invalid")
			}

			// Check that the first parameter must be one of the token pairs.
			if token := f.Params[2]; token != token0 && token != token1 {
				return idx, errors.New("func: swap token invalid")
			}

			derection := f.Params[4]
			if derection != "exactIn" && derection != "exactOut" {
				return idx, errors.New("func: swap derection invalid")
			}

			slippage := f.Params[6]
			if _, ok := CheckAmountVerify(slippage, 3); !ok {
				return idx, errors.New("func: swap slippage invalid")
			}

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_DECREASE_APPROVAL {
			if len(f.Params) != 2 {
				return idx, errors.New("func: decrease approval params invalid")
			}

			token := f.Params[0]
			if _, err := utils.GetValidUniqueLowerTickerTicker(token); err != nil {
				return idx, errors.New("func: decrease approval token invalid")
			}

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_SEND {
			if len(f.Params) != 3 {
				return idx, errors.New("func: send params invalid")
			}

			addressTo := f.Params[0]
			if _, err := utils.GetPkScriptByAddress(addressTo, conf.GlobalNetParams); err != nil {
				return idx, errors.New("send addr invalid")
			}

			token := f.Params[1]
			if _, err := utils.GetValidUniqueLowerTickerTicker(token); err != nil {
				return idx, errors.New("func: send token invalid")
			}

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_SENDLP {
			if len(f.Params) != 4 {
				return idx, errors.New("func: sendlp params invalid")
			}

			addressTo := f.Params[0]
			if _, err := utils.GetPkScriptByAddress(addressTo, conf.GlobalNetParams); err != nil {
				return idx, errors.New("send addr invalid")
			}

			token0, token1 := f.Params[1], f.Params[2]
			if token0 == token1 {
				return idx, errors.New("func: sendlp same tokens")
			}
			if _, err := utils.GetValidUniqueLowerTickerTicker(token0); err != nil {
				return idx, errors.New("func: sendlp token0 invalid")
			}
			if _, err := utils.GetValidUniqueLowerTickerTicker(token1); err != nil {
				return idx, errors.New("func: sendlp token1 invalid")
			}

			poolPair := fmt.Sprintf("%s/%s", token0, token1)
			if _, _, err := utils.DecodeTokensFromSwapPair(poolPair); err != nil {
				return idx, errors.New("func: sendlp poolPair invalid")
			}

			tokenAmtStr := f.Params[3]
			if _, ok := CheckAmountVerify(tokenAmtStr, 18); !ok {
				return idx, errors.New(fmt.Sprintf("func: sendlp amtLp invalid, %s", tokenAmtStr))
			}

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_LOCK || f.Function == constant.BRC20_SWAP_FUNCTION_UNLOCK {
			if len(f.Params) != 3 {
				return idx, errors.New("func: lock/unlock params invalid")
			}

			token0, token1 := f.Params[0], f.Params[1]
			if token0 == token1 {
				return idx, errors.New("func: lock/unlock same tokens")
			}
			if _, err := utils.GetValidUniqueLowerTickerTicker(token0); err != nil {
				return idx, errors.New("func: lock/unlock token0 invalid")
			}
			if _, err := utils.GetValidUniqueLowerTickerTicker(token1); err != nil {
				return idx, errors.New("func: lock/unlock token1 invalid")
			}

			poolPair := fmt.Sprintf("%s/%s", token0, token1)
			if _, _, err := utils.DecodeTokensFromSwapPair(poolPair); err != nil {
				return idx, errors.New("func: lock/unlock pair invalid")
			}
			tokenAmtStr := f.Params[2]
			if _, ok := CheckAmountVerify(tokenAmtStr, 18); !ok {
				return idx, errors.New(fmt.Sprintf("func: lock/unlock amtLp invalid, %s", tokenAmtStr))
			}

		} else {
			log.Printf("ProcessInscribeCommit commit[%d] invalid function: %s. id: %s", idx, f.Function, f.ID)
			return idx, errors.New("func invalid")
		}
	}

	return 0, nil
}

func (g *BRC20ModuleIndexer) ProcessCommitVerify(commitId string, body *model.InscriptionBRC20ModuleSwapCommitContent,
	results []*model.SwapFunctionResultCheckState) (index int, critical bool, err error) {

	// check module exist
	moduleInfo, ok := g.ModulesInfoMap[body.Module]
	if !ok {
		return -1, true, errors.New("commit, module not exist")
	}

	// check empty parent
	if body.Parent == "" {
		if len(moduleInfo.CommitIdMap) > 0 {
			return -1, true, errors.New("commit, missing parent")
		}
	} else {
		// invalid if reusing 'parent'
		if _, ok := moduleInfo.CommitIdChainMap[body.Parent]; ok {
			return -1, true, errors.New("commit, parent already sattled")
		}

		// invalid if parent commit not exist
		if _, ok := moduleInfo.CommitIdMap[body.Parent]; !ok {
			return -1, true, errors.New("commit, parent invalid")
		}
	}

	gasPriceAmt, _ := g.CheckTickVerify(moduleInfo.GasTick, body.GasPrice)

	moduleInfo.FeeRateSwapMut = moduleInfo.FeeRateSwap
	if body.FeeRateSwap != "" {
		moduleInfo.FeeRateSwapMut = body.FeeRateSwap
	}

	var errFunc error
	firstInvalidIdx := 0
	for idx, f := range body.Data {
		if pkScript, err := utils.GetPkScriptByAddress(f.Address, conf.GlobalNetParams); err != nil {
			return idx, true, errors.New("commit, addr invalid")
		} else {
			f.PkScript = string(pkScript)
		}

		process, _ := g.GetCommitFunctionProcess(f.Function)

		// gas fee
		if gasPriceAmt.Sign() > 0 {
			gasAmt := gasPriceAmt
			// log.Printf("process commit[%d] size: %d, gas fee: %s, module[%s]", idx, size, gasAmt.String(), body.Module)
			if err := g.ProcessCommitFunctionGasFee(moduleInfo, f.PkScript, gasAmt); err != nil { // has update
				log.Printf("process commit[%d] gas failed: %s", idx, err)
				if errFunc == nil {
					errFunc = err
					firstInvalidIdx = idx
				}
				f.Invalid = true
				continue
			}
		}

		if err := process(moduleInfo, f); err != nil {
			log.Printf("process commit[%d] %s failed: %s, module[%s]", idx, f.Function, err, body.Module)
			if errFunc == nil {
				errFunc = err
				firstInvalidIdx = idx
			}
			f.Invalid = true
		}

		// instant verify
		if len(results) == len(body.Data) {
			if err = g.BRC20ResultsPreVerify(moduleInfo, results[idx]); err != nil {
				log.Printf("commit verify failed: result[%d] %s", idx, err)
				return idx, false, err
			}
		}

		// verify test result
		if GResultsExternal == nil {
			continue
		}
		for _, result := range GResultsExternal {
			if result.CommitId != commitId {
				continue
			}

			if result.FunctionIdx != idx {
				continue
			}

			if err = g.BRC20ResultsPreVerify(moduleInfo, result); err != nil {
				log.Printf("commit verify failed: result[%d] %s", idx, err)
				return idx, false, err
			}
		}
	}
	if errFunc != nil {
		return firstInvalidIdx, false, errFunc
	}

	return 0, false, nil
}

func (g *BRC20ModuleIndexer) InitCherryPickFilter(body *model.InscriptionBRC20ModuleSwapCommitContent, pickUsersPkScript, pickTokensTick, pickPoolsPair map[string]bool) (index int, err error) {
	// check module exist
	moduleInfo, ok := g.ModulesInfoMap[body.Module]
	if !ok {
		return -1, errors.New("module invalid")
	}

	pickUsersPkScript[string(moduleInfo.GasToPkScript)] = true
	pickUsersPkScript[string(moduleInfo.LpFeePkScript)] = true
	pickUsersPkScript[string(moduleInfo.SequencerPkScript)] = true
	pickUsersPkScript[string(moduleInfo.DeployerPkScript)] = true

	pickTokensTick[moduleInfo.GasTick] = true

	for idx, f := range body.Data {
		if pkScript, err := utils.GetPkScriptByAddress(f.Address, conf.GlobalNetParams); err != nil {
			return idx, errors.New("addr invalid")
		} else {
			pickUsersPkScript[string(pkScript)] = true
		}

		// function process
		if f.Function == constant.BRC20_SWAP_FUNCTION_DEPLOY_POOL {
			if len(f.Params) != 2 {
				return idx, errors.New("func: deploy params invalid")
			}
			token0 := f.Params[0]
			token1 := f.Params[1]

			// pair
			poolPair := GetLowerInnerPairNameByToken(token0, token1)
			pickPoolsPair[poolPair] = true

			// tick
			token0 = strings.ToLower(token0)
			token1 = strings.ToLower(token1)
			pickTokensTick[token0] = true
			pickTokensTick[token1] = true

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_ADD_LIQ {
			if len(f.Params) != 6 {
				return idx, errors.New("func: addLiq params invalid")
			}

			token0, token1 := f.Params[0], f.Params[1]

			// pair
			poolPair := GetLowerInnerPairNameByToken(token0, token1)
			pickPoolsPair[poolPair] = true

			// tick
			token0 = strings.ToLower(token0)
			token1 = strings.ToLower(token1)
			pickTokensTick[token0] = true
			pickTokensTick[token1] = true

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_REMOVE_LIQ {
			if len(f.Params) != 6 {
				return idx, errors.New("func: removeLiq params invalid")
			}

			token0, token1 := f.Params[0], f.Params[1]

			// pair
			poolPair := GetLowerInnerPairNameByToken(token0, token1)
			pickPoolsPair[poolPair] = true

			// tick
			token0 = strings.ToLower(token0)
			token1 = strings.ToLower(token1)
			pickTokensTick[token0] = true
			pickTokensTick[token1] = true

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_SWAP {
			if len(f.Params) != 7 {
				return idx, errors.New("func: swap params invalid")
			}

			token0, token1 := f.Params[0], f.Params[1]

			// pair
			poolPair := GetLowerInnerPairNameByToken(token0, token1)
			pickPoolsPair[poolPair] = true

			// tick
			token0 = strings.ToLower(token0)
			token1 = strings.ToLower(token1)
			pickTokensTick[token0] = true
			pickTokensTick[token1] = true

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_DECREASE_APPROVAL {
			if len(f.Params) != 2 {
				return idx, errors.New("func: decrease approval params invalid")
			}

			token0 := f.Params[0]
			token0 = strings.ToLower(token0)
			pickTokensTick[token0] = true

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_SEND {
			if len(f.Params) != 3 {
				return idx, errors.New("func: send params invalid")
			}

			addressTo := f.Params[0]
			if pk, err := utils.GetPkScriptByAddress(addressTo, conf.GlobalNetParams); err != nil {
				return idx, errors.New("send addr invalid")
			} else {
				pickUsersPkScript[string(pk)] = true
			}

			token0 := f.Params[1]
			token0 = strings.ToLower(token0)
			pickTokensTick[token0] = true

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_SENDLP {
			if len(f.Params) != 4 {
				return idx, errors.New("func: send params invalid")
			}

			addressTo := f.Params[0]
			if pk, err := utils.GetPkScriptByAddress(addressTo, conf.GlobalNetParams); err != nil {
				return idx, errors.New("send addr invalid")
			} else {
				pickUsersPkScript[string(pk)] = true
			}

			token0, token1 := f.Params[1], f.Params[2]
			poolPair := GetLowerInnerPairNameByToken(token0, token1)
			pickPoolsPair[poolPair] = true

			// tick
			token0 = strings.ToLower(token0)
			token1 = strings.ToLower(token1)
			pickTokensTick[token0] = true
			pickTokensTick[token1] = true

		} else if f.Function == constant.BRC20_SWAP_FUNCTION_LOCK || f.Function == constant.BRC20_SWAP_FUNCTION_UNLOCK {
			if len(f.Params) != 3 {
				return idx, errors.New("func: lock/unlock params invalid")
			}

			token0, token1 := f.Params[0], f.Params[1]
			poolPair := GetLowerInnerPairNameByToken(token0, token1)
			pickPoolsPair[poolPair] = true

			// tick
			token0 = strings.ToLower(token0)
			token1 = strings.ToLower(token1)
			pickTokensTick[token0] = true
			pickTokensTick[token1] = true

		} else {
			log.Printf("ProcessInscribeCommit commit[%d] invalid function: %s. id: %s", idx, f.Function, f.ID)
			return idx, errors.New("func invalid")
		}
	}

	return 0, nil
}
