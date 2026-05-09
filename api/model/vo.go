package model

import brc20Model "github.com/unisat-wallet/libbrc20-indexer/model"

type Welcome struct {
	Contact string `json:"contact"`
	Job     string `json:"job"`
	Github  string `json:"github"`
}

////////////////

type MempoolInfoResp struct {
	TxCount int `json:"ntx"` // Number of transactions in the mempool.
}

type BlockchainInfoResp struct {
	Chain         string `json:"chain"`         // main/test
	Blocks        int    `json:"blocks"`        // Latest block count.
	Headers       int    `json:"headers"`       // Latest block header count.
	BestBlockHash string `json:"bestBlockHash"` // Latest block ID.
	PrevBlockHash string `json:"prevBlockHash"` // Previous block ID.
	Difficulty    string `json:"difficulty"`
	MedianTime    int    `json:"medianTime"`
	Chainwork     string `json:"chainwork"`
}

type FractalSupplyResp struct {
	Blocks int `json:"blocks"` // Latest block count.
	Supply int `json:"supply"`
}

type BlockInfoResp struct {
	Height         int    `json:"height"`      // Current block height.
	Version        int    `json:"version"`     // Current block version.
	AuxPow         bool   `json:"auxpow"`      // Whether the current block is auxpow.
	BlockIdHex     string `json:"id"`          // Current block ID.
	PrevBlockIdHex string `json:"prev"`        // Previous block ID.
	NextBlockIdHex string `json:"next"`        // Next block ID.
	MerkleRootHex  string `json:"merkle"`      // Merkle Tree
	TxCount        int    `json:"ntx"`         // Number of transactions in the block.
	InSatoshi      int    `json:"inSatoshi"`   // Total input amount in the block, excluding the block reward.
	OutSatoshi     int    `json:"outSatoshi"`  // Total output amount in the block, excluding the block reward and fee.
	CoinbaseOut    int    `json:"coinbaseOut"` // Block reward.
	BlockTime      int    `json:"timestamp"`   // Block timestamp.
	Bits           int    `json:"bits"`
	BlockSize      int    `json:"size"` // Block size in bytes.
}

type TxOutResp struct {
	TxIdHex       string `json:"txid"`       // Current txid.
	Vout          int    `json:"vout"`       // Current output index.
	Address       string `json:"address"`    // Current output address.
	CodeType      int    `json:"codeType"`   // Contract type of the current output: 0 None, 1 FT, 2 Unique, 3 NFT.
	Satoshi       int    `json:"satoshi"`    // Satoshis in the current output.
	ScriptTypeHex string `json:"scriptType"` // Locking script type of the current output.
	ScriptPkHex   string `json:"scriptPk"`   // Locking script of the current output.
	Height        int    `json:"height"`     // Block height where the current transaction was mined.
	Idx           int    `json:"idx"`        // Index of the current transaction in the block.

	CreatePointOfNFTs []*NFTCreatePoint `json:"-"`
	NFTCount          int64             `json:"nftCount"`
	Inscriptions      []*NFTData        `json:"inscriptions"` // Positions of all inscriptions on the UTXO.
}

// nft create point on create
type NFTData struct {
	InscriptionNumber int64  `json:"inscriptionNumber"` // Current inscription number.
	InscriptionId     string `json:"inscriptionId"`     // Current inscription ID.
	Offset            uint64 `json:"offset"`            // sat offset in utxo

	IsStrip     bool   `json:"isStrip"`     // the NFT is strip
	HasMoved    bool   `json:"moved"`       // the NFT has been moved after created
	Sequence    uint16 `json:"sequence"`    // sequence>0 the NFT has been moved after created
	IsCursed    bool   `json:"isCursed"`    // the NFT is cursed
	IsVindicate bool   `json:"isVindicate"` // the NFT is vindicate
	IsBRC20Tran bool   `json:"isBRC20Tran"` // the NFT is BRC20 transfer
	IsBRC20Mint bool   `json:"isBRC20Mint"` // the NFT is BRC20 mint
	IsBRC20Ext  bool   `json:"isBRC20Ext"`  // the NFT is BRC20Ext
	IsBRC20     bool   `json:"isBRC20"`     // the NFT is BRC20

	ContentType string `json:"contentType"`
}

