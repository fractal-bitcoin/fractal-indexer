package loader

import (
	"bufio"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/unisat-wallet/libbrc20-indexer/conf"
	"github.com/unisat-wallet/libbrc20-indexer/constant"
	"github.com/unisat-wallet/libbrc20-indexer/model"
	"github.com/unisat-wallet/libbrc20-indexer/utils"
)

func DumpBRC20InputData(fname string, brc20Datas chan interface{}, hexBody bool) {
	file, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		log.Fatalf("open block index file failed, %s", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	for dataIn := range brc20Datas {
		data := dataIn.(*model.InscriptionBRC20Data)

		var body, address string
		if hexBody {
			body = hex.EncodeToString(data.ContentBody)
			address = hex.EncodeToString([]byte(data.PkScript))
		} else {
			body = strings.ReplaceAll(string(data.ContentBody), "\n", " ")
			address, err = utils.GetAddressFromScript([]byte(data.PkScript), conf.GlobalNetParams)
			if err != nil {
				address = hex.EncodeToString([]byte(data.PkScript))
			}
		}

		fmt.Fprintf(writer, "%t %s %d %d %d %d %s %d %s %x %d %d %d %d\n",
			data.IsTransfer,

			hex.EncodeToString([]byte(data.TxId)),
			data.Idx,
			data.Vout,
			data.Offset,
			data.Satoshi,
			address,
			data.InscriptionNumber,
			body,
			data.CreateIdxKey,
			data.Height,
			data.TxIdx,
			data.BlockTime,
			data.Sequence,
		)
	}
	writer.Flush()
}

func DumpBRC20HistoryEventData(fname string, brc20Datas chan *model.BRC20History) {
	file, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		log.Fatalf("open block index file failed, %s", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	lastAddressPk := ""
	lastAddress := ""
	for data := range brc20Datas {
		addressFrom := "-"
		if len(data.PkScriptFrom) != 0 {
			addressFrom, err = utils.GetAddressFromScript([]byte(data.PkScriptFrom), conf.GlobalNetParams)
			if err != nil {
				addressFrom = hex.EncodeToString([]byte(data.PkScriptFrom))
			}
		}

		addressTo := "-"
		if lastAddressPk == data.PkScriptTo {
			addressTo = lastAddress
		} else {
			addressTo, err = utils.GetAddressFromScript([]byte(data.PkScriptTo), conf.GlobalNetParams)
			if err != nil {
				addressTo = hex.EncodeToString([]byte(data.PkScriptTo))
			}
			lastAddressPk = data.PkScriptTo
			lastAddress = addressTo
		}

		fmtstring := "%d %s %d %s %s %s %s %s %s\n"
		if !data.Valid {
			fmtstring = "! %d %s %d %s %s %s %s %s %s\n"
		}
		fmt.Fprintf(writer, fmtstring,
			data.Height,
			utils.HashString([]byte(data.TxId)),
			data.Inscription.InscriptionNumber,
			constant.BRC20_HISTORY_TYPE_NAMES[data.Type],
			data.Inscription.Data.BRC20Tick,

			addressFrom,
			addressTo,

			data.AvailableBalance,
			data.TransferableBalance,
		)
	}
	writer.Flush()
}

func DumpTickerInfoMap(fname string,
	redisClient redis.UniversalClient,
	baseHistoryCount uint32,
	historyData [][]byte,
	inscriptionsTickerInfoMap map[string]*model.BRC20TokenInfo,
	userTokensBalanceData map[string]map[string]*model.BRC20TokenBalance,
	tokenUsersBalanceData map[string]map[string]*model.BRC20TokenBalance,
) {

	file, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		log.Fatalf("open block index file failed, %s", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	var allTickers []string
	for ticker := range inscriptionsTickerInfoMap {
		allTickers = append(allTickers, ticker)
	}
	sort.SliceStable(allTickers, func(i, j int) bool {
		return allTickers[i] < allTickers[j]
	})

	for _, ticker := range allTickers {
		info := inscriptionsTickerInfoMap[ticker]
		nValid := 0

		fmt.Fprintf(writer, "%s history: %d, valid: %d, minted: %s, holders: %d\n",
			info.Ticker,
			len(info.History),
			nValid,
			info.Deploy.TotalMinted.String(),
			len(tokenUsersBalanceData[ticker]),
		)

		// holders
		var allHoldersPkScript []string
		for holder := range tokenUsersBalanceData[ticker] {
			allHoldersPkScript = append(allHoldersPkScript, holder)
		}
		// sort by holder address
		sort.SliceStable(allHoldersPkScript, func(i, j int) bool {
			return allHoldersPkScript[i] < allHoldersPkScript[j]
		})

		// holders
		for _, holder := range allHoldersPkScript {
			balanceData := tokenUsersBalanceData[ticker][holder]

			address, err := utils.GetAddressFromScript([]byte(balanceData.PkScript), conf.GlobalNetParams)
			if err != nil {
				address = hex.EncodeToString([]byte(balanceData.PkScript))
			}
			fmt.Fprintf(writer, "%s %s history: %d, transfer: %d, balance: %s\n",
				info.Ticker,
				address,
				len(balanceData.History),
				len(balanceData.ValidTransferMap),
				balanceData.OverallBalance().String(),
			)
		}
	}
	writer.Flush()
}

func DumpModuleInfoMap(fname string,
	modulesInfoMap map[string]*model.BRC20ModuleSwapInfo,
) {
	file, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		log.Fatalf("open module dump file failed, %s", err)
		return
	}
	defer file.Close()

	var allModules []string
	for moduleId := range modulesInfoMap {
		allModules = append(allModules, moduleId)
	}
	sort.SliceStable(allModules, func(i, j int) bool {
		return allModules[i] < allModules[j]
	})

	for _, moduleId := range allModules {
		info := modulesInfoMap[moduleId]
		nValid := 0
		for _, h := range info.History {
			if h.Valid {
				nValid++
			}
		}

		fmt.Fprintf(file, "module %s(%s) nHistory: %d, nValidHistory: %d, nCommit: %d, nTickers: %d, nHolders: %d, swap: %d, lpholders: %d\n",
			info.Name,
			info.ID,
			len(info.History),
			nValid,
			len(info.CommitIdChainMap),
			len(info.TokenUsersBalanceDataMap),
			len(info.UsersTokenBalanceDataMap),

			len(info.LPTokenUsersBalanceMap),
			len(info.UsersLPTokenBalanceMap),
		)

		DumpModuleTickInfoMap(file, info.TokenUsersBalanceDataMap, info.UsersTokenBalanceDataMap)

		DumpModuleSwapInfoMap(file, info.SwapPoolTotalBalanceDataMap, info.LPTokenUsersBalanceMap, info.UsersLPTokenBalanceMap)
	}
}

func DumpModuleTickInfoMap(file *os.File,
	inscriptionsTickerInfoMap, userTokensBalanceData map[string]map[string]*model.BRC20ModuleTokenBalance) {

	var allTickers []string
	for ticker := range inscriptionsTickerInfoMap {
		allTickers = append(allTickers, ticker)
	}
	sort.SliceStable(allTickers, func(i, j int) bool {
		return allTickers[i] < allTickers[j]
	})

	for _, ticker := range allTickers {
		holdersMap := inscriptionsTickerInfoMap[ticker]

		nHistory := 0
		nValid := 0

		var allHoldersPkScript []string
		for holder, data := range holdersMap {
			nHistory += len(data.History)
			for _, h := range data.History {
				if h.Valid {
					nValid++
				}
			}
			allHoldersPkScript = append(allHoldersPkScript, holder)
		}
		sort.SliceStable(allHoldersPkScript, func(i, j int) bool {
			return allHoldersPkScript[i] < allHoldersPkScript[j]
		})

		fmt.Fprintf(file, " %s nHistory: %d, valid: %d, nHolders: %d\n",
			ticker,
			nHistory,
			nValid,
			// TokenTotalBalance[tick], // fixme
			len(holdersMap),
		)

		// holders
		for _, holder := range allHoldersPkScript {
			balanceData := holdersMap[holder]

			address, err := utils.GetAddressFromScript([]byte(balanceData.PkScript), conf.GlobalNetParams)
			if err != nil {
				address = hex.EncodeToString([]byte(balanceData.PkScript))
			}
			fmt.Fprintf(file, "  %s %s nHistory: %d, bnAvai: %s, bnSwap: %s",
				ticker,
				address,
				len(balanceData.History),
				balanceData.AvailableBalance.String(),
				balanceData.SwapAccountBalance.String(),
			)

			if len(balanceData.ReadyToWithdrawMap) > 0 {
				fmt.Fprintf(file, ", nWithdraw: %d", len(balanceData.ReadyToWithdrawMap))
			}
			fmt.Fprintf(file, "\n")
		}
	}

	fmt.Fprintf(file, "\n")
}

func DumpModuleSwapInfoMap(file *os.File,
	swapPoolTotalBalanceDataMap map[string]*model.BRC20ModulePoolTotalBalance,
	inscriptionsTickerInfoMap, userTokensBalanceData map[string]map[string]*model.BRC20ModuleLPTokenBalance) {

	var allTickers []string
	for ticker := range inscriptionsTickerInfoMap {
		allTickers = append(allTickers, ticker)
	}
	sort.SliceStable(allTickers, func(i, j int) bool {
		return allTickers[i] < allTickers[j]
	})

	for _, ticker := range allTickers {
		holdersMap := inscriptionsTickerInfoMap[ticker]

		var allHoldersPkScript []string
		for holder := range holdersMap {
			allHoldersPkScript = append(allHoldersPkScript, holder)
		}
		sort.SliceStable(allHoldersPkScript, func(i, j int) bool {
			return allHoldersPkScript[i] < allHoldersPkScript[j]
		})

		swap := swapPoolTotalBalanceDataMap[ticker]

		fmt.Fprintf(file, " pool: %s nHistory: %d, nLPholders: %d, lp: %s, %s: %s, %s: %s\n",
			ticker,
			len(swap.History),
			len(holdersMap),
			swap.LpBalance,
			swap.Tick[0],
			swap.TickBalance[0],
			swap.Tick[1],
			swap.TickBalance[1],
		)

		// holders
		for _, holder := range allHoldersPkScript {
			balanceData := holdersMap[holder]

			address, err := utils.GetAddressFromScript([]byte(holder), conf.GlobalNetParams)
			if err != nil {
				address = hex.EncodeToString([]byte(holder))
			}
			fmt.Fprintf(file, "  pool: %s %s lp: %s, swaps: %d\n",
				ticker,
				address,
				balanceData.OverallBalance().String(),
				len(userTokensBalanceData[holder]),
			)
		}
	}
}

func GetHistoryByIdx(redisClient redis.UniversalClient, historyIdx, baseHistoryCount uint32, HistoryData [][]byte) (history *model.BRC20History, err error) {
	if historyIdx >= baseHistoryCount && historyIdx-baseHistoryCount < uint32(len(HistoryData)) {
		buf := HistoryData[historyIdx-baseHistoryCount]
		history = &model.BRC20History{}
		history.Unmarshal(buf)
		return history, nil
	}

	ctx := context.Background()
	res, err := redisClient.Get(ctx, fmt.Sprintf("h%d", historyIdx)).Result()
	if err != nil {
		return nil, errors.New("db failed")
	}

	history = &model.BRC20History{}
	history.Unmarshal([]byte(res))
	return history, nil
}
