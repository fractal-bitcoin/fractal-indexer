package utils

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/unisat-wallet/libbrc20-indexer/conf"
)

func DecodeTokensFromSwapPair(tickPair string) (token0, token1 string, err error) {
	slashIdx := strings.Index(tickPair, "/")
	if len(tickPair) < conf.TICK_MIN_LEN*2+1 ||
		len(tickPair) > conf.TICK_MAX_LEN*2+1 ||
		slashIdx < conf.TICK_MIN_LEN ||
		slashIdx > conf.TICK_MAX_LEN ||
		len(tickPair)-(slashIdx+1) < conf.TICK_MIN_LEN {
		return "", "", errors.New("func: removeLiq tickPair invalid")
	}
	token0 = tickPair[:slashIdx]
	token1 = tickPair[slashIdx+1:]

	return token0, token1, nil
}

func GetValidUniqueLowerTickerTicker(ticker string) (lowerTicker string, err error) {
	if len(ticker) < conf.TICK_MIN_LEN || len(ticker) > conf.TICK_MAX_LEN {
		return "", errors.New("ticker len invalid")
	}

	for _, b := range []byte(ticker) {
		if TickerB63[b] > 63 {
			return "", errors.New("ticker invalid")
		}
	}

	lowerTicker = strings.ToLower(ticker)
	return lowerTicker, nil
}

// single sha256 hash
func GetSha256(data []byte) (hash []byte) {
	sha := sha256.New()
	sha.Write(data[:])
	hash = sha.Sum(nil)
	return
}

func HashString(data []byte) (res string) {
	length := 32
	var reverseData [32]byte

	// need reverse
	for i := 0; i < length; i++ {
		reverseData[i] = data[length-i-1]
	}
	return hex.EncodeToString(reverseData[:])
}

func ReverseBytes(data []byte) (result []byte) {
	for _, b := range data {
		result = append([]byte{b}, result...)
	}
	return result
}

// PayToTaprootScript creates a pk script for a pay-to-taproot output key.
func PayToTaprootScript(taprootKey *btcec.PublicKey) ([]byte, error) {
	return txscript.NewScriptBuilder().
		AddOp(txscript.OP_1).
		AddData(schnorr.SerializePubKey(taprootKey)).
		Script()
}

// PayToWitnessScript creates a pk script for a pay-to-wpkh output key.
func PayToWitnessScript(pubkey *btcec.PublicKey) ([]byte, error) {
	return txscript.NewScriptBuilder().
		AddOp(txscript.OP_0).
		AddData(btcutil.Hash160(pubkey.SerializeCompressed())).
		Script()
}

// PayToWitnessScriptPubkeyHash creates a pk script for a pay-to-wpkh output key.
func PayToWitnessScriptPubkeyHash(pubkey []byte) ([]byte, error) {
	return txscript.NewScriptBuilder().
		AddOp(txscript.OP_0).
		AddData(btcutil.Hash160(pubkey)).
		Script()
}

// PayToScriptPubkeyHash creates a pk script for a pay-to-pkh output key.
func PayToScriptPubkeyHash(pubkey []byte) ([]byte, error) {
	return txscript.NewScriptBuilder().
		AddOp(txscript.OP_DUP).
		AddOp(txscript.OP_HASH160).
		AddData(btcutil.Hash160(pubkey)).
		AddOp(txscript.OP_EQUALVERIFY).
		AddOp(txscript.OP_CHECKSIG).
		Script()
}

// PayToScriptHash creates a pk script for a pay-to-pkh output key.
func PayToScriptHash(pubkey []byte) ([]byte, error) {
	return txscript.NewScriptBuilder().
		AddOp(txscript.OP_HASH160).
		AddData(btcutil.Hash160(pubkey)).
		AddOp(txscript.OP_EQUAL).
		Script()
}

const (
	BRC20_PUBKEY_ADDRESS_P2TR_SCRIPT      uint8 = 0x51
	BRC20_PUBKEY_ADDRESS_P2WPKH_EVEN      uint8 = 0x52
	BRC20_PUBKEY_ADDRESS_P2WPKH_ODD       uint8 = 0x53
	BRC20_PUBKEY_ADDRESS_P2PKH_EVEN       uint8 = 0x54
	BRC20_PUBKEY_ADDRESS_P2PKH_ODD        uint8 = 0x55
	BRC20_PUBKEY_ADDRESS_P2SH_P2WPKH_EVEN uint8 = 0x56
	BRC20_PUBKEY_ADDRESS_P2SH_P2WPKH_ODD  uint8 = 0x57
	BRC20_PUBKEY_ADDRESS_P2TR_KEY         uint8 = 0x58
)

