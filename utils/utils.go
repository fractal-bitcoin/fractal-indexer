package utils

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fractal-indexer/constant"
	"fractal-indexer/model"
	scriptDecoder "fractal-indexer/parser/script"
	"hash"
	"sync"
)

var sha256Pool = sync.Pool{
	New: func() interface{} { return sha256.New() },
}

func CalcBlockSubsidy(height uint32) uint64 {
	var SubsidyReductionInterval uint32 = 210000
	var SatoshiPerBitcoin uint64 = 100000000
	var baseSubsidy = 50 * SatoshiPerBitcoin
	if constant.CHAIN_TYPE == constant.CHAIN_TYPE_FRACTAL {
		SubsidyReductionInterval = 2100000
		baseSubsidy = 25 * SatoshiPerBitcoin
		if height == 0 {
			return baseSubsidy * 2
		}
		if height == 1 {
			return baseSubsidy * 2 * uint64(SubsidyReductionInterval)
		}
	}
	// Equivalent to: baseSubsidy / 2^(height/subsidyHalvingInterval)
	return baseSubsidy >> uint(height/SubsidyReductionInterval)
}

func DecodeVarIntForBlock(raw []byte) (cnt uint, cnt_size uint) {
	if raw[0] < 0xfd {
		return uint(raw[0]), 1
	} else if raw[0] == 0xfd {
		return uint(binary.LittleEndian.Uint16(raw[1:3])), 3
	} else if raw[0] == 0xfe {
		return uint(binary.LittleEndian.Uint32(raw[1:5])), 5
	} else {
		return uint(binary.LittleEndian.Uint64(raw[1:9])), 9
	}
}

func EncodeVarIntForBlock(cnt uint64, raw []byte) (cnt_size int) {
	if cnt < 0xfd {
		raw[0] = byte(cnt)
		return 1
	} else if cnt <= 0xffff {
		raw[0] = 0xfd
		binary.LittleEndian.PutUint16(raw[1:3], uint16(cnt))
		return 3
	} else if cnt <= 0xffffffff {
		raw[0] = 0xfe
		binary.LittleEndian.PutUint32(raw[1:5], uint32(cnt))
		return 5
	} else {
		raw[0] = 0xff
		binary.LittleEndian.PutUint64(raw[1:9], uint64(cnt))
		return 9
	}
}

func NewTxWit(txwitraw []byte) (wits []*model.TxWit, offset uint) {
	txWitcnt, txWitcntsize := DecodeVarIntForBlock(txwitraw[0:])
	offset = txWitcntsize

	wits = make([]*model.TxWit, txWitcnt)
	for witIndex := uint(0); witIndex < txWitcnt; witIndex++ {
		txWitScriptcnt, txWitScriptcntsize := DecodeVarIntForBlock(txwitraw[offset:])
		offset += txWitScriptcntsize

		txwit := new(model.TxWit)
		txwit.Script = txwitraw[offset : offset+txWitScriptcnt]

		wits[witIndex] = txwit
		offset += txWitScriptcnt
	}
	return
}

// getWitNFTScript extracts the NFT script from witness data without allocating.
// It tracks the last 3 script positions using a ring buffer.
func getWitNFTScript(witraw []byte) (nftScript []byte, ok bool) {
	cnt, cntsize := DecodeVarIntForBlock(witraw)
	offset := cntsize

	if cnt < 2 {
		return nil, false
	}

	type span struct{ start, end uint }
	var ring [3]span

	for i := uint(0); i < cnt; i++ {
		slen, slensize := DecodeVarIntForBlock(witraw[offset:])
		offset += slensize
		ring[2] = ring[1]
		ring[1] = ring[0]
		ring[0] = span{offset, offset + slen}
		offset += slen
	}

	if offset != uint(len(witraw)) {
		return nil, false
	}

	last := ring[0]
	hasAnnex := last.end > last.start && witraw[last.start] == 0x50
	if hasAnnex && cnt < 3 {
		return nil, false
	}

	if hasAnnex {
		return witraw[ring[2].start:ring[2].end], true
	}
	return witraw[ring[1].start:ring[1].end], true
}

