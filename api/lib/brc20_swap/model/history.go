package model

import "encoding/binary"

type BRC20HistoryBase struct {
	Type  uint8 // inscribe-deploy/inscribe-mint/inscribe-transfer/transfer/send/receive
	Valid bool

	TxId   string
	Idx    uint32
	Vout   uint32
	Offset uint64

	PkScriptFrom string
	PkScriptTo   string
	Satoshi      uint64
	Fee          int64

	Height    uint32
	TxIdx     uint32
	BlockTime uint32
}

// history
type BRC20History struct {
	BRC20HistoryBase

	Inscription InscriptionBRC20TickInfoHistory

	// param
	Amount string

	// state
	OverallBalance      string
	TransferableBalance string
	AvailableBalance    string
}

func NewBRC20History(historyType uint8, isValid bool, isTransfer bool,
	from *InscriptionBRC20TickInfo, bal *BRC20TokenBalance, to *InscriptionBRC20Data) *BRC20History {
	history := &BRC20History{
		BRC20HistoryBase: BRC20HistoryBase{
			Type:      historyType,
			Valid:     isValid,
			Height:    to.Height,
			TxIdx:     to.TxIdx,
			BlockTime: to.BlockTime,
			Fee:       to.Fee,
		},
		Inscription: InscriptionBRC20TickInfoHistory{
			Height:            from.Height,
			Data:              from.Data,
			InscriptionNumber: from.InscriptionNumber,
			TxId:              from.TxId,
			Idx:               from.Idx,
			Satoshi:           from.Satoshi,
		},
		Amount: from.Amount.String(),
	}
	if isTransfer {
		history.TxId = to.TxId
		history.Vout = to.Vout
		history.Offset = to.Offset
		history.Idx = to.Idx
		history.PkScriptFrom = from.PkScript
		history.PkScriptTo = to.PkScript
		history.Satoshi = to.Satoshi
		if history.Satoshi == 0 {
			history.PkScriptTo = history.PkScriptFrom
		}

	} else {
		history.TxId = from.TxId
		history.Vout = from.Vout
		history.Offset = from.Offset
		history.Idx = from.Idx
		history.PkScriptTo = from.PkScript
		history.Satoshi = from.Satoshi
	}

	if bal != nil {
		history.OverallBalance = bal.OverallBalance().String()
		history.TransferableBalance = bal.TransferableBalance.String()
		history.AvailableBalance = bal.AvailableBalance.String()
	} else {
		history.OverallBalance = "0"
		history.TransferableBalance = "0"
		history.AvailableBalance = "0"
	}
	return history
}

func appendFixedString(buf []byte, value string, size int) []byte {
	offset := len(buf)
	buf = append(buf, make([]byte, size)...)
	copy(buf[offset:], value)
	return buf
}

func appendUint32(buf []byte, value uint32) []byte {
	var raw [4]byte
	binary.LittleEndian.PutUint32(raw[:], value)
	return append(buf, raw[:]...)
}

func appendUint64(buf []byte, value uint64) []byte {
	var raw [8]byte
	binary.LittleEndian.PutUint64(raw[:], value)
	return append(buf, raw[:]...)
}

func appendInt64(buf []byte, value int64) []byte {
	return appendUint64(buf, uint64(value))
}

func appendBytes(buf []byte, value string) []byte {
	buf = appendUint32(buf, uint32(len(value)))
	return append(buf, value...)
}

func appendStringWithLimit(buf []byte, value string, limit int) []byte {
	n := len(value)
	if n < limit {
		buf = append(buf, uint8(n))
		return append(buf, value...)
	}
	return append(buf, 0)
}

func (h *BRC20History) Marshal() (result []byte) {
	buf := make([]byte, 0, 1024)
	buf = append(buf, h.Type)
	if h.Valid {
		buf = append(buf, 1)
	} else {
		buf = append(buf, 0)
	}
	buf = appendFixedString(buf, h.TxId, 32)

	buf = appendUint32(buf, h.Idx)
	buf = appendUint32(buf, h.Vout)
	buf = appendUint64(buf, h.Offset)
	buf = appendBytes(buf, h.PkScriptFrom)
	buf = appendBytes(buf, h.PkScriptTo)
	buf = appendUint64(buf, h.Satoshi)
	buf = appendInt64(buf, h.Fee)
	buf = appendUint32(buf, h.Height)
	buf = appendUint32(buf, h.TxIdx)
	buf = appendUint32(buf, h.BlockTime)

	buf = appendStringWithLimit(buf, h.Amount, 40)
	buf = appendStringWithLimit(buf, h.OverallBalance, 40)
	buf = appendStringWithLimit(buf, h.TransferableBalance, 40)
	buf = appendStringWithLimit(buf, h.AvailableBalance, 40)

	buf = appendUint32(buf, h.Inscription.Height)
	buf = appendInt64(buf, h.Inscription.InscriptionNumber)
	buf = appendUint64(buf, h.Inscription.Satoshi)
	buf = appendUint32(buf, h.Inscription.Idx)
	buf = appendStringWithLimit(buf, h.Inscription.TxId, 70)

	data := h.Inscription.Data
	if data == nil {
		return buf
	}

	buf = appendStringWithLimit(buf, data.BRC20Tick, 16)
	buf = appendStringWithLimit(buf, data.BRC20Max, 40)
	buf = appendStringWithLimit(buf, data.BRC20Limit, 40)
	buf = appendStringWithLimit(buf, data.BRC20Amount, 40)
	buf = appendStringWithLimit(buf, data.BRC20Decimal, 8)
	buf = appendStringWithLimit(buf, data.BRC20Minted, 40)
	buf = appendStringWithLimit(buf, data.BRC20SelfMint, 8)
	return buf
}

