package model

import (
	"encoding/binary"
	"encoding/hex"
	"fractal-indexer/constant"
	scriptDecoder "fractal-indexer/parser/script"
	"strconv"

	"github.com/golang/snappy"
	"go.uber.org/zap/zapcore"
)

const HEIGHT_MUTIPLY_NBIT = 20
const TXO_DATA_INSCRIPTION_SIZE = 19

// Channel-based pools — immune to GC clearing unlike sync.Pool.
// Buffer 128 > max pipeline depth (64 parallel + 1 serial end).
var (
	txSlabChan      = make(chan []Tx, 128)
	txInSlabChan    = make(chan []TxIn, 128)
	txOutSlabChan   = make(chan []TxOut, 128)
	nftDataSlabChan = make(chan []scriptDecoder.NFTData, 128)

	// snappy decode scratch buffer pool — lifetime is strictly within Unmarshal.
	snappyDecodeBufChan = make(chan []byte, 128)
)

func GetTxSlab() []Tx {
	select {
	case s := <-txSlabChan:
		return s
	default:
		return nil
	}
}

func GetTxInSlab() []TxIn {
	select {
	case s := <-txInSlabChan:
		return s
	default:
		return nil
	}
}

func GetTxOutSlab() []TxOut {
	select {
	case s := <-txOutSlabChan:
		return s
	default:
		return nil
	}
}

func GetNFTDataSlab() []scriptDecoder.NFTData {
	select {
	case s := <-nftDataSlabChan:
		return s
	default:
		return nil
	}
}

// PutTxSlabs returns Tx/TxIn/TxOut slabs to pool for reuse.
func PutTxSlabs(txSlab []Tx, inSlab []TxIn, outSlab []TxOut) {
	if txSlab != nil {
		clear(txSlab)
		select {
		case txSlabChan <- txSlab[:0]:
		default:
		}
	}
	if inSlab != nil {
		clear(inSlab)
		select {
		case txInSlabChan <- inSlab[:0]:
		default:
		}
	}
	if outSlab != nil {
		clear(outSlab)
		select {
		case txOutSlabChan <- outSlab[:0]:
		default:
		}
	}
}

// PutNFTDataSlabs returns NFTData slabs to pool for reuse.
func PutNFTDataSlabs(slabs [][]scriptDecoder.NFTData) {
	for _, slab := range slabs {
		if slab == nil {
			continue
		}
		clear(slab)
		select {
		case nftDataSlabChan <- slab[:0]:
		default:
		}
	}
}

type Tx struct {
	Raw          []byte
	TxId         []byte // 32
	Size         uint32
	WitOffset    uint32
	LockTime     uint32
	Version      uint32
	TxInCnt      uint32
	TxOutCnt     uint32
	InputsValue  uint64
	OutputsValue uint64
	TxIns        []TxIn
	TxOuts       []TxOut

	NewNFTDataCreated []scriptDecoder.NFTData
	NFTInputsCnt      uint64
	NFTOutputsCnt     uint64
	NFTLostCnt        uint64

	OpInRBF       bool
	GenesisNewNFT bool
	IsLowFee      bool
}

type TxIn struct {
	InputHash []byte // 32
	InputVout uint32
	ScriptSig []byte
	Sequence  uint32

	ScriptWitness []byte

	// other:
	CreatePointOfNFTs         []NFTCreatePoint // input NFTs
	CreatePointCountOfNewNFTs uint32           // newly created NFTs; duplicates are marked invalid

	InputOutpointKey string // short key: txid[:16] + variable vout
	InputOutpoint    []byte // 32 + 4
}

func (t *TxIn) InputHashHex() string {
	const length = 32
	var rev [length]byte
	for i := 0; i < length; i++ {
		rev[i] = t.InputHash[length-i-1]
	}
	return hex.EncodeToString(rev[:])
}

func (t *Tx) TxIdHex() string {
	const length = 32
	var rev [length]byte
	for i := 0; i < length; i++ {
		rev[i] = t.TxId[length-i-1]
	}
	return hex.EncodeToString(rev[:])
}

func (t *TxIn) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("t", t.InputHashHex())
	enc.AddUint32("i", t.InputVout)
	return nil
}

type TxOut struct {
	Satoshi  uint64
	PkScript []byte

	OutpointKey string // short key: txid[:16] + variable vout
	ScriptType  []byte

	CreatePointOfNFTs []NFTCreatePoint

	LockingScriptUnspendable bool
}

type TxWit struct {
	Script []byte
}

