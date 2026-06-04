package model

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"strings"
	"testing"
)

func testBRC20History() *BRC20History {
	return &BRC20History{
		BRC20HistoryBase: BRC20HistoryBase{
			Type:         5,
			Valid:        true,
			TxId:         "0123456789abcdef0123456789abcdef",
			Idx:          12345,
			Vout:         3,
			Offset:       9876543210,
			PkScriptFrom: string([]byte{0x00, 0x14, 0x01, 0x02, 0x03}),
			PkScriptTo:   string([]byte{0x51, 0x20, 0xaa, 0xbb, 0xcc, 0xdd}),
			Satoshi:      546,
			Fee:          -1000,
			Height:       840000,
			TxIdx:        77,
			BlockTime:    1710000000,
		},
		Inscription: InscriptionBRC20TickInfoHistory{
			Height:            839999,
			Data:              nil,
			InscriptionNumber: -42,
			TxId:              "fedcba9876543210fedcba9876543210i0",
			Idx:               9,
			Satoshi:           330,
		},
		Amount:              "123456789",
		OverallBalance:      "200000000",
		TransferableBalance: "100000000",
		AvailableBalance:    "100000000",
	}
}

func TestBRC20HistoryMarshalRoundTrip(t *testing.T) {
	history := testBRC20History()
	history.Inscription.Data = &InscriptionBRC20InfoResp{
		BRC20Tick:     "ordi",
		BRC20Max:      "21000000",
		BRC20Limit:    "1000",
		BRC20Amount:   "123",
		BRC20Decimal:  "18",
		BRC20Minted:   "456",
		BRC20SelfMint: "false",
	}

	var decoded BRC20History
	decoded.Unmarshal(history.Marshal())

	if !reflect.DeepEqual(&decoded, history) {
		t.Fatalf("round trip mismatch\nwant: %#v\ngot:  %#v", history, &decoded)
	}
}

func TestBRC20HistoryMarshalRoundTripWithoutInscriptionData(t *testing.T) {
	history := testBRC20History()

	var decoded BRC20History
	decoded.Unmarshal(history.Marshal())

	if !reflect.DeepEqual(&decoded, history) {
		t.Fatalf("round trip mismatch\nwant: %#v\ngot:  %#v", history, &decoded)
	}
	if decoded.Inscription.Data != nil {
		t.Fatalf("decoded inscription data = %#v, want nil", decoded.Inscription.Data)
	}
}

func TestBRC20HistoryMarshalRoundTripEmptyFields(t *testing.T) {
	history := &BRC20History{
		BRC20HistoryBase: BRC20HistoryBase{
			TxId: "00000000000000000000000000000000",
		},
		Inscription: InscriptionBRC20TickInfoHistory{
			Data: &InscriptionBRC20InfoResp{},
		},
	}

	var decoded BRC20History
	decoded.Unmarshal(history.Marshal())

	if !reflect.DeepEqual(&decoded, history) {
		t.Fatalf("round trip mismatch\nwant: %#v\ngot:  %#v", history, &decoded)
	}
}

