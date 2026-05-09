package indexer

import (
	"errors"
	"log"
	"strings"

	"github.com/unisat-wallet/libbrc20-indexer/conf"
	"github.com/unisat-wallet/libbrc20-indexer/constant"
	"github.com/unisat-wallet/libbrc20-indexer/model"
	"github.com/unisat-wallet/libbrc20-indexer/uint128"
	"github.com/unisat-wallet/libbrc20-indexer/utils"
)

func (g *BRC20ModuleIndexer) GetTransferInfoByKey(createIdxKey uint64) (
	transferInfo *model.InscriptionBRC20TickInfo, isInvalid bool) {
	var ok bool
	// transfer
	transferInfo, ok = g.InscriptionsValidTransferMap[createIdxKey]
	if !ok {
		transferInfo, ok = g.InscriptionsInvalidTransferMap[createIdxKey]
		if !ok {
			transferInfo = nil
		} else {
			delete(g.InscriptionsInvalidTransferMap, createIdxKey)
		}
		isInvalid = true
	} else {
		delete(g.InscriptionsValidTransferMap, createIdxKey)
	}

	return transferInfo, isInvalid
}

func (g *BRC20ModuleIndexer) ProcessTransfer(data *model.InscriptionBRC20Data, transferInfo *model.InscriptionBRC20TickInfo, latestHeight int, isInvalid bool) error {
	// ticker
	uniqueLowerTicker := strings.ToLower(transferInfo.Tick)
	tokenInfo, ok := g.InscriptionsTickerInfoMap[uniqueLowerTicker]
	if !ok {
		// log.Printf("ProcessBRC20Transfer send transfer, but ticker invalid. txid: %s",
		// 	hex.EncodeToString(utils.ReverseBytes([]byte(data.TxId))),
		// )
		return errors.New("transfer, invalid ticker")
	}

	// to
	senderPkScript := transferInfo.PkScript
	receiverPkScript := data.PkScript
	if data.Satoshi == 0 {
		receiverPkScript = senderPkScript
		data.PkScript = senderPkScript
	}

	// global history
	if g.EnableHistory {
		historyObj := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_TRANSFER, !isInvalid, true, transferInfo, nil, data)
		history := g.UpdateHistoryHeightAndGetHistoryIndex(historyObj)

		if !conf.PruneHistoryList {
			tokenInfo.History = append(tokenInfo.History, history)
			tokenInfo.HistoryTransfer = append(tokenInfo.HistoryTransfer, history)
		}
	}

	// from
	// get user's tokens to update
	fromTokenBalance := g.GetUserTokenBalance(transferInfo.Tick, senderPkScript)
	if isInvalid {
		if g.EnableHistory {
			historyObj := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_SEND, false, true, transferInfo, fromTokenBalance, data)
			fromHistory := g.UpdateHistoryHeightAndGetHistoryIndex(historyObj)

			if !conf.PruneHistoryList {
				fromTokenBalance.History = append(fromTokenBalance.History, fromHistory)
				fromTokenBalance.HistorySend = append(fromTokenBalance.HistorySend, fromHistory)

				userHistory := g.GetBRC20HistoryByUser(senderPkScript)
				userHistory.History = append(userHistory.History, fromHistory)
				userHistory.HistorySend = append(userHistory.HistorySend, fromHistory)
			}
		}
		return nil
	}

	if _, ok := fromTokenBalance.ValidTransferMap[data.CreateIdxKey]; !ok {
		// log.Printf("ProcessBRC20Transfer send from transfer missing(dup transfer?). height: %d, txidx: %d",
		// 	data.Height,
		// 	data.TxIdx,
		// )
		return errors.New("transfer, invalid transfer")
	}

	// set from
	fromTokenBalance.TransferableBalance = fromTokenBalance.TransferableBalance.Sub(transferInfo.Amount)
	fromTokenBalance.Balance = fromTokenBalance.OverallBalance().Float64()
	delete(fromTokenBalance.ValidTransferMap, data.CreateIdxKey)

	if g.EnableHistory {
		historyObj := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_SEND, true, true, transferInfo, fromTokenBalance, data)
		fromHistory := g.UpdateHistoryHeightAndGetHistoryIndex(historyObj)

		if !conf.PruneHistoryList {
			fromTokenBalance.History = append(fromTokenBalance.History, fromHistory)
			fromTokenBalance.HistorySend = append(fromTokenBalance.HistorySend, fromHistory)

			userHistoryFrom := g.GetBRC20HistoryByUser(senderPkScript)
			userHistoryFrom.History = append(userHistoryFrom.History, fromHistory)
			userHistoryFrom.HistorySend = append(userHistoryFrom.HistorySend, fromHistory)
		}
	}

	// to
	// get user's tokens to update
	tokenBalance := g.GetUserTokenBalance(transferInfo.Tick, receiverPkScript)
	// set to
	if data.BlockTime > 0 {
		tokenBalance.AvailableBalanceSafe = tokenBalance.AvailableBalanceSafe.Add(transferInfo.Amount)
	}
	tokenBalance.AvailableBalance = tokenBalance.AvailableBalance.Add(transferInfo.Amount)
	tokenBalance.Balance = tokenBalance.OverallBalance().Float64()

	// burn
	if len(receiverPkScript) == 1 && []byte(receiverPkScript)[0] == 0x6a {
		tokenInfo.Deploy.Burned = tokenInfo.Deploy.Burned.Add(transferInfo.Amount)
	}

	if g.EnableHistory {
		historyObj := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_RECEIVE, true, true, transferInfo, tokenBalance, data)
		toHistory := g.UpdateHistoryHeightAndGetHistoryIndex(historyObj)

		if !conf.PruneHistoryList {
			tokenBalance.History = append(tokenBalance.History, toHistory)
			tokenBalance.HistoryReceive = append(tokenBalance.HistoryReceive, toHistory)

			userHistoryTo := g.GetBRC20HistoryByUser(receiverPkScript)
			userHistoryTo.History = append(userHistoryTo.History, toHistory)
			userHistoryTo.HistoryReceive = append(userHistoryTo.HistoryReceive, toHistory)
		}
	}

	////////////////////////////////////////////////////////////////
	g.ProcessModuleDeposit(senderPkScript, receiverPkScript, transferInfo, data, latestHeight)

	return nil
}

