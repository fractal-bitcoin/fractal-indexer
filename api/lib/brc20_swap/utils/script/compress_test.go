package script

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func mustDecodeHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func TestVLQRoundTrip(t *testing.T) {
	tests := []uint64{0, 1, 127, 128, 16511, 16512, 1<<32 - 1, ^uint64(0)}

	for _, val := range tests {
		var encoded [10]byte
		n := PutVLQ(encoded[:], val)
		got, bytesRead := DeserializeVLQ(encoded[:n])
		if got != val {
			t.Fatalf("DeserializeVLQ(%x) = %d, want %d", encoded[:n], got, val)
		}
		if bytesRead != n {
			t.Fatalf("DeserializeVLQ(%x) read %d bytes, want %d", encoded[:n], bytesRead, n)
		}
	}
}

func TestScriptCompressionRoundTrip(t *testing.T) {
	tests := []struct {
		name         string
		uncompressed []byte
		compressed   []byte
	}{
		{
			name:         "p2pkh",
			uncompressed: mustDecodeHex("76a9141018853670f9f3b0582c5b9ee8ce93764ac32b9388ac"),
			compressed:   mustDecodeHex("001018853670f9f3b0582c5b9ee8ce93764ac32b93"),
		},
		{
			name:         "p2sh",
			uncompressed: mustDecodeHex("a914da1745e9b549bd0bfa1a569971c77eba30cd5a4b87"),
			compressed:   mustDecodeHex("01da1745e9b549bd0bfa1a569971c77eba30cd5a4b"),
		},
		{
			name:         "p2pk compressed",
			uncompressed: mustDecodeHex("2102192d74d0cb94344c9569c2e77901573d8d7903c3ebec3a957724895dca52c6b4ac"),
			compressed:   mustDecodeHex("02192d74d0cb94344c9569c2e77901573d8d7903c3ebec3a957724895dca52c6b4"),
		},
		{
			name:         "p2pk uncompressed legacy format",
			uncompressed: mustDecodeHex("4104192d74d0cb94344c9569c2e77901573d8d7903c3ebec3a957724895dca52c6b40d45264838c0bd96852662ce6a847b197376830160c6d2eb5e6a4c44d33f453eac"),
			compressed:   mustDecodeHex("04192d74d0cb94344c9569c2e77901573d8d7903c3ebec3a957724895dca52c6b40d45264838c0bd96852662ce6a847b197376830160c6d2eb5e6a4c44d33f453e"),
		},
	}

	for _, test := range tests {
		gotSize := compressedScriptSize(test.uncompressed)
		if gotSize != len(test.compressed) {
			t.Fatalf("%s compressedScriptSize = %d, want %d", test.name, gotSize, len(test.compressed))
		}

		gotCompressed := make([]byte, gotSize)
		bytesWritten := PutCompressedScript(gotCompressed, test.uncompressed)
		if bytesWritten != len(test.compressed) {
			t.Fatalf("%s PutCompressedScript wrote %d bytes, want %d", test.name, bytesWritten, len(test.compressed))
		}
		if !bytes.Equal(gotCompressed, test.compressed) {
			t.Fatalf("%s PutCompressedScript = %x, want %x", test.name, gotCompressed, test.compressed)
		}

		gotDecodedSize := DecodeCompressedScriptSize(test.compressed)
		if gotDecodedSize != len(test.compressed) {
			t.Fatalf("%s DecodeCompressedScriptSize = %d, want %d", test.name, gotDecodedSize, len(test.compressed))
		}

		gotDecompressed := DecompressScript(test.compressed)
		if !bytes.Equal(gotDecompressed, test.uncompressed) {
			t.Fatalf("%s DecompressScript = %x, want %x", test.name, gotDecompressed, test.uncompressed)
		}
	}
}
