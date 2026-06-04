package indexer

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"fractal-indexer/api/lib/brc20_swap/conf"
	"fractal-indexer/api/lib/brc20_swap/constant"
	"fractal-indexer/api/lib/brc20_swap/model"
	"fractal-indexer/api/lib/brc20_swap/uint128"
	"fractal-indexer/api/lib/brc20_swap/utils"
)

func (g *BRC20ModuleIndexer) GetWithdrawInfoByKey(createIdxKey uint64) (
	withdrawInfo *model.InscriptionBRC20SwapInfo) {
	var ok bool
	// withdraw
	withdrawInfo, ok = g.InscriptionsWithdrawMap[createIdxKey]
	if !ok {
		withdrawInfo = nil
	}

	return withdrawInfo
}

func (g *BRC20ModuleIndexer) ProcessWithdraw(data *model.InscriptionBRC20Data, withdrawInfo *model.InscriptionBRC20SwapInfo) error {
	// ticker
	uniqueLowerTicker := strings.ToLower(withdrawInfo.Tick)
	tokenInfo, ok := g.InscriptionsTickerInfoMap[uniqueLowerTicker]
	if !ok {
		log.Printf("ProcessWithdraw send withdraw, but ticker invalid. txid: %s",
			utils.HashString([]byte(data.TxId)),
		)
		return errors.New("transfer, invalid ticker")
	}

	moduleInfo, ok := g.ModulesInfoMap[withdrawInfo.Module]
	if !ok {
		log.Printf("ProcessBRC20Withdraw send withdraw, but ticker invalid. txid: %s",
			hex.EncodeToString(utils.ReverseBytes([]byte(data.TxId))),
		)
		return errors.New("withdraw, module invalid")
	}

	var isInvalid bool

	// History from the ticker perspective; BRC-20 level.
	if g.EnableHistory {
		brc20TickInfo := model.NewInscriptionBRC20TickInfo(withdrawInfo.Tick, constant.BRC20_OP_MODULE_WITHDRAW, withdrawInfo.Data)
		brc20TickInfo.Amount = withdrawInfo.Amount
		historyObj := model.NewBRC20History(constant.BRC20_HISTORY_MODULE_TYPE_N_WITHDRAW, !isInvalid, true, brc20TickInfo, nil, data)
		history := g.UpdateHistoryHeightAndGetHistoryIndex(historyObj)

		if !conf.PruneHistoryList {
			tokenInfo.History = append(tokenInfo.History, history)
			tokenInfo.HistoryWithdraw = append(tokenInfo.HistoryWithdraw, history)
		}
	}

	// from
	// get user's tokens to update
	fromUserTokens, ok := moduleInfo.UsersTokenBalanceDataMap[string(withdrawInfo.Data.PkScript)]
	if !ok {
		log.Printf("ProcessBRC20Withdraw send from user missing. height: %d, txidx: %d",
			data.Height,
			data.TxIdx,
		)
		return errors.New("withdraw, send from user missing")
	}
	// get tokenBalance to update
	fromTokenBalance, ok := fromUserTokens[uniqueLowerTicker]
	if !ok {
		log.Printf("ProcessBRC20Withdraw send from ticker missing. height: %d, txidx: %d",
			data.Height,
			data.TxIdx,
		)
		return errors.New("withdraw, send from ticker missing")
	}

	// Cross-check whether the withdraw-inscription exists.
	if _, ok := fromTokenBalance.ReadyToWithdrawMap[data.CreateIdxKey]; !ok {
		log.Printf("ProcessBRC20Withdraw send from withdraw missing(dup withdraw?). height: %d, txidx: %d",
			data.Height,
			data.TxIdx,
		)
		return errors.New("withdraw, send from withdraw missing(dup)")
	}

	// available > amt
	balanceWithdraw := withdrawInfo.Amount
	fromTokenBalance.ReadyToWithdrawAmount = fromTokenBalance.ReadyToWithdrawAmount.Sub(balanceWithdraw)
	delete(fromTokenBalance.ReadyToWithdrawMap, data.CreateIdxKey)

	if fromTokenBalance.AvailableBalance.Cmp(balanceWithdraw) < 0 { // invalid
		isInvalid = true
	}

	// to address
	receiverPkScript := string(data.PkScript)
	if data.Satoshi == 0 {
		receiverPkScript = string(withdrawInfo.Data.PkScript)
		data.PkScript = receiverPkScript
	}

	// global history
	historyData := &model.BRC20SwapHistoryWithdrawData{
		Tick:   withdrawInfo.Tick,
		Amount: withdrawInfo.Amount.String(),
	}
	history := model.NewBRC20ModuleHistory(true, constant.BRC20_HISTORY_MODULE_TYPE_N_WITHDRAW, withdrawInfo.Data, data, historyData, !isInvalid)
	moduleInfo.History = append(moduleInfo.History, history)
	if isInvalid {
		// Invalid withdraw history for the from address in the module; BRC-20 module level.
		fromHistory := model.NewBRC20ModuleHistory(true, constant.BRC20_HISTORY_MODULE_TYPE_N_WITHDRAW_FROM, withdrawInfo.Data, data, nil, false)
		fromTokenBalance.History = append(fromTokenBalance.History, fromHistory)
		return errors.New("withdraw, insufficient available balance")
	}
	g.AllModulesWithdrawHistory = append(g.AllModulesWithdrawHistory, history)

	// set from
	// The available balance here needs to be directly deducted and transferred to WithdrawableBalance.
	fromTokenBalance.AvailableBalanceSafe = fromTokenBalance.AvailableBalanceSafe.Sub(balanceWithdraw)
	fromTokenBalance.AvailableBalance = fromTokenBalance.AvailableBalance.Sub(balanceWithdraw)

	fromHistory := model.NewBRC20ModuleHistory(true, constant.BRC20_HISTORY_MODULE_TYPE_N_WITHDRAW_FROM, withdrawInfo.Data, data, nil, true)
	// Valid withdraw history for the from address in the module; BRC-20 module level.
	fromTokenBalance.History = append(fromTokenBalance.History, fromHistory)

	moduleScript, ok := utils.GetScriptFromModuleId(withdrawInfo.Module)
	if ok {
		// Update the module's own balance.
		fromTokenBalanceBrc20 := g.GetUserTokenBalance(withdrawInfo.Tick, string(moduleScript))
		if data.BlockTime > 0 {
			fromTokenBalanceBrc20.AvailableBalanceSafe = fromTokenBalanceBrc20.AvailableBalanceSafe.Sub(withdrawInfo.Amount)
		}
		fromTokenBalanceBrc20.AvailableBalance = fromTokenBalanceBrc20.AvailableBalance.Sub(withdrawInfo.Amount)
		fromTokenBalanceBrc20.Balance = fromTokenBalanceBrc20.OverallBalance().Float64()

		if g.EnableHistory {
			// Treat the module as an address as well. Users check the module balance to determine whether withdrawals are available, so record the module withdraw history; BRC-20 level.
			brc20TickInfo := model.NewInscriptionBRC20TickInfo(withdrawInfo.Tick, constant.BRC20_OP_MODULE_WITHDRAW, withdrawInfo.Data)
			brc20TickInfo.Amount = withdrawInfo.Amount
			historyObj := model.NewBRC20History(constant.BRC20_HISTORY_MODULE_TYPE_N_WITHDRAW, true, true, brc20TickInfo, fromTokenBalanceBrc20, data)
			toHistory := g.UpdateHistoryHeightAndGetHistoryIndex(historyObj)

			if !conf.PruneHistoryList {
				fromTokenBalanceBrc20.History = append(fromTokenBalanceBrc20.History, toHistory)
				fromTokenBalanceBrc20.HistoryWithdraw = append(fromTokenBalanceBrc20.HistoryWithdraw, toHistory)
				fromTokenBalanceBrc20.HistoryReceive = append(fromTokenBalanceBrc20.HistoryReceive, toHistory)

				userHistoryFrom := g.GetBRC20HistoryByUser(string(moduleScript))
				userHistoryFrom.History = append(userHistoryFrom.History, toHistory)
				userHistoryFrom.HistoryWithdraw = append(userHistoryFrom.HistoryWithdraw, toHistory)
				userHistoryFrom.HistoryReceive = append(userHistoryFrom.HistoryReceive, toHistory)
			}
		}
	}

	// to
	tokenBalance := g.GetUserTokenBalance(withdrawInfo.Tick, receiverPkScript)
	if data.BlockTime > 0 {
		tokenBalance.AvailableBalanceSafe = tokenBalance.AvailableBalanceSafe.Add(withdrawInfo.Amount)
	}
	tokenBalance.AvailableBalance = tokenBalance.AvailableBalance.Add(withdrawInfo.Amount)
	tokenBalance.Balance = tokenBalance.OverallBalance().Float64()

	// Withdraw history for the to address in the module; BRC-20 module level.
	{
		toModuleAddress := moduleInfo.UsersTokenBalanceDataMap[receiverPkScript]
		if toModuleAddressTokenBalance, ok := toModuleAddress[uniqueLowerTicker]; ok && toModuleAddressTokenBalance != nil {
			toHistory := model.NewBRC20ModuleHistory(true, constant.BRC20_HISTORY_MODULE_TYPE_N_WITHDRAW_TO, withdrawInfo.Data, data, nil, true)
			toModuleAddressTokenBalance.History = append(toModuleAddressTokenBalance.History, toHistory)
		}
	}

	// burn
	if len(receiverPkScript) == 1 && []byte(receiverPkScript)[0] == 0x6a {
		tokenInfo.Deploy.Burned = tokenInfo.Deploy.Burned.Add(withdrawInfo.Amount)
	}

	if g.EnableHistory {
		brc20TickInfo := model.NewInscriptionBRC20TickInfo(withdrawInfo.Tick, constant.BRC20_OP_MODULE_WITHDRAW, withdrawInfo.Data)
		brc20TickInfo.Amount = withdrawInfo.Amount
		historyObj := model.NewBRC20History(constant.BRC20_HISTORY_MODULE_TYPE_N_WITHDRAW, true, true, brc20TickInfo, tokenBalance, data)
		toHistory := g.UpdateHistoryHeightAndGetHistoryIndex(historyObj)

		if !conf.PruneHistoryList {
			// Valid withdraw history for the address in the module; BRC-20 level.
			tokenBalance.History = append(tokenBalance.History, toHistory)
			tokenBalance.HistoryWithdraw = append(tokenBalance.HistoryWithdraw, toHistory)
			tokenBalance.HistoryReceive = append(tokenBalance.HistoryReceive, toHistory)

			userHistoryTo := g.GetBRC20HistoryByUser(receiverPkScript)
			userHistoryTo.History = append(userHistoryTo.History, toHistory)
			userHistoryTo.HistoryWithdraw = append(userHistoryTo.HistoryWithdraw, toHistory)
			userHistoryTo.HistoryReceive = append(userHistoryTo.HistoryReceive, toHistory)

			if receiverPkScript != string(withdrawInfo.Data.PkScript) && receiverPkScript != string(moduleScript) { // to != from
				userHistoryFrom := g.GetBRC20HistoryByUser(string(withdrawInfo.Data.PkScript))
				userHistoryFrom.History = append(userHistoryFrom.History, toHistory)
				userHistoryFrom.HistoryWithdraw = append(userHistoryFrom.HistoryWithdraw, toHistory)
				userHistoryFrom.HistoryReceive = append(userHistoryFrom.HistoryReceive, toHistory)
			}
		}
	}

	// fixme: add user module history
	// toHistory := model.NewBRC20ModuleHistory(true, constant.BRC20_HISTORY_MODULE_TYPE_N_WITHDRAW_TO, withdrawInfo.Data, data, nil, true)
	// tokenBalance.History = append(tokenBalance.History, toHistory)

	////////////////////////////////////////////////////////////////
	// withdraw to a module, is NOT deposit
	return nil
}

