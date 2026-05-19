package model

import (
	"encoding/binary"
	"encoding/json"
	"fractal-indexer/api/constant"
	scriptDecoder "fractal-indexer/api/lib/blkparser/script"

	"github.com/golang/snappy"
)

type TxRequest struct {
	TxHex   string `json:"txHex"`
	ByTxHex string `json:"byTxHex"`
}

type TxResponse struct {
	TxId    string `json:"txId"`
	Index   int    `json:"index"`
	ByTxId  string `json:"byTxId"`
	Sig     string `json:"sigBE"`
	Padding string `json:"padding"`
	Payload string `json:"payload"`
}

type Utxo struct {
	Txid  string `json:"txid"`
	Index uint32 `json:"index"`
}

type UnlockUnavailableUtxoReq struct {
	OutpointList []Utxo `json:"outpointList"`
	Address      string `json:"address"`
	Signature    string `json:"signature"`
	Pubkey       string `json:"pubkey"`
	Nonce        int    `json:"nonce"`
}

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func (t *Response) MarshalJSON() ([]byte, error) {
	return json.Marshal(*t)
}

// nft create point on create
type NFTCreatePoint struct {
	IsStrip bool // the NFT is strip

	Height     uint32 // Height of NFT show in block onCreate
	IdxInBlock uint32 // Index of NFT show in block onCreate
	Offset     uint64 // sat offset in utxo
	Sequence   uint16 // sequence>0 the NFT has been moved after created

	IsCursed    bool // the NFT is IsCursed
	IsVindicate bool
	IsBRC20Tran bool // the NFT is BRC20 transfer
	IsBRC20Mint bool // the NFT is BRC20 mint
	IsBRC20Ext  bool // the NFT is BRC20
	IsBRC20     bool // the NFT is BRC20
	IsText      bool // the NFT is Text

	ContentType []byte
	Content     []byte
}

func (p *NFTCreatePoint) GetCreateIdxKey() uint64 {
	return uint64(p.Height)<<constant.HEIGHT_MUTIPLY_NBIT + uint64(p.IdxInBlock)
}

type NewInscriptionInfo struct {
	NFTData       scriptDecoder.NFTData // type/data
	CreatePoint   NFTCreatePoint
	TxIdx         uint32 // txidx in block
	TxId          []byte // create txid
	IdxInTx       uint32 // nft idx inside tx
	InTxVout      uint32 // nft outgoing(vout) inside tx
	InputsValue   uint64
	OutputsValue  uint64
	Ordinal       uint64
	Number        uint64
	BlockTime     uint32
	ContentLength uint32
}

func (d *NewInscriptionInfo) LoadString(data []byte) bool {
	if len(data) < 108 {
		return false
	}
	d.CreatePoint.Height = binary.LittleEndian.Uint32(data[0:4])
	d.BlockTime = binary.LittleEndian.Uint32(data[4:8])
	d.InputsValue = binary.LittleEndian.Uint64(data[8:16])
	d.OutputsValue = binary.LittleEndian.Uint64(data[16:24])
	d.Ordinal = binary.LittleEndian.Uint64(data[24:32])
	d.Number = binary.LittleEndian.Uint64(data[32:40])

	d.TxId = data[40:72]

	d.IdxInTx = binary.LittleEndian.Uint32(data[72:76])
	d.NFTData.InTxVin = binary.LittleEndian.Uint32(data[76:80])
	d.NFTData.Pointer = binary.LittleEndian.Uint64(data[80:88])

	flags := binary.LittleEndian.Uint32(data[104:108])
	d.NFTData.SetFlags(flags)

	offset := 108
	length := int(binary.LittleEndian.Uint16(data[98:100]))
	d.NFTData.ContentType = data[offset : offset+length]
	offset += length

	if d.NFTData.HasParent {
		length := int(data[88])
		d.NFTData.Parent = data[offset : offset+length]
		offset += length
	}
	if d.NFTData.HasDeligate {
		length := int(data[89])
		d.NFTData.Deligate = data[offset : offset+length]
		offset += length
	}
	if d.NFTData.HasMetaProtocal {
		length := int(binary.LittleEndian.Uint16(data[90:92]))
		d.NFTData.MetaProtocol = data[offset : offset+length]
		offset += length
	}
	if d.NFTData.HasMetadata {
		length := int(binary.LittleEndian.Uint32(data[92:96]))
		d.NFTData.Metadata = data[offset : offset+length]
		offset += length
	}
	if d.NFTData.HasContentEncoding {
		length := int(binary.LittleEndian.Uint16(data[96:98]))
		d.NFTData.ContentEncoding = data[offset : offset+length]
		offset += length
	}

	d.ContentLength = binary.LittleEndian.Uint32(data[100:104])

	return true
}

