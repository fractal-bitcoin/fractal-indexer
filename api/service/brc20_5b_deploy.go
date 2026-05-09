package service

import (
	"fractal-query/logger"
	"fractal-query/model"
	"sort"

	brc20Model "github.com/unisat-wallet/libbrc20-indexer/model"
)

func ProcessUpdateLatest5dTickerHoldersSummary() {
	logger.Log.Info("start ProcessUpdateLatest5dTickerHoldersSummary")
	if model.GSwap == nil {
		logger.Log.Error("ProcessUpdateLatest5dTickerHoldersSummary but brc20 not ready")
		return
	}
	var statusInfo []*brc20Model.BRC20TokenInfo
	for _, info := range model.GSwap.InscriptionsTickerInfoMap {
		if info.SelfMint {
			statusInfo = append(statusInfo, info)
		}
	}
	// sort by deploy height
	sort.Slice(statusInfo, func(i, j int) bool {
		idxi := uint64(statusInfo[i].Deploy.Height)*4294967296 + uint64(statusInfo[i].Deploy.TxIdx)
		idxj := uint64(statusInfo[j].Deploy.Height)*4294967296 + uint64(statusInfo[j].Deploy.TxIdx)
		return idxi < idxj
	})

	var deployNFTIndexList []uint64
	for _, info := range statusInfo {
		deployNFTIndexList = append(deployNFTIndexList, info.Deploy.CreateIdxKey)
	}

	inscriptionsAddressSummaryMap := make(map[string]*model.DeployInscriptionsAddressHoldSummary, 0)

	//////////////// ogdeploys
	deploysRsp, err := getInscriptionPointFromRedis(deployNFTIndexList)
	if err != nil {
		return
	}
	var utxoOutpointsForDeploys []string
	for _, nft := range deploysRsp {
		utxoOutpointsForDeploys = append(utxoOutpointsForDeploys, nft.UtxoOutpoint)
	}
	// Get UTXO details from a list of UTXO keys.
	txOutsForDeploysRsp, err := getNonTokenUtxoFromRedisReturnAddressOnly("process brc20 deploy holder", utxoOutpointsForDeploys)
	if err != nil {
		return
	}

	for idx, txo := range txOutsForDeploysRsp {
		summary, ok := inscriptionsAddressSummaryMap[txo.Address]
		if !ok {
			summary = &model.DeployInscriptionsAddressHoldSummary{}
			inscriptionsAddressSummaryMap[txo.Address] = summary
		}
		summary.DeployCount += 1
		info := statusInfo[idx]
		summary.DeployInscriptions = append(summary.DeployInscriptions,
			&model.DeployInscriptionsSummary{
				Data:              *info.Deploy.Data,
				InscriptionNumber: info.Deploy.InscriptionNumber,
				InscriptionId:     info.Deploy.GetInscriptionId(),
				Satoshi:           info.Deploy.Satoshi,
				Height:            info.Deploy.Height,
			})
	}
	model.Global5bDeployInscriptionsAddressSummaryMap = inscriptionsAddressSummaryMap

	logger.Log.Info("ProcessUpdateLatest5dTickerHoldersSummary finished")
}
