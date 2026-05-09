package model

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/unisat-wallet/libbrc20-indexer/constant"
	"github.com/unisat-wallet/libbrc20-indexer/decimal"
	"github.com/unisat-wallet/libbrc20-indexer/uint128"
	"github.com/unisat-wallet/libbrc20-indexer/utils"
)

var (
	CacheBody       map[string]*InscriptionBRC20ProtocalContent
	CacheBodyMint   map[string]*InscriptionBRC20MintTransferContent
	CacheAmountMint [19]map[string]uint128.Decimal
)

// nft create point on create
type NFTCreateIdxKey struct {
	Height     uint32 // Height of NFT show in block onCreate
	IdxInBlock uint32 // Index of NFT show in block onCreate
}

func (p *NFTCreateIdxKey) Uint64() uint64 {
	return uint64(p.Height)<<constant.HEIGHT_MUTIPLY_NBIT + uint64(p.IdxInBlock)
}

// event raw data
type InscriptionBRC20Data struct {
	IsTransfer bool
	TxId       string `json:"-"`
	Idx        uint32 `json:"-"`
	Vout       uint32 `json:"-"`
	Offset     uint64 `json:"-"`

	Satoshi  uint64 `json:"-"`
	PkScript string `json:"-"`
	Fee      int64  `json:"-"`

	TapScriptPk []byte `json:"-"`
	AddressType uint8  `json:"-"`
	NFTType     uint8  `json:"-"`

	InscriptionNumber int64
	Parent            []byte
	ContentBody       []byte
	CreateIdxKey      uint64

	Height    uint32 // Height of NFT show in block onCreate
	TxIdx     uint32
	BlockTime uint32
	Sequence  uint16

	// for cache
	InscriptionId string
}

func (data *InscriptionBRC20Data) GetInscriptionId() string {
	if data.InscriptionId == "" {
		data.InscriptionId = fmt.Sprintf("%si%d", utils.HashString([]byte(data.TxId)), data.Idx)
	}
	return data.InscriptionId
}

type InscriptionBRC20InfoResp struct {
	Operation     string `json:"op,omitempty"`
	BRC20Tick     string `json:"tick,omitempty"`
	BRC20Max      string `json:"max,omitempty"`
	BRC20Limit    string `json:"lim,omitempty"`
	BRC20Amount   string `json:"amt,omitempty"`
	BRC20Decimal  string `json:"decimal,omitempty"`
	BRC20Minted   string `json:"minted,omitempty"`
	BRC20SelfMint string `json:"self_mint,omitempty"`
}

// decode protocal
type InscriptionBRC20ProtocalContent struct {
	Proto     string `json:"p,omitempty"`
	Operation string `json:"op,omitempty"`
}

func BRC20ProtocalContentUnmarshal(contentBody []byte) (body *InscriptionBRC20ProtocalContent, err error) {
	if len(contentBody) < 128 {
		body, ok := CacheBody[string(contentBody)]
		if ok {
			return body, nil
		}
	}

	// protocal, lower case only
	var bodyMap map[string]interface{} = make(map[string]interface{}, 8)
	if err := json.Unmarshal(contentBody, &bodyMap); err != nil {
		return nil, err
	}

	body = new(InscriptionBRC20ProtocalContent)
	if v, ok := bodyMap["p"].(string); ok {
		body.Proto = v
	}
	if v, ok := bodyMap["op"].(string); ok {
		body.Operation = v
	}

	if len(contentBody) < 128 {
		CacheBody[string(contentBody)] = body
	}

	return body, nil
}

// decode mint/transfer
type InscriptionBRC20MintTransferContent struct {
	Proto       string `json:"p,omitempty"`
	Operation   string `json:"op,omitempty"`
	BRC20Tick   string `json:"tick,omitempty"`
	BRC20Amount string `json:"amt,omitempty"`
}

