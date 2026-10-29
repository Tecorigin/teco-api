# 使用日志改动说明：上游请求 ID 与流状态筛选

记录日期：2026-10-29。本文记录当前代码实现、改动原因、部署限制和验证情况，不代表已经更新运行中的服务或完成生产数据库迁移。

## 一、改动概览

| 问题 | 最终处理方式 |
| --- | --- |
| 成功请求能看到 Upstream Request ID，400 等失败请求看不到 | 统一提取响应头 ID；错误响应体补充提取；错误日志同步独立字段；详情兼容历史 `other` 字段 |
| 消费日志显示流错误，例如 `client_gone`，但无法筛选 | 增加流状态筛选，列表、总数和统计统一使用独立的 `logs.stream_status` 列 |
| 在 `other` JSON 中查询流状态存在性能风险 | 移除查询阶段的 JSON 解析，写入时提取状态，并建立索引 |

## 二、Upstream Request ID 问题

### 2.1 原因

这不是简单的“非 200 响应就不保存”。原有普通聊天路径已经在判断 HTTP 状态码之前提取部分上游响应头，但记录和显示存在不一致：

- 提取器将 ID 写入 `RelayInfo.UpstreamRequestId` 和 Gin context 的 `upstream_request_id`。
- 错误日志通过 `common.UpstreamRequestIdKey` 读取独立字段，该 key 与 `upstream_request_id` 不同。
- 错误日志的 `other.upstream_request_id` 可能已有 ID，但独立列 `logs.upstream_request_id` 没有同步。
- 详情页原来只读取独立列，因此已经存进 `other` 的 ID 也不会显示。
- 部分音频、嵌入、重排等处理路径没有调用统一提取器。
- 重试复用同一份 `RelayInfo` 和 Gin context，旧 ID 可能影响下一次尝试。

### 2.2 响应头统一提取

实现位置：[relay/helper/common.go](../relay/helper/common.go) 的 `CaptureUpstreamRequestID`。

按以下顺序查找第一个非空 ID，去掉首尾空白，并保存来源：

1. `openai-request-id`
2. `anthropic-request-id`
3. `x-request-id`
4. `request-id`
5. `x-amzn-requestid`
6. `x-amzn-request-id`
7. `x-goog-request-id`
8. `X-Oneapi-Request-Id`，即 `common.RequestIdKey`
9. `cf-ray`

同时同步以下位置：

- `RelayInfo.UpstreamRequestId` 和 `RelayInfo.UpstreamRequestSource`。
- Gin context 的 `common.UpstreamRequestIdKey`。
- Gin context 的 `upstream_request_id` 和 `upstream_request_id_source`。

响应头来源统一记录为小写名称，例如 `header:x-request-id`。同一次尝试中，已有 ID 不会被后续候选覆盖。

[relay/channel/api_request.go](../relay/channel/api_request.go) 的公共 HTTP 请求层在收到响应后调用提取器，不以 HTTP 200 为前提。模型 handler 也在错误处理前捕获 ID，以覆盖不同适配器路径。

相关 handler 包括普通聊天、Claude、Gemini、图像、音频、嵌入、重排、Responses，以及 Chat Completions 转 Responses 路径。

### 2.3 JSON 错误响应体补充提取

实现位置：[service/error.go](../service/error.go) 的 `RelayErrorHandler`。

只有在错误响应 JSON 可解析、且 context 中没有已捕获 ID 时，才继续检查：

1. 响应体顶层的 `request_id`，其次是 `trace_id`。
2. 响应体 `error` 对象中的 `request_id`，其次是 `trace_id`。

只接受非空字符串，并保存 `body:request_id` 或 `body:trace_id` 来源。响应头优先于响应体。

此处不是任意字段、任意嵌套结构的递归扫描，也不从错误消息文本中猜测 ID。成功响应体中的 ID 仍由已有的供应商适配器逻辑处理，本次通用响应体补充针对错误处理路径。

