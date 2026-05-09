package serial

import (
	"context"
	"fmt"
	"fractal-indexer/constant"
	"fractal-indexer/logger"
	"fractal-indexer/model"
	scriptDecoder "fractal-indexer/parser/script"
	"fractal-indexer/rdb"
	"fractal-indexer/utils"

	redis "github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

func getTxFee(tx *model.Tx, spentUtxoDataMap map[string]*model.TxoData) (satInputAmount, satOutputAmount uint64) {
	for vin, input := range tx.TxIns {
		objData, ok := spentUtxoDataMap[input.InputOutpointKey]
		if !ok {
			logger.Log.Info("tx-input-err",
				zap.String("txin", "input missing utxo"),
				zap.String("txid", tx.TxIdHex()),
				zap.Int("vin", vin),

				zap.String("utxid", input.InputHashHex()),
				zap.Uint32("vout", input.InputVout),
			)
			continue
		}
		satInputAmount += objData.Satoshi
	}
	for _, output := range tx.TxOuts {
		satOutputAmount += output.Satoshi
	}

	return satInputAmount, satOutputAmount
}

func setReinscriptionNFTByHeightTxIdx(txidx uint32, tx *model.Tx, txidxData map[uint32]struct{}) {
	if len(tx.NewNFTDataCreated) == 0 {
		return
	}
	if _, ok := txidxData[txidx]; ok {
		nft := &tx.NewNFTDataCreated[0]
		nft.IsCursed = true
		nft.IsReinscription = true
	}
}

func setCursedTxIfRecreateNFT(tx *model.Tx, spentUtxoDataMap map[string]*model.TxoData) {
	for i := range tx.NewNFTDataCreated {
		nft := &tx.NewNFTDataCreated[i]
		if nft.IsCursed {
			break
		}

		input := tx.TxIns[0]
		objData, ok := spentUtxoDataMap[input.InputOutpointKey]
		if !ok {
			logger.Log.Info("tx-input-err",
				zap.String("txin", "input missing utxo"),
				zap.String("txid", tx.TxIdHex()),
				zap.Int("vin", 0),
				zap.String("utxid", input.InputHashHex()),
				zap.Uint32("vout", input.InputVout),
			)
			break
		}

		nftpointIsCursed := false
		count := 0
		for _, nftpoint := range objData.CreatePointOfNFTs {
			if nftpoint.Offset != 0 {
				continue
			}
			if nftpoint.IsCursed || nftpoint.IsVindicate {
				nftpointIsCursed = true
			}
			count++
		}
		if count == 0 {
			continue
		}
		nft.IsReinscription = true

		if count > 1 || !nftpointIsCursed {
			nft.IsCursed = true
			continue
		}
	}
}

func updateTxCreateNFTPointerOffset(tx *model.Tx, spentUtxoDataMap map[string]*model.TxoData) {
	// update nft pointer offset
	satInputOffset := uint64(0)
	for vin, input := range tx.TxIns {
		objData, ok := spentUtxoDataMap[input.InputOutpointKey]
		if !ok {
			logger.Log.Info("tx-input-err",
				zap.String("txin", "input missing utxo"),
				zap.String("txid", tx.TxIdHex()),
				zap.Int("vin", vin),

				zap.String("utxid", input.InputHashHex()),
				zap.Uint32("vout", input.InputVout),
			)
			continue
		}

		if objData.Satoshi == 0 {
			for i := range tx.NewNFTDataCreated {
				nft := &tx.NewNFTDataCreated[i]
				if nft.InTxVin == uint32(vin) {
					nft.Is0SatInput = true
				}
			}
			continue
		}
		for i := range tx.NewNFTDataCreated {
			nft := &tx.NewNFTDataCreated[i]
			if nft.InTxVin == uint32(vin) {
				nft.InputOffset = satInputOffset
			}
		}
		satInputOffset += objData.Satoshi
	}
}

func getParentNFTId2CreateIdxInBlock(txs []model.Tx) (newParentId2CreateKey map[string]uint64) {
	newParent2Id := make(map[string]struct{})
	newParentId2CreateKey = make(map[string]uint64)

	pipe := rdb.RdbClient.Pipeline()
	defer pipe.Close()

	m := map[string]*redis.StringCmd{}
	ctx := context.Background()

	hasParent := false
	for _, tx := range txs {
		for i := range tx.NewNFTDataCreated {
			nft := &tx.NewNFTDataCreated[i]
			if len(nft.ParentsId) == 0 {
				continue
			}

			for _, id := range nft.ParentsId {
				hasParent = true
				newParent2Id[id] = struct{}{}
			}
		}
	}
	if !hasParent {
		return
	}

	for id := range newParent2Id {
		shortId := scriptDecoder.GetNFTBinIdFromRaw([]byte(id))
		m[id] = pipe.Get(ctx, "i"+shortId) // pika key uses short binId
	}
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		panic(err)
	}
	for id, v := range m {
		createPoint, err := v.Uint64()
		if err == redis.Nil {
			continue
		} else if err != nil {
			panic(err)
		}
		newParentId2CreateKey[id] = createPoint
	}
	return
}

