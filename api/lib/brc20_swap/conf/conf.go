package conf

import "github.com/btcsuite/btcd/chaincfg"

var (
	DEBUG                             = false
	MODULE_SWAP_SOURCE_INSCRIPTION_ID = "d2a30f6131324e06b1366876c8c089d7ad2a9c2b0ea971c5b0dc6198615bda2ei0"
	MODULE_SWAP_INSCRIPTION_ID        = "66801a4a8352e84ed8485ec231aee88c20983bf442aa04d398ac6c89c92abc8ci0"
	GlobalNetParams                   = &chaincfg.MainNetParams
	TICKS_ENABLED                     = ""
	TICK_MIN_LEN                      = 6
	TICK_MAX_LEN                      = 12
	BRC20_MODULE_SAFE_CONFIRMATION    = 5
	PruneBRC20MintHistory             = false
	// PruneHistoryList: skip appending to in-memory History slices on
	// BRC20TokenBalance, BRC20TokenInfo, and BRC20UserHistory.
	// HistoryData/pika writes are unaffected.
	// Env: BRC20_PRUNE_HISTORY_LIST=true
	PruneHistoryList = false
	// PruneValidBRC20DataMap: skip populating InscriptionsValidBRC20DataMap.
	// Saves ~3-5GB memory. Disables /nft inscription info API lookups.
	// Env: BRC20_PRUNE_VALID_DATA_MAP=true
	PruneValidBRC20DataMap = false

	BRC20_SWAP_MANDATORY_COMMIT_BEFORE_HEIGHT = 1000000
	PikaRewriteHistoryStartIndex              = -1

	BRC20_SINGLE_STEP_TRANSFER_HEIGHT                  = 930930
	BRC20_ACCEPT_VINDICATED_INSCRIPTION_HEIGHT         = 1050000
	BRC20_SINGLE_STEP_TRANSFER_NON_TRANSFERABLE_HEIGHT = 1357000
	BRC20_SWAP_FEERATE_15_HEIGHT                       = 600000

	BRC20_LOAD_AFTER_MEMPOOL_HEIGHT = 0
)
