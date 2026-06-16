package constant

const (
	TASK_INFO_KEYNAME       = "info"
	TASK_BLOCK              = "block"
	TASK_BLOCK_HEIGHT       = "block_height"
	TASK_REVERT_HEIGHT      = "revert_height"      // WAL: revert data committed up to this height
	TASK_REVERT_LAST_HEIGHT = "revert_last_height" // WAL: last revert data committed up to this height, need to remove
	TASK_NFT_POINTER        = "block_height_nft_pointer"
	TASK_NFT_ID             = "block_height_nft_id"

	TASK_UTXO_TOTAL               = "utxo_total"
	TASK_UTXO_WITH_BTC_TOTAL      = "utxo_with_btc_total"
	TASK_UTXO_WITH_DUST_BTC_TOTAL = "utxo_with_dust_btc_total"
	TASK_UTXO_WITH_BTC_NFT_TOTAL  = "utxo_with_btc_nft_total"
	TASK_UTXO_WITH_NFT_TOTAL      = "utxo_with_nft_total"

	TASK_UTXO_TOTAL_MEMPOOL = "utxo_total_mempool"
)
