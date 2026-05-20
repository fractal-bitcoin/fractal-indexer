package script

import (
	"testing"

	"github.com/btcsuite/btcd/txscript"
)

// buildWitness encodes witness items in the raw blob format used by NewTxWits:
// <item_count_varint> <item_len_varint><item_bytes>...
func buildWitness(items ...[]byte) []byte {
	var out []byte
	out = append(out, byte(len(items)))
	for _, item := range items {
		out = append(out, byte(len(item)))
		out = append(out, item...)
	}
	return out
}

// makeSig builds a DER-header-valid stub signature of total length n (including
// sighash byte). Layout: 0x30 <inner_len=n-3> 0x02 ... followed by zeros.
// Required by the stricter InferDustFromInput DER validation.
func makeSig(n int) []byte {
	b := make([]byte, n)
	b[0] = 0x30        // DER SEQUENCE tag
	b[1] = byte(n - 3) // inner length
	b[2] = 0x02        // INTEGER tag for r
	return b
}

func makePubkey33() []byte {
	b := make([]byte, 33)
	b[0] = 0x02 // compressed pubkey prefix required by InferDustFromInput
	return b
}
func makePubkey65() []byte { return make([]byte, 65) }

// ClassifyStandardDust tests

func TestClassifyStandardDust_P2WPKH_294(t *testing.T) {
	stype := []byte{txscript.OP_0, txscript.OP_DATA_20}
	if !ClassifyStandardDust(stype, 294) {
		t.Fatal("expected true for P2WPKH+294")
	}
}

func TestClassifyStandardDust_P2TR_330(t *testing.T) {
	stype := []byte{txscript.OP_1, txscript.OP_DATA_32}
	if !ClassifyStandardDust(stype, 330) {
		t.Fatal("expected true for P2TR+330")
	}
}

func TestClassifyStandardDust_P2PKH_546(t *testing.T) {
	stype := []byte{txscript.OP_DUP, txscript.OP_HASH160, txscript.OP_DATA_20, txscript.OP_EQUALVERIFY, txscript.OP_CHECKSIG}
	if !ClassifyStandardDust(stype, 546) {
		t.Fatal("expected true for P2PKH+546")
	}
}

func TestClassifyStandardDust_WrongSatoshi(t *testing.T) {
	stype := []byte{txscript.OP_0, txscript.OP_DATA_20}
	if ClassifyStandardDust(stype, 700) {
		t.Fatal("expected false for P2WPKH+700 (non-dust amount)")
	}
}

func TestClassifyStandardDust_P2WSH(t *testing.T) {
	stype := []byte{txscript.OP_0, txscript.OP_DATA_32}
	if ClassifyStandardDust(stype, 294) {
		t.Fatal("expected false for P2WSH")
	}
}

// InferDustFromInput tests

func TestInferDustFromInput_P2WPKH(t *testing.T) {
	sig := makeSig(72)
	pub := makePubkey33()
	wit := buildWitness(sig, pub)
	sat, ok := InferDustFromInput(nil, wit)
	if !ok || sat != 294 {
		t.Fatalf("P2WPKH: got (%d, %v), want (294, true)", sat, ok)
	}
}

func TestInferDustFromInput_P2WPKH_ScriptSigEmpty(t *testing.T) {
	sig := makeSig(71)
	pub := makePubkey33()
	wit := buildWitness(sig, pub)
	sat, ok := InferDustFromInput([]byte{}, wit)
	if !ok || sat != 294 {
		t.Fatalf("P2WPKH empty scriptSig: got (%d, %v)", sat, ok)
	}
}

func TestInferDustFromInput_P2TR_Keypath_64(t *testing.T) {
	schnorrSig := make([]byte, 64)
	wit := buildWitness(schnorrSig)
	sat, ok := InferDustFromInput(nil, wit)
	if !ok || sat != 330 {
		t.Fatalf("P2TR keypath 64B: got (%d, %v)", sat, ok)
	}
}

func TestInferDustFromInput_P2TR_Keypath_65(t *testing.T) {
	schnorrSig := make([]byte, 65)
	wit := buildWitness(schnorrSig)
	sat, ok := InferDustFromInput(nil, wit)
	if !ok || sat != 330 {
		t.Fatalf("P2TR keypath 65B: got (%d, %v)", sat, ok)
	}
}