type Block struct {
	Raw          []byte
	RawBlockBuf  []byte // original buffer for pool return
	Hash         []byte // 32 bytes
	HashHex      string
	Height       uint32
	Txs          []Tx
	TxSlab       []Tx                      // for pool return (may differ from Txs if pooled)
	TxInSlab     []TxIn                    // for pool return
	TxOutSlab    []TxOut                   // for pool return
	NFTDataSlabs [][]scriptDecoder.NFTData // for pool return
	Version      uint32
	MerkleRoot   []byte // 32 bytes
	BlockTime    uint32
	Bits         uint32
	Nonce        uint32
	Size         uint32
	TxCnt        uint32
	Parent       []byte // 32 bytes
	ParentHex    string
	ParseData    *ProcessBlock
}

type BlockIndex struct {
	Height    uint32
	HashHex   string
	ParentHex string
}

type BlockIndexInfo struct {
	Height  uint32
	HashHex string
}

// //////////////
type ProcessBlock struct {
	Height             uint32
	NftEndNumber       int64
	NftCursedEndNumber int64

	SpentUtxoKeysMap     map[string]struct{}
	SpentUtxoDataMap     map[string]*TxoData
	NewUtxoDataMap       map[string]*TxoData
	NewInscriptions      []*NewInscriptionInfo // index: createBlockNFTIndex;  nft: IncriptionID
	NewEventInscriptions []*NewInscriptionInfo
	NftTransferCount     int // total inscription transfers (including stripped/dropped)

	// Prefetched and deserialized UTXO data from Redis, populated in parallel stage.
	SpentUtxoPrefetch map[string]*TxoData
}

type NewInscriptionInfo struct {
	NFTData      *scriptDecoder.NFTData // type/data
	CreatePoint  NFTCreatePoint
	Height       uint32 // current height
	TxIdx        uint32 // current txidx in block
	TxId         []byte // current txid
	IdxInTx      uint32 // nft idx inside tx
	InTxVout     uint32 // nft outgoing(vout) inside tx
	InputsValue  uint64
	OutputsValue uint64
	Satoshi      uint64
	PkScript     []byte
	PkScriptFrom []byte
	InputIdx     uint32
	Ordinal      uint64
	Number       int64
	IsDefer      bool
	BlockTime    uint32
}

func (d *NewInscriptionInfo) GetId() string {
	const length = 32
	var rev [length]byte
	for i := 0; i < length; i++ {
		rev[i] = d.TxId[length-1-i]
	}
	var buf [75]byte // 64 hex + 'i' + max 10 digits
	n := hex.Encode(buf[:], rev[:])
	buf[n] = 'i'
	result := strconv.AppendUint(buf[:n+1], uint64(d.IdxInTx), 10)
	return string(result)
}

func (d *NewInscriptionInfo) Copy() (c *NewInscriptionInfo) {
	c = &NewInscriptionInfo{
		NFTData:      d.NFTData,
		CreatePoint:  d.CreatePoint,
		Height:       d.Height,
		TxIdx:        d.TxIdx,
		TxId:         d.TxId,
		IdxInTx:      d.IdxInTx,
		InTxVout:     d.InTxVout,
		InputsValue:  d.InputsValue,
		OutputsValue: d.OutputsValue,
		Satoshi:      d.Satoshi,
		PkScript:     d.PkScript,
		Ordinal:      d.Ordinal,
		Number:       d.Number,
		BlockTime:    d.BlockTime,
	}
	return c
}

