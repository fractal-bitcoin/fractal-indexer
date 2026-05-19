package utils

import (
	"encoding/hex"
	"errors"
	scriptDecoder "fractal-indexer/api/lib/blkparser/script"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/btcsuite/btcd/btcutil/bech32"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"golang.org/x/crypto/ripemd160"
)

func GetReversedStringHex(data string) (result string) {
	return hex.EncodeToString(ReverseBytes([]byte(data)))
}

func ReverseBytes(data []byte) (result []byte) {
	n := len(data)
	result = make([]byte, n)
	for i := 0; i < n; i++ {
		result[i] = data[n-1-i]
	}
	return result
}

const (
	PubKeyHashAddrIDMainNet = byte(0x00) // starts with 1
	PubKeyHashAddrIDTestNet = byte(0x6f) // starts with m or n

	P2SHAddrIDMainNet = byte(0x05) // starts with 3
	P2SHAddrIDTestNet = byte(0xc4) // starts with x

	PubKeyHashAddrHrpMainNet = "bc" // starts with bc1
	PubKeyHashAddrHrpTestNet = "tb" // starts with tb1
)

var (
	is_testnet          = os.Getenv("TESTNET")
	ErrChecksumMismatch = errors.New("checksum mismatch")
	empty               = make([]byte, ripemd160.Size)
	empty32             = make([]byte, 32)

	PubKeyHashAddrID  = PubKeyHashAddrIDMainNet
	P2SHAddrID        = P2SHAddrIDMainNet
	PubKeyHashAddrHrp = PubKeyHashAddrHrpMainNet
)

func init() {
	if is_testnet != "" {
		PubKeyHashAddrID = PubKeyHashAddrIDTestNet
		P2SHAddrID = P2SHAddrIDTestNet
		PubKeyHashAddrHrp = PubKeyHashAddrHrpTestNet
	}
	rand.Seed(time.Now().UnixNano())
}

func EncodeAddressByCodeType(addressPk []byte, codeType uint32) string {
	if codeType == scriptDecoder.CodeType_P2PKH {
		if len(addressPk) < ripemd160.Size+3 {
			return ""
		}
		return base58.CheckEncode(addressPk[3:3+ripemd160.Size], PubKeyHashAddrID)
	} else if codeType == scriptDecoder.CodeType_P2PK {
		if len(addressPk) < 32 {
			return ""
		}
		return base58.CheckEncode(btcutil.Hash160(addressPk[1:len(addressPk)-1]), PubKeyHashAddrID)
	} else if codeType == scriptDecoder.CodeType_P2SH {
		if len(addressPk) < ripemd160.Size+2 {
			return ""
		}
		return base58.CheckEncode(addressPk[2:2+ripemd160.Size], P2SHAddrID)
	} else if codeType == scriptDecoder.CodeType_P2WPKH {
		if len(addressPk) < ripemd160.Size+2 {
			return ""
		}

		addressData, _ := bech32.ConvertBits(addressPk[2:2+ripemd160.Size], 8, 5, true)
		address, _ := bech32.Encode(PubKeyHashAddrHrp, append([]byte{byte(bech32.Version0)}, addressData...))
		return address
	} else if codeType == scriptDecoder.CodeType_P2WSH {
		if len(addressPk) < 32+2 {
			return ""
		}
		addressData, _ := bech32.ConvertBits(addressPk[2:2+32], 8, 5, true)
		address, _ := bech32.Encode(PubKeyHashAddrHrp, append([]byte{byte(bech32.Version0)}, addressData...))
		return address
	} else if codeType == scriptDecoder.CodeType_P2TR {
		if len(addressPk) < 32+2 {
			return ""
		}
		addressData, _ := bech32.ConvertBits(addressPk[2:2+32], 8, 5, true)
		address, _ := bech32.EncodeM(PubKeyHashAddrHrp, append([]byte{byte(bech32.VersionM)}, addressData...))
		return address
	} else if codeType == scriptDecoder.CodeType_P2A {
		if len(addressPk) != 2+2 {
			return ""
		}
		addressData, _ := bech32.ConvertBits(addressPk[2:2+2], 8, 5, true)
		address, _ := bech32.EncodeM(PubKeyHashAddrHrp, append([]byte{byte(bech32.VersionM)}, addressData...))
		return address
	}
	return ""
}

func GetPkScriptByAddress(addr string) (pk []byte, err error) {
	if len(addr) == 0 {
		return nil, errors.New("decoded address empty")
	}

	if addr == "6a" || (len(addr) == 68 && strings.HasPrefix(addr, "6a20")) {
		// check full hex
		pkHex, err := hex.DecodeString(addr)
		if err != nil {
			return nil, errors.New("decoded address is of unknown format")
		}
		return pkHex, nil
	}

	netParams := &chaincfg.MainNetParams
	if is_testnet != "" {
		netParams = &chaincfg.TestNet3Params
	}
	addressObj, err := btcutil.DecodeAddress(addr, netParams)
	if err != nil {
		return nil, errors.New("decoded address is of unknown format")
	}
	addressPkScript, err := txscript.PayToAddrScript(addressObj)
	if err != nil {
		return nil, errors.New("decoded address is of unknown format")
	}
	return addressPkScript, nil
}

func GenerateRandomNonce() int {
	return rand.Intn(1000000)
}