func TestInferDustFromInput_P2TR_ScriptPath(t *testing.T) {
	// script-path: [<input>, <script>, <control_block>]
	input := make([]byte, 32)
	script := make([]byte, 10)
	controlBlock := make([]byte, 33)
	controlBlock[0] = 0xc0 // control block marker
	wit := buildWitness(input, script, controlBlock)
	sat, ok := InferDustFromInput(nil, wit)
	if !ok || sat != 330 {
		t.Fatalf("P2TR script-path: got (%d, %v)", sat, ok)
	}
}

func TestInferDustFromInput_P2PKH_Compressed(t *testing.T) {
	sig := makeSig(72)
	pub := makePubkey33()
	scriptSig := append([]byte{byte(len(sig))}, sig...)
	scriptSig = append(scriptSig, byte(len(pub)))
	scriptSig = append(scriptSig, pub...)
	sat, ok := InferDustFromInput(scriptSig, nil)
	if !ok || sat != 546 {
		t.Fatalf("P2PKH compressed: got (%d, %v)", sat, ok)
	}
}

func TestInferDustFromInput_P2PKH_Uncompressed(t *testing.T) {
	sig := makeSig(71)
	pub := makePubkey65()
	scriptSig := append([]byte{byte(len(sig))}, sig...)
	scriptSig = append(scriptSig, byte(len(pub)))
	scriptSig = append(scriptSig, pub...)
	sat, ok := InferDustFromInput(scriptSig, nil)
	if !ok || sat != 546 {
		t.Fatalf("P2PKH uncompressed: got (%d, %v)", sat, ok)
	}
}

func TestInferDustFromInput_P2SH_P2WPKH_Rejected(t *testing.T) {
	// Both scriptSig and witness present → P2SH-wrapped, not standard dust.
	sig := makeSig(72)
	pub := makePubkey33()
	wit := buildWitness(sig, pub)
	scriptSig := []byte{0x16, 0x00, 0x14} // P2SH redeemScript push
	scriptSig = append(scriptSig, make([]byte, 20)...)
	_, ok := InferDustFromInput(scriptSig, wit)
	if ok {
		t.Fatal("P2SH-P2WPKH must not be inferred as standard dust")
	}
}

func TestInferDustFromInput_EmptyWitness_NilSig(t *testing.T) {
	_, ok := InferDustFromInput(nil, nil)
	if ok {
		t.Fatal("empty input must return false")
	}
}

func TestInferDustFromInput_ZeroItemWitness(t *testing.T) {
	// Non-segwit tx has witness blob [0x00] for each input.
	wit := []byte{0x00}
	_, ok := InferDustFromInput(nil, wit)
	if ok {
		t.Fatal("zero-item witness with nil scriptSig must return false")
	}
}

func TestInferDustFromInput_P2PKH_WrongSigLen(t *testing.T) {
	// sigLen = 80 is outside the DER max (73) — must be rejected.
	sig := make([]byte, 80)
	sig[0] = 0x30
	sig[1] = 77 // would-be inner length
	sig[2] = 0x02
	pub := makePubkey33()
	scriptSig := append([]byte{byte(len(sig))}, sig...)
	scriptSig = append(scriptSig, byte(len(pub)))
	scriptSig = append(scriptSig, pub...)
	_, ok := InferDustFromInput(scriptSig, nil)
	if ok {
		t.Fatal("sig longer than 73B must not be inferred as P2PKH")
	}
}

func TestInferDustFromInput_P2WSH_Rejected(t *testing.T) {
	// P2WSH keypath would have many witness items — won't match our patterns.
	// Simulate a 2-item witness where item[1] is 32 bytes (P2WSH redeemScript hash), not 33B pubkey.
	sig := makeSig(72)
	witnessScript := make([]byte, 32) // not a 33B pubkey
	wit := buildWitness(sig, witnessScript)
	_, ok := InferDustFromInput(nil, wit)
	if ok {
		t.Fatal("P2WSH with 32B second item must not be inferred as P2WPKH")
	}
}

