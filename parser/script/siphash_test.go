package script

import (
	"encoding/binary"
	"testing"
)

// Official SipHash-2-4 test vectors from https://github.com/veorq/SipHash/blob/master/vectors.h
// Key: k[i] = i for i in 0..15 (k0=0x0706050403020100, k1=0x0f0e0d0c0b0a0908)
// Input n: in[i] = i for i in 0..n-1
// Values are the raw 8 bytes interpreted as little-endian uint64.
var siphashVectors = []uint64{
	0x726fdb47dd0e0e31, // len=0
	0x74f839c593dc67fd, // len=1
	0x0d6c8009d9a94f5a, // len=2
	0x85676696d7fb7e2d, // len=3
	0xcf2794e0277187b7, // len=4
	0x18765564cd99a68d, // len=5
	0xcbc9466e58fee3ce, // len=6
	0xab0200f58b01d137, // len=7
	0x93f5f5799a932462, // len=8
	0x9e0082df0ba9e4b0, // len=9
	0x7a5dbbc594ddb9f3, // len=10
	0xf4b32f46226bada7, // len=11
	0x751e8fbc860ee5fb, // len=12
	0x14ea5627c0843d90, // len=13
	0xf723ca908e7af2ee, // len=14
	0xa129ca6149be45e5, // len=15
}

func TestSipHash24Vectors(t *testing.T) {
	for n, expected := range siphashVectors {
		input := make([]byte, n)
		for i := range input {
			input[i] = byte(i)
		}
		got := siphash24(input)
		if got != expected {
			t.Errorf("siphash24(len=%d): got 0x%016x, want 0x%016x", n, got, expected)
		}
	}
}

func TestGetNFTBinIdFromTxIdAndIdx(t *testing.T) {
	txid := make([]byte, 32)
	for i := range txid {
		txid[i] = byte(i)
	}

	// Test fixed 16-byte output
	binId := GetNFTBinIdFromTxIdAndIdx(txid, 0)
	if len(binId) != NFTBinIdSize {
		t.Fatalf("binId length = %d, want %d", len(binId), NFTBinIdSize)
	}

	// First 8 bytes must be txid[:8]
	if binId[:8] != string(txid[:8]) {
		t.Errorf("binId prefix does not match txid[:8]")
	}

	// Different idx must produce different binId
	binId2 := GetNFTBinIdFromTxIdAndIdx(txid, 1)
	if binId == binId2 {
		t.Errorf("idx=0 and idx=1 produced same binId")
	}
	if len(binId2) != NFTBinIdSize {
		t.Errorf("binId2 length = %d, want %d", len(binId2), NFTBinIdSize)
	}

	// Same prefix (txid[:8])
	if binId2[:8] != binId[:8] {
		t.Errorf("same txid should have same 8-byte prefix")
	}

	// Different hash part
	if binId2[8:] == binId[8:] {
		t.Errorf("different idx should have different siphash")
	}
}

func TestGetNFTBinIdFromRaw(t *testing.T) {
	txid := make([]byte, 32)
	for i := range txid {
		txid[i] = byte(i + 0x10)
	}

	// idx=0: raw = txid (32 bytes)
	raw0 := make([]byte, 32)
	copy(raw0, txid)
	expected0 := GetNFTBinIdFromTxIdAndIdx(txid, 0)
	got0 := GetNFTBinIdFromRaw(raw0)
	if got0 != expected0 {
		t.Errorf("idx=0: GetNFTBinIdFromRaw != GetNFTBinIdFromTxIdAndIdx")
	}

	// idx=1: raw = txid + 0x01 (33 bytes)
	raw1 := make([]byte, 33)
	copy(raw1, txid)
	raw1[32] = 0x01
	expected1 := GetNFTBinIdFromTxIdAndIdx(txid, 1)
	got1 := GetNFTBinIdFromRaw(raw1)
	if got1 != expected1 {
		t.Errorf("idx=1: GetNFTBinIdFromRaw != GetNFTBinIdFromTxIdAndIdx")
	}

	// idx=256: raw = txid + 0x00 0x01 (34 bytes)
	raw256 := make([]byte, 34)
	copy(raw256, txid)
	binary.LittleEndian.PutUint16(raw256[32:], 256)
	expected256 := GetNFTBinIdFromTxIdAndIdx(txid, 256)
	got256 := GetNFTBinIdFromRaw(raw256)
	if got256 != expected256 {
		t.Errorf("idx=256: GetNFTBinIdFromRaw != GetNFTBinIdFromTxIdAndIdx")
	}

	// invalid lengths
	if GetNFTBinIdFromRaw(txid[:31]) != "" {
		t.Errorf("len=31 should return empty")
	}
	if GetNFTBinIdFromRaw(make([]byte, 37)) != "" {
		t.Errorf("len=37 should return empty")
	}
}

func BenchmarkSipHash24(b *testing.B) {
	var input [36]byte
	for i := range input {
		input[i] = byte(i)
	}
	for b.Loop() {
		siphash24(input[:])
	}
}

func BenchmarkGetNFTBinId(b *testing.B) {
	txid := make([]byte, 32)
	for i := range txid {
		txid[i] = byte(i)
	}
	for b.Loop() {
		GetNFTBinIdFromTxIdAndIdx(txid, 42)
	}
}