### 2.4 为什么改成传 Gin context

调用由：

```go
service.RelayErrorHandler(c.Request.Context(), httpResp, false)
```

改为：

```go
service.RelayErrorHandler(c, httpResp, false)
```

函数签名仍接收 `context.Context`，但新增逻辑需要将实际参数断言为 `*gin.Context`，通过 `c.Set(...)` 把响应体 ID 写回错误日志读取的 context。传 `c.Request.Context()` 无法完成这个断言和写回。

两个 context 并不完全等价：Gin 的取消、超时方法是否转发给 HTTP 请求 context，取决于 `ContextWithFallback` 配置。当前函数使用 context 写日志和保存 ID，没有依赖其取消、超时方法；这次也没有替换实际发送上游请求时的请求 context。以后若在该函数中新增取消或超时逻辑，需要重新检查这个接口设计。

### 2.5 错误日志、重试和前端

- [model/log.go](../model/log.go)：`RecordErrorLog` 优先采用非空的 `other.upstream_request_id` 写入独立列，避免已有 ID 只停留在 `other` 中。
- [controller/relay.go](../controller/relay.go)：每次模型请求尝试前清空 `RelayInfo` 中的 ID、来源，以及对应 Gin context key，防止跨尝试串用。
- [details-dialog.tsx](../web/default/src/features/usage-logs/components/dialogs/details-dialog.tsx)：详情优先读取独立列，缺失时兼容读取 `other.upstream_request_id`。
- [usage-logs/types.ts](../web/default/src/features/usage-logs/types.ts)：补齐日志 `other` 中的 ID、来源类型。

上游请求 ID 原有独立列和查询索引继续使用，本次没有为这个问题新增数据库列。新写入的错误日志可以通过独立 ID 字段查询；历史日志仅在 `other` 中有 ID 的，详情可以显示，但没有自动回填独立列，因此独立字段检索不保证命中这些旧记录。

### 2.6 边界

- 上游未返回支持的 ID 字段，或请求在收到响应前就失败，无法凭空生成供应商请求 ID。
- 错误日志仍受 `ERROR_LOG_ENABLED` 和原有错误记录策略控制，并非所有本地失败都会强制记录。
- 这次不改变计费、状态码映射或原有错误消息处理规则。

## 三、流状态筛选问题

### 3.1 原因与日志语义

HTTP 请求成功进入流式回复后，客户端仍可能中途断开，或者流出现软错误。这种请求可能已经产生 token 和消耗，所以仍是消费日志，而不是普通“错误日志”类型。

[service/log_info_generate.go](../service/log_info_generate.go) 的 `appendStreamStatus` 根据流是否正常结束、有无错误，生成 `ok` 或 `error`，详细信息保存在：

```json
{
  "stream_status": {
    "status": "error",
    "end_reason": "client_gone"
  }
}
```

现有耗时列依据 `is_stream` 和 `other.stream_status` 显示流错误提示。因此仅选择日志类型“错误”，无法找到所有这类已计费的流异常请求。

### 3.2 最终数据库字段

[model/log.go](../model/log.go) 的 `Log` 增加：

| 项目 | 说明 |
| --- | --- |
| Go 字段 | `StreamStatus string` |
| 数据库列 | `logs.stream_status` |
| 关系数据库类型 | `varchar(16)` |
| 已知值 | `ok`、`error`；未知或无状态时为空 |
| HTTP JSON 输出 | `json:"-"`，不直接暴露新列 |

非流日志不保存流状态。历史关系数据库记录新增列后可能是 NULL，均不会匹配 `stream_status = 'error'`。

详细诊断仍保留在 `other`，例如结束原因、错误条数、错误消息。普通用户日志原有的字段剥离规则保持不变，不会因为增加筛选列而公开这些诊断信息。

### 3.3 写入方式

