package script

import "encoding/binary"

// ClassifyStandardDust reports whether a UTXO with the given scriptType and satoshi
// qualifies as "standard dust" and can be omitted from Pika storage.
// Criteria: one of {P2WPKH=294, P2TR=330, P2PKH=546} with the exact dust satoshi.
// Callers must ensure len(CreatePointOfNFTs)==0 before calling.
func ClassifyStandardDust(scriptType []byte, satoshi uint64) bool {
	switch {
	case isPayToWitnessPubKeyHash(scriptType) && satoshi == 294:
		return true
	case isPayToTaproot(scriptType) && satoshi == 330:
		return true
	case isPubkeyHash(scriptType) && satoshi == 546:
		return true
	}
	return false
}

// InferDustFromInput infers the satoshi of a standard-dust UTXO from the spending
// input's scriptSig and witness blob. Returns (sat, true) on success.
// Only called when Pika returns nil for an outpoint key.
//
// Inference rules (must be symmetric with ClassifyStandardDust):
//   P2WPKH: witness=2 items [sig, 33B pubkey], scriptSig empty   → 294
//   P2TR keypath: witness=1 item [64/65B sig], scriptSig empty   → 330
//   P2TR script-path: witness≥2 items, last byte[0]≥0xc0        → 330
//   P2PKH: scriptSig=[sig][pubkey], witness empty/absent         → 546
func InferDustFromInput(scriptSig, witness []byte) (uint64, bool) {
	itemCount := witnessItems(witness)
	hasSig := len(scriptSig) > 0

	// P2SH-wrapped segwit has both scriptSig and witness — not standard dust.
	if hasSig && itemCount > 0 {
		return 0, false
	}

	if itemCount > 0 && !hasSig {
		off := uint(witnessVarIntLen(witness[0])) // skip item count varint

		if itemCount == 1 {
			// P2TR keypath: single 64 or 65 byte schnorr signature
			item0Len, _ := decodeWitnessVarInt(witness[off:])
			if item0Len == 64 || item0Len == 65 {
				return 330, true
			}
			return 0, false
		}

		if itemCount == 2 {
			// Two-item witness can be:
			//   P2WPKH:           [sig, compressed_pubkey(33B, first byte 0x02/0x03)]
			//   P2TR script-path: [script, control_block(33B+ first byte >=0xc0)]
			item0Len, s0 := decodeWitnessVarInt(witness[off:])
			off += s0 + item0Len
			if off >= uint(len(witness)) {
				return 0, false
			}
			item1Len, s1 := decodeWitnessVarInt(witness[off:])
			off += s1
			if item1Len == 33 && off < uint(len(witness)) {
				lead := witness[off]
				switch {
				case lead == 0x02 || lead == 0x03:
					return 294, true // P2WPKH: compressed pubkey
				case lead >= 0xc0:
					return 330, true // P2TR script-path: single-leaf control block (33B = 1+32)
				}
				return 0, false // ambiguous 33B item, not a known standard type
			}
			// item1 is not 33B (e.g., 65B control block with 1 merkle sibling) —
			// fall through to the general script-path check below.
		}

		// P2TR script-path with itemCount >= 2: the last item is the control block
		// whose first byte is >= 0xc0. Reset cursor to start of items.
		cur := uint(witnessVarIntLen(witness[0]))
		for i := uint(0); i < itemCount-1; i++ {
			l, s := decodeWitnessVarInt(witness[cur:])
			if s == 0 {
				return 0, false
			}
			cur += s + l
		}
		lastLen, s := decodeWitnessVarInt(witness[cur:])
		cur += s
		if s == 0 || lastLen == 0 || cur >= uint(len(witness)) {
			return 0, false
		}
		if witness[cur] >= 0xc0 { // control block leading byte: 0xc0 | leaf_version
			return 330, true
		}
		return 0, false
	}

	if hasSig && itemCount == 0 {
		// P2PKH scriptSig: <push_sig><DER_sig><push_pubkey><pubkey>
		// DER sig layout: 0x30 <inner_len> 0x02 <r_len> <r> 0x02 <s_len> <s> <sighash>
		// Total sigLen = 7 + r_len + s_len; real-world range roughly [39, 73]
		// (depends on r/s size after low-r/low-s enforcement and optional DER padding).
		if len(scriptSig) < 10 {
			return 0, false
		}
		sigLen := int(scriptSig[0])
		if sigLen < 8 || sigLen > 73 {
			return 0, false
		}
		if 1+sigLen >= len(scriptSig) {
			return 0, false
		}
		if scriptSig[1] != 0x30 { // DER SEQUENCE tag
			return 0, false
		}
		if int(scriptSig[2]) != sigLen-3 { // inner length = sigLen - 2(outer header) - 1(sighash)
			return 0, false
		}
		if scriptSig[3] != 0x02 { // INTEGER tag for r
			return 0, false
		}
		pubOff := 1 + sigLen
		pubLen := int(scriptSig[pubOff])
		if pubLen != 33 && pubLen != 65 {
			return 0, false
		}
		if pubOff+1+pubLen != len(scriptSig) {
			return 0, false
		}
		return 546, true
	}

	return 0, false
}

// witnessItems returns the number of witness stack items encoded in the blob.
// Returns 0 for nil/empty witness or a zero-item witness (non-segwit inputs).
func witnessItems(witness []byte) uint {
	if len(witness) == 0 {
		return 0
	}
	cnt, _ := decodeWitnessVarInt(witness)
	return cnt
}

// decodeWitnessVarInt decodes a standard Bitcoin varint (not script pushdata encoding).
func decodeWitnessVarInt(b []byte) (value uint, size uint) {
	if len(b) == 0 {
		return 0, 0
	}
	switch {
	case b[0] < 0xfd:
		return uint(b[0]), 1
	case b[0] == 0xfd && len(b) >= 3:
		return uint(binary.LittleEndian.Uint16(b[1:3])), 3
	case b[0] == 0xfe && len(b) >= 5:
		return uint(binary.LittleEndian.Uint32(b[1:5])), 5
	case len(b) >= 9:
		return uint(binary.LittleEndian.Uint64(b[1:9])), 9
	}
	return 0, 0
}

// witnessVarIntLen returns the byte length of a varint given its first byte.
func witnessVarIntLen(b byte) uint {
	switch {
	case b < 0xfd:
		return 1
	case b == 0xfd:
		return 3
	case b == 0xfe:
		return 5
	default:
		return 9
	}
}
