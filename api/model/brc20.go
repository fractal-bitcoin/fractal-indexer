package model

import (
	brc20swapIndexer "github.com/unisat-wallet/libbrc20-indexer/indexer"
	brc20Model "github.com/unisat-wallet/libbrc20-indexer/model"
)

var (
	GlobalBlocksHash []string
	GlobalBlocksTime []uint32

	GSwap     *brc20swapIndexer.BRC20ModuleIndexer
	GSwapBase *brc20swapIndexer.BRC20ModuleIndexer

	Global5bDeployInscriptionsAddressSummaryMap map[string]*DeployInscriptionsAddressHoldSummary
)

// ticker summary
type DeployInscriptionsAddressHoldSummary struct {
	Address            string
	DeployCount        uint64
	DeployInscriptions []*DeployInscriptionsSummary
}

type BRC20TickerSummaryResp struct {
	Height int                          `json:"height"` // synced block height
	Total  int                          `json:"total"`
	Start  int                          `json:"start"`
	Detail []*DeployInscriptionsSummary `json:"detail"`
}

type BRC20HeatMapResp struct {
	Height int            `json:"height"` // synced block height
	Total  int            `json:"total"`
	Start  int            `json:"start"`
	Detail []BRC20HeatMap `json:"detail"`
}

type BRC20HeatMap struct {
	Ticker     string                `json:"ticker"`
	Count      int64                 `json:"count"`
	StatusInfo BRC20TickerStatusInfo `json:"statusInfo"`
}

type DeployInscriptionsSummary struct {
	Data              brc20Model.InscriptionBRC20InfoResp `json:"data"`
	InscriptionNumber int64                               `json:"inscriptionNumber"`
	InscriptionId     string                              `json:"inscriptionId"`
	Satoshi           uint64                              `json:"satoshi"`
	Height            uint32                              `json:"height"`
	Confirmations     int                                 `json:"confirmations"`
}

// search
type InscriptionBRC20TickInfoForSearch struct {
	Data   brc20Model.InscriptionBRC20InfoResp
	Max    uint64
	Limit  uint64
	Minted uint64
}

// status
type BRC20TickerStatusInfo struct {
	Ticker   string `json:"ticker"`
	SelfMint bool   `json:"selfMint"`

	HoldersCount int `json:"holdersCount"`
	HistoryCount int `json:"historyCount"`

	InscriptionNumber int64  `json:"inscriptionNumber"`
	InscriptionId     string `json:"inscriptionId"`

	Max    string `json:"max"`
	Limit  string `json:"limit"`
	Minted string `json:"minted"`

	TotalMinted        string `json:"totalMinted"`
	ConfirmedMinted    string `json:"confirmedMinted"`
	ConfirmedMinted1h  string `json:"confirmedMinted1h"`
	ConfirmedMinted24h string `json:"confirmedMinted24h"`
	MintTimes          uint32 `json:"mintTimes"`
	Decimal            uint8  `json:"decimal"`

	CreatorAddress string `json:"creator"`

	TxIdHex         string `json:"txid"`
	DeployHeight    uint32 `json:"deployHeight"`
	DeployBlockTime uint32 `json:"deployBlocktime"`

	CompleteHeight    uint32 `json:"completeHeight"`
	CompleteBlockTime uint32 `json:"completeBlocktime"`

	InscriptionNumberStart int64 `json:"inscriptionNumberStart"`
	InscriptionNumberEnd   int64 `json:"inscriptionNumberEnd"`
}
type BRC20TickerStatusResp struct {
	Height int                      `json:"height"` // synced block height
	Total  int                      `json:"total"`
	Start  int                      `json:"start"`
	Detail []*BRC20TickerStatusInfo `json:"detail"`
}

type BRC20TickerListResp struct {
	Height int      `json:"height"` // synced block height
	Total  int      `json:"total"`
	Start  int      `json:"start"`
	Detail []string `json:"detail"`
}

type BRC20TickerBestHeightResp struct {
	Height     int    `json:"height"` // synced block height
	BlockIdHex string `json:"blockid"`
	BlockTime  int    `json:"timestamp"` // block time
	Total      int    `json:"total"`
}

type BRC20StateHashEntry struct {
	Height    int    `json:"height"`
	BlockHash string `json:"blockHash"`
	StateHash string `json:"stateHash"`
}

type BRC20StateHashResp struct {
	Height int                    `json:"height"` // synced block height
	Total  int                    `json:"total"`
	Detail []*BRC20StateHashEntry `json:"detail"`
}

