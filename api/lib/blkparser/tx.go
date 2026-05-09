package blkparser

import (
	"encoding/binary"
	"fractal-query/model"
)

func NewTx(rawtx []byte) (tx *model.Tx, offset uint) {
	tx = new(model.Tx)
	tx.Version = binary.LittleEndian.Uint32(rawtx[0:4])
	offset = 4

	// check witness
	isWit := false
	if rawtx[4] == 0 && rawtx[5] == 1 {
		isWit = true
		offset += 2
	}

	txincnt, txincntsize := DecodeVarIntForBlock(rawtx[offset:])
	offset += txincntsize

	tx.TxInCnt = uint32(txincnt)
	tx.TxIns = make([]*model.TxIn, txincnt)

	txoffset := uint(0)
	for i := range tx.TxIns {
		tx.TxIns[i], txoffset = NewTxIn(rawtx[offset:])
		offset += txoffset
	}

	txoutcnt, txoutcntsize := DecodeVarIntForBlock(rawtx[offset:])
	offset += txoutcntsize

	tx.TxOutCnt = uint32(txoutcnt)
	tx.TxOuts = make([]*model.TxOut, txoutcnt)
	for i := range tx.TxOuts {
		tx.TxOuts[i], txoffset = NewTxOut(rawtx[offset:])
		offset += txoffset
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

func NewTxIn(txinraw []byte) (txin *model.TxIn, offset uint) {
	txin = new(model.TxIn)
	txin.InputHash = make([]byte, 32)
	copy(txin.InputHash, txinraw[0:32])
	txin.InputHashHex = HashString(txin.InputHash)
	txin.InputVout = binary.LittleEndian.Uint32(txinraw[32:36])
	offset = 36

	scriptsig, scriptsigsize := DecodeVarIntForBlock(txinraw[offset:])
	offset += scriptsigsize

	txin.ScriptSig = make([]byte, scriptsig)
	copy(txin.ScriptSig, txinraw[offset:offset+scriptsig])
	offset += scriptsig

	txin.Sequence = binary.LittleEndian.Uint32(txinraw[offset : offset+4])
	offset += 4

	// process Parallel
	txin.InputOutpointKey = string(txinraw[0:36])
	txin.InputOutpoint = make([]byte, 36)
	copy(txin.InputOutpoint, txinraw[0:36])
	return
}

func NewTxOut(txoutraw []byte) (txout *model.TxOut, offset uint) {
	txout = new(model.TxOut)
	txout.Satoshi = binary.LittleEndian.Uint64(txoutraw[0:8])
	offset = 8

	pkscript, pkscriptsize := DecodeVarIntForBlock(txoutraw[offset:])
	offset += pkscriptsize

	txout.PkScript = make([]byte, pkscript)
	copy(txout.PkScript, txoutraw[offset:offset+pkscript])

	offset += pkscript
	return
}

func NewTxWits(txwitraw []byte) (wits []byte, offset uint) {
	txWitcnt, txWitcntsize := DecodeVarIntForBlock(txwitraw[0:])
	offset = txWitcntsize

	for witIndex := uint(0); witIndex < txWitcnt; witIndex++ {
		txWitScriptcnt, txWitScriptcntsize := DecodeVarIntForBlock(txwitraw[offset:])
		offset += txWitScriptcntsize
		offset += txWitScriptcnt
	}

	wits = txwitraw[:offset]

	return
}