func invalidTxParentNFT(tx *model.Tx, spentUtxoDataMap map[string]*model.TxoData, newParentId2CreateKey map[string]uint64) {
	checkParentId2CreateKey := make(map[string]uint64)

	hasParent := false
	for i := range tx.NewNFTDataCreated {
		nft := &tx.NewNFTDataCreated[i]
		if len(nft.ParentsId) == 0 {
			continue
		}

		for idx, id := range nft.ParentsId {
			if key, ok := newParentId2CreateKey[id]; ok {
				hasParent = true
				checkParentId2CreateKey[id] = key
				// exist
				continue
			}
			// must check here, don't put in func getParentNFTId2CreateIdxInBlock
			shortId := scriptDecoder.GetNFTBinIdFromRaw([]byte(id))
			if key, ok := model.GlobalNewInscriptionsId2CreateIdxKeyMap[shortId]; ok {
				hasParent = true
				checkParentId2CreateKey[id] = key
				continue
			}
			nft.ParentsId[idx] = ""
			nft.Parents[idx] = nil
		}
	}

	if !hasParent {
		return
	}

	// all nft exist in inputs
	validCreateIdxKey := make(map[uint64]struct{}, 0)
	for vin, input := range tx.TxIns {
		objData, ok := spentUtxoDataMap[input.InputOutpointKey]
		if !ok {
			logger.Log.Info("tx-input-err",
				zap.String("txin", "input missing utxo"),
				zap.String("txid", tx.TxIdHex()),
				zap.Int("vin", vin),

				zap.String("utxid", input.InputHashHex()),
				zap.Uint32("vout", input.InputVout),
			)
			continue
		}

		for _, nftpoint := range objData.CreatePointOfNFTs {
			if nftpoint.IsStrip {
				continue
			}
			validCreateIdxKey[nftpoint.GetCreateIdxUint64()] = struct{}{}
		}
	}

	// check if parent valid
	for i := range tx.NewNFTDataCreated {
		nft := &tx.NewNFTDataCreated[i]
		if len(nft.ParentsId) == 0 {
			continue
		}
		for idx, id := range nft.ParentsId {
			if id == "" {
				continue
			}
			if key, ok := checkParentId2CreateKey[id]; ok {
				if _, ok := validCreateIdxKey[key]; !ok {
					nft.ParentsId[idx] = ""
					nft.Parents[idx] = nil
				}
			}
		}
	}
}

