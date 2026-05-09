package store

import (
	"context"
	"fmt"
	"fractal-indexer/loader/clickhouse"
	"fractal-indexer/logger"
	"strconv"
	"time"

	"go.uber.org/zap"
)

var (
	createAllSQLs = []string{
		// block list
		// ================================================================
		"DROP TABLE IF EXISTS blk_height",
		"DROP TABLE IF EXISTS blk_height_new",
		`
CREATE TABLE IF NOT EXISTS blk_height (
	height       UInt32,
	version      UInt32,
	blkid        FixedString(32),
	previd       FixedString(32),
	merkle       FixedString(32),
	ntx          UInt32,
	nftnew       UInt64,        -- without coinbase, newly created NFT count, including invalid duplicate creations
	nftin        UInt64,        -- without coinbase, input NFT count
	nftout       UInt64,        -- without coinbase, output NFT count; valid creations = nftlost + nftout - nftin
	nftlost      UInt64,        -- NFTs dropped by regular transactions and collected by coinbase
	invalue      UInt64,        -- without coinbase
	outvalue     UInt64,        -- without coinbase
	coinbase_out UInt64,
	blocktime    UInt32,
	bits         UInt32,
	blocksize    UInt32,
	witsize      UInt32
) engine=MergeTree()
ORDER BY height
PARTITION BY intDiv(height, 2100)
`,

		"DROP TABLE IF EXISTS blk",
		`
CREATE TABLE IF NOT EXISTS blk (
	height       UInt32,
	version      UInt32,
	blkid        FixedString(32),
	previd       FixedString(32),
	merkle       FixedString(32),
	ntx          UInt32,
	nftnew       UInt64,        -- without coinbase, newly created NFT count, including invalid creations
	nftin        UInt64,        -- without coinbase, input NFT count
	nftout       UInt64,        -- without coinbase, output NFT count; valid creations = nftlost + nftout - nftin
	nftlost      UInt64,        -- NFTs dropped by regular transactions and collected by coinbase
	invalue      UInt64,        -- without coinbase
	outvalue     UInt64,        -- without coinbase
	coinbase_out UInt64,
	blocktime    UInt32,
	bits         UInt32,
	blocksize    UInt32,
	witsize      UInt32
) engine=MergeTree()
ORDER BY blkid
PARTITION BY intDiv(height, 2100)
`,

		// nft event list
		// ================================================================
		// All NFT events in blocks; create/move partitions are sorted and indexed by block height for fast height queries.
		"DROP TABLE IF EXISTS blkevent_height",
		"DROP TABLE IF EXISTS blkevent_height_new",
		`
CREATE TABLE IF NOT EXISTS blkevent_height (
	txid         FixedString(32),  -- inscription create txid
	idx          UInt32,           -- inscription create index, aka, output sat offset
	vin          UInt32,           -- revaled at input index
	vout         UInt32,           -- created at output index
	offset       UInt64,           -- created sat offset at this output
	satoshi      UInt64,
	script_pk    String,
	script_pk_from String,
	tapscript_pk String,              -- inscription create tapscript prefix: pubkey + checksig/checksigverify + n
	input_idx	 UInt32,
	invalue      UInt64,
	outvalue     UInt64,

	pointer      UInt64,
    nftflag      UInt32,
    parent       String,
    deligate     String,
    metaprotocol String,
    metadata     String,
	content_encoding String,

	content_type String,
	content_len  UInt32,
	content      String,
	height       UInt32,
	eventidx     UInt64,           -- event sequence within the block // fixme
	txidx        UInt32,
	blocktime    UInt32,
	nftidx       UInt32,            -- NFT creation index within the block
	nftnumber    Int64,
	nftheight    UInt32,            -- NFT creation height within the block; 0 means create, >0 means move
	sequence     UInt16,            -- NFT event sequence; 0 means create, >0 means move, and stops increasing after 65535
	nfttype      UInt8              -- NFT type: 1=text, 3=brc20
) engine=MergeTree()
ORDER BY (height, eventidx)         -- fixme: change txidx to eventidx
PARTITION BY intDiv(height, 2100)
`,

		// block revert data
		// ================================================================
		"DROP TABLE IF EXISTS blkrevert_height",
		"DROP TABLE IF EXISTS blkrevert_height_new",
		`
CREATE TABLE IF NOT EXISTS blkrevert_height (
	height    UInt32,
	op        UInt8,
	outpoint  String,
	utxo_data String
) ENGINE = MergeTree()
ORDER BY (height, op)
PARTITION BY intDiv(height, 2100)
`,
	}

	processAllSQLs = []string{
		// Build the block ID index.
		"INSERT INTO blk SELECT * FROM blk_height",
	}

	removeOrphanPartSQLs = []string{
		// No-op when there are no orphan blocks.
		"ALTER TABLE blk_height DELETE WHERE height >= ",
		"ALTER TABLE blk DELETE WHERE height >= ",

		"ALTER TABLE blkevent_height DELETE WHERE height >= ",
	}

	removeOrphanRevertPartSQLs = []string{
		"ALTER TABLE blkrevert_height DELETE WHERE height >= ",
	}

	createPartSQLs = []string{
		"CREATE TABLE IF NOT EXISTS blk_height_new AS blk_height",
		"CREATE TABLE IF NOT EXISTS blkevent_height_new AS blkevent_height",
		"CREATE TABLE IF NOT EXISTS blkrevert_height_new AS blkrevert_height",

		"TRUNCATE TABLE IF EXISTS blk_height_new",
		"TRUNCATE TABLE IF EXISTS blkevent_height_new",
		"TRUNCATE TABLE IF EXISTS blkrevert_height_new",
	}

	processPartSQLs = []string{
		"INSERT INTO blkevent_height SELECT * FROM blkevent_height_new",

		// Update the block ID index.
		"INSERT INTO blk SELECT * FROM blk_height_new",
		"INSERT INTO blk_height SELECT * FROM blk_height_new",

		// Optimize the blk table so height-ordered queries are consistent.
		// "OPTIMIZE TABLE blk_height FINAL",

		"TRUNCATE TABLE IF EXISTS blk_height_new",
		"TRUNCATE TABLE IF EXISTS blkevent_height_new",
	}

	// WAL phase: move revert data from staging to final table
	processRevertPartSQLs = []string{
		"INSERT INTO blkrevert_height SELECT * FROM blkrevert_height_new",
		"TRUNCATE TABLE IF EXISTS blkrevert_height_new",
	}
)