- `buildConsumeLog` 从已经解析好的 `params.Other["stream_status"]` 直接读取状态，避免正常消费写入时再次解析较大的 `other` JSON。
- GORM `Log.BeforeCreate` 为其他单条、批量创建路径补齐状态。
- 钩子对非流日志清空状态；已有非空状态时不重复提取。
- 状态缺失或 JSON 无法解析时，不因提取状态而阻止日志写入。
- `createLog`、`RecordBatchLog`、`RecordConsumeLogBatch` 等通过 GORM 创建日志的路径可使用该钩子；绕过 GORM 钩子的直接 SQL 写入需自行同步状态。

### 3.4 查询方式

最初的实现曾通过不同数据库的 JSON 函数解析 `other`。因查询性能风险，该实现已移除，最终条件为：

```sql
logs.is_stream = true AND logs.stream_status = 'error'
```

不再在查询阶段解析 `other`，也没有“独立列为空时回退查询 JSON”的逻辑。

以下查询统一复用 `applyStreamErrorLogFilter`：

- `GetAllLogs`：管理员列表和分页总数。
- `GetUserLogs`：用户自己的列表和分页总数，保留原用户隔离条件。
- `SumUsedQuota`：消耗额度及 RPM、TPM 查询。

筛选不改变日志类型，可以同时选择“消耗”和“流状态：错误”。统计中的消耗额度仍按原有消费日志规则计算，RPM/TPM 保持原有最近一分钟统计口径，本次只增加流错误条件。

### 3.5 筛选区和接口参数

桌面、移动筛选区新增“流状态：全部 / 错误”。选择错误后点击查询，会进入以下链路：

```text
筛选草稿 streamError='1'
  -> URL 搜索参数 streamError='1'
  -> API 查询参数 stream_error=1
  -> Controller 解析为 streamErrorOnly
  -> Model 查询独立列
```

选择全部或重置时移除条件；筛选状态参与草稿同步、激活状态和移动筛选计数，并通过 URL 保存。

涉及前端文件：

- [common-logs-filter-bar.tsx](../web/default/src/features/usage-logs/components/common-logs-filter-bar.tsx)
- [lib/filter.ts](../web/default/src/features/usage-logs/lib/filter.ts)
- [lib/utils.ts](../web/default/src/features/usage-logs/lib/utils.ts)
- [types.ts](../web/default/src/features/usage-logs/types.ts)
- [日志路由](../web/default/src/routes/_authenticated/usage-logs/$section.tsx)

[controller/log.go](../controller/log.go) 的管理员、用户日志接口及统计接口均识别 `stream_error=1`。未传参数时保持原有查询行为。页面复用现有 `Stream Status`、`All`、`Error` 翻译，本次没有新增翻译键。

### 3.6 索引与数据库迁移

SQLite、MySQL、PostgreSQL 通过现有日志库 `AutoMigrate` 创建列与联合索引：

| 索引 | 列顺序 |
| --- | --- |
| `idx_logs_stream_status_created_at` | `(stream_status, created_at)` |
| `idx_logs_user_stream_status_created_at` | `(user_id, stream_status, created_at)` |

[model/main.go](../model/main.go) 中的 ClickHouse 迁移同时处理新表与已存在的表：

- 新增 `stream_status LowCardinality(String) DEFAULT ''`。
- 新增 `idx_logs_stream_status`，类型为 `set(3)`，`GRANULARITY 1`。
- 已有表使用 `ADD COLUMN IF NOT EXISTS`、`ADD INDEX IF NOT EXISTS`。
- 沿用已有时间分区和排序键，不重建表，不执行 `MATERIALIZE INDEX`。

ClickHouse 跳数索引不是关系数据库的 B-tree 联合索引，其效果取决于错误在 granule 中的分布，不能承诺同等查询性能。

### 3.7 性能与历史数据限制

独立字段避免了查询时逐条解析 JSON，但不能保证查询和部署“零性能影响”：