// before jubilee
func EncodeTxNFT(tx *model.Tx) {
	for vin := range tx.TxIns {
		// Only supports the NFT in the first input.
		if vin != 0 {
			break
		}
		if len(tx.TxIns[vin].ScriptWitness) == 0 {
			break
		}

		nftScript, ok := getWitNFTScript(tx.TxIns[vin].ScriptWitness)
		if !ok {
			break
		}

		if nft, ok := scriptDecoder.ExtractPkScriptForNFT(nftScript); ok {
			nft.InTxVin = uint32(vin)
			if isTextContentType(&nft) {
				nft.IsText = true
				nft.IsBRC20, nft.IsBRC20Ext, nft.IsBRC20Mint, nft.IsBRC20Tran = isBRC20(&nft)
			}
			tx.TxIns[vin].CreatePointCountOfNewNFTs += 1
			tx.NewNFTDataCreated = append(tx.NewNFTDataCreated, nft)
			tx.GenesisNewNFT = true
		}
	}
}

// after jubilee — appends NFTData to slab, assigns sub-slice to tx.NewNFTDataCreated
func EncodeTxNFTAfterJubilee(tx *model.Tx, height uint32, slab []scriptDecoder.NFTData) []scriptDecoder.NFTData {
	txStart := len(slab)
	for vin := range tx.TxIns {
		if len(tx.TxIns[vin].ScriptWitness) == 0 {
			continue
		}

		nftScript, ok := getWitNFTScript(tx.TxIns[vin].ScriptWitness)
		if !ok {
			continue
		}

		prevLen := len(slab)
		slab = scriptDecoder.ExtractPkScriptForNFTJubilee(nftScript, slab)
		for idx := prevLen; idx < len(slab); idx++ {
			nft := &slab[idx]
			nft.InTxVin = uint32(vin)
			if isCursed(nft) {
				nft.IsCursed = true
			}
			if idx-prevLen != 0 {
				nft.IsCursed = true
			}

			if isTextContentType(nft) {
				nft.IsText = true
				if !nft.IsCursed || height >= constant.BRC20_SINGLE_STEP_TRANSFER_HEIGHT {
					nft.IsBRC20, nft.IsBRC20Ext, nft.IsBRC20Mint, nft.IsBRC20Tran = isBRC20(nft)
				}
			}
			tx.TxIns[vin].CreatePointCountOfNewNFTs += 1
			tx.GenesisNewNFT = true
		}
	}
	if len(slab) > txStart {
		tx.NewNFTDataCreated = slab[txStart:len(slab)]
	}
	return slab
}

func isTextContentType(nft *scriptDecoder.NFTData) bool {
	if len(nft.ContentBody) > 400*1024*1024 {
		return false
	}
	if bytes.HasPrefix(nft.ContentType, []byte("text/plain;")) {
		return true
	}
	if bytes.Equal(nft.ContentType, []byte("text/plain")) {
		return true
	}
	if bytes.Equal(nft.ContentType, []byte("application/json")) {
		return true
	}
	return false
}

func isCursed(nft *scriptDecoder.NFTData) bool {
	if nft.IsUnrecognizedEven { // This is not only cursed, but also unbound.
		return true
	}
	if nft.IsPushnum {
		return true
	}
	if nft.IsDuplicateField {
		return true
	}
	if nft.IsIncompleteField {
		return true
	}

	if nft.InTxVin != 0 {
		return true
	}

	if nft.HasPointer {
		return true
	}

	if nft.IsStutter {
		return true
	}

	// Defer further handling to serial processing; input UTXO data is unavailable here, so this cannot be identified.
	// if nft.IsReinscription {
	// 	return true
	// }

	return false
}