func (body *InscriptionBRC20MintTransferContent) Unmarshal(contentBody []byte) (err error) {
	var bodyMap map[string]interface{} = make(map[string]interface{}, 8)
	if err := json.Unmarshal(contentBody, &bodyMap); err != nil {
		return err
	}

	if v, ok := bodyMap["p"].(string); ok {
		body.Proto = v
	}
	if v, ok := bodyMap["op"].(string); ok {
		body.Operation = v
	}
	if v, ok := bodyMap["tick"].(string); ok {
		body.BRC20Tick = v
	}
	if v, ok := bodyMap["amt"].(string); ok {
		body.BRC20Amount = v
	}
	return nil
}

func BRC20MintTransferContentUnmarshal(contentBody []byte) (body *InscriptionBRC20MintTransferContent, err error) {
	if len(contentBody) < 128 {
		body, ok := CacheBodyMint[string(contentBody)]
		if ok {
			return body, nil
		}
	}

	var bodyMap map[string]interface{} = make(map[string]interface{}, 8)
	if err := json.Unmarshal(contentBody, &bodyMap); err != nil {
		return nil, err
	}

	body = new(InscriptionBRC20MintTransferContent)
	if v, ok := bodyMap["p"].(string); ok {
		body.Proto = v
	}
	if v, ok := bodyMap["op"].(string); ok {
		body.Operation = v
	}
	if v, ok := bodyMap["tick"].(string); ok {
		body.BRC20Tick = v
	}
	if v, ok := bodyMap["amt"].(string); ok {
		body.BRC20Amount = v
	}

	if len(contentBody) < 128 {
		CacheBodyMint[string(contentBody)] = body
	}
	return body, nil
}

func GetUint128FromString(amt string, decimal int) (uint128.Decimal, error) {
	n, ok := CacheAmountMint[decimal][amt]
	if ok {
		return n, nil
	}
	n, err := uint128.FromString(amt, decimal)
	if err != nil {
		return n, err
	}
	CacheAmountMint[decimal][amt] = n
	return n, nil
}

// decode deploy data
type InscriptionBRC20DeployContent struct {
	Proto         string `json:"p,omitempty"`
	Operation     string `json:"op,omitempty"`
	BRC20Tick     string `json:"tick,omitempty"`
	BRC20Max      string `json:"max,omitempty"`
	BRC20Limit    string `json:"lim,omitempty"`
	BRC20Decimal  string `json:"dec,omitempty"`
	BRC20SelfMint string `json:"self_mint,omitempty"`
}

func (body *InscriptionBRC20DeployContent) Unmarshal(contentBody []byte) (err error) {
	var bodyMap map[string]interface{} = make(map[string]interface{}, 8)
	if err := json.Unmarshal(contentBody, &bodyMap); err != nil {
		return err
	}
	if v, ok := bodyMap["p"].(string); ok {
		body.Proto = v
	}
	if v, ok := bodyMap["op"].(string); ok {
		body.Operation = v
	}
	if v, ok := bodyMap["tick"].(string); ok {
		body.BRC20Tick = v
	}
	if _, ok := bodyMap["self_mint"]; !ok {
		body.BRC20SelfMint = "false"
	} else {
		if v, ok := bodyMap["self_mint"].(string); ok {
			body.BRC20SelfMint = v
		}
	}
	if v, ok := bodyMap["max"].(string); ok {
		body.BRC20Max = v
	}
	if _, ok := bodyMap["lim"]; !ok {
		body.BRC20Limit = body.BRC20Max
	} else {
		if v, ok := bodyMap["lim"].(string); ok {
			body.BRC20Limit = v
		}
	}

	if _, ok := bodyMap["dec"]; !ok {
		body.BRC20Decimal = decimal.MAX_PRECISION_STRING
	} else {
		if v, ok := bodyMap["dec"].(string); ok {
			body.BRC20Decimal = v
		}
	}

	return nil
}

// all ticker (state and history)
type BRC20TokenInfo struct {
	Ticker   string
	SelfMint bool
	Deploy   *InscriptionBRC20TickInfo // fixme: Move static content out so access does not require an extra pointer dereference.

	History                 []uint32
	HistoryDeploy           []uint32
	HistoryMint             []uint32
	HistoryInscribeTransfer []uint32
	HistoryTransfer         []uint32
	HistoryWithdraw         []uint32 // fixme
}

