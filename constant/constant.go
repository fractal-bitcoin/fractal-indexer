package constant

var ORDINALS_ACTIVATION_HEIGHT uint32 = 767400
var JUBILEE_ACTIVATION_HEIGHT uint32 = 824544
var REINSCRIPTION_ACTIVATION_HEIGHT uint32 = 824544

var BRC20_SINGLE_STEP_TRANSFER_HEIGHT uint32 = 930930

var CHAIN_TYPE string

const (
	CHAIN_TYPE_BTC     = "BTC"
	CHAIN_TYPE_FRACTAL = "Fractal"
)

var ORDINALS_INSCRIPTION_COUNTS_BY_HEIGHT = "nft_counts"               // zser: nft counts -> height
var ORDINALS_INSCRIPTION_CURSED_COUNTS_BY_HEIGHT = "nft_cursed_counts" // zser: nft counts -> height

// Block metric IDs are stored in blkmetric_height and used by both indexer and API.
const (
	BlockMetricTxCount = iota
	BlockMetricTxWithWitness
	BlockMetricTxWithOpReturn
	BlockMetricTxWithInscription
	BlockMetricTxWithRunesRunestone
	BlockMetricTxWithRunesEtching
	BlockMetricTxWithTacit
	BlockMetricTxWithAlkanes
)

var (
	ZIP_INSCRIPTIONS_MAX_COUNT   = 10000
	ZSET_INSCRIPTION_MIN_SATOSHI = uint64(10000)
)

var (
	DUST_UTXO_ZSET_MIN_SATOSHI = uint64(546)
)