func TestBRC20HistoryMarshalUsesFixedWidthNumbersAndRawScripts(t *testing.T) {
	history := &BRC20History{
		BRC20HistoryBase: BRC20HistoryBase{
			Type:         7,
			Valid:        true,
			TxId:         "abcdefghijklmnopqrstuvwxyz123456",
			Idx:          0x11223344,
			Vout:         0x55667788,
			Offset:       0x1122334455667788,
			PkScriptFrom: string([]byte{0x00, 0x14, 0x01, 0x02}),
			PkScriptTo:   string([]byte{0x51, 0x20, 0xff}),
			Satoshi:      0x0102030405060708,
			Fee:          -2,
			Height:       0x99aabbcc,
			TxIdx:        0xddeeff00,
			BlockTime:    0x12345678,
		},
		Inscription: InscriptionBRC20TickInfoHistory{
			Height:            1,
			InscriptionNumber: -3,
			Satoshi:           4,
			Idx:               5,
		},
	}

	encoded := history.Marshal()
	if encoded[0] != history.Type {
		t.Fatalf("type byte = %d, want %d", encoded[0], history.Type)
	}
	if encoded[1] != 1 {
		t.Fatalf("valid byte = %d, want 1", encoded[1])
	}
	if got := string(encoded[2:34]); got != history.TxId {
		t.Fatalf("txid bytes = %q, want %q", got, history.TxId)
	}

	offset := 34
	if got := binary.LittleEndian.Uint32(encoded[offset:]); got != history.Idx {
		t.Fatalf("idx = %#x, want %#x", got, history.Idx)
	}
	offset += 4
	if got := binary.LittleEndian.Uint32(encoded[offset:]); got != history.Vout {
		t.Fatalf("vout = %#x, want %#x", got, history.Vout)
	}
	offset += 4
	if got := binary.LittleEndian.Uint64(encoded[offset:]); got != history.Offset {
		t.Fatalf("offset = %#x, want %#x", got, history.Offset)
	}
	offset += 8

	fromScript := []byte(history.PkScriptFrom)
	if got := binary.LittleEndian.Uint32(encoded[offset:]); got != uint32(len(fromScript)) {
		t.Fatalf("from script length = %d, want %d", got, len(fromScript))
	}
	offset += 4
	if !bytes.Equal(encoded[offset:offset+len(fromScript)], fromScript) {
		t.Fatalf("from script bytes = %x, want %x", encoded[offset:offset+len(fromScript)], fromScript)
	}
	offset += len(fromScript)

	toScript := []byte(history.PkScriptTo)
	if got := binary.LittleEndian.Uint32(encoded[offset:]); got != uint32(len(toScript)) {
		t.Fatalf("to script length = %d, want %d", got, len(toScript))
	}
	offset += 4
	if !bytes.Equal(encoded[offset:offset+len(toScript)], toScript) {
		t.Fatalf("to script bytes = %x, want %x", encoded[offset:offset+len(toScript)], toScript)
	}
	offset += len(toScript)

	if got := binary.LittleEndian.Uint64(encoded[offset:]); got != history.Satoshi {
		t.Fatalf("satoshi = %#x, want %#x", got, history.Satoshi)
	}
	offset += 8
	if got := int64(binary.LittleEndian.Uint64(encoded[offset:])); got != history.Fee {
		t.Fatalf("fee = %d, want %d", got, history.Fee)
	}
	offset += 8
	if got := binary.LittleEndian.Uint32(encoded[offset:]); got != history.Height {
		t.Fatalf("height = %#x, want %#x", got, history.Height)
	}
	offset += 4
	if got := binary.LittleEndian.Uint32(encoded[offset:]); got != history.TxIdx {
		t.Fatalf("txidx = %#x, want %#x", got, history.TxIdx)
	}
	offset += 4
	if got := binary.LittleEndian.Uint32(encoded[offset:]); got != history.BlockTime {
		t.Fatalf("blocktime = %#x, want %#x", got, history.BlockTime)
	}
}

func TestBRC20HistoryMarshalLengthLimitedStrings(t *testing.T) {
	history := testBRC20History()
	history.Amount = strings.Repeat("a", 40)
	history.OverallBalance = strings.Repeat("b", 40)
	history.TransferableBalance = strings.Repeat("c", 40)
	history.AvailableBalance = strings.Repeat("d", 40)
	history.Inscription.TxId = strings.Repeat("e", 70)
	history.Inscription.Data = &InscriptionBRC20InfoResp{
		BRC20Tick:     strings.Repeat("f", 16),
		BRC20Max:      strings.Repeat("g", 40),
		BRC20Limit:    strings.Repeat("h", 40),
		BRC20Amount:   strings.Repeat("i", 40),
		BRC20Decimal:  strings.Repeat("j", 8),
		BRC20Minted:   strings.Repeat("k", 40),
		BRC20SelfMint: strings.Repeat("l", 8),
	}

	var decoded BRC20History
	decoded.Unmarshal(history.Marshal())

	if decoded.Amount != "" || decoded.OverallBalance != "" ||
		decoded.TransferableBalance != "" || decoded.AvailableBalance != "" ||
		decoded.Inscription.TxId != "" {
		t.Fatalf("limited history strings were not omitted: %#v", decoded)
	}
	if decoded.Inscription.Data == nil {
		t.Fatal("decoded inscription data is nil")
	}
	if *decoded.Inscription.Data != (InscriptionBRC20InfoResp{}) {
		t.Fatalf("limited data strings were not omitted: %#v", decoded.Inscription.Data)
	}
	if decoded.PkScriptFrom != history.PkScriptFrom || decoded.PkScriptTo != history.PkScriptTo {
		t.Fatalf("scripts changed after limited string decode")
	}
}

func TestBRC20HistoryUnmarshalTruncatedDataDoesNotPanic(t *testing.T) {
	history := testBRC20History()
	history.Inscription.Data = &InscriptionBRC20InfoResp{
		BRC20Tick:     "ordi",
		BRC20Max:      "21000000",
		BRC20Limit:    "1000",
		BRC20Amount:   "123",
		BRC20Decimal:  "18",
		BRC20Minted:   "456",
		BRC20SelfMint: "false",
	}
	encoded := history.Marshal()

	for n := 0; n < len(encoded); n++ {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Unmarshal panicked for prefix length %d: %v", n, r)
				}
			}()

			var decoded BRC20History
			decoded.Unmarshal(encoded[:n])
		}()
	}
}