func (h *BRC20History) Unmarshal(buf []byte) {
	if len(buf) < 34 {
		return
	}

	h.Type = buf[0]
	h.Valid = (buf[1] == 1)

	h.TxId = string(buf[2 : 2+32])

	offset := 34

	readUint32 := func() (uint32, bool) {
		if len(buf[offset:]) < 4 {
			return 0, false
		}
		value := binary.LittleEndian.Uint32(buf[offset:])
		offset += 4
		return value, true
	}
	readUint64 := func() (uint64, bool) {
		if len(buf[offset:]) < 8 {
			return 0, false
		}
		value := binary.LittleEndian.Uint64(buf[offset:])
		offset += 8
		return value, true
	}
	readInt64 := func() (int64, bool) {
		value, ok := readUint64()
		return int64(value), ok
	}
	readBytes := func() (string, bool) {
		n, ok := readUint32()
		if !ok {
			return "", false
		}
		if uint64(len(buf[offset:])) < uint64(n) {
			return "", false
		}
		value := string(buf[offset : offset+int(n)])
		offset += int(n)
		return value, true
	}
	readStringWithLimit := func() (string, bool) {
		if len(buf[offset:]) < 1 {
			return "", false
		}
		n := int(buf[offset])
		offset += 1
		if len(buf[offset:]) < n {
			return "", false
		}
		value := string(buf[offset : offset+n])
		offset += n
		return value, true
	}

	var ok bool
	if h.Idx, ok = readUint32(); !ok {
		return
	}
	if h.Vout, ok = readUint32(); !ok {
		return
	}
	if h.Offset, ok = readUint64(); !ok {
		return
	}
	if h.PkScriptFrom, ok = readBytes(); !ok {
		return
	}
	if h.PkScriptTo, ok = readBytes(); !ok {
		return
	}
	if h.Satoshi, ok = readUint64(); !ok {
		return
	}
	if h.Fee, ok = readInt64(); !ok {
		return
	}
	if h.Height, ok = readUint32(); !ok {
		return
	}
	if h.TxIdx, ok = readUint32(); !ok {
		return
	}
	if h.BlockTime, ok = readUint32(); !ok {
		return
	}
	if h.Amount, ok = readStringWithLimit(); !ok {
		return
	}
	if h.OverallBalance, ok = readStringWithLimit(); !ok {
		return
	}
	if h.TransferableBalance, ok = readStringWithLimit(); !ok {
		return
	}
	if h.AvailableBalance, ok = readStringWithLimit(); !ok {
		return
	}
	if h.Inscription.Height, ok = readUint32(); !ok {
		return
	}
	if h.Inscription.InscriptionNumber, ok = readInt64(); !ok {
		return
	}
	if h.Inscription.Satoshi, ok = readUint64(); !ok {
		return
	}
	if h.Inscription.Idx, ok = readUint32(); !ok {
		return
	}
	if h.Inscription.TxId, ok = readStringWithLimit(); !ok {
		return
	}

	if len(buf[offset:]) == 0 {
		return
	}
	data := &InscriptionBRC20InfoResp{}
	h.Inscription.Data = data

	if data.BRC20Tick, ok = readStringWithLimit(); !ok {
		return
	}
	if data.BRC20Max, ok = readStringWithLimit(); !ok {
		return
	}
	if data.BRC20Limit, ok = readStringWithLimit(); !ok {
		return
	}
	if data.BRC20Amount, ok = readStringWithLimit(); !ok {
		return
	}
	if data.BRC20Decimal, ok = readStringWithLimit(); !ok {
		return
	}
	if data.BRC20Minted, ok = readStringWithLimit(); !ok {
		return
	}
	if data.BRC20SelfMint, ok = readStringWithLimit(); !ok {
		return
	}
}