func (g *BRC20ModuleIndexer) ProcessModuleDeposit(senderPkScript, receiverPkScript string, transferInfo *model.InscriptionBRC20TickInfo, data *model.InscriptionBRC20Data, latestHeight int) {
	// module deposit
	moduleId, ok := utils.GetModuleFromScript([]byte(receiverPkScript))
	if !ok {
		// errors.New("module transfer, not module")
		return
	}
	moduleInfo, ok := g.ModulesInfoMap[moduleId]
	if !ok { // invalid module
		return
		// return errors.New(fmt.Sprintf("module transfer, module(%s) not exist", moduleId))
	}

	// global history
	mHistory := model.NewBRC20ModuleHistory(true, constant.BRC20_HISTORY_TYPE_N_TRANSFER, transferInfo.Meta, data, nil, true)
	moduleInfo.History = append(moduleInfo.History, mHistory)
	mHistory.PkScriptFrom = senderPkScript

	// get user's tokens to update
	moduleTokenBalance := moduleInfo.GetUserTokenBalance(transferInfo.Tick, senderPkScript)
	// set module deposit
	if (latestHeight - int(data.Height) + 1) >= conf.BRC20_MODULE_SAFE_CONFIRMATION { // how many confirmes ok
		moduleTokenBalance.SwapAccountBalanceSafe = moduleTokenBalance.SwapAccountBalanceSafe.Add(transferInfo.Amount)
	}
	moduleTokenBalance.SwapAccountBalance = moduleTokenBalance.SwapAccountBalance.Add(transferInfo.Amount)

}

