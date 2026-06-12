# fractal-indexer

`fractal-indexer` provides indexing and query services for Fractal Bitcoin.

- `fractal-indexer`: runs the block indexer, reads node block/RPC data, parses Ordinals/BRC-20 data, and writes to ClickHouse and Pika/Redis.
- `fractal-indexer -api`: runs the query HTTP/BRC20 service, reads ClickHouse and Pika/Redis data written by the indexer, and exposes query APIs.

## Architecture Overview

Runtime is split into two roles.

### Indexer

The indexer reads blocks from a Fractal/Bitcoin node and builds indexes:

1. Reads chain, block file, RPC/ZMQ, activation height, and related settings from `conf/chain.yaml`.
2. Reads blocks from node RPC or local block files.
3. Parses transactions, UTXOs, inscriptions, and BRC-20 related events.
4. Writes blocks and events to ClickHouse.
5. Writes UTXOs, NFT ID mappings, sync heights, and rollback state to Pika/Redis.

Core storage:

- ClickHouse: block metadata, inscription events, and WAL/reorg data.
- Pika/Redis: UTXOs, NFT ID mappings, sync heights, and query-side BRC-20 state/history.

### Query API

The query API reads indexer outputs and exposes HTTP queries:

1. Uses the root `conf/db.yaml` to connect to ClickHouse.
2. Uses the root `conf/kvdb.yaml` to read primary Pika/Redis data written by the indexer.
3. Uses `conf/api/kvdb_brc20.yaml` to store query-side BRC-20 state/history.
4. Uses `conf/api/conf.yaml` for query-specific configuration.
5. Uses `conf/chain.yaml` to apply chain-specific report settings.

API routes include `/blockchain/info`, inscription, BRC20, report, admin, and other endpoints.

## Configuration Files

Runtime configuration files are under the root `conf/` directory. API-specific configuration lives under `conf/api/`, while primary DB/KVDB settings are shared through the root configs.

| File | Used by | Description |
|------|---------|-------------|
| `conf/chain.yaml` | indexer + query | Chain type, blocks path, RPC/ZMQ, activation heights, and query report chain settings. |
| `conf/db.yaml` | indexer + query | ClickHouse connection configuration. The current config uses the Docker-internal hostname `clickhouse`. |
| `conf/kvdb.yaml` | indexer + query | Primary Pika/Redis configuration for shared indexer data such as UTXOs, NFTs, and sync heights. The default `addrs` value is a single-item list: `["pika:6390"]`. |
| `conf/api/conf.yaml` | query | API-owned configuration such as HTTP/cache timeout and query logger settings. |
| `conf/api/kvdb_brc20.yaml` | query | Pika/Redis configuration for BRC-20 state/history. The default value is `["pika:6390"]`. |

Service addresses in the configuration default to Docker-internal DNS names:

- `clickhouse`
- `pika`
- `fractald`

If you run directly on the host machine, change these hosts to addresses reachable from the host, or map them through local DNS/hosts.

## Build

Per project constraints, do not use `go build ./...`, because that also compiles some unrelated code under `tools/`.

Recommended root binary build:

```sh
go build -v fractal-indexer 2>&1
```

If the default local Go build cache has permission issues, set it explicitly:

```sh
GOCACHE=$(pwd)/.gocache go build -v fractal-indexer 2>&1
```

## Run Indexer

Start the indexer in default mode:

```sh
LISTEN=:8000 ./fractal-indexer
```

Common flags:

| Flag | Description |
|------|-------------|
| `-full` | Fully rebuild from the genesis block. This clears Redis/Pika and initializes ClickHouse sync tables. |
| `-start <height>` | Set the sync start height. In `metric_only -full` mode, this overrides `metrics_start_height`. |
| `-end <height>` | Set the sync end height. This disables lag behavior. |
| `-once` | Exit after one sync round. Useful when an external supervisor periodically starts the process. |
| `-lag <n>` | Keep the latest `n` blocks unsynced. `-span` is a backward-compatible alias. |
| `-nblock <n>` | Number of block IDs fetched from RPC per round. Default is 256. |
| `-reorg` | Reorg test mode. |

Metric-only full sync can rebuild the metric table from a configured height:

```yaml
index_mode: metric_only
metrics_start_height: 800000
```

When `index_mode: metric_only`, `-full` rebuilds only `blkmetric_height` and starts at `metrics_start_height`. Passing `-start <height>` on the command line takes precedence. Business full sync still starts from genesis because UTXO and inscription state depend on earlier blocks.

