# Indexer/API 重复代码重构清单

目的：记录原 indexer 和 API 两个模块合并后产生的重复代码，方便后续 Codex 按模块逐步消除重复，同时保持现有行为不变。

## 基本原则

- 每次只处理一组模块，保持改动小且可验证。
- API 优先直接复用 root/indexer 现有包；只有 root 包语义不合适、会引入循环依赖，或会把 API 专属行为污染到 root 时，才新增中立 helper 包。
- 已经出现语义分叉的包不要直接合并，先补测试或确认行为边界。
- API response/view model 继续留在 API 层，除非能明确抽出稳定共享契约。
- 不要提交本地或生成目录，例如 `.gocache/`、`skills/`。

## 优先级 1：完全相同或接近相同的重复代码

### NFT content code 表

- 重复文件：
  - `constant/nft_content_code.go`
  - `api/constant/nft_content_code.go`
- 当前状态：已完成。
- 完成方式：
  - 表和 helper 函数保留在 root `constant` 包。
  - API 的 NFT content 解码调用点直接使用 root `constant`，不再保留 API 侧 wrapper。
  - 已删除 `api/constant/nft_content_code.go` 和临时中立包 `constant/nftcontent`。

### Script compression / opcode helper

- 重复文件：
  - `parser/script/compress.go`
  - `api/lib/blkparser/script/compress.go`
  - `api/lib/brc20_swap/utils/script/compress.go`
  - 对应的 `compress_test.go`
  - `parser/script/opcode.go`
  - `api/lib/blkparser/script/opcode.go`
  - `api/lib/brc20_swap/utils/script/opcode.go`
- 当前状态：上述文件在三处检查中为完全相同。
- 建议合并方式：
  - 优先评估 API 是否能直接复用 root `parser/script` 的稳定导出能力。
  - 若 root `parser/script` 的 package init、测试依赖或已分叉 parser 语义不适合 API 直接复用，再只抽 compression/opcode 这类无状态底层实现，原 API 包保留薄 wrapper。
- 验证建议：
  - 删除旧副本前，先跑现有 script compression 测试。

### Middleware metrics / handler

- 重复文件：
  - `lib/midware/handler.go`
  - `api/lib/midware/handler.go`
  - `lib/midware/metrics.go`
  - `api/lib/midware/metrics.go`
- 当前状态：同源复制后 API 侧已有额外行为和注释差异。
- API 侧额外内容：
  - `api/lib/midware/sign.go`
  - `api/lib/midware/metrics.go` 增加了 `memory_heap_inuse`、metric item 超限时 logger/zap 记录、`ReportError`、`ErrorType`。
- 建议合并方式：
  - 将通用 metrics core 移到共享包。
  - API 专属的 error reporting / sign middleware 保留在 API 包，或做 thin wrapper。
  - 需要决定 indexer 是否也应获得 API 侧的超限日志和 `memory_heap_inuse` 指标。

## 优先级 2：同领域但已经分叉

### Logger

- 重复文件：
  - `logger/logger.go`
  - `api/logger/logger.go`
- 当前状态：几乎相同，但生命周期不同。
- 关键差异：
  - root logger 通过 `init()` 自动初始化。
  - API logger 暴露 `Init()`，由 `api.Run()` 调用。
- 建议合并方式：
  - 抽出共享构造函数，例如 `NewLoggerFromEnv()`。
  - root/API 保留各自生命周期 wrapper。
- 风险：
  - 改动 root 隐式初始化可能影响命令启动顺序。

### Redis 初始化

- 重复文件：
  - `rdb/init.go`
  - `api/dao/rdb/init.go`
- 当前状态：配置解析形态相似，但运行语义不同。
- root indexer 行为：
  - 单个 `RdbClient`。
  - 支持 cluster。
  - 包含 `FlushdbInRedis()`。
- API 行为：
  - `RdbRedisClient` 和 `RdbBrc20StateClient` 两个 client。
  - `InitClients()` 加载 `conf/kvdb.yaml` 和 `conf/api/kvdb_brc20.yaml`。
  - 总是使用 `redis.NewUniversalClient`。
  - 初始化时执行 ping。
- 建议合并方式：
  - 优先在 root `rdb` 暴露可复用的配置加载和 client 构造 helper。
  - 如果 root `rdb` 的全局 client 或 `FlushdbInRedis()` 语义会污染 API，再抽最小配置/client helper。
  - exported client globals 和 destructive flush 行为继续留在原包。

### ClickHouse 初始化

- 重复文件：
  - `loader/clickhouse/init.go`
  - `api/dao/clickhouse/init.go`
- 当前状态：使用同一类配置文件，但 client 类型不同。
- root indexer 行为：
  - 使用 native `driver.Conn`。
  - 初始化 HTTP RowBinary bulk insert 支持。
- API 行为：
  - 使用 `database/sql`，通过 `clickhouse.OpenDB` 创建连接。
  - 通过 `clickhImpl` 包装 DB。
- 建议合并方式：
  - 优先在 root ClickHouse 相关包暴露可复用的配置解析 helper。
  - 如果 root 包强绑定 native `driver.Conn` 或 bulk insert 初始化，再只抽最小配置解析 helper。
  - 除非两侧能接受同一个抽象，否则连接构造继续分开。

### Utility 函数