// history
type BRC20TickerHistoryInfoInsideSwapModule struct {
	Ticker string `json:"ticker"`
	Type   string `json:"type"` // send/receive

	AddressFrom string `json:"from"`
	AddressTo   string `json:"to"`
	Amount      string `json:"amount"`

	FunctionIndex uint32 `json:"functionIndex"` // function index

	TxIdHex           string `json:"txid"`
	Idx               uint32 `json:"idx"` // inscription index
	Vout              uint32 `json:"vout"`
	Offset            uint64 `json:"offset"`
	InscriptionNumber int64  `json:"inscriptionNumber"`
	InscriptionId     string `json:"inscriptionId"`

	Height       uint32 `json:"height"`
	TxIdx        uint32 `json:"txidx"` // txidx in block
	BlockHashHex string `json:"blockhash"`
	BlockTime    uint32 `json:"blocktime"`
}

// history
type BRC20TickerHistoryInfo struct {
	Ticker string `json:"ticker"`
	Type   string `json:"type"` // inscribe-deploy/inscribe-mint/inscribe-transfer/transfer/send/receive
	Valid  bool   `json:"valid"`

	TxIdHex           string `json:"txid"`
	Idx               uint32 `json:"idx"` // inscription index
	Vout              uint32 `json:"vout"`
	Offset            uint64 `json:"offset"`
	InscriptionNumber int64  `json:"inscriptionNumber"`
	InscriptionId     string `json:"inscriptionId"`

	AddressFrom string `json:"from"`
	AddressTo   string `json:"to"`
	Satoshi     uint64 `json:"satoshi"`
	Fee         int64  `json:"fee"`

	Amount              string `json:"amount"`
	OverallBalance      string `json:"overallBalance"`
	TransferableBalance string `json:"transferBalance"`
	AvailableBalance    string `json:"availableBalance"`

	Height       uint32 `json:"height"`
	TxIdx        uint32 `json:"txidx"` // txidx in block
	BlockHashHex string `json:"blockhash"`
	BlockTime    uint32 `json:"blocktime"`
	History      uint32 `json:"h"`
}
type BRC20TickerHistoryResp struct {
	Height int                       `json:"height"` // synced block height
	Total  int                       `json:"total"`
	Start  int                       `json:"start"`
	Detail []*BRC20TickerHistoryInfo `json:"detail"`
}

// holders
type BRC20TickerHoldersInfo struct {
	Address                string `json:"address"`
	OverallBalance         string `json:"overallBalance"`
	TransferableBalance    string `json:"transferableBalance"`
	AvailableBalance       string `json:"availableBalance"`
	AvailableBalanceSafe   string `json:"availableBalanceSafe"`
	AvailableBalanceUnSafe string `json:"availableBalanceUnSafe"`
}

type BRC20TickerHoldersResp struct {
	Height int                       `json:"height"` // synced block height
	Total  int                       `json:"total"`
	Start  int                       `json:"start"`
	Detail []*BRC20TickerHoldersInfo `json:"detail"`
}

type BRC20TickerSwapBalance struct {
	SwapAccountBalanceSafe   string
	ModuleAccountBalanceSafe string

	// with unconfirmed balance
	SwapAccountBalance   string
	AvailableBalanceSafe string
	AvailableBalance     string
}

// summary
type BRC20TokenSummaryInfo struct {
	Ticker                 string `json:"ticker"`
	OverallBalance         string `json:"overallBalance"`
	TransferableBalance    string `json:"transferableBalance"`
	AvailableBalance       string `json:"availableBalance"`
	AvailableBalanceSafe   string `json:"availableBalanceSafe"`
	AvailableBalanceUnSafe string `json:"availableBalanceUnSafe"`
	Decimal                int    `json:"decimal"`
	SelfMint               bool   `json:"selfMint"`

	SwapBalance *BRC20TickerSwapBalance `json:"swapBalance,omitempty"`
}

type BRC20TokenSummaryResp struct {
	Height int                      `json:"height"` // synced block height
	Total  int                      `json:"total"`
	Start  int                      `json:"start"`
	Detail []*BRC20TokenSummaryInfo `json:"detail"`
}

// address info
type BRC20TickerStatusInfoOfAddressResp struct {
	Ticker   string `json:"ticker"`
	SelfMint bool   `json:"selfMint"`

	OverallBalance         string `json:"overallBalance"`
	TransferableBalance    string `json:"transferableBalance"`
	AvailableBalance       string `json:"availableBalance"`
	AvailableBalanceSafe   string `json:"availableBalanceSafe"`
	AvailableBalanceUnSafe string `json:"availableBalanceUnSafe"`

	SwapBalance *BRC20TickerSwapBalance `json:"swapBalance,omitempty"`

	TransferableCount        int                                        `json:"transferableCount"`
	TransferableInscriptions []*brc20Model.InscriptionBRC20TickInfoResp `json:"transferableInscriptions"`
	HistoryCount             int                                        `json:"historyCount"`
	HistoryInscriptions      []*brc20Model.InscriptionBRC20TickInfoResp `json:"historyInscriptions"`
}

// inscriptions
type BRC20TickerInscriptionsResp struct {
	Height int                                        `json:"height"` // synced block height
	Total  int                                        `json:"total"`
	Start  int                                        `json:"start"`
	Detail []*brc20Model.InscriptionBRC20TickInfoResp `json:"detail"`
}
