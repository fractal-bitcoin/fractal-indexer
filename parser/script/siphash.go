package script

import (
	"encoding/binary"
	"math/bits"
)

// Fixed SipHash key (arbitrary, only needs to be deterministic).
const (
	sipKey0 = 0x0706050403020100
	sipKey1 = 0x0f0e0d0c0b0a0908
)

// siphash24 computes SipHash-2-4 with the fixed key.
func siphash24(data []byte) uint64 {
	var v0 uint64 = sipKey0 ^ 0x736f6d6570736575
	var v1 uint64 = sipKey1 ^ 0x646f72616e646f6d
	var v2 uint64 = sipKey0 ^ 0x6c7967656e657261
	var v3 uint64 = sipKey1 ^ 0x7465646279746573

	length := len(data)
	blocks := length / 8

	for i := 0; i < blocks; i++ {
		m := binary.LittleEndian.Uint64(data[i*8:])
		v3 ^= m
		v0, v1, v2, v3 = sipRound(v0, v1, v2, v3)
		v0, v1, v2, v3 = sipRound(v0, v1, v2, v3)
		v0 ^= m
	}

	// last block: remaining bytes + length in top byte
	var last uint64
	tail := data[blocks*8:]
	for i := range tail {
		last |= uint64(tail[i]) << (uint(i) * 8)
	}
	last |= uint64(length) << 56

	v3 ^= last
	v0, v1, v2, v3 = sipRound(v0, v1, v2, v3)
	v0, v1, v2, v3 = sipRound(v0, v1, v2, v3)
	v0 ^= last

	// finalization: 4 rounds
	v2 ^= 0xff
	v0, v1, v2, v3 = sipRound(v0, v1, v2, v3)
	v0, v1, v2, v3 = sipRound(v0, v1, v2, v3)
	v0, v1, v2, v3 = sipRound(v0, v1, v2, v3)
	v0, v1, v2, v3 = sipRound(v0, v1, v2, v3)

	return v0 ^ v1 ^ v2 ^ v3
}

func sipRound(v0, v1, v2, v3 uint64) (uint64, uint64, uint64, uint64) {
	v0 += v1
	v1 = bits.RotateLeft64(v1, 13)
	v1 ^= v0
	v0 = bits.RotateLeft64(v0, 32)
	v2 += v3
	v3 = bits.RotateLeft64(v3, 16)
	v3 ^= v2
	v0 += v3
	v3 = bits.RotateLeft64(v3, 21)
	v3 ^= v0
	v2 += v1
	v1 = bits.RotateLeft64(v1, 17)
	v1 ^= v2
	v2 = bits.RotateLeft64(v2, 32)
	return v0, v1, v2, v3
}