func CreateAllSyncCk() bool {
	logger.Log.Info("create sql: all")
	return ProcessSyncCk(createAllSQLs)
}

func ProcessAllSyncCk() bool {
	logger.Log.Info("sync sql: all")
	return ProcessSyncCk(processAllSQLs)
}

// Check whether ClickHouse delete operations have finished.
func CheckRemoveOrphanPartSyncCkDone(startBlockHeight uint32, removeOrphanPartSQLsWithHeight []string) bool {
	interTime := time.Now().Unix()
	logger.Log.Info("CheckRemoveOrphanPartSyncCkDone Inter", zap.Int64("Inter time", interTime))
	for _, command := range removeOrphanPartSQLsWithHeight {
		for {
			// Warn if the delete has been blocked for more than one minute.
			if time.Now().Unix()-interTime > 60 {
				logger.Log.Error("CheckRemoveOrphanPartSyncCkDone more than 60s", zap.Int64("stock time(s)", time.Now().Unix()-interTime))
			}

			// 1. Check mutation status in system tables; no row or is_done != 0 means the operation is complete.
			tableName := parseTableName(command)
			sql := `SELECT is_done
					FROM system.mutations
					WHERE database = currentDatabase()
					AND table = ?
					AND command = ?
					ORDER BY create_time DESC`

			// Use v2 driver's direct query
			rows, err := clickhouse.Query(context.Background(), sql, tableName, fmt.Sprintf("DELETE WHERE height >= %d", startBlockHeight))
			if err != nil {
				logger.Log.Info("query system.mutations failed", zap.Error(err), zap.String("command", command))
				return false
			}

			var isDone uint8
			hasResult := false
			if rows.Next() {
				if err := rows.Scan(&isDone); err != nil {
					rows.Close()
					logger.Log.Info("scan system.mutations failed", zap.Error(err))
					return false
				}
				hasResult = true
			}
			rows.Close()

			if hasResult && isDone == 0 {
				// Delete is not finished; wait and retry.
				time.Sleep(1 * time.Second)
				continue
			}

			// 2. Check whether matching rows still exist.
			sql = fmt.Sprintf(`SELECT count(*)
				FROM %v
				WHERE height >= ?
			`, tableName)

			rows, err = clickhouse.Query(context.Background(), sql, startBlockHeight)
			if err != nil {
				logger.Log.Info("query table count(*) failed", zap.Error(err), zap.String("tableName", tableName))
				return false
			}

			var count uint64
			if rows.Next() {
				if err := rows.Scan(&count); err != nil {
					rows.Close()
					logger.Log.Info("scan count failed", zap.Error(err))
					return false
				}
			}
			rows.Close()

			if count != 0 {
				// Delete is not finished; wait and retry.
				time.Sleep(1 * time.Second)
				continue
			}
			// Delete is complete; check the next table.
			break
		}
	}
	logger.Log.Info("CheckRemoveOrphanPartSyncCkDone Done", zap.Int64("Done time", time.Now().Unix()))
	return true
}