// //////////////
type TxoData struct {
	UTxid       []byte
	Vout        uint32
	BlockHeight uint32
	TxIdx       uint32 // txidx in block
	Satoshi     uint64
	PkScript    []byte
	ScriptType  []byte
	OpInRBF     bool

	CreatePointOfNFTs []*NFTCreatePoint
}

func (d *TxoData) Unmarshal(zbuf []byte) bool {
	if len(zbuf) < 4 {
		return false
	}

	zipflag := binary.LittleEndian.Uint32(zbuf[:4]) // 4
	offset := 4

	buf := zbuf
	if zipflag != 0 {
		var err error
		buf, err = snappy.Decode(nil, zbuf)
		if err != nil {
			return false
		}
	}

	if len(buf) < 4+20 {
		return false
	}

	d.BlockHeight = binary.LittleEndian.Uint32(buf[offset:]) // 4
	offset += 4

	d.TxIdx = binary.LittleEndian.Uint32(buf[offset:]) // 4
	offset += 4

	d.Satoshi = binary.LittleEndian.Uint64(buf[offset:]) // 8
	offset += 8

	scriptSize := binary.LittleEndian.Uint32(buf[offset:]) // 4
	offset += 4

	if len(buf) < 4+20+int(scriptSize) {
		return false
	}

	d.PkScript = make([]byte, scriptSize)
	copy(d.PkScript, buf[offset:offset+int(scriptSize)])
	offset += int(scriptSize)

	if d.BlockHeight == constant.MEMPOOL_HEIGHT {
		if buf[offset] == 0x01 {
			d.OpInRBF = true
		}
		offset += 1
	}

	offset += d.LoadNFTCreatePointsFromRaw(buf[offset:])
	return true
}

func (d *TxoData) UnmarshalWithoutInscriptions(zbuf []byte) bool {
	if len(zbuf) < 4 {
		return false
	}

	zipflag := binary.LittleEndian.Uint32(zbuf[:4]) // 4
	offset := 4

	buf := zbuf
	if zipflag != 0 {
		var err error
		buf, err = snappy.Decode(nil, zbuf)
		if err != nil {
			return false
		}
	}

	if len(buf) < 4+20 {
		return false
	}

	d.BlockHeight = binary.LittleEndian.Uint32(buf[offset:]) // 4
	offset += 4

	d.TxIdx = binary.LittleEndian.Uint32(buf[offset:]) // 4
	offset += 4

	d.Satoshi = binary.LittleEndian.Uint64(buf[offset:]) // 8
	offset += 8

	scriptSize := binary.LittleEndian.Uint32(buf[offset:]) // 4
	offset += 4

	if len(buf) < 4+20+int(scriptSize) {
		return false
	}

	d.PkScript = make([]byte, scriptSize)
	copy(d.PkScript, buf[offset:offset+int(scriptSize)])
	offset += int(scriptSize)

	if d.BlockHeight == constant.MEMPOOL_HEIGHT {
		if buf[offset] == 0x01 {
			d.OpInRBF = true
		}
		offset += 1
	}

	return true
}

// load nft
func (d *TxoData) LoadNFTCreatePointsFromRaw(buf []byte) (offset int) {
	lastOffset := int64(0)
	for {
		if len(buf[offset:]) == 0 {
			return
		}

		isStrip := false

		isCursed := false
		isVindicate := false
		isBRC20Tran := false
		isBRC20Mint := false
		isBRC20Ext := false
		isBRC20 := false
		isText := false

		var flag byte = buf[offset]

		if flag >= 0x80 {
			isStrip = true
			flag -= 0x80
		}

		if flag >= 0x40 {
			isCursed = true
			flag -= 0x40
		}
		if flag >= 0x20 {
			isVindicate = true
			flag -= 0x20
		}

		if flag == 0x0f {
			isBRC20Tran = true
			isBRC20 = true
			isText = true
		} else if flag == 0x0b {
			isBRC20Mint = true
			isBRC20 = true
			isText = true
		} else if flag == 0x07 {
			isBRC20Ext = true
			isBRC20 = true
			isText = true
		} else if flag == 0x03 {
			isBRC20 = true
			isText = true
		} else if flag == 0x01 {
			isText = true
		}
		offset += 1

		nft := &NFTCreatePoint{
			IsStrip:     isStrip,
			IsCursed:    isCursed,
			IsVindicate: isVindicate,
			IsBRC20Tran: isBRC20Tran,
			IsBRC20Mint: isBRC20Mint,
			IsBRC20Ext:  isBRC20Ext,
			IsBRC20:     isBRC20,
			IsText:      isText,
		}

		span := int64(0)
		if isStrip {
			if 8 > len(buf[offset:]) {
				return
			}

			span = int64(binary.LittleEndian.Uint64(buf[offset:])) // 8
			offset += 8

			nft.Sequence = 1 // always moved
		} else {
			if 18 > len(buf[offset:]) {
				return
			}

			nft.Height = binary.LittleEndian.Uint32(buf[offset:]) // 4
			offset += 4

			nft.IdxInBlock = binary.LittleEndian.Uint32(buf[offset:]) // 4
			offset += 4

			span = int64(binary.LittleEndian.Uint64(buf[offset:])) // 8
			offset += 8

			nft.Sequence = binary.LittleEndian.Uint16(buf[offset:]) // 2
			offset += 2
		}

		lastOffset += span
		nft.Offset = uint64(lastOffset)

		d.CreatePointOfNFTs = append(d.CreatePointOfNFTs, nft)
	}
}

