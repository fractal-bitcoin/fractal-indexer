package script

import "github.com/btcsuite/btcd/txscript"

const (
	runesMagicOpcode = txscript.OP_13
	runesTagFlags    = 2
	runesTagMint     = 20
	runesFlagEtching = 1
)

func IsRunestone(pkScript []byte) bool {
	if len(pkScript) < 2 {
		return false
	}
	return pkScript[0] == txscript.OP_RETURN && pkScript[1] == runesMagicOpcode
}

func IsRunesEtching(pkScript []byte) bool {
	payload, ok := runestonePayload(pkScript)
	if !ok {
		return false
	}
	return runestoneHasEtching(payload)
}

func IsRunesMint(pkScript []byte) bool {
	payload, ok := runestonePayload(pkScript)
	if !ok {
		return false
	}
	return runestoneHasMint(payload)
}

func runestonePayload(pkScript []byte) ([]byte, bool) {
	if !IsRunestone(pkScript) {
		return nil, false
	}

	payload := make([]byte, 0, len(pkScript)-2)
	for offset := uint(2); offset < uint(len(pkScript)); {
		size, data, isPush, isOpcode := GetOpcodeFormScript(pkScript[offset:])
		if size == 0 || data == nil {
			return nil, false
		}
		offset += size
		if !isPush || isOpcode {
			return nil, false
		}
		payload = append(payload, data...)
	}
	return payload, true
}

func runestoneHasEtching(payload []byte) bool {
	for offset := 0; offset < len(payload); {
		tag, n := decodeRunestoneVarint(payload[offset:])
		if n == 0 {
			return false
		}
		offset += n
		if tag == 0 {
			return false
		}

		value, n := decodeRunestoneVarint(payload[offset:])
		if n == 0 {
			return false
		}
		offset += n

		if tag == runesTagFlags && value&runesFlagEtching != 0 {
			return true
		}
	}
	return false
}

func runestoneHasMint(payload []byte) bool {
	mintValues := 0
	for offset := 0; offset < len(payload); {
		tag, n := decodeRunestoneVarint(payload[offset:])
		if n == 0 {
			return false
		}
		offset += n
		if tag == 0 {
			break
		}

		_, n = decodeRunestoneVarint(payload[offset:])
		if n == 0 {
			return false
		}
		offset += n

		if tag == runesTagMint {
			mintValues++
			if mintValues >= 2 {
				return true
			}
		}
	}
	return false
}

func decodeRunestoneVarint(data []byte) (uint64, int) {
	var value uint64
	for i, b := range data {
		if i > 18 {
			return 0, 0
		}
		value |= uint64(b&0x7f) << (7 * i)
		if b&0x80 == 0 {
			return value, i + 1
		}
	}
	return 0, 0
}