func RemoveOrphanPartSyncCk(startBlockHeight uint32, moveRevert bool) bool {
	logger.Log.Info("remove sql: part")
	removeOrphanPartSQLsWithHeight := []string{}

	orphanPartSQLs := removeOrphanPartSQLs
	if moveRevert {
		orphanPartSQLs = removeOrphanRevertPartSQLs
	}
	for _, psql := range orphanPartSQLs {
		removeOrphanPartSQLsWithHeight = append(removeOrphanPartSQLsWithHeight,
			psql+strconv.Itoa(int(startBlockHeight)),
		)
	}
	ok := ProcessSyncCk(removeOrphanPartSQLsWithHeight)
	if !ok {
		return ok
	}

	ok = CheckRemoveOrphanPartSyncCkDone(startBlockHeight, removeOrphanPartSQLsWithHeight)

	return ok
}

func CreatePartSyncCk() bool {
	// logger.Log.Info("create sql: part")
	return ProcessSyncCk(createPartSQLs)
}

func ProcessPartSyncCk() bool {
	// logger.Log.Info("sync sql: part")
	return ProcessSyncCk(processPartSQLs)
}

// ProcessRevertPartSyncCk moves revert data from staging to final table (WAL phase).
func ProcessRevertPartSyncCk() bool {
	return ProcessSyncCk(processRevertPartSQLs)
}

func ProcessSyncCk(processSQLs []string) bool {
	for _, psql := range processSQLs {
		partLen := len(psql)
		if partLen > 96 {
			partLen = 96
		}
		// logger.Log.Info("sync exec: " + psql[:partLen])
		if err := clickhouse.Exec(psql); err != nil {
			logger.Log.Info("sync exec err",
				zap.String("sql", psql[:partLen]), zap.Error(err))
			return false
		}
	}
	return true
}

// parseTableName extracts the table name from the ALTER TABLE template
func parseTableName(tmpl string) string {
	// tmpl format: "ALTER TABLE <table> DELETE WHERE ..."
	var table string
	_, err := fmt.Sscanf(tmpl, "ALTER TABLE %s DELETE WHERE", &table)
	if err != nil {
		return ""
	}
	return table
}

// DropOldRevertPartitions drops blkrevert_height partitions older than 4200 blocks (2 partitions).
func DropOldRevertPartitions(currentHeight uint32) {
	if currentHeight < 4200 {
		return
	}
	cutoff := (currentHeight - 4200) / 2100

	psql := fmt.Sprintf(`
SELECT partition FROM system.parts
WHERE database = currentDatabase()
  AND table = 'blkrevert_height'
  AND active
  AND toUInt32(partition) < %d
GROUP BY partition
ORDER BY partition`, cutoff)

	rows, err := clickhouse.Query(context.Background(), psql)
	if err != nil {
		logger.Log.Info("query revert partitions failed", zap.Error(err))
		return
	}
	defer rows.Close()

	for rows.Next() {
		var partition string
		if err := rows.Scan(&partition); err != nil {
			logger.Log.Info("scan revert partition failed", zap.Error(err))
			return
		}
		dropSQL := fmt.Sprintf("ALTER TABLE blkrevert_height DROP PARTITION %s", partition)
		if err := clickhouse.Exec(dropSQL); err != nil {
			logger.Log.Info("drop revert partition failed",
				zap.String("partition", partition), zap.Error(err))
		} else {
			logger.Log.Info("dropped revert partition", zap.String("partition", partition))
		}
	}
}
