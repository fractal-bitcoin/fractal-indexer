# Indexer/API 去重与收敛重构清单

目的：记录原 indexer 和 API 两套代码合并后仍残留的重复实现，并指导后续按小步、可验证的方式继续收敛。当前阶段的目标不是再维护两套相似模块，而是逐步把 root/indexer 与 API 当作同一个项目来整理：能删除的重复代码优先删除，能直接复用 root 包的 API 代码优先复用 root 包。

## 当前阶段结论

- 优先级 1 的明显重复代码已经基本处理完：NFT content code 表、script compression/opcode helper、middleware handler/metrics、logger。
- 仍需要处理的主要重复集中在初始化配置、通用 helper、parser/model 边界和常量归属。
- 后续方向应从“保留 API/root 两份近似实现”转为“API 和 root 共用同一套稳定实现”。
- `api/lib/brc20_swap` 暂时作为边界例外处理：只删除明显无用或已经被替代的重复代码，不强行把 BRC20 swap 内部逻辑并入 root 主路径，避免引入大 diff 和业务回归。

## 重要原则

- 尽量删除多余代码。优先删除无调用点、纯 wrapper、重复 alias、已经被新格式替代的兼容实现；不要为了“共享”新增一层没有价值的转发代码。
- API 和 root 要往一个项目方向合并。API 优先复用 root 中已经稳定的包、常量和 helper，而不是继续在 `api/` 下维护一份相同逻辑。
- 每次只处理一组模块，保持 diff 小。不要把 Redis、ClickHouse、utils、parser/model 一次性混在同一个变更里。
- 小 diff 优先于一次性完美抽象。每一步都应便于 code review、问题定位和回滚。
- 不做无关重构。不要顺手改命名、格式、目录结构或导入风格，除非这是当前去重步骤必须的。
- 行为不明确时先保守。已经明显分叉的代码不要硬合并；先补测试、确认调用语义，再决定删除、复用或抽 shared helper。
- 删除优先，抽象其次。只有当两边都需要同一逻辑，且直接复用 root 会引入循环依赖或污染运行语义时，才新增职责明确的小 helper。
- 保留进程边界语义。全局 client、启动初始化、环境变量、destructive 操作等仍应留在拥有该行为的包内，不为了去重牺牲清晰边界。
- `api/lib/brc20_swap` 边界内的业务逻辑先保持独立。可以删除已废弃工具、复用通用纯函数，但不要把 swap 业务模型和 root ingest 模型强行合并。
- 不提交本地或生成目录，例如 `.gocache/`、`skills/`。

## 已完成项

### NFT content code 表

- 原重复文件：
  - `constant/nft_content_code.go`
  - `api/constant/nft_content_code.go`
- 当前状态：已完成。
- 完成方式：
  - 表和 helper 函数保留在 root `constant` 包。
  - API NFT content 解码调用点直接使用 root `constant`。
  - 已删除 `api/constant/nft_content_code.go` 和临时中立包 `constant/nftcontent`。

### Script compression / opcode helper

- 原重复文件：
  - `parser/script/compress.go`
  - `api/lib/blkparser/script/compress.go`
  - `api/lib/brc20_swap/utils/script/compress.go`
  - 对应的 `compress_test.go`
  - `parser/script/opcode.go`
  - `api/lib/blkparser/script/opcode.go`
  - `api/lib/brc20_swap/utils/script/opcode.go`
- 当前状态：已完成。
- 完成方式：
  - root `parser/script` 的 compression 导出面无当前仓库生产调用点，已删除 `parser/script/compress.go` 和 `parser/script/compress_test.go`。
  - `api/lib/blkparser/script/compress.go` 只是 root `parser/script` 的简单封装，且 API 侧无调用点，已删除。
  - BRC20 history 已改为固定宽度整数和 raw pkScript 新格式，不再需要兼容旧 Pika history 数据；已删除 `api/lib/brc20_swap/utils/script/compress.go`。
  - opcode 常量不再保留重复实现或别名；root/API 调用点直接使用 `txscript.OP_*`。

### Middleware metrics / handler

- 原重复文件：
  - `lib/midware/handler.go`
  - `api/lib/midware/handler.go`
  - `lib/midware/metrics.go`
  - `api/lib/midware/metrics.go`
