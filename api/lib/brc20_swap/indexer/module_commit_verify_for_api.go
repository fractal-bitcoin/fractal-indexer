package indexer

import (
	"encoding/json"
	"errors"
	"log"

	"fractal-indexer/api/lib/brc20_swap/model"
)

func (g *BRC20ModuleIndexer) BRC20ModulePrepareSwapCommitContent(
	commitsStr []string,
	commitsObj []*model.InscriptionBRC20ModuleSwapCommitContent) {

	total := len(commitsStr)
	if total < 2 {
		return
	}

	for idx, commitStr := range commitsStr[:total-1] {
		nextCommitObj := commitsObj[idx+1]

		if _, ok := g.InscriptionsValidCommitMapById[nextCommitObj.Parent]; ok {
			continue
		}

		data := &model.InscriptionBRC20Data{
			InscriptionId: nextCommitObj.Parent,
			ContentBody:   []byte(commitStr),
		}
		g.InscriptionsValidCommitMapById[nextCommitObj.Parent] = data
	}
}

func (g *BRC20ModuleIndexer) ProcessCommitCheck(data *model.InscriptionBRC20Data) (int, error) {
	var body *model.InscriptionBRC20ModuleSwapCommitContent
	if err := json.Unmarshal(data.ContentBody, &body); err != nil {
		return -1, errors.New("json")
	}

	// check module exist
	moduleInfo, ok := g.ModulesInfoMap[body.Module]
	if !ok {
		return -1, errors.New("commit, module not exist")
	}

	inscriptionId := data.GetInscriptionId()
	log.Printf("ProcessCommitVerify commit[%s] ", inscriptionId)
	idx, _, err := g.ProcessCommitVerify(inscriptionId, body, nil)
	if err != nil {
		return idx, err
	}

	// set commit id
	moduleInfo.CommitIdMap[inscriptionId] = struct{}{}
	moduleInfo.CommitIdChainMap[body.Parent] = struct{}{}

	// Delete the already sent commit
	delete(g.InscriptionsValidCommitMapById, inscriptionId)

	return 0, nil
}

func getCommitParentFromData(data *model.InscriptionBRC20Data) (string, error) {
	var body *model.InscriptionBRC20ModuleSwapCommitContent
	if err := json.Unmarshal(data.ContentBody, &body); err != nil {
		return "", errors.New("json")
	}
	return body.Parent, nil
}

func (g *BRC20ModuleIndexer) BRC20ModuleVerifySwapCommitContent(
	commitStr string,
	commitObj *model.InscriptionBRC20ModuleSwapCommitContent,
	results []*model.SwapFunctionResultCheckState) (idx int, critical bool, err error) {

	if len(commitObj.Data) != len(results) {
		return -1, false, errors.New("commit verify, function results different size")
	}

	idx, err = g.ProcessInscribeCommitPreVerify(commitObj)
	if err != nil {
		log.Printf("commit verify failed: inscribe pre function[%d] %s", idx, err)
		return idx, true, err
	}

	// Verifying commit that was not moved in the middle
	// check module exist
	moduleInfo, ok := g.ModulesInfoMap[commitObj.Module]
	if !ok {
		return -1, true, errors.New("commit, module not exist")
	}

	parentId := commitObj.Parent
	// invalid if parent commit not exist
	commitIdsToCheck := []string{}
	for parentId != "" {
		if _, ok := moduleInfo.CommitIdMap[parentId]; ok {
			break
		}
		commitIdsToCheck = append([]string{parentId}, commitIdsToCheck...)

		parentCommitData, ok := g.InscriptionsValidCommitMapById[parentId]
		if !ok {
			return -1, false, errors.New("commit, parent body missing")
		}

		parentId, err = getCommitParentFromData(parentCommitData)
		if err != nil {
			return -1, true, errors.New("commit, parent json invalid")
		}
	}

	for _, parentId := range commitIdsToCheck {
		parentCommitData, ok := g.InscriptionsValidCommitMapById[parentId]
		if !ok {
			return -1, false, errors.New("commit, parent body not ready")
		}

		if idx, err := g.ProcessCommitCheck(parentCommitData); err != nil {
			return idx, true, err
		}
	}

	// verify current commit
	idx, critical, err = g.ProcessCommitVerify("", commitObj, results)
	if err != nil {
		log.Printf("commit verify failed, send function[%d] invalid", idx)
		return idx, critical, err
	}
	return 0, false, nil
}