type TxStandardOutResp struct {
	TxIdHex       string `json:"txid"`       // Current txid.
	Vout          int    `json:"vout"`       // Current output index.
	Satoshi       int    `json:"satoshi"`    // Satoshis in the current output.
	ScriptTypeHex string `json:"scriptType"` // Locking script type of the current output.
	ScriptPkHex   string `json:"scriptPk"`   // Locking script of the current output.
	CodeType      int    `json:"codeType"`   // Script type of the current output: 0 None, 1 FT, 2 Unique, 3 NFT, 4 P2PK, 5 P2PKH, 6 P2SH, 7 P2WPKH, 8 P2WSH, 9 P2TR, 10 P2A.
	Address       string `json:"address"`    // Current output address.
	Height        int    `json:"height"`     // Block height where the current transaction was mined.
	TxIdx         int    `json:"idx"`        // Index of the spending transaction in its block.
	OpInRBF       bool   `json:"isOpInRBF"`  // Whether the current transaction is an RBF transaction.

	CreatePointOfNFTs []*NFTCreatePoint `json:"-"`
	InscriptionsCount int               `json:"inscriptionsCount"`
	Inscriptions      []*NFTData        `json:"inscriptions"` // Positions of all inscriptions on the UTXO.
}

type InscriptionResp struct {
	UtxoOutpoint string             `json:"-"`       //
	UTXO         *TxStandardOutResp `json:"utxo"`    // UTXO result.
	Address      string             `json:"address"` // Current output address.
	CreateIdxKey uint64             `json:"-"`       //

	Offset            uint64 `json:"offset"`            // sat offset in utxo
	InscriptionIndex  uint64 `json:"inscriptionIndex"`  // Current inscription index; duplicate number, where 0 means first occurrence.
	InscriptionNumber int64  `json:"inscriptionNumber"` // Current inscription number.
	InscriptionId     string `json:"inscriptionId"`     // Current inscription ID.

	IsStrip            bool   `json:"isStrip"`
	HasPointer         bool   `json:"hasPointer"`
	HasParent          bool   `json:"hasParent"`
	HasDeligate        bool   `json:"hasDeligate"`
	HasMetaProtocal    bool   `json:"hasMetaProtocal"`
	HasMetadata        bool   `json:"hasMetadata"`
	HasContentEncoding bool   `json:"hasContentEncoding"`
	Pointer            uint64 `json:"pointer"`
	Parent             string `json:"parent"`
	Deligate           string `json:"deligate"`
	MetaProtocol       string `json:"metaprotocol"`
	Metadata           string `json:"metadata"`
	ContentEncoding    string `json:"contentEncoding"`

	ContentType   string `json:"contentType"`   //
	ContentLength int    `json:"contentLength"` //
	ContentBody   string `json:"contentBody"`   //
	Height        uint32 `json:"height"`        // Height where the current inscription was mined.
	IdxInBlock    uint32 `json:"idxInBlock"`    // Index of NFT show in block onCreate
	BlockTime     int    `json:"timestamp"`     // Block timestamp.
	InSatoshi     int    `json:"inSatoshi"`     // Total input amount in GenesisTx.
	OutSatoshi    int    `json:"outSatoshi"`    // Total output amount in GenesisTx.

	BRC20           *brc20Model.InscriptionBRC20InfoResp `json:"brc20"`           // BRC-20 information; included only for a valid transfer.
	BRC20BestHeight uint32                               `json:"brc20BestHeight"` // for /inscription/info/:id only

	Detail *InscriptionContentForCheckResp `json:"detail"`
}

// feerate estimate
type BlockFeeRateResp struct {
	Height    int     `json:"height"`    // Current block height.
	BlockTime int     `json:"timestamp"` // Block timestamp.
	FeeRate   float64 `json:"feerate"`   // Average block fee rate.
}

type BlockFeeRateEstimateResp struct {
	Blocks  int     `json:"blocks"`  // Expected confirmation blocks.
	FeeRate float64 `json:"feerate"` // Fee rate.
}

type BlockFeeInfoResp struct {
	BlocksFeeRateEstimate []*BlockFeeRateEstimateResp
	BlocksAvgFeeRate      []*BlockFeeRateResp
	BTCPrice              float64
}