- 当前状态：已完成。
- 完成方式：
  - API 改为直接使用 root `lib/midware`。
  - API 侧 metrics 增量已合入 root `lib/midware`：`memory_heap_inuse`、metric item 超限日志、`ReportError`、`ErrorType`。
  - 已删除 `api/lib/midware/handler.go` 和 `api/lib/midware/metrics.go`。
  - `api/lib/midware/sign.go` 保留在 API 包。

### Logger

- 原重复文件：
  - `logger/logger.go`
  - `api/logger/logger.go`
- 当前状态：已完成。
- 完成方式：
  - API 改为直接使用 root `logger`。
  - root `logger` 增加 `Init()`，同时保留 `init()` 自动初始化。
  - 已删除 `api/logger` 重复实现和重复测试。

## 待处理项

### 1. Redis 初始化

- 相关文件：
  - `rdb/init.go`
  - `api/dao/rdb/init.go`
- 当前状态：仍有重复的 viper 配置读取、Redis options 组装、client 构造逻辑。
- root 行为：
  - 单个 `RdbClient`。
  - 支持 cluster。
  - 包含 `FlushdbInRedis()`。
- API 行为：
  - `RdbRedisClient` 和 `RdbBrc20StateClient` 两个 client。
  - `InitClients()` 加载 `conf/kvdb.yaml` 和 `conf/api/kvdb_brc20.yaml`。
  - 总是使用 `redis.NewUniversalClient`。
  - 初始化时执行 ping。
- 推荐处理方式：
  - 第一步只抽共享配置结构和 options 构造，保持 root/API 的 globals、ping、flush、cluster 选择逻辑不变。
  - 如果直接放进 root `rdb` 会让 API 依赖 root 的全局 client 或 `FlushdbInRedis()`，则抽一个更小、更中立的配置 helper。
  - 不要在同一个 diff 里调整调用方业务逻辑。

### 2. ClickHouse 初始化

- 相关文件：
  - `loader/clickhouse/init.go`
  - `api/dao/clickhouse/init.go`
- 当前状态：读取同一类 `conf/db.yaml` 配置，但连接类型不同。
- root 行为：
  - 使用 native `driver.Conn`。
  - 初始化 HTTP RowBinary bulk insert 支持。
- API 行为：
  - 使用 `database/sql`，通过 `clickhouse.OpenDB` 创建连接。
  - 通过 `clickhImpl` 包装 DB。
- 推荐处理方式：
  - 第一步只抽配置读取和 `clickhouse.Options` 组装。
  - root 继续创建 `driver.Conn` 并初始化 RowBinary HTTP client。
  - API 继续创建 `database/sql` DB 并包装为 `clickhImpl`。
  - 不要为了去重强行统一连接抽象。

### 3. Utility helper

- 相关文件：
  - `utils/utils.go`
  - `api/lib/utils/utils.go`
  - `parser/script/model.go`
  - `api/lib/blkparser/script/model.go`
  - `api/lib/brc20_swap/utils/utils.go`
  - `api/lib/brc20_swap/utils/bip322/verify.go`
- 已完成：
  - 删除 `api/lib/blkparser.HashString`，API block/utxo/tx 解析调用点直接使用 root `utils.HashString`。
  - 删除 `api/lib/blkparser/script.IsFalseOpreturn`，当前仓库内无调用点。
  - 删除 `api/lib/blkparser/script.HashString`，API inscription ID 解码复用 root `parser/script.HashString`，保留 API 原有解码语义。
- 仍存在的重复或相似函数：
  - `HashString`
  - `ReverseBytes`
  - `DecodeInscriptionFromBin`
  - `GetSha256`
  - `GetPkScriptByAddress`
  - `GetAddressFromScript`
  - `GetPkScriptByPubkeyAndType`
  - `FindMaxLessThan`
- 推荐处理顺序：
  - 先处理纯函数：hash、byte reverse、inscription ID encode/decode。这类行为稳定，最适合小 diff 去重。
  - 再处理 address/script helper。这里涉及网络参数、OP_RETURN 特例、API 环境变量和 BRC20 swap 调用，需要单独审核。
  - 最后处理 parser-only helper。不要把 parser 语义塞进通用 `utils`。
