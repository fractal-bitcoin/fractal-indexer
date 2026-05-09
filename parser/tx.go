package parser

import (
	"encoding/binary"
	"fractal-indexer/constant"
	"fractal-indexer/model"
	scriptDecoder "fractal-indexer/parser/script"
	"fractal-indexer/utils"
	"sync"
)

func NewTxs(txsraw []byte, height uint32) (txs []model.Tx, txSlabOut []model.Tx, inSlabOut []model.TxIn, outSlabOut []model.TxOut, nftSlabs [][]scriptDecoder.NFTData, ok bool) {
	offset := uint(0)
	txcnt, txcnt_size := utils.DecodeVarIntForBlock(txsraw[offset:])
	offset += txcnt_size

	// try to reuse pooled Tx slab
	if s := model.GetTxSlab(); s != nil && uint(cap(s)) >= txcnt {
		txs = s[:txcnt]
	}
	if txs == nil {
		txs = make([]model.Tx, txcnt)
	}

	// pre-scan to get exact total input/output/pkscript counts, then allocate slabs once
	totalIns, totalOuts, totalPkBytes := preScanTxCounts(txsraw[offset:], txcnt)

	// try to reuse pooled slabs
	var inSlab []model.TxIn
	if s := model.GetTxInSlab(); s != nil && uint(cap(s)) >= totalIns {
		inSlab = s[:totalIns]
	}
	if inSlab == nil {
		inSlab = make([]model.TxIn, totalIns)
	}

	var outSlab []model.TxOut
	if s := model.GetTxOutSlab(); s != nil && uint(cap(s)) >= totalOuts {
		outSlab = s[:totalOuts]
	}
	if outSlab == nil {
		outSlab = make([]model.TxOut, totalOuts)
	}

	// block-level slabs: PkScript, InputOutpointKey, TxId
	pkSlab := make([]byte, totalPkBytes)
	pkOff := uint(0)

	// one big buffer for all InputOutpointKeys, convert to string for substring sharing
	inKeyBuf := make([]byte, totalIns*model.OutpointKeySize)
	inKeyOff := uint(0)

	txIdSlab := make([]byte, txcnt*32)

	inOff := uint(0)
	outOff := uint(0)

	for i := range txs {
		txoffset := NewTx(&txs[i], txsraw[offset:], inSlab[inOff:], outSlab[outOff:], pkSlab[pkOff:], inKeyBuf[inKeyOff:])

		tx := &txs[i]
		// check tx valid
		if tx.TxInCnt == 0 || tx.TxOutCnt == 0 {
			return nil, nil, nil, nil, nil, false
		}

		tx.Raw = txsraw[offset : offset+txoffset]
		offset += txoffset

		// advance slab offsets
		for j := range tx.TxOuts {
			pkOff += uint(len(tx.TxOuts[j].PkScript))
		}
		inKeyOff += uint(tx.TxInCnt) * model.OutpointKeySize
		inOff += uint(tx.TxInCnt)
		outOff += uint(tx.TxOutCnt)

		// TxId into slab
		dst := txIdSlab[uint(i)*32 : uint(i)*32+32]
		if tx.WitOffset > 0 {
			utils.GetWitnessHash256Into(dst, tx.Raw, tx.WitOffset)
		} else {
			utils.GetHash256Into(dst, tx.Raw)
		}
		tx.TxId = dst
		tx.Size = uint32(txoffset)
	}

	// convert inKeyBuf to one big string, then assign substrings (shared backing)
	inKeyStr := string(inKeyBuf)
	idx := uint(0)
	for i := range txs {
		for j := range txs[i].TxIns {
			txs[i].TxIns[j].InputOutpointKey = inKeyStr[idx : idx+model.OutpointKeySize]
			idx += model.OutpointKeySize
		}
	}

	if height < constant.ORDINALS_ACTIVATION_HEIGHT {
		return txs, txs, inSlab, outSlab, nil, true
	}

	// nft decode, after txidHex
	const numWorkers = 4
	txChan := make(chan *model.Tx, 128)
	var wg sync.WaitGroup
	nftSlabsArr := make([][]scriptDecoder.NFTData, numWorkers)
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func(workerIdx int) {
			defer wg.Done()
			// get slab from pool
			var slab []scriptDecoder.NFTData
			if s := model.GetNFTDataSlab(); s != nil {
				slab = s[:0]
			}
			for tx := range txChan {
				slab = utils.EncodeTxNFTAfterJubilee(tx, height, slab)
			}
			nftSlabsArr[workerIdx] = slab
		}(i)
	}

	for i := range txs {
		if txs[i].WitOffset == 0 {
			continue
		}
		txChan <- &txs[i]
	}
	close(txChan)
	wg.Wait()

	return txs, txs, inSlab, outSlab, nftSlabsArr, true
}