// ParseBlockTxNFTsInAndOutSerial all tx input/output info
func ParseBlockTxNFTsInAndOutSerial(block *model.Block) {
	var coinbaseCreatePointOfNFTs []model.NFTCreatePoint
	var coinbaseNewEventInscriptions []*model.NewInscriptionInfo

	newParentId2CreateKey := getParentNFTId2CreateIdxInBlock(block.Txs[1:])

	// pre-allocate slab for NewInscriptionInfo (created NFTs)
	totalNewNFTs := 0
	for i := range block.Txs[1:] {
		totalNewNFTs += len(block.Txs[i+1].NewNFTDataCreated)
	}
	niiSlab := make([]model.NewInscriptionInfo, totalNewNFTs)
	niiIdx := 0

	satFeeOffset := utils.CalcBlockSubsidy(block.Height)
	nftIndexInBlock := uint32(0)
	// Skip coinbase.

	reinscriptionTxidxData, needSetReinscription := constant.MarkBRC20ReinscriptionByHeightTxidxMap[block.Height]
	for idx := range block.Txs[1:] {
		txIdx := idx + 1
		tx := &block.Txs[txIdx]
		var txCreateNewEventInscriptions []*model.NewInscriptionInfo

		satInputAmount, satOutputAmount := getTxFee(tx, block.ParseData.SpentUtxoDataMap)

		updateTxCreateNFTPointerOffset(tx, block.ParseData.SpentUtxoDataMap)
		if needSetReinscription {
			setReinscriptionNFTByHeightTxIdx(uint32(txIdx), tx, reinscriptionTxidxData)
		}
		setCursedTxIfRecreateNFT(tx, block.ParseData.SpentUtxoDataMap)
		invalidTxParentNFT(tx, block.ParseData.SpentUtxoDataMap, newParentId2CreateKey)

		// insert created NFT
		for createIdxInTx := range tx.NewNFTDataCreated {
			nft := &tx.NewNFTDataCreated[createIdxInTx]
			if block.Height >= constant.JUBILEE_ACTIVATION_HEIGHT {
				if nft.IsCursed {
					nft.IsCursed = false
					nft.IsVindicate = true
				}
			}

			createPoint := model.NFTCreatePoint{
				Height:     uint32(block.Height),
				IdxInBlock: nftIndexInBlock + uint32(createIdxInTx),
				Sequence:   0,
			}
			createPoint.SetCreatePointFlags(nft)
			newInscriptionInfo := &niiSlab[niiIdx]
			niiIdx++
			*newInscriptionInfo = model.NewInscriptionInfo{
				NFTData:     nft,
				CreatePoint: createPoint,
				TxIdx:       uint32(txIdx),
				TxId:        tx.TxId,
				IdxInTx:     uint32(createIdxInTx),

				InputsValue:  satInputAmount,
				OutputsValue: satOutputAmount,
				Ordinal:      0, // fixme: missing ordinal, todo
				InputIdx:     nft.InTxVin,
				BlockTime:    block.BlockTime,
			}
			if createPoint.IsBRC20Mint {
				createPoint.IsStrip = true
			}

			block.ParseData.NewInscriptions = append(block.ParseData.NewInscriptions, newInscriptionInfo)

			binId := scriptDecoder.GetNFTBinIdFromTxIdAndIdx(newInscriptionInfo.TxId, int(newInscriptionInfo.IdxInTx))
			model.GlobalNewInscriptionsId2CreateIdxKeyMap[binId] = newInscriptionInfo.CreatePoint.GetCreateIdxUint64()

			inFee := true
			satOutputOffset := uint64(0)
			satPointer := nft.InputOffset
			if nft.HasPointer && nft.Pointer < satOutputAmount {
				satPointer = nft.Pointer
			}
			for vout := range tx.TxOuts {
				output := &tx.TxOuts[vout]
				if satPointer < satOutputOffset+output.Satoshi {
					createPoint.Offset = satPointer - satOutputOffset
					inFee = false
					// create event, with a creator
					// Add the event for an NFT paid as fee; if the NFT has an unrecognized even tag or a 0-sat input, no event is generated.
					if nft.IsUnrecognizedEven || nft.Is0SatInput {
						break
					}

					// Drop stripped mint inscriptions.
					if constant.REINSCRIPTION_ACTIVATION_HEIGHT > 0 &&
						block.Height >= constant.REINSCRIPTION_ACTIVATION_HEIGHT &&
						createPoint.IsBRC20Mint {
						// drop
					} else {
						output.CreatePointOfNFTs = append(output.CreatePointOfNFTs, createPoint)
					}

					newInscriptionInfo.CreatePoint = createPoint
					newInscriptionInfo.InTxVout = uint32(vout)
					newInscriptionInfo.Satoshi = output.Satoshi
					newInscriptionInfo.PkScript = output.PkScript
					txCreateNewEventInscriptions = append(txCreateNewEventInscriptions, newInscriptionInfo)
					break
				}
				satOutputOffset += output.Satoshi
			}

			// create nft may in fee
			if inFee {
				tx.NFTLostCnt += 1
				// create event in fee, without creator. if Satoshi=0
				newInscriptionInfo.InTxVout = tx.TxOutCnt
				newInscriptionInfo.IsDefer = true

				// Add the event for an NFT paid as fee; if the NFT has an unrecognized even tag or a 0-sat input, no event is generated.
				if nft.IsUnrecognizedEven || nft.Is0SatInput {
					continue
				}

				// global fee offset in coinbase
				createPoint.Offset = satPointer - satOutputOffset + satFeeOffset
				coinbaseCreatePointOfNFTs = append(coinbaseCreatePointOfNFTs, createPoint)

				// transfer event, direct to miner
				if createPoint.IsStripBRC20EventForCoinbase() {
					coinbaseNewEventInscriptions = append(coinbaseNewEventInscriptions, nil)
				} else {
					txCreateNewEventInscriptions = append(txCreateNewEventInscriptions, newInscriptionInfo)

					newInscriptionInfoCopy := newInscriptionInfo.Copy()
					newInscriptionInfoCopy.CreatePoint = createPoint
					coinbaseNewEventInscriptions = append(coinbaseNewEventInscriptions, newInscriptionInfoCopy)
				}
			}
		}
		nftIndexInBlock += uint32(len(tx.NewNFTDataCreated))

		// insert exist NFT
		// Pre-count NFTs per output to avoid append reallocation.
		// Only activate for large NFT counts to avoid overhead on normal txs.
		{
			totalInputNFTs := 0
			for _, input := range tx.TxIns {
				if objData, ok := block.ParseData.SpentUtxoDataMap[input.InputOutpointKey]; ok {
					totalInputNFTs += len(objData.CreatePointOfNFTs)
				}
			}
			if totalInputNFTs > 256 {
				perOutput := make([]int, tx.TxOutCnt)
				tmpSatInputOffset := uint64(0)
				for _, input := range tx.TxIns {
					objData, ok := block.ParseData.SpentUtxoDataMap[input.InputOutpointKey]
					if !ok {
						continue
					}
					for _, nftpoint := range objData.CreatePointOfNFTs {
						sat := tmpSatInputOffset + nftpoint.Offset
						tmpSatOutputOffset := uint64(0)
						for vout := range tx.TxOuts {
							if sat < tmpSatOutputOffset+tx.TxOuts[vout].Satoshi {
								if !(constant.REINSCRIPTION_ACTIVATION_HEIGHT > 0 &&
									block.Height >= constant.REINSCRIPTION_ACTIVATION_HEIGHT &&
									(nftpoint.IsBRC20Mint || nftpoint.IsBRC20Tran)) {
									perOutput[vout]++
								}
								break
							}
							tmpSatOutputOffset += tx.TxOuts[vout].Satoshi
						}
					}
					tmpSatInputOffset += objData.Satoshi
				}
				for vout := range tx.TxOuts {
					if c := perOutput[vout]; c > 0 {
						old := tx.TxOuts[vout].CreatePointOfNFTs
						grown := make([]model.NFTCreatePoint, len(old), len(old)+c)
						copy(grown, old)
						tx.TxOuts[vout].CreatePointOfNFTs = grown
					}
				}
			}
		}

		satInputOffset := uint64(0)
		for vin, input := range tx.TxIns {
			objData, ok := block.ParseData.SpentUtxoDataMap[input.InputOutpointKey]
			if !ok {
				logger.Log.Info("tx-input-err",
					zap.String("txin", "input missing utxo"),
					zap.String("txid", tx.TxIdHex()),
					zap.Int("vin", vin),
					zap.String("utxid", input.InputHashHex()),
					zap.Uint32("vout", input.InputVout),
				)
				continue
			}
			block.ParseData.NftTransferCount += len(objData.CreatePointOfNFTs)
			for _, nftpoint := range objData.CreatePointOfNFTs {
				sat := satInputOffset + nftpoint.Offset
				inFee := true
				satOutputOffset := uint64(0)
				for vout := range tx.TxOuts {
					output := &tx.TxOuts[vout]
					if uint64(sat) < satOutputOffset+output.Satoshi {
						movetoCreatePoint := nftpoint
						movetoCreatePoint.Offset = uint64(sat - satOutputOffset)
						if movetoCreatePoint.Sequence < 0xffff {
							movetoCreatePoint.Sequence += 1
						}
						if movetoCreatePoint.IsBRC20Tran {
							movetoCreatePoint.IsStrip = true
						}

						inFee = false

						// Drop stripped mint/transfer inscriptions.
						if constant.REINSCRIPTION_ACTIVATION_HEIGHT > 0 &&
							block.Height >= constant.REINSCRIPTION_ACTIVATION_HEIGHT &&
							(movetoCreatePoint.IsBRC20Mint || movetoCreatePoint.IsBRC20Tran) {
							// drop
						} else {
							output.CreatePointOfNFTs = append(output.CreatePointOfNFTs, movetoCreatePoint)
						}

						// without record event
						if movetoCreatePoint.IsStripBRC20Event() {
							break
						}
						// record event first transfer
						newInscriptionInfo := &model.NewInscriptionInfo{
							NFTData:     &scriptDecoder.NFTData{},
							CreatePoint: movetoCreatePoint,

							Height: uint32(block.Height),
							TxIdx:  uint32(txIdx),
							TxId:   tx.TxId,

							InputsValue:  satInputAmount,
							OutputsValue: satOutputAmount,
							Ordinal:      0, // fixme: missing ordinal, todo

							InTxVout:     uint32(vout),
							Satoshi:      output.Satoshi,
							PkScript:     output.PkScript,
							PkScriptFrom: objData.PkScript,
							InputIdx:     uint32(vin),
							BlockTime:    block.BlockTime,
						}
						block.ParseData.NewEventInscriptions = append(block.ParseData.NewEventInscriptions, newInscriptionInfo)
						break
					}
					satOutputOffset += output.Satoshi
				} // fixme: create nft may in fee

				// move nft may in fee
				if inFee {
					tx.NFTLostCnt += 1

					movetoCreatePoint := nftpoint
					movetoCreatePoint.Offset = 0
					if movetoCreatePoint.Sequence < 0xffff {
						movetoCreatePoint.Sequence += 1
					}
					if movetoCreatePoint.IsBRC20Tran {
						movetoCreatePoint.IsStrip = true
					}

					// record event first transfer
					newInscriptionInfo := &model.NewInscriptionInfo{
						NFTData:     &scriptDecoder.NFTData{},
						CreatePoint: movetoCreatePoint,

						Height: uint32(block.Height),
						TxIdx:  uint32(txIdx),
						TxId:   tx.TxId,

						InputsValue:  satInputAmount,
						OutputsValue: satOutputAmount,
						Ordinal:      0, // fixme: missing ordinal, todo

						InTxVout:     tx.TxOutCnt,
						Satoshi:      0,
						PkScriptFrom: objData.PkScript,
						InputIdx:     uint32(vin),
						BlockTime:    block.BlockTime,
					}

					// record event by type
					if !movetoCreatePoint.IsStripBRC20Event() {
						block.ParseData.NewEventInscriptions = append(block.ParseData.NewEventInscriptions, newInscriptionInfo)
					}

					movetoCreatePoint.Offset = uint64(sat) - satOutputOffset + satFeeOffset // global fee offset in coinbase
					coinbaseCreatePointOfNFTs = append(coinbaseCreatePointOfNFTs, movetoCreatePoint)

					// additional events in coinbase
					if movetoCreatePoint.IsStripBRC20EventForCoinbase() {
						coinbaseNewEventInscriptions = append(coinbaseNewEventInscriptions, nil)
					} else {
						newInscriptionInfoCopy := newInscriptionInfo.Copy()
						newInscriptionInfoCopy.CreatePoint = movetoCreatePoint
						coinbaseNewEventInscriptions = append(coinbaseNewEventInscriptions, newInscriptionInfoCopy)
					}
				}
			}
			satInputOffset += objData.Satoshi
		}

		satFeeOffset += satInputAmount - satOutputAmount

		block.ParseData.NewEventInscriptions = append(block.ParseData.NewEventInscriptions, txCreateNewEventInscriptions...)

		// store utxo nft point
		for vout := range tx.TxOuts {
			output := &tx.TxOuts[vout]
			if output.Satoshi == 0 || len(output.CreatePointOfNFTs) == 0 {
				continue
			}

			if objData, ok := block.ParseData.SpentUtxoDataMap[output.OutpointKey]; ok {
				// not spent in self block
				objData.CreatePointOfNFTs = output.CreatePointOfNFTs
			} else if objData, ok := block.ParseData.NewUtxoDataMap[output.OutpointKey]; ok {
				objData.CreatePointOfNFTs = output.CreatePointOfNFTs
			} else {
				logger.Log.Info("tx-output-restore-nft-err",
					zap.String("txout", "output missing utxo"),
					zap.String("txid", tx.TxIdHex()),
					zap.Int("vout", vout),
				)
			}
		}
	}

	// coinbase
	coinbaseTx := &block.Txs[0]

	// update coinbase input nft
	coinbaseTx.TxIns[0].CreatePointOfNFTs = coinbaseCreatePointOfNFTs
	for idx, nftpoint := range coinbaseCreatePointOfNFTs {
		inFee := true
		sat := nftpoint.Offset
		satOutputOffset := uint64(0)
		for vout := range coinbaseTx.TxOuts {
			output := &coinbaseTx.TxOuts[vout]
			if uint64(sat) < satOutputOffset+output.Satoshi {

				createPoint := nftpoint
				createPoint.Offset = uint64(sat - satOutputOffset)
				if createPoint.Sequence < 0xffff {
					createPoint.Sequence += 1
				}

				output.CreatePointOfNFTs = append(output.CreatePointOfNFTs, createPoint)
				inFee = false

				event := coinbaseNewEventInscriptions[idx]
				// without record event
				if event == nil {
					break
				}
				event.TxIdx = block.TxCnt // need to be at last
				event.TxId = coinbaseTx.TxId
				event.InputsValue = coinbaseTx.InputsValue
				event.OutputsValue = coinbaseTx.OutputsValue

				event.CreatePoint = createPoint
				event.InTxVout = uint32(vout)
				event.Satoshi = output.Satoshi
				event.PkScript = output.PkScript
				event.InputIdx = 0
				block.ParseData.NewEventInscriptions = append(block.ParseData.NewEventInscriptions, event)
				break
			}
			satOutputOffset += output.Satoshi
		}
		if inFee {
			coinbaseTx.NFTLostCnt += 1
			// realy lost
		}
	}

	// store utxo nft point
	for vout := range coinbaseTx.TxOuts {
		output := &coinbaseTx.TxOuts[vout]
		if output.Satoshi == 0 || len(output.CreatePointOfNFTs) == 0 {
			continue
		}

		if objData, ok := block.ParseData.NewUtxoDataMap[output.OutpointKey]; ok {
			objData.CreatePointOfNFTs = output.CreatePointOfNFTs
		} else {
			logger.Log.Info("coinbase-output-restore-nft-err",
				zap.String("txout", "output is not utxo"),
				zap.String("txid", coinbaseTx.TxIdHex()),
				zap.Int("vout", vout),
			)
		}
	}
}

// GetNFTCountBeforeHeight gets the inscription count before the specified height, excluding height.
// The zset stores the inscription count at each height; subtract 1 to get the count before the given height.
func GetNFTCountBeforeHeight(key string, height uint32) int64 {
	// logger.Log.Info("GetNFTCountBeforeHeight",
	// 	zap.Uint32("height", height),
	// )

	count, err := rdb.RdbClient.ZScore(context.Background(), key, fmt.Sprintf("%d", height-1)).Result()
	if err != nil && err != redis.Nil {
		logger.Log.Error("GetNFTCountBeforeHeight failed", zap.Error(err))
		panic(err)
	}
	if err == redis.Nil {
		return 0
	}

	return int64(count)
}
