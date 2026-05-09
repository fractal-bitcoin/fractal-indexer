package model

import (
	"fractal-query/constant"
	"regexp"
	"sync"

	brc20Model "github.com/unisat-wallet/libbrc20-indexer/model"
)

type SearchTypeAndInscriptionContentForSearch struct {
	Type string
	Exp  *regexp.Regexp
	Data []*InscriptionContentForSearch
}

const (
	NAME_TYPE_FB         int = 0
	NAME_TYPE_INFINITYAI int = 1
	NAME_TYPE_ALL        int = 2

	TOTAL_NAME_TYPES int = 3
)

const (
	SEARCH_TYPE_FB         = "fb"
	SEARCH_TYPE_BRC20      = "brc20"
	SEARCH_TYPE_NUMBER     = "number"
	SEARCH_TYPE_WORD       = "word"
	SEARCH_TYPE_OTHER      = "other"
	SEARCH_TYPE_JSON       = "json"
	SEARCH_TYPE_INFINITYAI = constant.INFINITYAI_NAME_SUFFIX
	SEARCE_TYPE_ALL        = "all"
)

var GlobalDomainNamesCategoryToIndexMap map[string]int = map[string]int{
	SEARCH_TYPE_FB:         NAME_TYPE_FB,
	SEARCH_TYPE_INFINITYAI: NAME_TYPE_INFINITYAI,
	SEARCE_TYPE_ALL:        NAME_TYPE_ALL,
}

func NFTNumberFix(number uint64) (fixedNumber int64) {
	// return NFTNumberSkipFix(number)
	return int64(number)
}

type InscribeFeeSummaryResp struct {
	SatsCount           uint64 `json:"satsCount"`
	UniSatCount         uint64 `json:"unisatCount"`
	OGPassCount         uint64 `json:"ogPassCount"`
	UniSatFeeCutPercent int    `json:"unisatFeeCutPercent"`
	OGPassFeeCutPercent int    `json:"ogPassFeeCutPercent"`
}

type InscribeSummaryResp struct {
	OGPassCount         uint64 `json:"ogPassCount"`
	OGPassConfirmations int    `json:"ogPassConfirmations"`
	SatsCount           uint64 `json:"satsCount"`
	UniSatCount         uint64 `json:"unisatCount"`
	InscribeCount       uint64 `json:"inscribeCount"`
}

type NameSummaryResp struct {
	CategoryIdx int
	Category    string `json:"category"`
	Count       uint64 `json:"count"`
}

type NameInscriptionsSummary struct {
	Count        uint64
	Inscriptions []*InscriptionContentForCheck
}

type InscriptionsAddressHoldSummary struct {
	Address         string
	OGPassCount     uint64
	OGPassMinHeight int

	NamesInfo [TOTAL_NAME_TYPES]*NameInscriptionsSummary
}

func NewInscriptionsAddressHoldSummary() (s *InscriptionsAddressHoldSummary) {
	s = new(InscriptionsAddressHoldSummary)
	for _, category := range GlobalDomainNamesCategoryToIndexMap {
		s.NamesInfo[category] = new(NameInscriptionsSummary)
	}
	return s
}

type AddressSatsInscriptionResp struct {
	Cursor      int                               `json:"cursor"` // Inscription result offset.
	Total       int                               `json:"total"`  // Total inscription count.
	Inscription []*InscriptionContentForCheckResp `json:"detail"` // Inscription result list.
}

// exist req
type GetNamesByAddressRequest struct {
	Address  string   `json:"address"`
	Category []string `json:"suffixes"`
}

type GetNamesByAddressResponse struct {
	Name        string           `json:"name"`
	Category    string           `json:"suffix"`
	Inscription *InscriptionResp `json:"inscription"`
}

type CheckInscriptionsExistenceResponse struct {
	Name         string           `json:"name"`
	Status       string           `json:"status"`
	CreateIdxKey uint64           `json:"-"`
	Inscription  *InscriptionResp `json:"inscription"`
}

type InscriptionContentForSearch struct {
	Type              string
	Name              string
	NameForSearch     string
	Index             uint64
	ContentType       string
	ContentBody       string
	CreateIdxKey      uint64
	InscriptionId     string
	InscriptionNumber int64
	Satoshi           uint64
	Height            uint32 // Height of NFT show in block onCreate
	Address           string `json:"address"` // Current output address.

	BRC20 *brc20Model.InscriptionBRC20InfoResp
}

type InscriptionContentForCheck struct {
	Type              string
	Name              string
	InscriptionId     string
	InscriptionNumber int64

	Address      string // Current output address.
	CreateIdxKey uint64
	Height       uint32 // Height of NFT show in block onCreate
}

type InscriptionContentForCheckResp struct {
	InscriptionType    string `json:"inscriptionType"`    // Current NFT type (sats/brc20/unisat/btc/news/number/word).
	InscriptionNumber  int64  `json:"inscriptionNumber"`  // Current inscription number.
	InscriptionId      string `json:"inscriptionId"`      // Current inscription ID.
	InscriptionName    string `json:"inscriptionName"`    // Current NFT name.
	InscriptionNameHex string `json:"inscriptionNameHex"` // Current NFT name in hex.
	BlockTime          int    `json:"timestamp"`          // Block timestamp.
}

type InscriptionContentFromDB struct {
	Type              string
	ContentType       string
	ContentBody       string
	CreateIdxKey      uint64
	InscriptionId     string
	InscriptionNumber int64
	Satoshi           uint64
	Height            uint32 // Height of NFT show in block onCreate
}