func NewTx(tx *model.Tx, rawtx []byte, inSlab []model.TxIn, outSlab []model.TxOut, pkSlab []byte, inKeyBuf []byte) (offset uint) {
	tx.Version = binary.LittleEndian.Uint32(rawtx[0:4])
	offset = 4

	// check witness
	isWit := false
	if rawtx[4] == 0 && rawtx[5] == 1 {
		isWit = true
		offset += 2
	}

	txincnt, txincntsize := utils.DecodeVarIntForBlock(rawtx[offset:])
	offset += txincntsize

	tx.TxInCnt = uint32(txincnt)
	tx.TxIns = inSlab[:txincnt] // slice from pre-allocated slab

	txoffset := uint(0)
	inKeyOff := uint(0)
	for i := range tx.TxIns {
		txoffset = NewTxIn(&tx.TxIns[i], rawtx[offset:], inKeyBuf[inKeyOff:])
		offset += txoffset
		inKeyOff += model.OutpointKeySize
	}

	txoutcnt, txoutcntsize := utils.DecodeVarIntForBlock(rawtx[offset:])
	offset += txoutcntsize

	tx.TxOutCnt = uint32(txoutcnt)
	tx.TxOuts = outSlab[:txoutcnt] // slice from pre-allocated slab
	for i := range tx.TxOuts {
		txoffset = NewTxOut(&tx.TxOuts[i], rawtx[offset:])
		offset += txoffset
	}

	// copy PkScripts into block-level slab (zero per-tx allocation)
	off := 0
	for i := range tx.TxOuts {
		n := copy(pkSlab[off:], tx.TxOuts[i].PkScript)
		tx.TxOuts[i].PkScript = pkSlab[off : off+n]
		off += n
	}

	if isWit {
		tx.WitOffset = uint32(offset)
		for i := range tx.TxIns {
			tx.TxIns[i].ScriptWitness, txoffset = NewTxWits(rawtx[offset:])
			offset += txoffset
		}
	}

	tx.LockTime = binary.LittleEndian.Uint32(rawtx[offset : offset+4])
	offset += 4
	return
}

func NewTxIn(txin *model.TxIn, txinraw []byte, inKeyDst []byte) (offset uint) {
	txin.InputHash = txinraw[0:32] // zero-copy slice into raw
	txin.InputVout = binary.LittleEndian.Uint32(txinraw[32:36])
	offset = 36

	scriptsig, scriptsigsize := utils.DecodeVarIntForBlock(txinraw[offset:])
	offset += scriptsigsize

	txin.ScriptSig = txinraw[offset : offset+scriptsig] // zero-copy slice into raw
	offset += scriptsig

	txin.Sequence = binary.LittleEndian.Uint32(txinraw[offset : offset+4])
	offset += 4

	// build fixed 20-byte outpoint key: txid[:16] + vout(4 LE)
	model.MakeOutpointKey(inKeyDst, txinraw[0:32], txin.InputVout)
	txin.InputOutpoint = txinraw[0:36] // zero-copy slice into raw (full 36 bytes preserved)
	return
}

func NewTxOut(txout *model.TxOut, txoutraw []byte) (offset uint) {
	txout.Satoshi = binary.LittleEndian.Uint64(txoutraw[0:8])
	offset = 8

	pkscript, pkscriptsize := utils.DecodeVarIntForBlock(txoutraw[offset:])
	offset += pkscriptsize

	txout.PkScript = txoutraw[offset : offset+pkscript] // zero-copy temporarily, batch-copied in NewTx

	offset += pkscript
	return
}

