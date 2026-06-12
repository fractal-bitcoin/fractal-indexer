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
	if IsRunesMint(script) {
		t.Fatal("unexpected mint")
	}
}

func TestRunestoneMint(t *testing.T) {
	script := []byte{0x6a, 0x5d, 0x04, 0x14, 0x01, 0x14, 0x02}
	if !IsRunestone(script) {
		t.Fatal("expected runestone")
	}
	if !IsRunesMint(script) {
		t.Fatal("expected mint")
	}
	if IsRunesEtching(script) {
		t.Fatal("unexpected etching")
	}
}

func TestRunestoneIncompleteMint(t *testing.T) {
	script := []byte{0x6a, 0x5d, 0x02, 0x14, 0x01}
	if !IsRunestone(script) {
		t.Fatal("expected runestone")
	}
	if IsRunesMint(script) {
		t.Fatal("unexpected mint")
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
	if IsRunesMint(script) {
		t.Fatal("unexpected mint for non-data-push payload")
	}
}
