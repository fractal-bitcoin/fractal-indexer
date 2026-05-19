package constant

import (
	"fractal-indexer/api/logger"
	"os"
	"strconv"

	"go.uber.org/zap"
)

var (
	MAX_STR_LEN_ALLOWED_TO_QUERY_FROM_RDB, _ = strconv.Atoi(os.Getenv("MAX_STR_LEN_ALLOWED_TO_QUERY_FROM_RDB"))
)

func InitEnv() {
	// 16384 = 2**14. Considering MAX_SCRIPT_SIZE=10000, this value must be greater than MAX_SCRIPT_SIZE + 4 + 20 + 1.
	// Calculate the compression ratio from the UTXO with 51,261,221 inscriptions: strlen is 21,798,053, about 0.425 per inscription.
	// With a length limit of 16382, the maximum inscription count is about (16382 - 55) / 0.425 = 38416. If each inscription is 1 KB, memory usage is 38 MB.
	maxStrLenAllowedToQueryFromRDBDefault := 16384
	if MAX_STR_LEN_ALLOWED_TO_QUERY_FROM_RDB < maxStrLenAllowedToQueryFromRDBDefault {
		MAX_STR_LEN_ALLOWED_TO_QUERY_FROM_RDB = maxStrLenAllowedToQueryFromRDBDefault
	}
	logger.Log.Info("MAX_STR_LEN_ALLOWED_TO_QUERY_FROM_RDB", zap.Int("MAX_STR_LEN_ALLOWED_TO_QUERY_FROM_RDB", MAX_STR_LEN_ALLOWED_TO_QUERY_FROM_RDB))
}