type InscriptionBRC20TransferInfo struct {
	Tick   string
	Amount uint128.Decimal // no use?
	Data   *InscriptionBRC20Data
}

// inscription info, with mint state
type InscriptionBRC20TickInfo struct {
	Data   *InscriptionBRC20InfoResp `json:"data"`
	Tick   string
	Amount uint128.Decimal `json:"-"`
	Meta   *InscriptionBRC20Data

	Max    uint128.Decimal `json:"-"`
	Max999 uint128.Decimal `json:"-"`
	Limit  uint128.Decimal `json:"-"`

	MaxMintTimes uint64 `json:"-"`

	TotalMinted     uint128.Decimal `json:"-"`
	ConfirmedMinted uint128.Decimal `json:"-"`
	Burned          uint128.Decimal `json:"-"`

	MintTimes uint32 `json:"-"`
	Decimal   uint8  `json:"-"`

	TxId   string `json:"-"`
	Idx    uint32 `json:"-"`
	Vout   uint32 `json:"-"`
	Offset uint64 `json:"-"`

	Satoshi  uint64 `json:"-"`
	PkScript string `json:"-"`

	InscriptionNumber int64  `json:"inscriptionNumber"`
	CreateIdxKey      uint64 `json:"-"`
	Height            uint32 `json:"-"`
	TxIdx             uint32 `json:"-"`
	BlockTime         uint32 `json:"-"`

	CompleteHeight    uint32 `json:"-"`
	CompleteBlockTime uint32 `json:"-"`

	InscriptionNumberStart int64 `json:"-"`
	InscriptionNumberEnd   int64 `json:"-"`

	// for cache
	InscriptionId string
}

func (d *InscriptionBRC20TickInfo) GetInscriptionId() string {
	if d.InscriptionId == "" {
		d.InscriptionId = fmt.Sprintf("%si%d", utils.HashString([]byte(d.TxId)), d.Idx)
	}
	return d.InscriptionId
}

func (in *InscriptionBRC20TickInfo) DeepCopy() (copy *InscriptionBRC20TickInfo) {
	copy = &InscriptionBRC20TickInfo{
		Tick: in.Tick,

		Data:    in.Data,
		Decimal: in.Decimal,

		TxId:   in.TxId,
		Idx:    in.Idx,
		Vout:   in.Vout,
		Offset: in.Offset,

		Satoshi:  in.Satoshi,
		PkScript: in.PkScript,

		InscriptionNumber: in.InscriptionNumber,
		CreateIdxKey:      in.CreateIdxKey,
		Height:            in.Height,
		TxIdx:             in.TxIdx,
		BlockTime:         in.BlockTime,

		// runtime value
		Max:             in.Max,
		Max999:          in.Max999,
		Limit:           in.Limit,
		TotalMinted:     in.TotalMinted,
		ConfirmedMinted: in.ConfirmedMinted,
		Burned:          in.Burned,
		Amount:          in.Amount,

		MintTimes:    in.MintTimes,
		MaxMintTimes: in.MaxMintTimes,

		CompleteHeight:    in.CompleteHeight,
		CompleteBlockTime: in.CompleteBlockTime,

		InscriptionNumberStart: in.InscriptionNumberStart,
		InscriptionNumberEnd:   in.InscriptionNumberEnd,
	}
	return copy
}

func NewInscriptionBRC20TickInfo(tick, operation string, data *InscriptionBRC20Data) *InscriptionBRC20TickInfo {
	info := &InscriptionBRC20TickInfo{
		Tick: tick,
		Data: &InscriptionBRC20InfoResp{
			BRC20Tick: tick,
			Operation: operation,
		},
		Decimal: 18,

		TxId:   data.TxId,
		Idx:    data.Idx,
		Vout:   data.Vout,
		Offset: data.Offset,

		Satoshi:  data.Satoshi,
		PkScript: data.PkScript,

		InscriptionNumber: data.InscriptionNumber,
		CreateIdxKey:      data.CreateIdxKey,
		Height:            data.Height,
		TxIdx:             data.TxIdx,
		BlockTime:         data.BlockTime,
	}
	return info
}

