package script

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

const (
	CodeType_NONE   uint32 = 0
	CodeType_FT     uint32 = 1
	CodeType_UNIQUE uint32 = 2
	CodeType_NFT    uint32 = 3

	CodeType_P2PK   uint32 = 4
	CodeType_P2PKH  uint32 = 5
	CodeType_P2SH   uint32 = 6
	CodeType_P2WPKH uint32 = 7
	CodeType_P2WSH  uint32 = 8
	CodeType_P2TR   uint32 = 9
	CodeType_P2A    uint32 = 10

	CodeType_SENSIBLE uint32 = 65536
)

var CodeTypeName []string = []string{
	"NONE",
	"FT",
	"UNIQUE",
	"NFT",

	"P2PK",
	"P2PKH",
	"P2SH",
	"P2WPKH",
	"P2WSH",
	"P2TR",
}

const (
	ORDINALS_TAG_CONTENT         = 0
	ORDINALS_TAG_CONTENTTYPE     = 1
	ORDINALS_TAG_POINTER         = 2
	ORDINALS_TAG_UNBOUND         = 66
	ORDINALS_TAG_PARENT          = 3
	ORDINALS_TAG_METADATA        = 5
	ORDINALS_TAG_METAPROTOCOL    = 7
	ORDINALS_TAG_CONTENTENCODING = 9
	ORDINALS_TAG_DELEGATE        = 11
)

type AddressData struct {
	HasAddress bool
	CodeType   uint32
	AddressPkh [20]byte
}

func (u *AddressData) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		HasAddress bool
		CodeType   uint32
		AddressPkh string
	}{
		HasAddress: u.HasAddress,
		CodeType:   u.CodeType,
		AddressPkh: hex.EncodeToString(u.AddressPkh[:]),
	})
}

// nft data from witness
type NFTData struct {
	IsStrip bool

	IsKeyVerify bool
	IsCursed    bool
	IsVindicate bool
	IsBRC20     bool
	IsBRC20Ext  bool
	IsBRC20Mint bool
	IsBRC20Tran bool
	IsText      bool

	Is0SatInput        bool
	IsUnrecognizedEven bool
	IsDuplicateField   bool
	IsIncompleteField  bool
	IsPushnum          bool
	IsStutter          bool
	IsReinscription    bool

	HasPointer         bool
	HasParent          bool
	HasDeligate        bool
	HasMetaProtocal    bool
	HasMetadata        bool
	HasContentEncoding bool

	InTxVin uint32 // nft vin inside tx

	Pointer         uint64 // 2
	InputOffset     uint64
	Parent          []byte // 3, max 32+4 bytes; note that this does not actually verify whether the parent exists.
	Deligate        []byte // 11, max 32+4 bytes.
	MetaProtocol    []byte // 7, max 520 bytes.
	Metadata        []byte // 5, may be very long (composed of multiple 520-byte chunks, each preceded by tag 5).
	ContentEncoding []byte // 9, max 520 bytes.
	ContentType     []byte // 1, max 520 bytes.
	ContentBody     []byte // 0, may be very long (composed of multiple 520-byte chunks, with no tag before each chunk).
}

func HashString(data []byte) (res string) {
	length := 32
	var reverseData [32]byte

	// need reverse
	for i := 0; i < length; i++ {
		reverseData[i] = data[length-i-1]
	}
	return hex.EncodeToString(reverseData[:])
}

func DecodeInscriptionFromBin(script []byte) (id string) {
	if len(script) < 32 || len(script) > 36 {
		return ""
	}

	var idx uint32
	if len(script) <= 32 {
		idx = uint32(0)
	} else if len(script) <= 33 {
		idx = uint32(script[32])
	} else if len(script) <= 34 {
		idx = uint32(binary.LittleEndian.Uint16(script[32:34]))
	} else if len(script) <= 35 {
		idx = uint32(script[32]) | uint32(script[33])<<8 | uint32(script[34])<<16
	} else if len(script) <= 36 {
		idx = binary.LittleEndian.Uint32(script[32:36])
	}

	id = fmt.Sprintf("%si%d", HashString(script[:32]), idx)
	return id
}

