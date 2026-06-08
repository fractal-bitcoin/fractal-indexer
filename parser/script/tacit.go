package script

import "github.com/btcsuite/btcd/txscript"

var tacitMagic = []byte("TACIT")

func WitnessHasTacitEnvelope(witness []byte) bool {
	leafScript, ok := witnessLeafScript(witness)
	if !ok {
		return false
	}
	return HasTacitEnvelope(leafScript)
}

func HasTacitEnvelope(script []byte) bool {
	for offset := uint(0); offset < uint(len(script)); {
		size, data, isPush, isOpcode := GetOpcodeFormScript(script[offset:])
		if size == 0 || data == nil {
			return false
		}
		offset += size
		if !isPush || isOpcode || len(data) != 32 {
			continue
		}
		if offset >= uint(len(script)) || script[offset] != txscript.OP_CHECKSIG {
			continue
		}
		offset++

		if !consumeOpcode(script, &offset, txscript.OP_FALSE) {
			continue
		}
		if !consumeOpcode(script, &offset, txscript.OP_IF) {
			continue
		}
		if !consumePushData(script, &offset, tacitMagic) {
			continue
		}
		if !consumePushData(script, &offset, []byte{0x01}) {
			continue
		}
		if envelopeEnds(script[offset:]) {
			return true
		}
	}
	return false
}

func witnessLeafScript(witness []byte) ([]byte, bool) {
	cnt, cntsize := decodeWitnessVarInt(witness)
	if cnt < 2 || cntsize == 0 {
		return nil, false
	}

	offset := uint(cntsize)
	type span struct{ start, end uint }
	var ring [3]span
	for i := uint(0); i < cnt; i++ {
		itemLen, itemSize := decodeWitnessVarInt(witness[offset:])
		if itemSize == 0 {
			return nil, false
		}
		offset += uint(itemSize)
		end := offset + uint(itemLen)
		if end > uint(len(witness)) {
			return nil, false
		}
		ring[2] = ring[1]
		ring[1] = ring[0]
		ring[0] = span{offset, end}
		offset = end
	}
	if offset != uint(len(witness)) {
		return nil, false
	}

	last := ring[0]
	hasAnnex := last.end > last.start && witness[last.start] == 0x50
	if hasAnnex {
		if cnt < 3 {
			return nil, false
		}
		return witness[ring[2].start:ring[2].end], true
	}
	return witness[ring[1].start:ring[1].end], true
}

func consumeOpcode(script []byte, offset *uint, opcode byte) bool {
	if *offset >= uint(len(script)) || script[*offset] != opcode {
		return false
	}
	*offset += 1
	return true
}

func consumePushData(script []byte, offset *uint, want []byte) bool {
	size, data, isPush, isOpcode := GetOpcodeFormScript(script[*offset:])
	if size == 0 || data == nil || !isPush || isOpcode {
		return false
	}
	if len(data) != len(want) {
		return false
	}
	for i := range want {
		if data[i] != want[i] {
			return false
		}
	}
	*offset += size
	return true
}

func envelopeEnds(script []byte) bool {
	for offset := uint(0); offset < uint(len(script)); {
		size, data, _, isOpcode := GetOpcodeFormScript(script[offset:])
		if size == 0 || data == nil {
			return false
		}
		offset += size
		if isOpcode && len(data) == 1 && data[0] == txscript.OP_ENDIF {
			return true
		}
	}
	return false
}
