package model

import (
	"encoding/binary"
	"runtime"
)

// OutpointKeyTxidSize is the truncated txid prefix length used in outpoint keys.
const OutpointKeyTxidSize = 16

// OutpointKeySize is the fixed internal outpoint key size: txid[:16] + vout(4 LE) = 20.
const OutpointKeySize = OutpointKeyTxidSize + 4

// MakeOutpointKey builds a fixed 20-byte outpoint key into buf.
// Format: txid[:16] + little-endian uint32 vout.
// Returns OutpointKeySize (20).
func MakeOutpointKey(buf []byte, txid []byte, vout uint32) {
	copy(buf[:OutpointKeyTxidSize], txid[:OutpointKeyTxidSize])
	binary.LittleEndian.PutUint32(buf[OutpointKeyTxidSize:], vout)
}

// TrimOutpointKey trims trailing zero bytes from a fixed 20-byte key.
// Used only at external storage boundaries (pika, ClickHouse).
func TrimOutpointKey(key string) string {
	n := len(key)
	for n > OutpointKeyTxidSize && key[n-1] == 0 {
		n--
	}
	return key[:n]
}

var (
	GlobalNewUtxoDataMap    map[string]*TxoData
	GlobalDeleteUtxoKeysMap []string
	GlobalSpentUtxoCount    int

	GlobalMempoolNewUtxoDataMap map[string]*TxoData

	GlobalNewInscriptionsId2CreateIdxKeyMap map[string]uint64
)

// lastBatch sizes for pre-allocation — updated by SnapshotBatchSizes before each CleanUtxoMap.
var (
	lastBatchSpentCount int
	lastBatchNewCount   int
)

// SnapshotBatchSizes records current batch sizes so CleanUtxoMap can pre-allocate next batch.
// Must be called immediately before CleanUtxoMap.
func SnapshotBatchSizes() {
	lastBatchSpentCount = len(GlobalDeleteUtxoKeysMap)
	lastBatchNewCount = len(GlobalNewUtxoDataMap)
}

func init() {
	CleanUtxoMap()
}

// Clears local map memory.
func CleanUtxoMap() {
	GlobalNewUtxoDataMap = nil
	GlobalDeleteUtxoKeysMap = nil
	GlobalNewInscriptionsId2CreateIdxKeyMap = nil
	runtime.GC()

	cap1 := lastBatchSpentCount
	if cap1 < 64*1024 {
		cap1 = 64 * 1024
	}
	cap2 := lastBatchNewCount
	if cap2 < 64*1024 {
		cap2 = 64 * 1024
	}
	GlobalNewUtxoDataMap = make(map[string]*TxoData, cap2)
	GlobalDeleteUtxoKeysMap = make([]string, 0, cap1)
	GlobalSpentUtxoCount = 0

	GlobalNewInscriptionsId2CreateIdxKeyMap = make(map[string]uint64, 0)
}

// Clears local map memory.
func CleanMempoolUtxoMap() {
	GlobalMempoolNewUtxoDataMap = nil

	runtime.GC()

	GlobalMempoolNewUtxoDataMap = make(map[string]*TxoData, 0)
}
