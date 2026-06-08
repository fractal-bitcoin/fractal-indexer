package script

import "testing"

func TestAlkanesProtocolDetection(t *testing.T) {
	script := alkanesTestRunestoneScript(t, 1, 0)

	if !IsRunestone(script) {
		t.Fatal("expected runestone")
	}
	if !IsAlkanes(script) {
		t.Fatal("expected alkanes")
	}
}

func TestAlkanesProtocolDetectionAcrossChunks(t *testing.T) {
	script := alkanesTestRunestoneScript(t, 1, 16, 81, 0, 81, 0, 81, 0, 81, 0, 81, 0, 81, 0, 81, 0, 81, 0)

	if !IsAlkanes(script) {
		t.Fatal("expected chunked alkanes")
	}
}

func TestAlkanesProtoburnDetection(t *testing.T) {
	script := alkanesTestRunestoneScript(t, 13, 2, 83, 1)

	if !IsAlkanes(script) {
		t.Fatal("expected alkanes protoburn")
	}
}

func TestNonAlkanesProtocolIsIgnored(t *testing.T) {
	script := alkanesTestRunestoneScript(t, 2, 0)

	if IsAlkanes(script) {
		t.Fatal("unexpected alkanes")
	}
}

func TestInvalidAlkanesProtocolPayload(t *testing.T) {
	script := []byte{0x6a, 0x5d, 0x02, 0xff, 0x7f}

	if IsAlkanes(script) {
		t.Fatal("unexpected alkanes for incomplete protocol field")
	}
}

func alkanesTestRunestoneScript(t *testing.T, protostoneIntegers ...uint64) []byte {
	t.Helper()

	raw := make([]byte, 0, len(protostoneIntegers))
	for _, value := range protostoneIntegers {
		raw = appendTestVarint128(raw, runestoneInteger{lo: value})
	}

	payload := make([]byte, 0, len(raw)+4)
	for offset := 0; offset < len(raw); offset += protostoneChunkBytes {
		end := offset + protostoneChunkBytes
		if end > len(raw) {
			end = len(raw)
		}

		chunk := runestoneInteger{}
		for i, b := range raw[offset:end] {
			if i < 8 {
				chunk.lo |= uint64(b) << (8 * i)
			} else {
				chunk.hi |= uint64(b) << (8 * (i - 8))
			}
		}

		payload = appendTestVarint128(payload, runestoneInteger{lo: runesTagProtocol})
		payload = appendTestVarint128(payload, chunk)
	}

	if len(payload) > 75 {
		t.Fatalf("test payload too large for direct push: %d", len(payload))
	}

	script := []byte{0x6a, 0x5d, byte(len(payload))}
	return append(script, payload...)
}

func appendTestVarint128(dst []byte, value runestoneInteger) []byte {
	for {
		b := byte(value.lo & 0x7f)
		next := runestoneInteger{
			lo: (value.lo >> 7) | (value.hi << 57),
			hi: value.hi >> 7,
		}
		if next.isZero() {
			return append(dst, b)
		}

		dst = append(dst, b|0x80)
		value = next
	}
}