- 注意事项：
  - `api/lib/brc20_swap` 里的 helper 可以逐步复用外部纯函数，但不要让 root 主路径依赖 BRC20 swap 包。
  - 避免新增一个更大的 `utils` 包来容纳所有东西；按职责拆小 helper 或直接复用已有 root 导出函数。

### 4. Script parser 包

- 相关包：
  - `parser/script`
  - `api/lib/blkparser/script`
- 当前状态：
  - compression/opcode 重复已删除。
  - `script.go`、`satotx.go`、`utils.go`、`model.go` 仍有相似或分叉实现。
- 主要分叉：
  - root `NFTData` 支持多个 `Parents`、`ParentsId`、short NFT binary ID helper，更偏 ingest 逻辑。
  - API `NFTData` 使用单个 `Parent`，包含 API 解码 helper，flag setter 也不同。
  - API `script.go` 的无调用点 `IsFalseOpreturn` 已删除。
  - root `utils.go` 保留 `GetOpcodeFormScript`，API 副本已移除。
- 推荐处理方式：
  - 先处理基础脚本判断和 satotx 逻辑，确认 API 是否可以直接复用 root `parser/script` 的稳定导出能力。
  - `NFTData` 这类模型差异先不要硬合并；需要测试覆盖和调用方梳理后再做。
  - 如果 API-only helper 实际无调用点，优先删除；如果有调用点，再考虑移动到 root。

### 5. Model 包

- 相关包：
  - `model`
  - `api/model`
- 当前状态：同源但已经大幅分叉。
- root model 侧重点：
  - ingest pipeline 的 block/tx/utxo/NFT 存储状态。
  - slab pools。
  - compressed txo data。
  - Redis/ClickHouse persistence shape。
- API model 侧重点：
  - request/response DTO。
  - query result model。
  - BRC20/API view。
- 推荐处理方式：
  - 不做整体合并。
  - 只抽稳定且两边确实共享的 primitive，例如 `NFTCreatePoint` 编码常量、字段完全一致的简单 block/tx DB shape。
  - API response/view model 继续留在 `api/model`。

### 6. Constants

- 相关包：
  - `constant`
  - `api/constant`
- 当前状态：
  - NFT content code 表已收敛到 root `constant`。
  - `api/constant/const.go` 仍包含 height bit packing、mempool height、SNS suffix、API 查询相关常量。
  - root `constant/constant.go`、`constant/redis.go`、`constant/brc20_reinscription.go` 保留 indexer 运行语义和激活高度。
- 推荐处理方式：
  - 稳定、不带运行时语义的常量优先迁移到 root `constant` 并让 API 复用。
  - `HEIGHT_MUTIPLY_NBIT`、`HEIGHT_MUTIPLY_MASK`、`MEMPOOL_HEIGHT` 这类跨 model/API 使用的 primitive 可以优先评估。
  - runtime-tuned env 值、activation height、API-only 查询限制继续留在拥有该行为的进程附近。

## 建议执行顺序

1. Redis 初始化：只抽配置/options helper，保持行为不变。
2. ClickHouse 初始化：只抽配置/options helper，连接类型继续分开。
3. Utility 纯函数：先去重 hash、reverse、inscription ID 相关函数。
4. Constants primitive：评估并迁移 height packing/mempool height 等稳定常量。
5. Script parser 基础能力：先处理 `script.go`、`satotx.go` 这类低风险重复。
6. Model/parser 深层收敛：等测试和行为边界更清楚后再推进。

## 每一步验收清单

- diff 只覆盖当前模块，不夹带无关格式化或命名调整。
- 能删除的重复文件或函数已删除，没有留下无意义 wrapper。
- API 调用 root 包时没有引入循环依赖。
- `api/lib/brc20_swap` 没有被强行并入 root 主路径。
- 对改动过的 Go 文件运行 `gofmt`。
- 运行 `GOCACHE=/private/tmp/fractal-indexer-gocache go build -v`。
- 运行受影响 package 的 targeted tests。
- 注意：当前全量 `go test ./...` 有既有失败，包括 script tests 依赖 `test.txt`，以及 `model.TestBrc20` 断言失败。