// load nft limit 500
func (d *TxoData) LoadNFTCreatePointsFromRawLimit500(buf []byte) (offset int) {
	lastOffset := int64(0)
	for {
		if len(buf[offset:]) == 0 {
			return
		}

		isStrip := false

		isCursed := false
		isVindicate := false
		isBRC20Tran := false
		isBRC20Mint := false
		isBRC20Ext := false
		isBRC20 := false
		isText := false

		var flag byte = buf[offset]

		if flag >= 0x80 {
			isStrip = true
			flag -= 0x80
		}

		if flag >= 0x40 {
			isCursed = true
			flag -= 0x40
		}
		if flag >= 0x20 {
			isVindicate = true
			flag -= 0x20
		}

		if flag == 0x0f {
			isBRC20Tran = true
			isBRC20 = true
			isText = true
		} else if flag == 0x0b {
			isBRC20Mint = true
			isBRC20 = true
			isText = true
		} else if flag == 0x07 {
			isBRC20Ext = true
			isBRC20 = true
			isText = true
		} else if flag == 0x03 {
			isBRC20 = true
			isText = true
		} else if flag == 0x01 {
			isText = true
		}
		offset += 1

		nft := &NFTCreatePoint{
			IsStrip:     isStrip,
			IsCursed:    isCursed,
			IsVindicate: isVindicate,
			IsBRC20Tran: isBRC20Tran,
			IsBRC20Mint: isBRC20Mint,
			IsBRC20Ext:  isBRC20Ext,
			IsBRC20:     isBRC20,
			IsText:      isText,
		}

		span := int64(0)
		if isStrip {
			if 8 > len(buf[offset:]) {
				return
			}

			span = int64(binary.LittleEndian.Uint64(buf[offset:])) // 8
			offset += 8

			nft.Sequence = 1 // always moved
		} else {
			if 18 > len(buf[offset:]) {
				return
			}

			nft.Height = binary.LittleEndian.Uint32(buf[offset:]) // 4
			offset += 4

			nft.IdxInBlock = binary.LittleEndian.Uint32(buf[offset:]) // 4
			offset += 4

			span = int64(binary.LittleEndian.Uint64(buf[offset:])) // 8
			offset += 8

			nft.Sequence = binary.LittleEndian.Uint16(buf[offset:]) // 2
			offset += 2
		}

		lastOffset += span
		nft.Offset = uint64(lastOffset)

		if !isStrip {
			d.CreatePointOfNFTs = append(d.CreatePointOfNFTs, nft)
		}

		if len(d.CreatePointOfNFTs) >= 500 {
			return
		}
	}
}

func NewTxoData(outpoint, res []byte) (txout *TxoData) {
	txout = &TxoData{}
	txout.Unmarshal(res)

	// Supplemental data.
	txout.UTxid = outpoint[:32]                            // 32
	txout.Vout = binary.LittleEndian.Uint32(outpoint[32:]) // 4
	txout.ScriptType = scriptDecoder.GetLockingScriptType(txout.PkScript)
	return
}

func NewTxoDataWithoutInscriptions(outpoint, res []byte) (txout *TxoData) {
	txout = &TxoData{}
	txout.UnmarshalWithoutInscriptions(res)

	// Supplemental data.
	txout.UTxid = outpoint[:32]                            // 32
	txout.Vout = binary.LittleEndian.Uint32(outpoint[32:]) // 4
	txout.ScriptType = scriptDecoder.GetLockingScriptType(txout.PkScript)
	return
}