func (g *BRC20ModuleIndexer) ProcessInscribeWithdraw(data *model.InscriptionBRC20Data, latestHeight int) error {
	var body model.InscriptionBRC20ModuleWithdrawContent
	if err := json.Unmarshal(data.ContentBody, &body); err != nil {
		log.Printf("parse module withdraw json failed. txid: %s",
			hex.EncodeToString(utils.ReverseBytes([]byte(data.TxId))),
		)
		return err
	}

	// lower case moduleid only
	if body.Module != strings.ToLower(body.Module) {
		return errors.New("module id invalid")
	}

	moduleInfo, ok := g.ModulesInfoMap[body.Module]
	if !ok { // invalid module
		return errors.New("module invalid")
	}

	uniqueLowerTicker, err := utils.GetValidUniqueLowerTickerTicker(body.Tick)
	if err != nil {
		return errors.New("tick invalid")
	}

	tokenInfo, ok := g.InscriptionsTickerInfoMap[uniqueLowerTicker]
	if !ok {
		return errors.New("tick not exist")
	}
	tinfo := tokenInfo.Deploy

	amt, err := uint128.FromString(body.Amount, int(tinfo.Decimal))
	if err != nil {
		return errors.New(fmt.Sprintf("withdraw amount invalid: %s", body.Amount))
	}
	if amt.Sign() <= 0 || amt.Cmp(tinfo.Max) > 0 {
		return errors.New("amount out of range")
	}

	balanceWithdraw := amt

	// Unify ticker case
	body.Tick = tokenInfo.Ticker
	// Set up withdraw data for subsequent use.
	withdrawInfo := &model.InscriptionBRC20SwapInfo{
		Data: data,
	}
	withdrawInfo.Module = body.Module
	withdrawInfo.Tick = tokenInfo.Ticker
	withdrawInfo.Amount = balanceWithdraw

	// global history
	historyData := &model.BRC20SwapHistoryWithdrawData{
		Tick:   withdrawInfo.Tick,
		Amount: withdrawInfo.Amount.String(),
	}
	history := model.NewBRC20ModuleHistory(false, constant.BRC20_HISTORY_MODULE_TYPE_N_INSCRIBE_WITHDRAW, data, data, historyData, true)
	moduleInfo.History = append(moduleInfo.History, history)

	// Check if the module balance is sufficient to withdraw
	moduleTokenBalance := moduleInfo.GetUserTokenBalance(withdrawInfo.Tick, data.PkScript)
	{

		moduleTokenBalance.ReadyToWithdrawAmount = moduleTokenBalance.ReadyToWithdrawAmount.Add(balanceWithdraw)

		history.Valid = true
		// Update personal withdraw lookup table ReadyToWithdrawMap
		if moduleTokenBalance.ReadyToWithdrawMap == nil {
			moduleTokenBalance.ReadyToWithdrawMap = make(map[uint64]*model.InscriptionBRC20Data, 1)
		}
		moduleTokenBalance.ReadyToWithdrawMap[data.CreateIdxKey] = data

		// Update global withdraw lookup table
		g.InscriptionsWithdrawMap[data.CreateIdxKey] = withdrawInfo
		// g.InscriptionsValidBRC20DataMap[data.CreateIdxKey] = withdrawInfo.Data  // fixme
	}

	return nil
}