func (g *BRC20ModuleIndexer) ProcessInscribeTransfer(data *model.InscriptionBRC20Data, latestHeight int) error {
	body := new(model.InscriptionBRC20MintTransferContent)
	if err := body.Unmarshal(data.ContentBody); err != nil {
		return nil
	}

	// check tick
	uniqueLowerTicker, err := utils.GetValidUniqueLowerTickerTicker(body.BRC20Tick)
	if err != nil {
		return nil
		// return errors.New("transfer, tick length not between 6 and 12")
	}

	tokenInfo, ok := g.InscriptionsTickerInfoMap[uniqueLowerTicker]
	if !ok {
		return nil
		// return errors.New(fmt.Sprintf("transfer %s, but tick not exist", body.BRC20Tick))
	}
	tinfo := tokenInfo.Deploy

	// check amount
	amt, err := uint128.FromString(body.BRC20Amount, int(tinfo.Decimal))
	if err != nil {
		return nil
		// return errors.New("transfer, but invalid amount")
	}
	if amt.Sign() <= 0 || amt.Cmp(tinfo.Max) > 0 {
		return nil
		// return errors.New("transfer, invalid amount(range)")
	}

	// get user's tokens to update
	fromTokenBalance := g.GetUserTokenBalance(tokenInfo.Ticker, data.PkScript)
	isSingleStep := data.AddressType > 0
	if data.Height < uint32(conf.BRC20_SINGLE_STEP_TRANSFER_HEIGHT) {
		isSingleStep = false
	}

	toTokenBalance := fromTokenBalance

	withDifferentReceiver := false
	senderPkScript := data.PkScript
	receiverPkScript := data.PkScript
	if isSingleStep {
		// single-step transfer
		var err error
		senderPkScriptByte, err := utils.GetPkScriptByPubkeyAndType(data.TapScriptPk[1:33], data.AddressType)
		if err != nil {
			log.Panicf("single-step transfer, invalid pubkey")
			return errors.New("single-step transfer, invalid pubkey")
		}
		if string(senderPkScriptByte) != senderPkScript {
			senderPkScript = string(senderPkScriptByte)
			withDifferentReceiver = true
			// get user's tokens to update
			fromTokenBalance = g.GetUserTokenBalance(tokenInfo.Ticker, senderPkScript)
		}
	} else {
		if []byte(data.PkScript)[0] == 0x6a {
			// ignore transfer on opreturn
			return nil
		}
	}

	balanceTransfer := amt

	body.BRC20Tick = tokenInfo.Ticker

	transferInfo := model.NewInscriptionBRC20TickInfo(body.BRC20Tick, body.Operation, data)
	transferInfo.Data.BRC20Amount = body.BRC20Amount
	transferInfo.Data.BRC20Limit = tinfo.Data.BRC20Limit
	transferInfo.Data.BRC20Decimal = tinfo.Data.BRC20Decimal

	transferInfo.Tick = tokenInfo.Ticker
	transferInfo.Amount = balanceTransfer
	transferInfo.Meta = data

	// If use the safe version of the available balance, it will cause the unconfirmed balance to not be able to be used to create a valid transfer inscription.
	historyValid := true
	if fromTokenBalance.AvailableBalance.Cmp(balanceTransfer) < 0 {
		historyValid = false
		g.InscriptionsInvalidTransferMap[data.CreateIdxKey] = transferInfo
	} else {
		// Update available balance

		// fixme: The available safe balance may not decrease, the current transfer usage of available balance source is not accurately distinguished.
		if fromTokenBalance.AvailableBalanceSafe.Cmp(balanceTransfer) > 0 {
			fromTokenBalance.AvailableBalanceSafe = fromTokenBalance.AvailableBalanceSafe.Sub(balanceTransfer)
		} else {
			fromTokenBalance.AvailableBalanceSafe = uint128.Zero
		}

		fromTokenBalance.AvailableBalance = fromTokenBalance.AvailableBalance.Sub(balanceTransfer)

		isTransferable := true
		if data.Height >= uint32(conf.BRC20_SINGLE_STEP_TRANSFER_NON_TRANSFERABLE_HEIGHT) && withDifferentReceiver {
			isTransferable = false
		}

		if isTransferable {
			toTokenBalance.TransferableBalance = toTokenBalance.TransferableBalance.Add(balanceTransfer)
		} else {
			if data.BlockTime > 0 {
				toTokenBalance.AvailableBalanceSafe = toTokenBalance.AvailableBalanceSafe.Add(balanceTransfer)
			}
			toTokenBalance.AvailableBalance = toTokenBalance.AvailableBalance.Add(balanceTransfer)
		}

		fromTokenBalance.Balance = fromTokenBalance.OverallBalance().Float64()
		if withDifferentReceiver {
			toTokenBalance.Balance = toTokenBalance.OverallBalance().Float64()

			// burn
			if len(receiverPkScript) == 1 && []byte(receiverPkScript)[0] == 0x6a {
				tokenInfo.Deploy.Burned = tokenInfo.Deploy.Burned.Add(transferInfo.Amount)
			}
		}
		if toTokenBalance.ValidTransferMap == nil {
			toTokenBalance.ValidTransferMap = make(map[uint64]struct{}, 1)
		}

		if isTransferable {
			toTokenBalance.ValidTransferMap[data.CreateIdxKey] = struct{}{}
			g.InscriptionsValidTransferMap[data.CreateIdxKey] = transferInfo
		} else {
			transferInfo.Data.Operation = constant.BRC20_DUMMY_OP_SINGLE_STEP_TRANSFER
		}
		if !conf.PruneValidBRC20DataMap {
			g.InscriptionsValidBRC20DataMap[data.CreateIdxKey] = transferInfo.Data
		}

		////////////////////////////////////////////////////////////////
		if withDifferentReceiver {
			g.ProcessModuleDeposit(senderPkScript, receiverPkScript, transferInfo, data, latestHeight)
		}
	}

	if g.EnableHistory {
		historyType := constant.BRC20_HISTORY_TYPE_N_INSCRIBE_TRANSFER
		if isSingleStep {
			historyType = constant.BRC20_HISTORY_TYPE_N_SINGLE_STEP_TRANSFER
		}

		history := g.HistoryCount
		historyObj := model.NewBRC20History(historyType, true, false, transferInfo, fromTokenBalance, data)
		// If use the safe version of the available balance, it will cause the unconfirmed balance to not be able to be used to create a valid transfer inscription.

		if !conf.PruneHistoryList {
			// user tick history
			fromTokenBalance.History = append(fromTokenBalance.History, history)
			fromTokenBalance.HistoryInscribeTransfer = append(fromTokenBalance.HistoryInscribeTransfer, history)
			// user history
			userHistory := g.GetBRC20HistoryByUser(string(senderPkScript))
			userHistory.History = append(userHistory.History, history)
			userHistory.HistoryInscribeTransfer = append(userHistory.HistoryInscribeTransfer, history)
			// global history
			tokenInfo.History = append(tokenInfo.History, history)
			tokenInfo.HistoryInscribeTransfer = append(tokenInfo.HistoryInscribeTransfer, history)
		}

		historyObj.Valid = historyValid
		if isSingleStep {
			historyObj.PkScriptFrom = senderPkScript
		}
		g.UpdateHistoryHeightAndGetHistoryIndex(historyObj)

		if withDifferentReceiver {
			history := g.HistoryCount
			historyObj := model.NewBRC20History(historyType, true, false, transferInfo, toTokenBalance, data)
			// If use the safe version of the available balance, it will cause the unconfirmed balance to not be able to be used to create a valid transfer inscription.

			if !conf.PruneHistoryList {
				// user tick history
				toTokenBalance.History = append(toTokenBalance.History, history)
				toTokenBalance.HistoryInscribeTransfer = append(toTokenBalance.HistoryInscribeTransfer, history)
				// user history
				userHistory := g.GetBRC20HistoryByUser(receiverPkScript)
				userHistory.History = append(userHistory.History, history)
				userHistory.HistoryInscribeTransfer = append(userHistory.HistoryInscribeTransfer, history)
			}

			historyObj.Valid = historyValid
			historyObj.PkScriptFrom = senderPkScript
			g.UpdateHistoryHeightAndGetHistoryIndex(historyObj)
		}
	}

	return nil
}