For a partial metric replay, run `index_mode: metric_only` without `-full` and pass `-start <height>`. The indexer deletes `blkmetric_height` rows where `height >= <height>`, reloads the boundary header from RPC, and resumes metrics from that height while preserving earlier metric rows.

Metric rows include the block timestamp in `blkmetric_height.blocktime`; rebuild this table after metric schema changes.

Environment variables:

| Variable | Description |
|----------|-------------|
| `LISTEN` | HTTP listen address for indexer metrics/pprof. |
| `ENABLE_WAL=true|1` | Enables the WAL/reorg recovery flow. |
| `MAX_REORG_BLOCKS` | Maximum reorg check depth. Default is 100. |

Full or range sync examples:

```sh
./fractal-indexer -full -end 100000
./fractal-indexer -start 100000 -end 200000
./fractal-indexer
```

When stopping the indexer, send `SIGINT`/`SIGTERM` so the program can finish the current batch and flush logs. Avoid force-killing the process during large sync runs.

## Run Query API

Start the query API:

```sh
LISTEN=:5555 ./fractal-indexer -api
```

Common API environment variables:

| Variable | Description |
|----------|-------------|
| `LISTEN` | API HTTP listen address. |
| `BASE_PATH` | Swagger base path. |
| `DISABLE_BRC20_PROCESS=true` | Start HTTP only and skip BRC20 background processing. Useful for quickly validating the API process and basic queries. |
| `DISABLE_BRC20_PROCESS=once` | Process BRC20 once, then exit the background loop. |
| `LISTEN_BEFORE_BRC20_PROCESS=true` | Open HTTP first, then wait for BRC20 background loading. |
| `DUMP_BRC20_DATA` | Switches related to BRC20 dumps. |
| `BRC20_PROCESS_BEFORE_HEIGHT` | Limit the BRC20 processing height. |
| `BRC20_DEBUG_LOG=true` | Enables BRC20 debug logs. |
| `TESTNET` | Uses testnet address parameters. |

Minimal validation:

```sh
DISABLE_BRC20_PROCESS=true LISTEN=:5555 ./fractal-indexer -api
curl -i http://127.0.0.1:5555/
curl -i http://127.0.0.1:5555/blockchain/info
```

BRC20-related validation:

```sh
LISTEN=:5555 ./fractal-indexer -api
curl -i http://127.0.0.1:5555/brc20/status
curl -i http://127.0.0.1:5555/brc20/bestheight
```

## Docker

The root Dockerfile builds the `fractal-indexer` binary. The current compose files start the indexer service:

```sh
docker-compose -f docker-compose-init.yaml up -d
docker-compose -f docker-compose-batch.yaml up -d
docker-compose up -d
```

These compose files mount the root `./conf` directory to `/data/conf` inside the container. To run the query API instead, change the entrypoint/command to:

```sh
./fractal-indexer -api
```

And set at least:

```sh
LISTEN=0.0.0.0:8000
```

## Notes

- Query primary data must read the same `conf/db.yaml` and `conf/kvdb.yaml` written by the indexer. Otherwise, the API will query a different data plane.
- `conf/api/kvdb_brc20.yaml` is the independent query BRC20 state configuration. It should not be accidentally pointed at the primary UTXO/NFT KVDB unless the deployment intentionally reuses the same Pika DB.
- Primary DB/KVDB configs must come from the root `conf/db.yaml` and `conf/kvdb.yaml`.
- The default `clickhouse`, `pika`, and `fractald` values are Docker-network service names. Direct host execution requires you to adjust name resolution yourself.

## Resource Requirements Reference

Actual resource usage depends on chain height, whether BRC20 history is retained, ClickHouse partition strategy, and Pika deployment. Historical reference values:

| Component | Minimum disk | Recommended disk | Minimum memory | Recommended memory |
|-----------|--------------|------------------|----------------|--------------------|
| indexer process | 1 GB | 1 GB | 8 GB | 16 GB+ |
| query API process | 50 GB | 50 GB | 48 GB | 64 GB+ |
| fractald | 2 TB | 3 TB+ | 8 GB | 16 GB+ |
| ClickHouse | 300 GB | 500 GB+ | 8 GB | 16 GB+ |
| Pika | 100 GB | 200 GB+ | 8 GB | 16 GB+ |
