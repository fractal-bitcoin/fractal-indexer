package model

type BlockDO struct {
	Height      uint32 `db:"height"`
	Version     uint32 `db:"version"`
	BlockId     []byte `db:"blkid"`
	PrevBlockId []byte `db:"previd"`
	NextBlockId []byte `db:"nextid"`
	MerkleRoot  []byte `db:"merkle"`
	TxCount     uint32 `db:"ntx"`
	InSatoshi   uint64 `db:"invalue"`
	OutSatoshi  uint64 `db:"outvalue"`
	CoinbaseOut uint64 `db:"coinbase_out"`
	BlockTime   uint32 `db:"blocktime"`
	Bits        uint32 `db:"bits"`
	BlockSize   uint32 `db:"blocksize"`
}

type TxDO struct {
	TxId         []byte `db:"txid"`
	InCount      uint32 `db:"nin"`
	OutCount     uint32 `db:"nout"`
	TxSize       uint32 `db:"txsize"`
	TxWitOffset  uint32 `db:"witoffset"`
	LockTime     uint32 `db:"locktime"`
	InSatoshi    uint64 `db:"invalue"`
	OutSatoshi   uint64 `db:"outvalue"`
	NFTNewCount  uint64 `db:"nftnew"`
	NFTInCount   uint64 `db:"nftin"`
	NFTOutCount  uint64 `db:"nftout"`
	NFTLostCount uint64 `db:"nftlost"`
	BlockTime    uint32 `db:"blocktime"`
	Height       uint32 `db:"height"`
	BlockId      []byte `db:"blkid"`
	Idx          uint32 `db:"idx"`
}

type TxInSpentDO struct {
	Height uint32 `db:"height"`
	TxId   []byte `db:"txid"`
	Idx    uint32 `db:"idx"`
	UtxId  []byte `db:"utxid"`
	Vout   uint32 `db:"vout"`
}

type TxInDO struct {
	Height     uint32 `db:"height"`
	TxId       []byte `db:"txid"`
	Idx        uint32 `db:"idx"`
	ScriptSig  []byte `db:"script_sig"`
	ScriptWits []byte `db:"script_wits"`
	Sequence   uint32 `db:"nsequence"`

	HeightTxo  uint32 `db:"height_txo"`
	UtxId      []byte `db:"utxid"`
	Vout       uint32 `db:"vout"`
	Address    []byte `db:"address"`
	Satoshi    uint64 `db:"satoshi"`
	ScriptType []byte `db:"script_type"`
	ScriptPk   []byte `db:"script_pk"`

	NFTCount          int64             `db:"nftin"`
	CreatePointOfNFTs []*NFTCreatePoint `db:"nftpoints"`
	ValuableNFTCount  int64             `db:"v_nftin"`
}

type TxOutDO struct {
	TxId       []byte `db:"txid"`
	Vout       uint32 `db:"vout"`
	Address    []byte `db:"address"`
	Satoshi    uint64 `db:"satoshi"`
	ScriptType []byte `db:"script_type"`
	ScriptPk   []byte `db:"script_pk"`
	Height     uint32 `db:"height"`
	Idx        uint32 `db:"txidx"`

	NFTCount          int64             `db:"nftout"`
	CreatePointOfNFTs []*NFTCreatePoint `db:"nftpoints"`
	ValuableNFTCount  int64             `db:"v_nftout"`
}

type TxOutHistoryDO struct {
	TxOutDO
	BlockTime uint32 `db:"blocktime"`
	IOType    uint8  `db:"io_type"` // 0: input; 1: output
}

type TxOutStatusDO struct {
	TxOutDO

	TxIdSpent   []byte `db:"txid_spent"`
	HeightSpent uint32 `db:"height_spent"`
}

type BlockFeeRateDO struct {
	Height    uint32  `db:"height"`
	BlockTime uint32  `db:"blocktime"`
	FeeRate   float64 `db:"feerate"`
}

type BlockHashDO struct {
	Height  uint32 `db:"height"`
	BlockId string `db:"blkid"`
}
