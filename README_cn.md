# fractal-indexer

`fractal-indexer` 为 Fractal Bitcoin 提供索引与查询服务。

- `fractal-indexer`：运行区块索引器，读取节点区块/RPC，解析 Ordinals/BRC-20 数据，写入 ClickHouse 和 Pika/Redis。
- `fractal-indexer -api`：运行 query HTTP/BRC20 服务，读取 indexer 写入的 ClickHouse 和 Pika/Redis 数据，对外提供查询 API。

## 架构概览

运行时分为两个角色。

### Indexer

Indexer 负责从 Fractal/Bitcoin 节点读取区块并构建索引：

1. 从 `conf/chain.yaml` 读取链、区块文件、RPC/ZMQ、激活高度等配置。
2. 从节点 RPC 或本地区块文件读取区块。
3. 解析交易、UTXO、inscription、BRC-20 相关事件。
4. 将区块和事件写入 ClickHouse。
5. 将 UTXO、NFT ID 映射、同步高度和回滚状态写入 Pika/Redis。

核心存储：

- ClickHouse：区块元数据、inscription 事件、WAL/reorg 数据。
- Pika/Redis：UTXO、NFT ID 映射、同步高度、query BRC-20 state/history。

### Query API

Query API 负责读取 indexer 产物并提供 HTTP 查询：

1. 使用同一个根 `conf/db.yaml` 连接 ClickHouse。
2. 使用同一个根 `conf/kvdb.yaml` 读取 indexer 写入的主 Pika/Redis 数据。
3. 使用 `conf/api/kvdb_brc20.yaml` 保存 query 侧 BRC-20 state/history。
4. 使用 `conf/api/conf.yaml` 读取 query 自有配置。
5. 使用 `conf/chain.yaml` 应用 report 的链相关设置。

API 路由包括 `/blockchain/info`、inscription、BRC20、report、admin 等接口。

## 配置文件

运行配置都放在根 `conf/` 目录下。API 自有配置放在 `conf/api/`，主 DB/KVDB 配置统一使用根目录配置。

| 文件 | 使用方 | 说明 |
|------|--------|------|
| `conf/chain.yaml` | indexer + query | 链类型、blocks 路径、RPC/ZMQ、激活高度，以及 query report 的链相关设置。 |
| `conf/db.yaml` | indexer + query | ClickHouse 连接配置。当前配置使用 Docker 内部域名 `clickhouse`。 |
| `conf/kvdb.yaml` | indexer + query | 主 Pika/Redis 配置，用于 UTXO/NFT/同步高度等 indexer 共享数据。默认 `addrs` 是单元素 list：`["pika:6390"]`。 |
| `conf/api/conf.yaml` | query | HTTP/cache timeout、query logger 等 API 自有配置。 |
| `conf/api/kvdb_brc20.yaml` | query | BRC-20 state/history Pika/Redis 配置，默认 `["pika:6390"]`。 |

配置中的服务地址默认按 Docker 内部 DNS 写法设置：

- `clickhouse`
- `pika`
- `fractald`

如果在宿主机直接运行，需要把这些 host 改成宿主机可访问地址，或通过本机 DNS/hosts 解析到对应服务。

## 构建

按项目约束，不要使用 `go build ./...`，因为会把 `tools/` 下部分无关代码一起编译。

推荐构建根二进制：

```sh
go build -v fractal-indexer 2>&1
```

如果本机默认 Go build cache 没权限，可以显式指定：

```sh
GOCACHE=$(pwd)/.gocache go build -v fractal-indexer 2>&1
```

## 运行 Indexer

默认启动 indexer：

```sh
LISTEN=:8000 ./fractal-indexer
```

常用参数：

| 参数 | 说明 |
|------|------|
| `-full` | 从创世块开始全量重建。会清空 Redis/Pika 并初始化 ClickHouse 同步表。 |
| `-start <height>` | 指定同步起始高度。在 `metric_only -full` 模式下会覆盖 `metrics_start_height`。 |
| `-end <height>` | 指定同步结束高度。设置后会关闭 lag 行为。 |
| `-once` | 同步一轮后退出，适合由外部 supervisor 周期拉起。 |
| `-lag <n>` | 保留最新 `n` 个块不同步。`-span` 是兼容旧参数的别名。 |
| `-nblock <n>` | 每轮从 RPC 拉取的 block id 数量，默认 256。 |
| `-reorg` | reorg 测试模式。 |

Metric-only 全量同步可以从配置高度开始重建指标表：

```yaml
index_mode: metric_only
metrics_start_height: 800000
```