func (d *NewInscriptionInfo) DumpString() (ret string) {
	var data [108]byte
	binary.LittleEndian.PutUint32(data[0:4], d.CreatePoint.Height) // fixme: may nil
	binary.LittleEndian.PutUint32(data[4:8], d.BlockTime)
	binary.LittleEndian.PutUint64(data[8:16], d.InputsValue)
	binary.LittleEndian.PutUint64(data[16:24], d.OutputsValue)
	binary.LittleEndian.PutUint64(data[24:32], d.Ordinal)
	binary.LittleEndian.PutUint64(data[32:40], uint64(d.Number))

	copy(data[40:72], d.TxId[:])

	binary.LittleEndian.PutUint32(data[72:76], d.IdxInTx)
	binary.LittleEndian.PutUint32(data[76:80], d.NFTData.InTxVin)
	binary.LittleEndian.PutUint64(data[80:88], d.NFTData.Pointer)

	parents := d.NFTData.GetParentsData()
	data[88] = uint8(len(parents))
	data[89] = uint8(len(d.NFTData.Deligate))

	binary.LittleEndian.PutUint16(data[90:92], uint16(len(d.NFTData.MetaProtocol)))
	binary.LittleEndian.PutUint32(data[92:96], uint32(len(d.NFTData.Metadata)))
	binary.LittleEndian.PutUint16(data[96:98], uint16(len(d.NFTData.ContentEncoding)))
	binary.LittleEndian.PutUint16(data[98:100], uint16(len(d.NFTData.ContentType)))
	binary.LittleEndian.PutUint32(data[100:104], uint32(len(d.NFTData.ContentBody)))

	binary.LittleEndian.PutUint32(data[104:108], d.NFTData.GetFlags()) // flags

	ret = string(data[:]) + string(d.NFTData.ContentType)
	if d.NFTData.HasParent {
		ret += string(parents[:])
	}
	if d.NFTData.HasDeligate {
		ret += string(d.NFTData.Deligate[:])
	}
	if d.NFTData.HasMetaProtocal {
		ret += string(d.NFTData.MetaProtocol[:])
	}
	if d.NFTData.HasMetadata {
		ret += string(d.NFTData.Metadata[:])
	}
	if d.NFTData.HasContentEncoding {
		ret += string(d.NFTData.ContentEncoding[:])
	}
	return ret
}

type TxData struct {
	Raw  []byte
	TxId []byte // 32
}

// nft create point on create, in utxo
type NFTCreatePoint struct {
	Height      uint32 // Height of NFT show in block onCreate
	IdxInBlock  uint32 // Index of NFT show in block onCreate
	Offset      uint64 // sat offset in utxo
	Sequence    uint16 // sequence>0 the NFT has been moved after created
	IsStrip     bool   // the NFT is strip
	IsCursed    bool   // the NFT is IsCursed
	IsVindicate bool
	IsBRC20Tran bool // the NFT is BRC20Mint, for mint/transfer
	IsBRC20Mint bool // the NFT is BRC20Mint, for mint/transfer
	IsBRC20Ext  bool // the NFT is BRC20Ext, for conditional-approve
	IsBRC20     bool // the NFT is BRC20, for brc20-base, brc20-module, brc20-swap, but without confitional-approve
	IsText      bool // the NFT is Text
}

func (p *NFTCreatePoint) IsStripBRC20Event() bool {
	if p.IsBRC20Mint { // mint does not record move events
		return true
	}

	if !p.IsBRC20 { // non-BRC20 records all move events
		return false
	}

	if p.IsBRC20Ext { // cond-approve records all move events
		return false
	}

	if p.Sequence > 1 { // other BRC20 records only one move event
		return true
	} else {
		return false
	}

}

func (p *NFTCreatePoint) IsStripBRC20EventForCoinbase() bool {
	if p.IsBRC20 {
		return true
	}
	return false
}

func (p *NFTCreatePoint) SetCreatePointFlags(nft *scriptDecoder.NFTData) {
	p.IsStrip = nft.IsStrip
	p.IsCursed = nft.IsCursed
	p.IsVindicate = nft.IsVindicate
	p.IsBRC20Tran = nft.IsBRC20Tran
	p.IsBRC20Mint = nft.IsBRC20Mint
	p.IsBRC20Ext = nft.IsBRC20Ext
	p.IsBRC20 = nft.IsBRC20
	p.IsText = nft.IsText
}

// IdxInBlock < 2**20
func (p *NFTCreatePoint) GetCreateIdxUint64() uint64 {
	return uint64(p.Height)<<HEIGHT_MUTIPLY_NBIT + uint64(p.IdxInBlock)
}

type TxIdxPkScriptData struct {
	TxIdx    uint32
	PkScript []byte
}

type TxoData struct {
	UTxid       []byte
	Vout        uint32
	BlockHeight uint32
	TxIdx       uint32
	Satoshi     uint64
	PkScript    []byte
	ScriptType  []byte
	OpInRBF     bool
	IsLowFee    bool
	// IsStandardDustStub marks a TxoData that was synthesized from the spending
	// input's witness/scriptSig because the UTXO was intentionally omitted from
	// Pika (standard dust: P2WPKH=294, P2TR=330, P2PKH=546 with no inscriptions).
	// Only Satoshi is valid; all other fields are zero. Never serialized.
	IsStandardDustStub bool

	CreatePointOfNFTs []NFTCreatePoint
}

func (d *TxoData) MarshalBufSize() int {
	return 4 + 20 + len(d.PkScript) + 1 + len(d.CreatePointOfNFTs)*TXO_DATA_INSCRIPTION_SIZE
}