- 重复或相似文件：
  - `utils/utils.go`
  - `api/lib/utils/utils.go`
  - `api/lib/brc20_swap/utils/utils.go`
  - `api/lib/brc20_swap/utils/bip322/verify.go`
  - `api/lib/blkparser/utils.go`
  - `parser/script/model.go`
  - `api/lib/blkparser/script/model.go`
- 发现的重复函数：
  - `HashString`
  - `ReverseBytes`
  - `DecodeInscriptionFromBin`
  - `GetSha256`
  - 地址/script helper，例如 `GetPkScriptByAddress`、`GetAddressFromScript`、`GetPkScriptByPubkeyAndType`
  - 二分 helper `FindMaxLessThan`
- 建议合并方式：
  - 优先让 API 直接复用 root 已稳定导出的 helper。
  - 对 root 未导出、语义不适合或会引入循环依赖的 helper，再按职责拆，不要继续扩大单个 `utils` 包：
    - hash/byte helper
    - inscription ID encode/decode
    - address/script 转换
    - parser-only transaction/NFT encode 逻辑
  - 避免把所有 helper 都堆进一个新的大 `utils` 包。

## 优先级 3：需要先设计边界的 model / parser 重复

### Script parser 包

- 相关包：
  - `parser/script`
  - `api/lib/blkparser/script`
- 完全重复文件：
  - `compress.go`
  - `compress_test.go`
  - `opcode.go`
- 已分叉文件：
  - `model.go`
  - `satotx.go`
  - `script.go`
  - `utils.go`
  - `decode_test.go`
- root-only 文件：
  - `inscription.go`
  - `inscription_jubilee.go`
  - `nft_panic_test.go`
  - `nft_test.go`
  - `siphash.go`
  - `siphash_test.go`
  - `standard.go`
  - `standard_test.go`
- API-only 文件：
  - `script_test.go`
- 主要分叉：
  - root `NFTData` 支持多个 `Parents`、`ParentsId`、short NFT binary ID helper，更偏 ingest 逻辑。
  - API `NFTData` 使用单个 `Parent`，包含 API 解码 helper，flag setter 也不同。
  - API `script.go` 增加 `IsFalseOpreturn`。
  - root `utils.go` 保留优化后的 `GetOpcodeFormScript`，API 副本已移除。
- 建议合并方式：
  - 先评估 API 是否可以直接复用 root `parser/script` 的稳定导出能力。
  - 若不能直接复用，再只抽出完全重复的 compression/opcode 文件，并让原包保留薄 wrapper。
  - 再只为稳定基础能力建立 shared parser core。
  - ingest-only inscription parsing 暂留 root，等 API 行为有测试覆盖后再考虑继续合并。

### Model 包

- 相关包：
  - `model`
  - `api/model`
- 同名文件：
  - `do.go`
  - `model.go`
- 当前状态：同源但已经大幅分叉。
- root model 侧重点：
  - ingest pipeline 的 block/tx/utxo/NFT 存储状态
  - slab pools
  - compressed txo data
  - Redis/ClickHouse persistence shape
- API model 侧重点：
  - request/response DTO
  - query result model
  - BRC20/API view
- 建议合并方式：
  - 不建议整体合并。
  - 只抽稳定且两边确实共享的 primitive，例如 `NFTCreatePoint` 编码常量、字段完全一致的简单 block/tx DB shape。
  - API response struct 继续留在 `api/model`。

### Constants

- 相关包：
  - `constant`
  - `api/constant`
- 完全重复：
  - `nft_content_code.go`（已保留在 root `constant`，API 直接复用）
- 已分叉文件：
  - root `constant/constant.go`、`constant/redis.go`、`constant/brc20_reinscription.go`
  - API `api/constant/const.go`、`api/constant/env.go`
- 建议合并方式：
  - 优先让 API 直接复用 root `constant` 中稳定、不带运行时语义的常量和 helper，例如 NFT content code 表。
  - 对 height bit packing（`HEIGHT_MUTIPLY_NBIT`、mask）、mempool height 等跨 API/model 使用的常量，先确认不会引入 root 运行时配置语义，再迁移到 root 可复用位置。
  - runtime-tuned env 值和 activation height 变量继续放在拥有该行为的进程附近。

## 建议执行顺序

1. 已完成：`nft_content_code.go` 保留在 root `constant`，API NFT content 调用点直接复用 root `constant`。
2. 评估 script compression/opcode 是否可直接复用 root `parser/script`；不能直接复用时，再抽最小底层 helper 并保留 API wrapper。
3. 已完成：API 直接复用 root `logger`；root 保留 `init()` 并新增 `Init()`。
4. 只抽 Redis 和 ClickHouse 的共享配置解析，client globals 保持分开。
5. 小型 utility helper 优先直接复用 root 已稳定导出能力；无法直接复用时，再拆到职责明确的小 helper。
6. 等 root ingest 和 API query 路径测试覆盖更完整后，再处理 parser/model 的进一步收敛。

## 每一步验证清单

- 对改动过的 Go 文件运行 `gofmt`。
- 运行 `GOCACHE=/private/tmp/fractal-indexer-gocache go build -v`。
- 运行受影响 package 的 targeted tests。
- 注意：当前全量 `go test ./...` 有既有失败，包括 script tests 依赖 `test.txt`，以及 `model.TestBrc20` 断言失败。