// jsonValueBytes finds "key":"value" in simple JSON and returns value as []byte (zero-alloc).
func jsonValueBytes(data []byte, key []byte) []byte {
	idx := bytes.Index(data, key)
	if idx < 0 {
		return nil
	}
	pos := idx + len(key)
	// skip whitespace and colon
	for pos < len(data) && (data[pos] == ' ' || data[pos] == '\t' || data[pos] == '\n' || data[pos] == '\r' || data[pos] == ':') {
		pos++
	}
	if pos >= len(data) || data[pos] != '"' {
		return nil
	}
	pos++ // skip opening quote
	end := pos
	for end < len(data) && data[end] != '"' {
		if data[end] == '\\' {
			end++ // skip escaped char
		}
		end++
	}
	return data[pos:end]
}

var (
	jsonKeyP    = []byte(`"p"`)
	jsonKeyOp   = []byte(`"op"`)
	jsonKeyTick = []byte(`"tick"`)

	valBRC20     = []byte("brc-20")
	valBRC20Mod  = []byte("brc20-module")
	valBRC20Swap = []byte("brc20-swap")
	valMint      = []byte("mint")
	valTransfer  = []byte("transfer")
	valCondAppr  = []byte("conditional-approve")

	// fast-path prefixes for common brc-20 mint/transfer
	mintPrefix     = []byte(`{"p":"brc-20","op":"mint","tick":"`)
	transferPrefix = []byte(`{"p":"brc-20","op":"transfer","tick":"`)
	amtInfix       = []byte(`","amt":"`)
)

// isTickChar returns true for valid BRC-20 tick characters: [0-9a-zA-Z_-]
func isTickChar(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c == '-'
}

// isAmtChar returns true for valid BRC-20 amt characters: [0-9.+-]
func isAmtChar(c byte) bool {
	return (c >= '0' && c <= '9') || c == '.' || c == '+' || c == '-'
}

// tryFastBRC20 attempts to fully validate a common-form brc-20 mint/transfer JSON.
// Expected format: {"p":"brc-20","op":"mint","tick":"XXXX","amt":"NNN"}
// Returns (matched, isMint, isTransfer). If matched=false, caller must use slow path.
func tryFastBRC20(content []byte) (matched, isMint, isTransfer bool) {
	var prefix []byte
	var mint, transfer bool

	if bytes.HasPrefix(content, mintPrefix) {
		prefix = mintPrefix
		mint = true
	} else if bytes.HasPrefix(content, transferPrefix) {
		prefix = transferPrefix
		transfer = true
	} else {
		return false, false, false
	}

	pos := len(prefix)
	rest := content[pos:]

	// validate tick: at least 1 char, all isTickChar, ends with "
	tickEnd := 0
	for tickEnd < len(rest) && isTickChar(rest[tickEnd]) {
		tickEnd++
	}
	if tickEnd == 0 || tickEnd >= len(rest) || rest[tickEnd] != '"' {
		return false, false, false
	}
	pos += tickEnd + 1 // skip tick + closing quote
	rest = content[pos:]

	// expect ,"amt":"
	if !bytes.HasPrefix(rest, amtInfix) {
		return false, false, false
	}
	pos += len(amtInfix)
	rest = content[pos:]

	// validate amt: at least 1 char, all isAmtChar, ends with "
	amtEnd := 0
	for amtEnd < len(rest) && isAmtChar(rest[amtEnd]) {
		amtEnd++
	}
	if amtEnd == 0 || amtEnd >= len(rest) || rest[amtEnd] != '"' {
		return false, false, false
	}
	pos += amtEnd + 1 // skip amt + closing quote
	rest = content[pos:]

	// expect exactly }
	if len(rest) != 1 || rest[0] != '}' {
		return false, false, false
	}

	return true, mint, transfer
}