var InscriptionContentFromDBPool = sync.Pool{
	New: func() interface{} { return new(InscriptionContentFromDB) },
}

// do
type InscriptionCountDO struct {
	ContentType []byte `db:"content_type"`
	Count       uint64 `db:"n"`
}

type InscriptionNumberRangeDO struct {
	MaxNumber int64 `db:"max_number"`
	MinNumber int64 `db:"min_number"`
}

type InscriptionStatusResp struct {
	Count            int64 `json:"count"`
	LastNumber       int64 `json:"lastNumber"`
	LastCursedNumber int64 `json:"lastCursedNumber"`
}

// vo
type InscriptionCountByContentTypeResp struct {
	ContentType  string             `json:"content_type"`
	Count        int                `json:"n"`
	Inscriptions []*InscriptionResp `json:"inscriptions"` // Latest 10 issued inscriptions.
}

type InscriptionsSummaryResp struct {
	Total  int                                  `json:"total"`
	Detail []*InscriptionCountByContentTypeResp `json:"detail"`
}

type InscriptionSearchResultsResp struct {
	Total      int                      `json:"total"`
	MatchCount int                      `json:"matchCount"`
	Start      int                      `json:"start"`
	Detail     []*InscriptionSearchResp `json:"detail"`
}

type InscriptionSearchResp struct {
	InscriptionType    string `json:"inscriptionType"`    // Current NFT type (sats/brc20/unisat/btc/news/number/word).
	InscriptionName    string `json:"inscriptionName"`    // Current NFT name.
	InscriptionNameHex string `json:"inscriptionNameHex"` // Current NFT name in hex.
	InscriptionIndex   uint64 `json:"inscriptionIndex"`   // Current inscription index; duplicate number, where 0 means first occurrence.
	InscriptionNumber  int64  `json:"inscriptionNumber"`  // Current inscription number.
	InscriptionId      string `json:"inscriptionId"`      // Current inscription ID.

	IsStrip            bool   `json:"isStrip"`
	HasPointer         bool   `json:"hasPointer"`
	HasParent          bool   `json:"hasParent"`
	HasDeligate        bool   `json:"hasDeligate"`
	HasMetaProtocal    bool   `json:"hasMetaProtocal"`
	HasMetadata        bool   `json:"hasMetadata"`
	HasContentEncoding bool   `json:"hasContentEncoding"`
	Pointer            uint64 `json:"pointer"`
	Parent             string `json:"parent"`
	Deligate           string `json:"deligate"`
	MetaProtocol       string `json:"metaprotocol"`
	Metadata           string `json:"metadata"`
	ContentEncoding    string `json:"contentEncoding"`

	Address       string `json:"address"`       // Current output address.
	ContentType   string `json:"contentType"`   //
	ContentLength int    `json:"contentLength"` //
	ContentBody   string `json:"contentBody"`   //
	BlockTime     int    `json:"timestamp"`     // Block timestamp.
	InSatoshi     int    `json:"inSatoshi"`     // Total input amount in GenesisTx.
	OutSatoshi    int    `json:"outSatoshi"`    // Total output amount in GenesisTx.

	BRC20 *brc20Model.InscriptionBRC20InfoResp `json:"brc20"` // BRC-20 information.
}

// exist req
type CheckInscriptionsExistenceRequest struct {
	Names []string `json:"names"`
}

// inscriptions info req
type GetInscriptionInfoBatchRequest struct {
	InscriptionIds []string `json:"inscriptionIds"`
}

type InscriptionEventsResultsResp struct {
	Total  int                     `json:"total"`
	Cursor int                     `json:"cursor"`
	Detail []*InscriptionEventResp `json:"detail"`
}

type InscriptionEventData struct {
	IsTransfer bool
	TxId       string `json:"-"`
	Idx        uint32 `json:"-"`
	Vout       uint32 `json:"-"`
	Offset     uint64 `json:"-"`
	Sequence   uint16 `json:"-"`

	Satoshi      uint64 `json:"-"`
	PkScript     string `json:"-"`
	PkScriptFrom []byte
	InputIdx     uint32

	InscriptionNumber int64
	ContentType       string
	ContentBody       string
	CreateIdxKey      uint64

	BlockTime  uint32
	InSatoshi  uint64
	OutSatoshi uint64
	Height     uint32 // Height of NFT show in block onCreate
	TxIdx      uint32
}

type InscriptionEventResp struct {
	IsTransfer bool   `json:"isTransfer"`
	TxId       string `json:"txid"`
	Idx        uint32 `json:"i"`
	Vout       uint32 `json:"vout"`
	Offset     uint64 `json:"offset"`
	Sequence   uint16 `json:"sequence"`

	InscriptionNumber int64  `json:"inscriptionNumber"` // Current inscription number.
	InscriptionId     string `json:"inscriptionId"`     // Current inscription ID.
	Address           string `json:"address"`           // Current output address.
	Satoshi           int    `json:"satoshi"`
	PkScriptHex       string `json:"pkScript"`
	AddressFrom       string `json:"addressFrom"`
	PkScriptFromHex   string `json:"pkScriptFrom"`
	InputIdx          uint32 `json:"inputIdx"`

	ContentType string `json:"contentType"` //
	ContentBody string `json:"contentBody"` //
	BlockTime   int    `json:"timestamp"`   // Block timestamp.
	InSatoshi   int    `json:"inSatoshi"`   // Total input amount in GenesisTx.
	OutSatoshi  int    `json:"outSatoshi"`  // Total output amount in GenesisTx.

	Height int `json:"height"`
	TxIdx  int `json:"txidx"`
}