当 `index_mode: metric_only` 时，`-full` 只会重建 `blkmetric_height`，并从 `metrics_start_height` 开始同步。命令行传入 `-start <height>` 时优先使用命令行高度。业务全量同步仍然从创世块开始，因为 UTXO 和 inscription 状态依赖历史区块。

如果只想局部重跑指标，使用 `index_mode: metric_only` 且不要加 `-full`，传入 `-start <height>`。indexer 会删除 `blkmetric_height` 中 `height >= <height>` 的行，从 RPC 重新加载边界区块头，并从该高度继续写入指标，同时保留更早的指标数据。

指标行会在 `blkmetric_height.blocktime` 保存真实区块时间；指标表结构变化后需要重建该表。

环境变量：

| 变量 | 说明 |
|------|------|
| `LISTEN` | indexer metrics/pprof HTTP 监听地址。 |
| `ENABLE_WAL=true|1` | 启用 WAL/reorg 恢复流程。 |
| `MAX_REORG_BLOCKS` | 最大 reorg 检查深度，默认 100。 |

全量或分段同步示例：

```sh
./fractal-indexer -full -end 100000
./fractal-indexer -start 100000 -end 200000
./fractal-indexer
```

停止 indexer 时应发送 `SIGINT`/`SIGTERM`，让程序完成当前批次和日志 flush。大量同步期间不要直接强杀进程。

## 运行 Query API

启动 query API：

```sh
LISTEN=:5555 ./fractal-indexer -api
```

常用 API 环境变量：

| 变量 | 说明 |
|------|------|
| `LISTEN` | API HTTP 监听地址。 |
| `BASE_PATH` | Swagger base path。 |
| `DISABLE_BRC20_PROCESS=true` | 只启动 HTTP，不跑 BRC20 后台处理。适合快速验证 API 进程和基础查询。 |
| `DISABLE_BRC20_PROCESS=once` | 单次处理 BRC20 后退出后台循环。 |
| `LISTEN_BEFORE_BRC20_PROCESS=true` | 先开放 HTTP，再等待 BRC20 后台加载。 |
| `DUMP_BRC20_DATA` | BRC20 dump 相关开关。 |
| `BRC20_PROCESS_BEFORE_HEIGHT` | 限制 BRC20 处理高度。 |
| `BRC20_DEBUG_LOG=true` | 打开 BRC20 debug 日志。 |
| `TESTNET` | 使用 testnet 地址参数。 |

最小验证：

```sh
DISABLE_BRC20_PROCESS=true LISTEN=:5555 ./fractal-indexer -api
curl -i http://127.0.0.1:5555/
curl -i http://127.0.0.1:5555/blockchain/info
```

BRC20 相关验证：

```sh
LISTEN=:5555 ./fractal-indexer -api
curl -i http://127.0.0.1:5555/brc20/status
curl -i http://127.0.0.1:5555/brc20/bestheight
```

## Docker

根目录 Dockerfile 构建的是 `fractal-indexer` 二进制。当前 compose 文件默认启动 indexer 服务：

```sh
docker-compose -f docker-compose-init.yaml up -d
docker-compose -f docker-compose-batch.yaml up -d
docker-compose up -d
```

这些 compose 会挂载根 `./conf` 到容器 `/data/conf`。如果要运行 query API，需要将 entrypoint/command 改成：

```sh
./fractal-indexer -api
```

并设置至少：

```sh
LISTEN=0.0.0.0:8000
```

## 注意事项

- Query 主数据必须读取 indexer 写入的同一套 `conf/db.yaml` 和 `conf/kvdb.yaml`，否则 API 会查询到不同数据面。
- `conf/api/kvdb_brc20.yaml` 是 query BRC20 state 独立配置，不应误写到主 UTXO/NFT KVDB，除非部署上明确复用同一个 Pika DB。
- 主 DB/KVDB 配置必须来自根目录 `conf/db.yaml` 和 `conf/kvdb.yaml`。
- 默认配置里的 `clickhouse`、`pika`、`fractald` 是 Docker 网络内服务名；宿主机直接运行需要自行调整解析。

## 资源需求参考

实际资源与链高度、是否保留 BRC20 history、ClickHouse 分区策略和 Pika 部署有关。历史经验值：

| 组件           | 磁盘最低 | 磁盘建议 | 内存最低 | 内存建议 |
|----------------|----------|----------|----------|----------|
| indexer 进程   | 1 GB     | 1 GB     | 8 GB     | 16 GB+   |
| query API 进程 | 50 GB    | 50 GB    | 48 GB    | 64 GB+   |
| fractald       | 2 TB     | 3 TB+    | 8 GB     | 16 GB+   |
| ClickHouse     | 300 GB   | 500 GB+  | 8 GB     | 16 GB+   |
| Pika           | 100 GB   | 200 GB+  | 8 GB     | 16 GB+   |