func (d *NFTData) GetParentId() string {
	return DecodeInscriptionFromBin(d.Parent)
}

func (d *NFTData) GetDeligateID() string {
	return DecodeInscriptionFromBin(d.Deligate)
}

func (d *NFTData) GetFlags() (flag uint32) {
	if d.IsStrip {
		flag += (1 << 21)
	}
	if d.IsKeyVerify {
		flag += (1 << 20)
	}
	if d.IsCursed {
		flag += (1 << 19)
	}
	if d.IsVindicate {
		flag += (1 << 18)
	}

	if d.IsBRC20Tran {
		flag += (1 << 17)
	}
	if d.IsBRC20Mint {
		flag += (1 << 16)
	}
	if d.IsBRC20Ext {
		flag += (1 << 15)
	}
	if d.IsBRC20 {
		flag += (1 << 14)
	}
	if d.IsText {
		flag += (1 << 13)
	}
	if d.Is0SatInput {
		flag += (1 << 12)
	}
	if d.IsUnrecognizedEven {
		flag += (1 << 11)
	}
	if d.IsDuplicateField {
		flag += (1 << 10)
	}
	if d.IsIncompleteField {
		flag += (1 << 9)
	}
	if d.IsPushnum {
		flag += (1 << 8)
	}
	if d.IsStutter {
		flag += (1 << 7)
	}
	if d.IsReinscription {
		flag += (1 << 6)
	}
	if d.HasPointer {
		flag += (1 << 5)
	}
	if d.HasParent {
		flag += (1 << 4)
	}
	if d.HasDeligate {
		flag += (1 << 3)
	}
	if d.HasMetaProtocal {
		flag += (1 << 2)
	}
	if d.HasMetadata {
		flag += (1 << 1)
	}
	if d.HasContentEncoding {
		flag += (1 << 0)
	}
	return flag
}

func (d *NFTData) SetFlags(flag uint32) {
	if flag&(1<<21) > 0 {
		d.IsStrip = true
	}
	if flag&(1<<20) > 0 {
		d.IsKeyVerify = true
	}
	if flag&(1<<19) > 0 {
		d.IsCursed = true
	}
	if flag&(1<<18) > 0 {
		d.IsVindicate = true
	}

	if flag&(1<<17) > 0 {
		d.IsBRC20Tran = true
	}
	if flag&(1<<16) > 0 {
		d.IsBRC20Mint = true
	}
	if flag&(1<<15) > 0 {
		d.IsBRC20Ext = true
	}
	if flag&(1<<14) > 0 {
		d.IsBRC20 = true
	}
	if flag&(1<<13) > 0 {
		d.IsText = true
	}
	if flag&(1<<12) > 0 {
		d.Is0SatInput = true
	}
	if flag&(1<<11) > 0 {
		d.IsUnrecognizedEven = true
	}
	if flag&(1<<10) > 0 {
		d.IsDuplicateField = true
	}
	if flag&(1<<9) > 0 {
		d.IsIncompleteField = true
	}
	if flag&(1<<8) > 0 {
		d.IsPushnum = true
	}
	if flag&(1<<7) > 0 {
		d.IsStutter = true
	}
	if flag&(1<<6) > 0 {
		d.IsReinscription = true
	}
	if flag&(1<<5) > 0 {
		d.HasPointer = true
	}
	if flag&(1<<4) > 0 {
		d.HasParent = true
	}
	if flag&(1<<3) > 0 {
		d.HasDeligate = true
	}
	if flag&(1<<2) > 0 {
		d.HasMetaProtocal = true
	}
	if flag&(1<<1) > 0 {
		d.HasMetadata = true
	}
	if flag&(1<<0) > 0 {
		d.HasContentEncoding = true
	}
}