func GetPkScriptByPubkeyAndType(pubKeyStr []byte, keyType uint8) (pk []byte, err error) {
	if keyType == BRC20_PUBKEY_ADDRESS_P2TR_SCRIPT || keyType == BRC20_PUBKEY_ADDRESS_P2TR_KEY {
		pubkey, err := schnorr.ParsePubKey(pubKeyStr)
		if err != nil {
			return nil, err
		}

		if keyType == BRC20_PUBKEY_ADDRESS_P2TR_SCRIPT {
			return PayToTaprootScript(txscript.ComputeTaprootKeyNoScript(pubkey))
		}

		if keyType == BRC20_PUBKEY_ADDRESS_P2TR_KEY {
			return PayToTaprootScript(pubkey)
		}
	}

	var prefix byte = 0x03
	if keyType == BRC20_PUBKEY_ADDRESS_P2WPKH_EVEN ||
		keyType == BRC20_PUBKEY_ADDRESS_P2PKH_EVEN ||
		keyType == BRC20_PUBKEY_ADDRESS_P2SH_P2WPKH_EVEN {
		prefix = 0x02
	}
	var pubkeyByte [33]byte
	pubkeyByte[0] = prefix
	copy(pubkeyByte[1:], pubKeyStr)

	if keyType == BRC20_PUBKEY_ADDRESS_P2WPKH_EVEN || keyType == BRC20_PUBKEY_ADDRESS_P2WPKH_ODD {
		return PayToWitnessScriptPubkeyHash(pubkeyByte[:])
	}

	if keyType == BRC20_PUBKEY_ADDRESS_P2PKH_EVEN || keyType == BRC20_PUBKEY_ADDRESS_P2PKH_ODD {
		return PayToScriptPubkeyHash(pubkeyByte[:])
	}

	if keyType == BRC20_PUBKEY_ADDRESS_P2SH_P2WPKH_EVEN || keyType == BRC20_PUBKEY_ADDRESS_P2SH_P2WPKH_ODD {
		scriptByte, err := PayToWitnessScriptPubkeyHash(pubkeyByte[:])
		if err != nil {
			return nil, err
		}
		return PayToScriptHash(scriptByte)
	}

	return nil, errors.New("pubkey type wrong")
}

func GetPkScriptByAddress(addr string, netParams *chaincfg.Params) (pk []byte, err error) {
	if len(addr) == 0 {
		return nil, errors.New("decoded address empty")
	}

	addressObj, err := btcutil.DecodeAddress(addr, netParams)
	if err != nil {
		if len(addr) != 68 || !strings.HasPrefix(addr, "6a20") {
			return nil, errors.New("decoded address is of unknown format")
		}
		// check full hex
		pkHex, err := hex.DecodeString(addr)
		if err != nil {
			return nil, errors.New("decoded address is of unknown format")
		}
		return pkHex, nil
	}
	addressPkScript, err := txscript.PayToAddrScript(addressObj)
	if err != nil {
		return nil, errors.New("decoded address is of unknown format")
	}
	return addressPkScript, nil
}

// GetAddressFromScript Use btcsuite to get address
func GetAddressFromScript(script []byte, params *chaincfg.Params) (string, error) {
	scriptClass, addresses, _, err := txscript.ExtractPkScriptAddrs(script, params)
	if err != nil {
		return "", fmt.Errorf("failed to get address: %v", err)
	}

	if len(addresses) == 0 {
		return "", fmt.Errorf("noaddress")
	}

	if scriptClass == txscript.NonStandardTy {
		return "", fmt.Errorf("non-standard")
	}

	return addresses[0].EncodeAddress(), nil
}

func GetModuleFromScript(script []byte) (module string, ok bool) {
	n := len(script)
	if n < 34 || n > 38 {
		return "", false
	}
	if script[0] != 0x6a {
		return "", false
	}
	if int(script[1])+2 != n {
		return "", false
	}

	// remove trailling 0
	if n > 34 && script[n-1] == 0 {
		return "", false
	}

	var idx uint32
	if script[1] <= 32 {
		idx = uint32(0)
	} else if script[1] <= 33 {
		idx = uint32(script[34])
	} else if script[1] <= 34 {
		idx = uint32(binary.LittleEndian.Uint16(script[34:36]))
	} else if script[1] <= 35 {
		idx = uint32(script[34]) | uint32(script[35])<<8 | uint32(script[36])<<16
	} else if script[1] <= 36 {
		idx = binary.LittleEndian.Uint32(script[34:38])
	}

	module = fmt.Sprintf("%si%d", HashString(script[2:34]), idx)
	return module, true
}

func GetScriptFromModuleId(module string) (script []byte, ok bool) {
	if len(module) != 64+2 {
		return nil, false
	}

	if !strings.HasSuffix(module, "i0") {
		return nil, false
	}

	txIdReverse, err := hex.DecodeString(module[:64])
	if err != nil || len(txIdReverse) != 32 {
		return nil, false
	}
	txid := ReverseBytes(txIdReverse)

	script = make([]byte, 34)
	script[0] = 0x6a
	script[1] = 0x20
	copy(script[2:], txid)

	return script, true
}

func DecodeInscriptionFromBin(script []byte) (id string) {
	n := len(script)
	if n < 32 || n > 36 {
		return ""
	}

	var idx uint32
	if n == 32 {
		idx = uint32(0)
	} else if n <= 33 {
		idx = uint32(script[32])
	} else if n <= 34 {
		idx = uint32(binary.LittleEndian.Uint16(script[32:34]))
	} else if n <= 35 {
		idx = uint32(script[32]) | uint32(script[33])<<8 | uint32(script[34])<<16
	} else if n <= 36 {
		idx = binary.LittleEndian.Uint32(script[32:36])
	}

	id = fmt.Sprintf("%si%d", HashString(script[:32]), idx)
	return id
}

func FindMaxLessThan(target uint32, arr []uint32) (uint32, bool) {
	if len(arr) == 0 || target <= arr[0] {
		return 0, false // No element is smaller than target.
	}

	left, right := 0, len(arr)-1
	result := uint32(0)
	found := false

	for left <= right {
		mid := (left + right) / 2
		if arr[mid] < target {
			result = arr[mid]
			found = true
			left = mid + 1
		} else {
			right = mid - 1
		}
	}

	return result, found
}
