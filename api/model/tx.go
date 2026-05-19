package model

import scriptDecoder "fractal-indexer/api/lib/blkparser/script"

type Tx struct {
	Raw          []byte
	TxIdHex      string // 64
	TxId         []byte // 32
	Size         uint32
	WitOffset    uint32
	LockTime     uint32
	Version      uint32
	TxInCnt      uint32
	TxOutCnt     uint32
	InputsValue  uint64
	OutputsValue uint64
	TxIns        []*TxIn
	TxOuts       []*TxOut

	NewNFTDataCreated []*scriptDecoder.NFTData
	NFTInputsCnt      uint64
	NFTOutputsCnt     uint64
	NFTLostCnt        uint64

	OpInRBF       bool
	GenesisNewNFT bool
}

type TxIn struct {
	InputHashHex string // 32
	InputHash    []byte // 32
	InputVout    uint32
	ScriptSig    []byte
	Sequence     uint32

	ScriptWitness []byte

	InputOutpointKey string // 32 + 4
	InputOutpoint    []byte // 32 + 4
	InputPoint       []byte // 32 + 4
}

type TxOut struct {
	Satoshi  uint64
	PkScript []byte

	Outpoint      []byte // 32 + 4
	OutpointKey   string // 32 + 4
	ScriptType    []byte
	ScriptTypeHex string

	AddressData *scriptDecoder.AddressData

	LockingScriptUnspendable bool
}

type TxWit struct {
	Script []byte
}