// P2TR script-path: 2-item witness [script, 33B control_block] (single-leaf, no sibling).
// Pre-fix code would misidentify this as P2WPKH (294); correct answer is P2TR (330).
func TestInferDustFromInput_P2TR_ScriptPath_TwoItems(t *testing.T) {
	script := make([]byte, 10)
	controlBlock := make([]byte, 33)
	controlBlock[0] = 0xc0 // leaf_version marker
	wit := buildWitness(script, controlBlock)
	sat, ok := InferDustFromInput(nil, wit)
	if !ok || sat != 330 {
		t.Fatalf("P2TR 2-item script-path: got (%d, %v), want (330, true)", sat, ok)
	}
}

// P2TR script-path: 2-item witness where control block is 65B (1 merkle sibling).
func TestInferDustFromInput_P2TR_ScriptPath_TwoItems_LargeControlBlock(t *testing.T) {
	script := make([]byte, 10)
	controlBlock := make([]byte, 65)
	controlBlock[0] = 0xc0
	wit := buildWitness(script, controlBlock)
	sat, ok := InferDustFromInput(nil, wit)
	if !ok || sat != 330 {
		t.Fatalf("P2TR 2-item 65B control block: got (%d, %v), want (330, true)", sat, ok)
	}
}

// Ambiguous 33B second item with a leading byte that is neither a compressed pubkey
// prefix (0x02/0x03) nor a control block marker (>=0xc0).
func TestInferDustFromInput_Ambiguous33B_OtherLeading(t *testing.T) {
	item0 := make([]byte, 72)
	item1 := make([]byte, 33)
	item1[0] = 0x04 // uncompressed pubkey prefix — not a valid P2WPKH spending input
	wit := buildWitness(item0, item1)
	_, ok := InferDustFromInput(nil, wit)
	if ok {
		t.Fatal("33B item with 0x04 leading byte must not be inferred as standard dust")
	}
}

// P2PKH regression: real-world low-r+low-s sigs can be shorter than 71B
// (e.g. 70B when s is 31B). Pre-fix code rejected these → missing-utxo errors.
func TestInferDustFromInput_P2PKH_Sig70(t *testing.T) {
	sig := makeSig(70)
	pub := makePubkey33()
	scriptSig := append([]byte{byte(len(sig))}, sig...)
	scriptSig = append(scriptSig, byte(len(pub)))
	scriptSig = append(scriptSig, pub...)
	sat, ok := InferDustFromInput(scriptSig, nil)
	if !ok || sat != 546 {
		t.Fatalf("P2PKH sig70: got (%d, %v), want (546, true)", sat, ok)
	}
}

// Boundary: DER sig at the upper end (73B) still accepted.
func TestInferDustFromInput_P2PKH_Sig73(t *testing.T) {
	sig := makeSig(73)
	pub := makePubkey33()
	scriptSig := append([]byte{byte(len(sig))}, sig...)
	scriptSig = append(scriptSig, byte(len(pub)))
	scriptSig = append(scriptSig, pub...)
	sat, ok := InferDustFromInput(scriptSig, nil)
	if !ok || sat != 546 {
		t.Fatalf("P2PKH sig73: got (%d, %v), want (546, true)", sat, ok)
	}
}

// DER inner-length mismatch must be rejected by the strengthened check.
func TestInferDustFromInput_P2PKH_BadDERInnerLen(t *testing.T) {
	sig := makeSig(71)
	sig[1] = 0x40 // corrupt inner length (should be 68 = 71-3)
	pub := makePubkey33()
	scriptSig := append([]byte{byte(len(sig))}, sig...)
	scriptSig = append(scriptSig, byte(len(pub)))
	scriptSig = append(scriptSig, pub...)
	_, ok := InferDustFromInput(scriptSig, nil)
	if ok {
		t.Fatal("scriptSig with wrong DER inner length must not be inferred as P2PKH")
	}
}

// INTEGER tag for r must be 0x02; anything else fails the DER check.
func TestInferDustFromInput_P2PKH_BadRTag(t *testing.T) {
	sig := makeSig(71)
	sig[2] = 0x04 // wrong r INTEGER tag (makeSig set it to 0x02)
	pub := makePubkey33()
	scriptSig := append([]byte{byte(len(sig))}, sig...)
	scriptSig = append(scriptSig, byte(len(pub)))
	scriptSig = append(scriptSig, pub...)
	_, ok := InferDustFromInput(scriptSig, nil)
	if ok {
		t.Fatal("scriptSig with wrong r INTEGER tag must not be inferred as P2PKH")
	}
}
