package script

import "testing"

func TestRunestoneDetection(t *testing.T) {
	script := []byte{0x6a, 0x5d, 0x02, 0x02, 0x01}
	if !IsRunestone(script) {
		t.Fatal("expected runestone")
	}
	if !IsRunesEtching(script) {
		t.Fatal("expected etching")
	}
}

func TestRunestoneWithoutEtching(t *testing.T) {
	script := []byte{0x6a, 0x5d, 0x02, 0x02, 0x00}
	if !IsRunestone(script) {
		t.Fatal("expected runestone")
	}
	if IsRunesEtching(script) {
		t.Fatal("unexpected etching")
	}
}

func TestInvalidRunestonePayload(t *testing.T) {
	script := []byte{0x6a, 0x5d, 0x51}
	if !IsRunestone(script) {
		t.Fatal("expected runestone prefix")
	}
	if IsRunesEtching(script) {
		t.Fatal("unexpected etching for non-data-push payload")
	}
}
