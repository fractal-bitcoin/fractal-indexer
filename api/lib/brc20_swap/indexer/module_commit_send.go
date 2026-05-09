package indexer

import (
	"errors"
	"log"

	"github.com/unisat-wallet/libbrc20-indexer/conf"
	"github.com/unisat-wallet/libbrc20-indexer/model"
	"github.com/unisat-wallet/libbrc20-indexer/utils"
)

func (g *BRC20ModuleIndexer) ProcessCommitFunctionSend(moduleInfo *model.BRC20ModuleSwapInfo, f *model.SwapFunctionData) error {
	addressTo := f.Params[0]
	pkScriptTo, _ := utils.GetPkScriptByAddress(addressTo, conf.GlobalNetParams)

	token := f.Params[1]
	tokenAmtStr := f.Params[2]

	tokenAmt, ok := g.CheckTickVerify(token, tokenAmtStr)
	if !ok {
		return errors.New("send: amt invalid")
	}

	tokenBalanceFrom := moduleInfo.GetUserTokenBalance(token, f.PkScript)

	// fixme: Must use the confirmed amount
	if tokenBalanceFrom.SwapAccountBalance.Cmp(tokenAmt) < 0 {
		log.Printf("token[%s] user[%s], balance %s", token, f.Address, tokenBalanceFrom)
		return errors.New("send: token balance insufficient")
	}

	tokenBalanceTo := moduleInfo.GetUserTokenBalance(token, string(pkScriptTo))

	// User Real-time Balance Update
	tokenBalanceFrom.SwapAccountBalance = tokenBalanceFrom.SwapAccountBalance.Sub(tokenAmt)
	tokenBalanceFrom.SwapAccountBalanceSafe = tokenBalanceFrom.SwapAccountBalanceSafe.Sub(tokenAmt)
	tokenBalanceTo.SwapAccountBalance = tokenBalanceTo.SwapAccountBalance.Add(tokenAmt)
	tokenBalanceTo.SwapAccountBalanceSafe = tokenBalanceTo.SwapAccountBalanceSafe.Add(tokenAmt)

	// log.Printf("pool send [%s] swappable: %s -> %s", token, tokenBalanceFrom.SwapAccountBalance, tokenBalanceTo.SwapAccountBalance)

	return nil
}