func (d *TxoData) MakeMarshalBuf() (buf []byte) {
	buf = make([]byte, d.MarshalBufSize())
	return buf
}

func (d *TxoData) Marshal(buf []byte) (zbuf []byte, size int) {
	offset := 4
	buf[0] = 0
	buf[1] = 0
	buf[2] = 0
	buf[3] = 0

	binary.LittleEndian.PutUint32(buf[offset:], d.BlockHeight) // 4
	offset += 4

	binary.LittleEndian.PutUint32(buf[offset:], d.TxIdx) // 4
	offset += 4

	binary.LittleEndian.PutUint64(buf[offset:], d.Satoshi) // 8
	offset += 8

	scriptSize := uint32(len(d.PkScript))
	binary.LittleEndian.PutUint32(buf[offset:], scriptSize) // 4
	offset += 4

	copy(buf[offset:], d.PkScript)
	offset += int(scriptSize)

	offset += DumpNFTCreatePoints(buf[offset:], d.CreatePointOfNFTs)

	if len(d.CreatePointOfNFTs) < constant.ZIP_INSCRIPTIONS_MAX_COUNT {
		return buf[:offset], offset
	}
	// zip if more than nft
	zbuf = snappy.Encode(nil, buf[:offset])
	return zbuf, offset
}

// dump nft
func DumpNFTCreatePoints(buf []byte, createPointOfNFTs []NFTCreatePoint) int {
	lastOffset := int64(0)

	offset := 0
	for _, nft := range createPointOfNFTs {
		var flag byte = 0x00
		if nft.IsStrip {
			flag = 0x80
		}
		if nft.IsCursed {
			flag += 0x40
		}
		if nft.IsVindicate {
			flag += 0x20
		}

		if nft.IsText {
			if nft.IsBRC20Tran {
				buf[offset] = flag + 0x0f
			} else if nft.IsBRC20Mint {
				buf[offset] = flag + 0x0b
			} else if nft.IsBRC20Ext {
				buf[offset] = flag + 0x07
			} else if nft.IsBRC20 {
				buf[offset] = flag + 0x03
			} else {
				buf[offset] = flag + 0x01
			}
		} else {
			buf[offset] = flag
		}

		offset += 1

		thisOffset := int64(nft.Offset)
		span := uint64(thisOffset - lastOffset)
		lastOffset = thisOffset

		if nft.IsStrip {
			binary.LittleEndian.PutUint64(buf[offset:], span) // 8
			offset += 8
		} else {
			binary.LittleEndian.PutUint32(buf[offset:], nft.Height) // 4
			offset += 4

			binary.LittleEndian.PutUint32(buf[offset:], nft.IdxInBlock) // 4
			offset += 4

			binary.LittleEndian.PutUint64(buf[offset:], span) // 8
			offset += 8

			binary.LittleEndian.PutUint16(buf[offset:], nft.Sequence) // 2
			offset += 2
		}
	}
	return offset
}

func (d *TxoData) Unmarshal(zbuf []byte) bool {
	if len(zbuf) < 4 {
		return false
	}

	zipflag := binary.LittleEndian.Uint32(zbuf[:4]) // 4
	offset := 4

	buf := zbuf
	if zipflag != 0 {
		var dst []byte
		select {
		case dst = <-snappyDecodeBufChan:
		default:
		}
		var err error
		buf, err = snappy.Decode(dst, zbuf)
		if err != nil {
			return false
		}
		defer func() {
			select {
			case snappyDecodeBufChan <- buf[:0]:
			default:
			}
		}()
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

	offset += d.LoadNFTCreatePointsFromRaw(buf[offset:])
	return true
}

// load nft
func (d *TxoData) LoadNFTCreatePointsFromRaw(buf []byte) (offset int) {
	if len(buf) > 0 {
		// min entry size = 9 bytes (stripped: 1 flag + 8 span), upper-bound estimate
		d.CreatePointOfNFTs = make([]NFTCreatePoint, 0, len(buf)/9)
	}
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

		nft := NFTCreatePoint{
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

func (d *TxoData) AllNFTStripedOrWithOutNFT() bool {
	for _, nft := range d.CreatePointOfNFTs {
		if !nft.IsStrip {
			return false
		}
	}
	return true
}

func (d *TxoData) GetStripedNFTType() (mintCount, transferCount int) {
	for _, nft := range d.CreatePointOfNFTs {
		if nft.IsStrip {
			if nft.IsBRC20Mint {
				mintCount++
			}
			if nft.IsBRC20Tran {
				transferCount++
			}
		}
	}
	return
}
