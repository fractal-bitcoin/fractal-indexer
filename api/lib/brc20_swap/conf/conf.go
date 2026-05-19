package conf

import "github.com/btcsuite/btcd/chaincfg"

var (
	DEBUG                             = false
	MODULE_SWAP_SOURCE_INSCRIPTION_ID = "34761af5d6aa0ab7f14589e6f0a905b505c40ba542c9a27d407c9d052499ffddi0"
	MODULE_SWAP_INSCRIPTION_ID        = "fd5bd482bed1b62d0702e2f19a1e3bdd4fb755fa5c9bed5d8d0f219a3219ee95i0"
	GlobalNetParams                   = &chaincfg.MainNetParams
	TICKS_ENABLED                     = ""
	TICK_MIN_LEN                      = 6
	TICK_MAX_LEN                      = 12
	BRC20_MODULE_SAFE_CONFIRMATION    = 5
	PruneBRC20MintHistory             = true
	// PruneHistoryList: skip appending to in-memory History slices on
	// BRC20TokenBalance, BRC20TokenInfo, and BRC20UserHistory.
	// HistoryData/pika writes are unaffected.
	// Env: BRC20_PRUNE_HISTORY_LIST=true
	PruneHistoryList = true
	// PruneValidBRC20DataMap: skip populating InscriptionsValidBRC20DataMap.
	// Saves ~3-5GB memory. Disables /nft inscription info API lookups.
	// Env: BRC20_PRUNE_VALID_DATA_MAP=true
	PruneValidBRC20DataMap = true

	BRC20_SWAP_MANDATORY_COMMIT_BEFORE_HEIGHT = 999999999
	PikaRewriteHistoryStartIndex              = -1

	BRC20_SINGLE_STEP_TRANSFER_HEIGHT                  = 930930
	BRC20_ACCEPT_VINDICATED_INSCRIPTION_HEIGHT         = 1050000
	BRC20_SINGLE_STEP_TRANSFER_NON_TRANSFERABLE_HEIGHT = 1375380
	BRC20_SWAP_FEERATE_15_HEIGHT                       = 600000

	BRC20_LOAD_AFTER_MEMPOOL_HEIGHT = 0
)
