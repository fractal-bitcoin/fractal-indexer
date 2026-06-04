package indexer

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"

	"fractal-indexer/api/lib/brc20_swap/conf"
	"fractal-indexer/api/lib/brc20_swap/constant"
	"fractal-indexer/api/lib/brc20_swap/model"
	"fractal-indexer/api/lib/brc20_swap/utils"
)

func (g *BRC20ModuleIndexer) GetCommitFunctionProcess(function string) (func(*model.BRC20ModuleSwapInfo, *model.SwapFunctionData) error, bool) {
	var process func(*model.BRC20ModuleSwapInfo, *model.SwapFunctionData) error
	if function == constant.BRC20_SWAP_FUNCTION_DEPLOY_POOL {
		process = g.ProcessCommitFunctionDeployPool
	} else if function == constant.BRC20_SWAP_FUNCTION_ADD_LIQ {
		process = g.ProcessCommitFunctionAddLiquidity
	} else if function == constant.BRC20_SWAP_FUNCTION_REMOVE_LIQ {
		process = g.ProcessCommitFunctionRemoveLiquidity
	} else if function == constant.BRC20_SWAP_FUNCTION_SWAP {
		process = g.ProcessCommitFunctionSwap
	} else if function == constant.BRC20_SWAP_FUNCTION_DECREASE_APPROVAL {
		process = g.ProcessCommitFunctionDecreaseApproval
	} else if function == constant.BRC20_SWAP_FUNCTION_SEND {
		process = g.ProcessCommitFunctionSend
	} else if function == constant.BRC20_SWAP_FUNCTION_SENDLP {
		process = g.ProcessCommitFunctionSendLp
	} else if function == constant.BRC20_SWAP_FUNCTION_LOCK {
		process = g.ProcessCommitFunctionLockLp
	} else if function == constant.BRC20_SWAP_FUNCTION_UNLOCK {
		process = g.ProcessCommitFunctionUnlockLp
	} else {
		return nil, false
	}
	return process, true
}

func (g *BRC20ModuleIndexer) GetCommitInfoByKey(createIdxKey uint64) (
	commitData *model.InscriptionBRC20Data, isInvalid bool) {
	var ok bool
	// commit
	commitData, ok = g.InscriptionsValidCommitMap[createIdxKey]
	if !ok {
		commitData, ok = g.InscriptionsInvalidCommitMap[createIdxKey]
		if !ok {
			commitData = nil
		}
		isInvalid = true
	}

	return commitData, isInvalid
}

func (g *BRC20ModuleIndexer) ProcessCommit(dataFrom, dataTo *model.InscriptionBRC20Data, isInvalid bool) error {
	inscriptionId := dataFrom.GetInscriptionId()
	// log.Printf("parse move commit. inscription id: %s", inscriptionId)

	// Delete the already sent commit
	delete(g.InscriptionsValidCommitMapById, inscriptionId)

	var body *model.InscriptionBRC20ModuleSwapCommitContent
	if err := json.Unmarshal(dataFrom.ContentBody, &body); err != nil {
		log.Printf("parse module commit json failed. txid: %s",
			hex.EncodeToString(utils.ReverseBytes([]byte(dataTo.TxId))),
		)
		return errors.New("json")
	}

	// Check the inscription reception address, it must be a module address.
	moduleId, ok := utils.GetModuleFromScript([]byte(dataTo.PkScript))
	if !ok || moduleId != body.Module {
		return errors.New("commit, not send to module")
	}

	// check module exist
	moduleInfo, ok := g.ModulesInfoMap[body.Module]
	if !ok {
		return errors.New("commit, module not exist")
	}

	// preset invalid
	moduleInfo.CommitInvalidMap[inscriptionId] = struct{}{}

	// Check the inscription sending address, it must be the sequencer address.
	if moduleInfo.SequencerPkScript != dataFrom.PkScript {
		return errors.New("module sequencer invalid")
	}

	// disable cherry pick after height
	if dataTo.Height < uint32(conf.BRC20_SWAP_MANDATORY_COMMIT_BEFORE_HEIGHT) {
		log.Printf("ProcessCommitVerify pick check commit[%d]: %s", len(moduleInfo.CommitIdMap), inscriptionId)
		// // Need to cherrypick, then verify on the copy.
		var pickUsersPkScript = make(map[string]bool, 0)
		var pickTokensTick = make(map[string]bool, 0)
		var pickPoolsPair = make(map[string]bool, 0)
		g.InitCherryPickFilter(body, pickUsersPkScript, pickTokensTick, pickPoolsPair)
		swapState := g.CherryPick(body.Module, pickUsersPkScript, pickTokensTick, pickPoolsPair)
		if idx, _, err := swapState.ProcessCommitVerify(inscriptionId, body, nil); err != nil {
			log.Printf("pick check commit[%d] invalid, function[%d] %s, txid: %s",
				len(moduleInfo.CommitIdMap),
				idx, err, hex.EncodeToString([]byte(dataTo.TxId)))
			return err
		}
	}

	historyData := &model.BRC20SwapHistoryCommitData{
		FunctionsTotal: len(body.Data),
	}
	log.Printf("ProcessCommitVerify final check commit[%d]: %s", len(moduleInfo.CommitIdMap), inscriptionId)
	// Execute in reality if successful.
	if idx, critical, err := g.ProcessCommitVerify(inscriptionId, body, nil); err != nil {
		log.Printf("final check commit[%d] invalid, function[%d] %s, txid: %s",
			len(moduleInfo.CommitIdMap),
			idx, err, hex.EncodeToString([]byte(dataTo.TxId)))
		if dataTo.Height < uint32(conf.BRC20_SWAP_MANDATORY_COMMIT_BEFORE_HEIGHT) {
			return err
		}
		if critical {
			return err
		}
		invalidList := make([]int, 0)
		for i, d := range body.Data {
			if d.Invalid {
				invalidList = append(invalidList, i)
			}
		}
		historyData.InvalidList = invalidList
	}

	// set commit id
	moduleInfo.CommitIdMap[inscriptionId] = struct{}{}
	moduleInfo.CommitIdChainMap[body.Parent] = struct{}{}

	// valid
	delete(moduleInfo.CommitInvalidMap, inscriptionId)

	history := model.NewBRC20ModuleHistory(true, constant.BRC20_HISTORY_SWAP_TYPE_N_COMMIT, dataFrom, dataTo, historyData, true)
	moduleInfo.History = append(moduleInfo.History, history)
	return nil
}