func isBRC20(nft *scriptDecoder.NFTData) (base, ext, mint, transfer bool) {
	if len(nft.ContentBody) < 40 {
		return
	}

	content := bytes.TrimSpace(nft.ContentBody)
	if len(content) < 2 || content[0] != '{' || content[len(content)-1] != '}' {
		return
	}

	// fast path: fully validated common-form mint/transfer (~90% of inscriptions)
	if matched, isMint, isTransfer := tryFastBRC20(content); matched {
		return true, false, isMint, isTransfer
	}

	// slow path: general byte scanning
	proto := jsonValueBytes(content, jsonKeyP)
	if proto == nil {
		return
	}

	if bytes.Equal(proto, valBRC20) {
		tick := jsonValueBytes(content, jsonKeyTick)
		if len(tick) == 0 {
			return
		}
		base = true
		op := jsonValueBytes(content, jsonKeyOp)
		if bytes.Equal(op, valMint) {
			mint = true
		} else if bytes.Equal(op, valTransfer) {
			transfer = true
		}
		return
	}

	if bytes.Equal(proto, valBRC20Mod) {
		base = true
		return
	}

	if bytes.Equal(proto, valBRC20Swap) {
		base = true
		op := jsonValueBytes(content, jsonKeyOp)
		if bytes.Equal(op, valCondAppr) {
			ext = true
		}
		return
	}

	return
}

func GetWitnessHash256(data []byte, witOffset uint32) []byte {
	sha := sha256Pool.Get().(hash.Hash)
	sha.Reset()
	sha.Write(data[:4]) // version
	// skip 2 bytes
	sha.Write(data[4+2 : witOffset]) // inputs/outputs
	// skip witness
	sha.Write(data[len(data)-4:]) // locktime
	var tmp [32]byte
	sha.Sum(tmp[:0])
	sha.Reset()
	sha.Write(tmp[:])
	result := sha.Sum(nil)
	sha256Pool.Put(sha)
	return result
}

// GetWitnessHash256Into writes the double-SHA256 witness hash into dst (must be >=32 bytes).
func GetWitnessHash256Into(dst []byte, data []byte, witOffset uint32) {
	sha := sha256Pool.Get().(hash.Hash)
	sha.Reset()
	sha.Write(data[:4])
	sha.Write(data[4+2 : witOffset])
	sha.Write(data[len(data)-4:])
	var tmp [32]byte
	sha.Sum(tmp[:0])
	sha.Reset()
	sha.Write(tmp[:])
	sha.Sum(dst[:0])
	sha256Pool.Put(sha)
}

func GetHash256(data []byte) []byte {
	sha := sha256Pool.Get().(hash.Hash)
	sha.Reset()
	sha.Write(data[:])
	var tmp [32]byte
	sha.Sum(tmp[:0])
	sha.Reset()
	sha.Write(tmp[:])
	result := sha.Sum(nil)
	sha256Pool.Put(sha)
	return result
}

// GetHash256Into writes the double-SHA256 hash into dst (must be >=32 bytes).
func GetHash256Into(dst []byte, data []byte) {
	sha := sha256Pool.Get().(hash.Hash)
	sha.Reset()
	sha.Write(data[:])
	var tmp [32]byte
	sha.Sum(tmp[:0])
	sha.Reset()
	sha.Write(tmp[:])
	sha.Sum(dst[:0])
	sha256Pool.Put(sha)
}

func HashString(data []byte) (res string) {
	const length = 32
	var reverseData [length]byte

	// need reverse
	for i := 0; i < length; i++ {
		reverseData[i] = data[length-i-1]
	}

	return hex.EncodeToString(reverseData[:])
}

func GetNFTIdForScript(txid string, idx uint32) (scriptId string) {
	if idx == 0 {
		return txid
	}

	var n [4]byte
	binary.LittleEndian.PutUint32(n[:], uint32(idx))

	if idx < 256 {
		return txid + string(n[:1])
	} else if idx < 65536 {
		return txid + string(n[:2])
	} else if idx < 16777216 {
		return txid + string(n[:3])
	} else {
		return txid + string(n[:])
	}
}