- 数据库是否采用索引取决于执行计划、查询范围和数据分布。
- 大范围 `COUNT` 和统计仍需处理大量匹配记录，应保留合理的时间范围。
- 首次加列、建索引可能扫描历史数据、产生 I/O 和锁等待，具体取决于数据库版本和表规模。
- 大日志库需在维护窗口迁移，或先按部署规范执行在线 DDL；代码没有提供通用的零停机索引创建方案。
- 没有启动时自动回填、历史整表更新或 JSON 查询回退。
- 新列为空的历史日志不会进入流错误筛选，即使其 `other` 中曾有错误状态。
- 若需要历史筛选，需另行设计由运维控制的限速、分批回填，本次没有执行或实现自动回填任务。

## 四、一次性断流脚本

脚本：[test/test_stream_disconnect.sh](../test/test_stream_disconnect.sh)。

从仓库根目录执行：

```bash
bash test/test_stream_disconnect.sh
```

脚本使用本地 `http://localhost:3000/v1/chat/completions` 和 `MiniMax-M2.5-AWQ`，从现有测试脚本读取 Authorization，不额外复制保存密钥。请求设置 `stream=true`、`max_tokens=4096`，要求模型输出长回复。

`curl -N -i` 输出响应头和 SSE 数据，管道后的 `head -n 60` 收到 60 行后退出；curl 后续写入关闭的管道时退出，模拟客户端中途断开。出现 `curl: (23)` 写入错误属于此测试的预期现象。

注意：60 行包含响应头，不等于 60 个 token。脚本设置 60 秒总超时；若接口报错、模型提前完成、或未收到回复就超时，不能据此认定已经验证了中途断流。

人工验证步骤：

1. 确认运行中的后端已经更新，并完成独立字段和索引迁移。
2. 执行脚本，确认已看到部分流式输出，且不是完整回复自然结束。
3. 从响应头记下网关请求 ID 或上游请求 ID。
4. 等待后端结束处理并写入日志。
5. 在使用日志中选择适当时间范围、“消耗”和“流状态：错误”，点击查询。
6. 找到对应请求并查看流结束原因；实际原因可能受断开时机和供应商行为影响，不能保证每次都恰好是 `client_gone`。

该脚本没有替用户发送过实际模型请求；它可能产生真实消耗，需人工执行。

## 五、验证记录与尚未验证项

| 项目 | 已完成情况 |
| --- | --- |
| 上游 ID 改动 | 当时已运行模型层、请求层和错误处理相关 Go 回归测试，覆盖 200/400/429/500 响应头、错误体 ID、优先级和错误日志保存 |
| 上游 ID 详情兼容 | 修改文件的精确 lint 通过；全量类型检查存在其他模块的错误 |
| 流筛选的初始 JSON 查询版 | 当时已运行 SQLite 回归，验证消费日志匹配、分页计数、用户范围和统计条件 |
| 最终独立字段版 | 按用户要求未运行测试；已完成 Go 文件格式化、补丁空白检查及编辑器静态诊断 |
| 最终独立字段回归用例 | 已更新，但未执行；覆盖写入后查询不依赖 `other` 内容、单条消费筛选及批量状态保存 |
| 前端筛选接入 | 当时类型检查没有新增报错，但全量检查仍有 13 个现有错误；涉及的查询工具文件另有既存的 API 循环导入 lint 问题，未作无关重构 |
| 断流脚本 | `bash -n` 语法检查通过，未发送实际请求 |
| 浏览器交互 | 开发页面启动过，但停留在登录页，未完成登录后的端到端筛选验证 |
| 数据库迁移和性能 | 未执行生产迁移、历史回填、最终独立字段版的跨数据库实测或执行计划验证 |

相关回归用例保存在 [model/log_format_test.go](../model/log_format_test.go)、[service/error_test.go](../service/error_test.go) 和 [relay/channel/api_request_test.go](../relay/channel/api_request_test.go)。部署后的实际断流效果、日志筛选和迁移耗时仍需在目标环境验证。