package indexer

import (
	"errors"
	"fmt"
	"log"

	"fractal-indexer/api/lib/brc20_swap/conf"
	"fractal-indexer/api/lib/brc20_swap/constant"
	"fractal-indexer/api/lib/brc20_swap/model"
	"fractal-indexer/api/lib/brc20_swap/utils"
)

func (g *BRC20ModuleIndexer) ProcessMint(data *model.InscriptionBRC20Data, latestHeight int) error {
	// body := new(model.InscriptionBRC20MintTransferContent)
	// if err := body.Unmarshal(data.ContentBody); err != nil {
	// 	return nil
	// }

	body, err := model.BRC20MintTransferContentUnmarshal(data.ContentBody)
	if err != nil {
		return nil
	}

	// check tick
	uniqueLowerTicker, err := utils.GetValidUniqueLowerTickerTicker(body.BRC20Tick)
	if err != nil {
		return nil
		// return errors.New("mint, tick length not between 6 and 12")
	}
	tokenInfo, ok := g.InscriptionsTickerInfoMap[uniqueLowerTicker]
	if !ok {
		return nil
		// return errors.New(fmt.Sprintf("mint %s, but tick not exist", body.BRC20Tick))
	}
	tinfo := tokenInfo.Deploy
	if tokenInfo.SelfMint {
		if utils.DecodeInscriptionFromBin(data.Parent) != tinfo.GetInscriptionId() {
			return errors.New(fmt.Sprintf("self mint %s, but parent invalid", body.BRC20Tick))
		}
	}

	amt, err := model.GetUint128FromString(body.BRC20Amount, int(tinfo.Decimal))
	// check mint amount
	// amt, err := uint128.FromString(body.BRC20Amount, int(tinfo.Decimal))
	if err != nil {
		return errors.New(fmt.Sprintf("mint %s, but invalid amount(%s)", body.BRC20Tick, body.BRC20Amount))
	}
	if amt.Sign() <= 0 || amt.Cmp(tinfo.Limit) > 0 {
		return errors.New(fmt.Sprintf("mint %s, invalid amount(%s), limit(%s)", body.BRC20Tick, body.BRC20Amount, tinfo.Limit))
	}

	// clean mint
	isCleanMint := data.AddressType > 0
	if data.Height < uint32(conf.BRC20_SINGLE_STEP_TRANSFER_HEIGHT) {
		isCleanMint = false
	}
	if isCleanMint {
		var err error
		receiverPkScriptByte, err := utils.GetPkScriptByPubkeyAndType(data.TapScriptPk[1:33], data.AddressType)
		if err != nil {
			log.Panicf("clean mint, invalid pubkey")
			return errors.New("clean mint, invalid pubkey")
		}
		data.PkScript = string(receiverPkScriptByte)
	}

	receiverPkScript := data.PkScript

	// get user's tokens to update
	tokenBalance := g.GetUserTokenBalance(tokenInfo.Ticker, receiverPkScript)

	body.BRC20Tick = tokenInfo.Ticker
	mintInfo := model.NewInscriptionBRC20TickInfo(body.BRC20Tick, body.Operation, data)
	mintInfo.Data.BRC20Amount = body.BRC20Amount
	mintInfo.Data.BRC20Minted = amt.String()
	mintInfo.Decimal = tinfo.Decimal
	mintInfo.Amount = amt
	if tinfo.TotalMinted.Cmp(tinfo.Max) >= 0 {
		// invalid history
		if !conf.PruneBRC20MintHistory && g.EnableHistory {
			historyObj := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_INSCRIBE_MINT, false, false, mintInfo, tokenBalance, data)
			// history :=
			g.UpdateHistoryHeightAndGetHistoryIndex(historyObj)

			// tokenBalance.History = append(tokenBalance.History, history)
			// tokenBalance.HistoryMint = append(tokenBalance.HistoryMint, history)
			// tokenInfo.History = append(tokenInfo.History, history)
			// tokenInfo.HistoryMint = append(tokenInfo.HistoryMint, history)
		}
		return errors.New(fmt.Sprintf("mint %s, but mint out", body.BRC20Tick))
	}

	// update tinfo
	// minted
	balanceMinted := amt
	if tinfo.TotalMinted.Add(amt).Cmp(tinfo.Max) > 0 {
		balanceMinted = tinfo.Max.Sub(tinfo.TotalMinted)
	}
	tinfo.TotalMinted = tinfo.TotalMinted.Add(balanceMinted)
	if tinfo.TotalMinted.Cmp(tinfo.Max) >= 0 {
		tinfo.CompleteHeight = data.Height
		tinfo.CompleteBlockTime = data.BlockTime
	}
	// confirmed minted
	if data.BlockTime > 0 {
		tinfo.ConfirmedMinted = tinfo.ConfirmedMinted.Add(balanceMinted)
	}
	// count
	tinfo.MintTimes++
	tinfo.Data.BRC20Minted = tinfo.TotalMinted.String()
	// valid mint inscriptionNumber range
	tinfo.InscriptionNumberEnd = data.InscriptionNumber

	// update mint info
	mintInfo.Data.BRC20Minted = balanceMinted.String()
	mintInfo.Amount = balanceMinted

	// update tokenBalance
	if data.BlockTime > 0 {
		tokenBalance.AvailableBalanceSafe = tokenBalance.AvailableBalanceSafe.Add(balanceMinted)
	}
	tokenBalance.AvailableBalance = tokenBalance.AvailableBalance.Add(balanceMinted)
	tokenBalance.Balance = tokenBalance.OverallBalance().Float64()

	// burn
	if len(receiverPkScript) == 1 && receiverPkScript[0] == 0x6a {
		tinfo.Burned = tinfo.Burned.Add(balanceMinted)
	}

	if !conf.PruneBRC20MintHistory && g.EnableHistory {
		// history
		historyObj := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_INSCRIBE_MINT, true, false, mintInfo, tokenBalance, data)
		// history :=
		g.UpdateHistoryHeightAndGetHistoryIndex(historyObj)

		// tick history
		// tokenBalance.History = append(tokenBalance.History, history)
		// tokenBalance.HistoryMint = append(tokenBalance.HistoryMint, history)
		// tokenInfo.History = append(tokenInfo.History, history)
		// tokenInfo.HistoryMint = append(tokenInfo.HistoryMint, history)
		// user address
		// userHistory := g.GetBRC20HistoryByUser(receiverPkScript)
		// userHistory.History = append(userHistory.History, history)
	}

	// if !conf.PruneBRC20MintHistory {
	// 	g.InscriptionsValidBRC20DataMap[data.CreateIdxKey] = mintInfo.Data
	// }

	if m, ok := g.TickerMintCountByHeight[data.Height]; !ok {
		mTickMintCount := make(map[string]uint64, 0)
		mTickMintCount[tokenInfo.Ticker] = 1
		g.TickerMintCountByHeight[data.Height] = mTickMintCount
	} else {
		m[tokenInfo.Ticker] += 1
	}
	return nil
}
