package constant

const MEMPOOL_HEIGHT = 0x3fffff // 4294967295 2^32-1; 3fffff, 2^22-1
const HEIGHT_MUTIPLY_NBIT = 20
const HEIGHT_MUTIPLY_MASK = 0x0fffff

const AUXPOW_VERSON_MASK = 0x80

const (
	FB_NAME_SUFFIX         = "fb"
	INFINITYAI_NAME_SUFFIX = "infinityai" // Different from FB rules; not SNS rules.
)

func IsValidSNSDomainName(name string) bool {
	if name == FB_NAME_SUFFIX {
		return true
	}
	return false
}

const TXO_DATA_INSCRIPTION_SIZE = 19

var ORDINALS_INSCRIPTION_LIST_RDB_KEY = "nfts"

var ORDINALS_INSCRIPTION_COUNTS_BY_HEIGHT = "nft_counts"               // zser: nft counts -> height
var ORDINALS_INSCRIPTION_CURSED_COUNTS_BY_HEIGHT = "nft_cursed_counts" // zser: nft counts -> height