func NewTxWits(txwitraw []byte) (wits []byte, offset uint) {
	txWitcnt, txWitcntsize := utils.DecodeVarIntForBlock(txwitraw[0:])
	offset = txWitcntsize

	for witIndex := uint(0); witIndex < txWitcnt; witIndex++ {
		txWitScriptcnt, txWitScriptcntsize := utils.DecodeVarIntForBlock(txwitraw[offset:])
		offset += txWitScriptcntsize
		offset += txWitScriptcnt
	}

	wits = txwitraw[:offset]

	return
}

// striped tx
func NewRawTx(tx *model.Tx, rawtx []byte) (offset int) {
	binary.LittleEndian.PutUint32(rawtx[0:4], tx.Version)
	offset = 4

	txincntsize := utils.EncodeVarIntForBlock(uint64(tx.TxInCnt), rawtx[offset:])
	offset += txincntsize

	txoffset := 0
	for i := range tx.TxIns {
		txoffset = NewRawTxIn(&tx.TxIns[i], rawtx[offset:])
		offset += txoffset
	}

	txoutcntsize := utils.EncodeVarIntForBlock(uint64(tx.TxOutCnt), rawtx[offset:])
	offset += txoutcntsize

	for i := range tx.TxOuts {
		txoffset = NewRawTxOut(&tx.TxOuts[i], rawtx[offset:])
		offset += txoffset
	}

	binary.LittleEndian.PutUint32(rawtx[offset:offset+4], tx.LockTime)
	offset += 4
	return
}

func NewRawTxIn(txin *model.TxIn, txinraw []byte) (offset int) {
	copy(txinraw[0:36], txin.InputOutpoint)
	offset = 36
	txinraw[offset] = 0x00
	offset += 1
	binary.LittleEndian.PutUint32(txinraw[offset:offset+4], txin.Sequence)
	offset += 4
	return
}

func NewRawTxOut(txout *model.TxOut, txoutraw []byte) (offset int) {
	binary.LittleEndian.PutUint64(txoutraw[0:8], txout.Satoshi)
	offset = 8

	if len(txout.PkScript) == 0 {
		txoutraw[offset] = 0x00
		offset += 1
	} else if scriptDecoder.IsOpreturn(txout.PkScript) {
		txoutraw[offset] = 0x01
		txoutraw[offset+1] = 0x6a
		offset += 2
	} else {
		pkscript := len(txout.PkScript)
		pkscriptsize := utils.EncodeVarIntForBlock(uint64(pkscript), txoutraw[offset:])
		offset += pkscriptsize

		copy(txoutraw[offset:offset+pkscript], txout.PkScript)
		offset += pkscript
	}
	return
}

// preScanTxCounts quickly scans raw block data to count total inputs, outputs, and pkscript bytes.
// Pure pointer arithmetic, zero allocations.
func preScanTxCounts(txsraw []byte, txcnt uint) (totalIns, totalOuts, totalPkBytes uint) {
	offset := uint(0)
	for i := uint(0); i < txcnt; i++ {
		offset += 4 // version
		isWit := txsraw[offset] == 0 && txsraw[offset+1] == 1
		if isWit {
			offset += 2
		}
		incnt, incntsize := utils.DecodeVarIntForBlock(txsraw[offset:])
		offset += incntsize
		totalIns += incnt
		for j := uint(0); j < incnt; j++ {
			offset += 36 // hash(32) + vout(4)
			scriptsig, scriptsigsize := utils.DecodeVarIntForBlock(txsraw[offset:])
			offset += scriptsigsize + scriptsig + 4 // scriptsig + sequence
		}
		outcnt, outcntsize := utils.DecodeVarIntForBlock(txsraw[offset:])
		offset += outcntsize
		totalOuts += outcnt
		for j := uint(0); j < outcnt; j++ {
			offset += 8 // value
			pkscript, pkscriptsize := utils.DecodeVarIntForBlock(txsraw[offset:])
			totalPkBytes += pkscript
			offset += pkscriptsize + pkscript
		}
		if isWit {
			for j := uint(0); j < incnt; j++ {
				witcnt, witcntsize := utils.DecodeVarIntForBlock(txsraw[offset:])
				offset += witcntsize
				for k := uint(0); k < witcnt; k++ {
					witlen, witlensize := utils.DecodeVarIntForBlock(txsraw[offset:])
					offset += witlensize + witlen
				}
			}
		}
		offset += 4 // locktime
	}
	return
}
