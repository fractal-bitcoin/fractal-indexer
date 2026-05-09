package indexer

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	"github.com/unisat-wallet/libbrc20-indexer/decimal"
	"github.com/unisat-wallet/libbrc20-indexer/model"
	"github.com/unisat-wallet/libbrc20-indexer/uint128"
	"github.com/unisat-wallet/libbrc20-indexer/utils"
	"github.com/unisat-wallet/libbrc20-indexer/utils/bip322"

	"github.com/btcsuite/btcd/wire"
)

// GetFunctionDataId Calculate ID hash, used for signing.
func GetFunctionDataContent(contentPrefix string, data *model.SwapFunctionData) (content string) {
	content = contentPrefix + fmt.Sprintf(`addr: %s
func: %s
params: %s
ts: %d
`, data.Address, data.Function, strings.Join(data.Params, " "), data.Timestamp)
	return content
}

func CheckFunctionSigVerify(contentPrefix string, data *model.SwapFunctionData, previous []string) (id string, ok bool) {
	if len(previous) != 0 {
		contentPrefix += fmt.Sprintf("prevs: %s\n", strings.Join(previous, " "))
	}

	content := GetFunctionDataContent(contentPrefix, data)
	// check id
	id = utils.HashString(utils.GetSha256([]byte(content)))
	message := GetFunctionDataContent(fmt.Sprintf("id: %s\n", id), data)

	signature, err := base64.StdEncoding.DecodeString(data.Signature)
	if err != nil {
		log.Println("CheckFunctionSigVerify decoding signature:", err)
		return id, false
	}

	var wit wire.TxWitness
	lenSignature := len(signature)
	if len(signature) == 66 {
		wit = wire.TxWitness{signature[2:]}
	} else if lenSignature > (2+64+34) && lenSignature <= (2+72+34) {
		wit = wire.TxWitness{signature[2 : lenSignature-34], signature[lenSignature-33 : lenSignature]}
	} else {
		fmt.Println("b64 sig:", hex.EncodeToString(signature))
		fmt.Println("pkScript:", hex.EncodeToString([]byte(data.PkScript)))
		fmt.Println("b64 sig length invalid")
		return id, false
	}

	// check sig
	if ok := bip322.VerifySignature(wit, []byte(data.PkScript), message); !ok {
		log.Printf("CheckFunctionSigVerify. content: %s", content)
		fmt.Println("sig invalid")
		return id, false
	}
	return id, true
}

// CheckAmountVerify Verify the legality of the brc20 tick amt.
func CheckAmountVerify(amtStr string, nDecimal uint8) (amt *decimal.Decimal, ok bool) {
	// check amount
	amt, err := decimal.FromString(amtStr, int(nDecimal))
	if err != nil {
		return nil, false
	}
	if amt.Sign() < 0 {
		return nil, false
	}

	return amt, true
}

// CheckTickVerify Verify the legality of the brc20 tick amt.
func (g *BRC20ModuleIndexer) CheckTickVerifyDecimal(tick string, amtStr string) (amount *decimal.Decimal, ok bool) {
	uniqueLowerTicker := strings.ToLower(tick)
	tokenInfo, ok := g.InscriptionsTickerInfoMap[uniqueLowerTicker]
	if !ok {
		return nil, false
	}

	if amtStr == "" {
		return nil, true
	}

	tinfo := tokenInfo.Deploy

	// check amount
	amt, err := uint128.FromString(amtStr, int(tinfo.Decimal))
	if err != nil {
		return nil, false
	}
	if amt.Sign() < 0 || amt.Cmp(tinfo.Max) > 0 {
		return nil, false
	}

	amount = &decimal.Decimal{Precision: tinfo.Decimal, Value: amt.Big()}
	return amount, true
}

// CheckTickVerify Verify the legality of the brc20 tick amt.
func (g *BRC20ModuleIndexer) CheckTickVerify(tick string, amtStr string) (amt uint128.Decimal, ok bool) {
	uniqueLowerTicker := strings.ToLower(tick)
	tokenInfo, ok := g.InscriptionsTickerInfoMap[uniqueLowerTicker]
	if !ok {
		return amt, false
	}

	if amtStr == "" {
		return amt, true
	}

	tinfo := tokenInfo.Deploy

	// check amount
	amt, err := uint128.FromString(amtStr, int(tinfo.Decimal))
	if err != nil {
		return amt, false
	}
	if amt.Sign() < 0 || amt.Cmp(tinfo.Max) > 0 {
		return amt, false
	}

	return amt, true
}

func GetLowerInnerPairNameByToken(token0, token1 string) (poolPair string) {
	token0 = strings.ToLower(token0)
	token1 = strings.ToLower(token1)

	if token0 > token1 {
		poolPair = fmt.Sprintf("%s%s%s", string([]byte{uint8(len(token1))}), token1, token0)
	} else {
		poolPair = fmt.Sprintf("%s%s%s", string([]byte{uint8(len(token0))}), token0, token1)
	}
	return poolPair
}
