package script

import "testing"

func TestTacitEnvelopeDetection(t *testing.T) {
	script := validTacitScript()
	if !HasTacitEnvelope(script) {
		t.Fatal("expected tacit envelope")
	}
	if !WitnessHasTacitEnvelope(testWitness(script, nil)) {
		t.Fatal("expected tacit envelope in witness")
	}
}

func TestTacitEnvelopeWithAnnex(t *testing.T) {
	script := validTacitScript()
	annex := []byte{0x50, 0x01}
	if !WitnessHasTacitEnvelope(testWitness(script, annex)) {
		t.Fatal("expected tacit envelope in witness with annex")
	}
}

func TestTacitEnvelopeRejectsWrongMagic(t *testing.T) {
	script := validTacitScript()
	copy(script[37:42], []byte("WRONG"))
	if HasTacitEnvelope(script) {
		t.Fatal("unexpected tacit envelope")
	}
}

func validTacitScript() []byte {
	script := make([]byte, 0, 48)
	script = append(script, 0x20)
	script = append(script, make([]byte, 32)...)
	script = append(script, 0xac, 0x00, 0x63)
	script = append(script, 0x05)
	script = append(script, []byte("TACIT")...)
	script = append(script, 0x01, 0x01)
	script = append(script, 0x01, 0x99)
	script = append(script, 0x68)
	return script
}

func testWitness(leafScript, annex []byte) []byte {
	items := [][]byte{
		{0x30},
		leafScript,
		{0xc0},
	}
	if annex != nil {
		items = [][]byte{
			{0x30},
			leafScript,
			annex,
			{0xc0},
		}
	}

	witness := []byte{byte(len(items))}
	for _, item := range items {
		witness = append(witness, byte(len(item)))
		witness = append(witness, item...)
	}
	return witness
}
