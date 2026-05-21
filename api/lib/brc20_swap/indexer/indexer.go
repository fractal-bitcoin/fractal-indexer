package indexer

import (
	"bytes"
	"context"
	"log"

	"fractal-indexer/api/lib/brc20_swap/conf"
	"fractal-indexer/api/lib/brc20_swap/constant"
	"fractal-indexer/api/lib/brc20_swap/model"
	"fractal-indexer/api/lib/brc20_swap/uint128"
	"github.com/go-redis/redis/v8"
)

func isJson(contentBody []byte) bool {
	if len(contentBody) < 40 {
		return false
	}

	content := bytes.TrimSpace(contentBody)
	if !bytes.HasPrefix(content, []byte("{")) {
		return false
	}
	if !bytes.HasSuffix(content, []byte("}")) {
		return false
	}

	return true
}

// ProcessUpdateLatestBRC20Loop
func (g *BRC20ModuleIndexer) ProcessUpdateLatestBRC20Loop(brc20Datas chan interface{}, endHeight, latestHeight int, client *redis.UniversalClient) {
	// totalDataCount := len(brc20Datas)
	// log.Printf("process swap update. total %d", totalDataCount)

	model.CacheBody = make(map[string]*model.InscriptionBRC20ProtocalContent, 0)
	model.CacheBodyMint = make(map[string]*model.InscriptionBRC20MintTransferContent, 0)

	for i := 0; i < 19; i++ {
		model.CacheAmountMint[i] = make(map[string]uint128.Decimal, 0)
	}

	tmpHeight := 0
	lastProcessedHeight := g.BestHeight
	for dataIn := range brc20Datas {
		data := dataIn.(*model.InscriptionBRC20Data)

		if int(data.Height) >= tmpHeight+1000 {
			tmpHeight = int(data.Height)
			log.Printf("(%d/%d) brc20 processing...", data.Height, endHeight)
		}

		// on height boundary: flush L1 overlay to L2 for the previous height
		if data.Height != lastProcessedHeight && lastProcessedHeight > 0 {
			g.MergeOverlayToPending()
		}
		lastProcessedHeight = data.Height

		// update latest height
		g.BestHeight = data.Height
		// is sending transfer
		if data.IsTransfer {
			// not first move
			if data.Sequence != 1 {
				continue
			}

			// transfer
			if transferInfo, isInvalid := g.GetTransferInfoByKey(data.CreateIdxKey); transferInfo != nil {
				if err := g.ProcessTransfer(data, transferInfo, latestHeight, isInvalid); err != nil {
					log.Printf("process transfer move failed: %s", err)
				}
				continue
			}

			// module withdraw
			if withdrawInfo := g.GetWithdrawInfoByKey(data.CreateIdxKey); withdrawInfo != nil {
				if err := g.ProcessWithdraw(data, withdrawInfo); err != nil {
					log.Printf("process withdraw move failed: %s", err)
				}
				continue
			}

			// module commit
			if commitFrom, isInvalid := g.GetCommitInfoByKey(data.CreateIdxKey); commitFrom != nil {
				if err := g.ProcessCommit(commitFrom, data, isInvalid); err != nil {
					log.Printf("process commit move failed: %s", err)
				}
				continue
			}

			continue
		}

		// inscribe as fee
		if data.Satoshi == 0 {
			continue
		}

		if ok := isJson(data.ContentBody); !ok {
			// log.Println("not json")
			continue
		}

		body, err := model.BRC20ProtocalContentUnmarshal(data.ContentBody)
		if err != nil {
			// log.Println("Unmarshal failed", err, string(data.ContentBody))
			continue
		}

		// is inscribe deploy/mint/transfer
		if body.Proto != constant.BRC20_P &&
			body.Proto != constant.BRC20_P_MODULE &&
			body.Proto != constant.BRC20_P_SWAP {
			// log.Println("not proto")
			continue
		}

		// brc20 not accept cursed (except reinscription enabled, or is single step transfer)
		if data.NFTType == 67 {
			if int(data.Height) < conf.BRC20_SINGLE_STEP_TRANSFER_HEIGHT {
				continue
			} else {
				// Accept cursed inscriptions, but reject batch inscriptions unless this is a single-step transfer.
				if !(data.AddressType > 0 && body.Proto == constant.BRC20_P && body.Operation == constant.BRC20_OP_TRANSFER) &&
					data.Idx > 0 {
					continue
				}
			}
		}

		var process func(*model.InscriptionBRC20Data, int) error
		if body.Proto == constant.BRC20_P && body.Operation == constant.BRC20_OP_DEPLOY {
			process = g.ProcessDeploy
		} else if body.Proto == constant.BRC20_P && body.Operation == constant.BRC20_OP_MINT {
			process = g.ProcessMint
		} else if body.Proto == constant.BRC20_P && body.Operation == constant.BRC20_OP_TRANSFER {
			process = g.ProcessInscribeTransfer
		} else if body.Proto == constant.BRC20_P_MODULE && body.Operation == constant.BRC20_OP_MODULE_DEPLOY {
			process = g.ProcessCreateModule
		} else if body.Proto == constant.BRC20_P_MODULE && body.Operation == constant.BRC20_OP_MODULE_WITHDRAW {
			process = g.ProcessInscribeWithdraw
		} else if body.Proto == constant.BRC20_P_SWAP && body.Operation == constant.BRC20_OP_SWAP_COMMIT {
			process = g.ProcessInscribeCommit
		} else {
			continue
		}

		if err := process(data, latestHeight); err != nil {
			if conf.DEBUG {
				log.Printf("(%d/%d) process failed: %s", data.Height, endHeight, err)
			}
		}

		if len(g.HistoryData) >= 500000 && client != nil {
			g.StoreHistoryIntoPika(*client)
		}

		if body.Proto == constant.BRC20_P && body.Operation == constant.BRC20_OP_MINT {
			data.InscriptionId = ""
			data.TapScriptPk = data.TapScriptPk[:0]
			data.Parent = data.Parent[:0]
			data.ContentBody = data.ContentBody[:0]
			CacheContentBodyPool.Put(data)
		}
	}

	// flush L1 overlay for the last processed height
	if g.BestHeight > 0 {
		g.MergeOverlayToPending()
	}

	if len(g.HistoryData) > 0 && client != nil {
		c := *client
		g.StoreHistoryIntoPika(c)
		c.HMSet(context.Background(), "info", "height", endHeight, "total", g.HistoryCount)
	}

	log.Printf("process swap finish. ticker: %d, users: %d(%d/%d), tokens: %d(%d/%d), validInscription: %d, validTransfer: %d, invalidTransfer: %d, history: %d",
		len(g.InscriptionsTickerInfoMap),
		len(g.UserTokensBalanceData), len(g.PendingUserTokensBalanceData), len(g.OverlayUserTokensBalanceData),
		len(g.TokenUsersBalanceData), len(g.PendingTokenUsersBalanceData), len(g.OverlayTokenUsersBalanceData),

		len(g.InscriptionsValidBRC20DataMap),

		len(g.InscriptionsValidTransferMap),
		len(g.InscriptionsInvalidTransferMap),

		g.HistoryCount,
	)

	nswap := 0
	for _, m := range g.ModulesInfoMap {
		nswap += len(m.SwapPoolTotalBalanceDataMap)
	}

	nuser := 0
	for _, m := range g.ModulesInfoMap {
		nuser += len(m.UsersTokenBalanceDataMap)
	}

	log.Printf("process swap finish. module: %d, swap: %d, users: %d, validCommit: %d, invalidCommit: %d",
		len(g.ModulesInfoMap),
		nswap,
		nuser,

		len(g.InscriptionsValidCommitMap),
		len(g.InscriptionsInvalidCommitMap),
	)
}

func (g *BRC20ModuleIndexer) Init() {
	g.initBRC20()
	g.initModule()
}
