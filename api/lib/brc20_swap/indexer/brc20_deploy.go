package indexer

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/unisat-wallet/libbrc20-indexer/conf"
	"github.com/unisat-wallet/libbrc20-indexer/constant"
	"github.com/unisat-wallet/libbrc20-indexer/decimal"
	"github.com/unisat-wallet/libbrc20-indexer/model"
	"github.com/unisat-wallet/libbrc20-indexer/uint128"
	"github.com/unisat-wallet/libbrc20-indexer/utils"
)

func (g *BRC20ModuleIndexer) ProcessDeploy(data *model.InscriptionBRC20Data, latestHeight int) error {
	body := new(model.InscriptionBRC20DeployContent)
	if err := body.Unmarshal(data.ContentBody); err != nil {
		return nil
	}

	// check tick
	uniqueLowerTicker, err := utils.GetValidUniqueLowerTickerTicker(body.BRC20Tick)
	if err != nil {
		return nil
		// return errors.New("deploy, tick length not between 6 and 32")
	}

	if body.BRC20SelfMint != "true" && body.BRC20SelfMint != "false" {
		return nil
		// return errors.New("deploy, set self_mint, but not true/false")
	}

	// tick enable, fixme: test only, not support space in ticker
	if conf.TICKS_ENABLED != "" {
		if strings.Contains(uniqueLowerTicker, " ") {
			return nil
		}
		if !strings.Contains(conf.TICKS_ENABLED, uniqueLowerTicker) {
			return nil
		}
	}

	if _, ok := g.InscriptionsTickerInfoMap[uniqueLowerTicker]; ok { // dup ticker
		return nil
		// return errors.New("deploy, but tick exist")
	}
	if body.BRC20Max == "" { // without max
		return errors.New(fmt.Sprintf("deploy, but max missing. ticker: %s", uniqueLowerTicker))
	}

	tinfo := model.NewInscriptionBRC20TickInfo(body.BRC20Tick, body.Operation, data)
	tinfo.Data.BRC20Max = body.BRC20Max
	tinfo.Data.BRC20Limit = body.BRC20Limit
	tinfo.Data.BRC20Decimal = body.BRC20Decimal
	tinfo.Data.BRC20Minted = "0"
	tinfo.InscriptionNumberStart = data.InscriptionNumber

	tinfoSelfMint := false
	if body.BRC20SelfMint == "true" {
		tinfoSelfMint = true
		tinfo.Data.BRC20SelfMint = "true"
	} else {
		tinfoSelfMint = false
		tinfo.Data.BRC20SelfMint = "false"
	}

	// dec
	if dec, err := strconv.ParseUint(tinfo.Data.BRC20Decimal, 10, 64); err != nil || dec > 18 {
		// dec invalid
		return errors.New(fmt.Sprintf("deploy, but dec invalid. ticker: %s, dec: %s", uniqueLowerTicker, tinfo.Data.BRC20Decimal))
	} else {
		tinfo.Decimal = uint8(dec)
	}

	// max
	if max, err := uint128.FromString(body.BRC20Max, int(tinfo.Decimal)); err != nil {
		// max invalid
		return errors.New(fmt.Sprintf("deploy, but max invalid. ticker: %s, max: '%s'", uniqueLowerTicker, body.BRC20Max))
	} else {
		if max.Sign() < 0 || max.IsOverflowUint64() {
			return nil
			// return errors.New("deploy, but max invalid (range)")
		}

		if max.Sign() == 0 {
			if tinfoSelfMint {
				tinfo.Max = max.GetMaxUint64()
			} else {
				return errors.New("deploy, but max invalid (0)")
			}
		} else {
			tinfo.Max = max
		}
		tinfo.Max999 = decimal.FromUint128(tinfo.Max).Mul(decimal.NewDecimal(999, 3)).Div(decimal.NewDecimal(1000, 3)).Uint128()
	}

	// lim
	if lim, err := uint128.FromString(tinfo.Data.BRC20Limit, int(tinfo.Decimal)); err != nil {
		// limit invalid
		return errors.New(fmt.Sprintf("deploy, but limit invalid. ticker: %s, limit: '%s'", uniqueLowerTicker, tinfo.Data.BRC20Limit))
	} else {
		if lim.Sign() < 0 || lim.IsOverflowUint64() {
			return errors.New("deploy, but lim invalid (range)")
		}
		if lim.Sign() == 0 {
			if tinfoSelfMint {
				tinfo.Limit = lim.GetMaxUint64()
			} else {
				return errors.New("deploy, but lim invalid (0)")
			}
		} else {
			tinfo.Limit = lim
		}
	}

	// maxmint times
	tinfo.MaxMintTimes = decimal.FromUint128(tinfo.Max).Div(decimal.FromUint128(tinfo.Limit)).Uint64()
	if tinfo.MaxMintTimes == 0 {
		tinfo.MaxMintTimes = 1
	}

	tokenInfo := &model.BRC20TokenInfo{Ticker: body.BRC20Tick, SelfMint: tinfoSelfMint, Deploy: tinfo}
	g.InscriptionsTickerInfoMap[uniqueLowerTicker] = tokenInfo

	// get user's tokens to update
	tokenBalance := g.GetUserTokenBalance(tokenInfo.Ticker, data.PkScript)

	if g.EnableHistory {
		historyObj := model.NewBRC20History(constant.BRC20_HISTORY_TYPE_N_INSCRIBE_DEPLOY, true, false, tinfo, nil, data)
		history := g.UpdateHistoryHeightAndGetHistoryIndex(historyObj)

		if !conf.PruneHistoryList {
			tokenBalance.History = append(tokenBalance.History, history)
			tokenBalance.HistoryDeploy = append(tokenBalance.HistoryDeploy, history)
			tokenInfo.History = append(tokenInfo.History, history)
			tokenInfo.HistoryDeploy = append(tokenInfo.HistoryDeploy, history)

			// user history
			userHistory := g.GetBRC20HistoryByUser(string(data.PkScript))
			userHistory.History = append(userHistory.History, history)
			userHistory.HistoryDeploy = append(userHistory.HistoryDeploy, history)
		}
	}

	if !conf.PruneValidBRC20DataMap {
		g.InscriptionsValidBRC20DataMap[data.CreateIdxKey] = tinfo.Data
	}
	return nil
}
