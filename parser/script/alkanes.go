package script

const (
	runesTagProtocol = 16383

	alkanesProtocolTag    = 1
	protorunesProtocolTag = 13

	protostoneTagBody = 0
	protostoneTagBurn = 83

	protostoneChunkBytes = 15
)

type runestoneInteger struct {
	lo uint64
	hi uint64
}

func (v runestoneInteger) isZero() bool {
	return v.lo == 0 && v.hi == 0
}

func (v runestoneInteger) equalsUint64(n uint64) bool {
	return v.lo == n && v.hi == 0
}

func (v runestoneInteger) toUint64() (uint64, bool) {
	if v.hi != 0 {
		return 0, false
	}
	return v.lo, true
}

func (v runestoneInteger) append15Bytes(dst []byte) []byte {
	for i := 0; i < 8; i++ {
		dst = append(dst, byte(v.lo>>(8*i)))
	}
	for i := 0; i < 7; i++ {
		dst = append(dst, byte(v.hi>>(8*i)))
	}
	return dst
}

func IsAlkanes(pkScript []byte) bool {
	payload, ok := runestonePayload(pkScript)
	if !ok {
		return false
	}

	protocolValues, ok := runestoneProtocolValues(payload)
	if !ok || len(protocolValues) == 0 {
		return false
	}
	return protostonesHaveAlkanes(protocolValues)
}

func runestoneProtocolValues(payload []byte) ([]runestoneInteger, bool) {
	values := make([]runestoneInteger, 0, 1)
	for offset := 0; offset < len(payload); {
		tag, n := decodeRunestoneVarint128(payload[offset:])
		if n == 0 {
			return nil, false
		}
		offset += n

		if tag.equalsUint64(protostoneTagBody) {
			break
		}

		value, n := decodeRunestoneVarint128(payload[offset:])
		if n == 0 {
			return nil, false
		}
		offset += n

		if tag.equalsUint64(runesTagProtocol) {
			values = append(values, value)
		}
	}
	return values, true
}

func protostonesHaveAlkanes(values []runestoneInteger) bool {
	raw := make([]byte, 0, len(values)*protostoneChunkBytes)
	for _, value := range values {
		raw = value.append15Bytes(raw)
	}

	integers, ok := decodeRunestoneIntegers(raw)
	if !ok {
		return false
	}

	for offset := 0; offset < len(integers); {
		protocolTag := integers[offset]
		offset++
		if protocolTag.isZero() {
			break
		}
		if offset >= len(integers) {
			return false
		}

		length, ok := integers[offset].toUint64()
		if !ok {
			return false
		}
		offset++
		if length > uint64(len(integers)-offset) {
			return false
		}

		fields := integers[offset : offset+int(length)]
		offset += int(length)

		if protocolTag.equalsUint64(alkanesProtocolTag) {
			return true
		}
		if protocolTag.equalsUint64(protorunesProtocolTag) && protostoneFieldsHaveAlkanesBurn(fields) {
			return true
		}
	}
	return false
}

func protostoneFieldsHaveAlkanesBurn(fields []runestoneInteger) bool {
	for offset := 0; offset < len(fields); {
		tag := fields[offset]
		offset++
		if tag.equalsUint64(protostoneTagBody) {
			break
		}
		if offset >= len(fields) {
			return false
		}

		value := fields[offset]
		offset++
		if tag.equalsUint64(protostoneTagBurn) && value.equalsUint64(alkanesProtocolTag) {
			return true
		}
	}
	return false
}

func decodeRunestoneIntegers(data []byte) ([]runestoneInteger, bool) {
	integers := make([]runestoneInteger, 0, len(data))
	for offset := 0; offset < len(data); {
		value, n := decodeRunestoneVarint128(data[offset:])
		if n == 0 {
			return nil, false
		}
		offset += n
		integers = append(integers, value)
	}
	return integers, true
}

func decodeRunestoneVarint128(data []byte) (runestoneInteger, int) {
	var value runestoneInteger
	for i, b := range data {
		if i >= 18 {
			return runestoneInteger{}, 0
		}

		part := uint64(b & 0x7f)
		shift := uint(i * 7)
		if shift == 126 && part > 3 {
			return runestoneInteger{}, 0
		}

		if shift < 64 {
			value.lo |= part << shift
			if shift > 57 {
				value.hi |= part >> (64 - shift)
			}
		} else {
			value.hi |= part << (shift - 64)
		}

		if b&0x80 == 0 {
			return value, i + 1
		}
	}
	return runestoneInteger{}, 0
}