// all history for user
type BRC20UserHistory struct {
	History                 []uint32
	HistoryDeploy           []uint32
	HistoryMint             []uint32 // Pruned.
	HistoryInscribeTransfer []uint32
	HistorySend             []uint32
	HistoryReceive          []uint32
	HistoryWithdraw         []uint32
}

// state of address for each tick, (balance and history)
type BRC20TokenBalance struct {
	Ticker               string
	PkScript             string
	AvailableBalance     uint128.Decimal
	AvailableBalanceSafe uint128.Decimal
	TransferableBalance  uint128.Decimal
	ValidTransferMap     map[uint64]struct{}
	Balance              float64

	History                 []uint32
	HistoryDeploy           []uint32
	HistoryMint             []uint32
	HistoryInscribeTransfer []uint32
	HistorySend             []uint32
	HistoryReceive          []uint32
	HistoryWithdraw         []uint32
}

func (bal *BRC20TokenBalance) OverallBalance() uint128.Decimal {
	return bal.AvailableBalance.Add(bal.TransferableBalance)
}

func (in *BRC20TokenBalance) DeepCopy() (tb *BRC20TokenBalance) {
	tb = &BRC20TokenBalance{
		Ticker:               in.Ticker,
		PkScript:             in.PkScript,
		AvailableBalanceSafe: in.AvailableBalanceSafe,
		AvailableBalance:     in.AvailableBalance,
		TransferableBalance:  in.TransferableBalance,
		Balance:              in.Balance,
	}

	tb.ValidTransferMap = make(map[uint64]struct{}, len(in.ValidTransferMap))
	for k := range in.ValidTransferMap {
		tb.ValidTransferMap[k] = struct{}{}
	}

	tb.History = make([]uint32, len(in.History))
	copy(tb.History, in.History)

	tb.HistoryDeploy = make([]uint32, len(in.HistoryDeploy))
	copy(tb.HistoryDeploy, in.HistoryDeploy)

	tb.HistoryMint = make([]uint32, len(in.HistoryMint))
	copy(tb.HistoryMint, in.HistoryMint)

	tb.HistoryInscribeTransfer = make([]uint32, len(in.HistoryInscribeTransfer))
	copy(tb.HistoryInscribeTransfer, in.HistoryInscribeTransfer)

	tb.HistorySend = make([]uint32, len(in.HistorySend))
	copy(tb.HistorySend, in.HistorySend)

	tb.HistoryReceive = make([]uint32, len(in.HistoryReceive))
	copy(tb.HistoryReceive, in.HistoryReceive)

	tb.HistoryWithdraw = make([]uint32, len(in.HistoryWithdraw))
	copy(tb.HistoryWithdraw, in.HistoryWithdraw)
	return tb
}

// Used for sorting the top K when there are many tickers.
type BRC20TokenBalanceSlice []*BRC20TokenBalance

func (s BRC20TokenBalanceSlice) Len() int {
	return len(s)
}

// desc order
func (s BRC20TokenBalanceSlice) Less(i, j int) bool {
	ret := s[i].Balance - s[j].Balance
	if ret == 0 {
		return strings.Compare(s[i].PkScript, s[j].PkScript) > 0
	}
	return ret > 0
}

func (s BRC20TokenBalanceSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type InscriptionBRC20TickInfoResp struct {
	Height            uint32                    `json:"-"`
	Data              *InscriptionBRC20InfoResp `json:"data"`
	InscriptionNumber int64                     `json:"inscriptionNumber"`
	InscriptionId     string                    `json:"inscriptionId"`
	Satoshi           uint64                    `json:"satoshi"`
	Confirmations     int                       `json:"confirmations"`
}

// history inscription info
type InscriptionBRC20TickInfoHistory struct {
	Height            uint32
	Data              *InscriptionBRC20InfoResp
	InscriptionNumber int64
	TxId              string
	Idx               uint32
	Satoshi           uint64
	Confirmations     int
}

func (d *InscriptionBRC20TickInfoHistory) GetInscriptionId() string {
	return fmt.Sprintf("%si%d", utils.HashString([]byte(d.TxId)), d.Idx)
}
