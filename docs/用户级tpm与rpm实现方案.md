# 用户级 TPM/RPM 限制实现方案

## 目标

为 API Relay 请求提供按分钟统计的请求数限制（RPM）和 Token 数限制（TPM）。管理员可在“系统设置 -> 安全与限制 -> RPM/TPM 限制”中，为以下四种范围配置规则：

1. 全局默认规则。
2. 用户组规则。
3. 指定用户规则。
4. 指定用户 + 指定模型规则。

任一请求命中限制时，系统在转发上游之前直接返回 HTTP 429。

## 规则语义

- `RPM`：最近 60 秒内允许的最大请求数量。
- `TPM`：最近 60 秒内允许的最大 Token 数量。
- `0`：对应维度不限制。
- RPM 与 TPM 同时配置时，必须同时满足；任一超限即拒绝请求。
- 请求开始时 TPM 先使用预估 Token 预占，确保并发请求不会绕过限制。
- 消费日志入口拿到最终 usage 后，窗口内该请求的预估值会替换为同一份实际 `prompt_tokens + completion_tokens`；不需要反查日志表。
- 当上游不返回 usage 时，既有日志链路会回退到预估输入 Token；此时 TPM 结算值也相应保持该回退值。
- 采用滑动时间窗口，而不是按自然分钟清零，避免分钟边界突发请求绕过限制。

## 匹配优先级

每次请求只匹配一条最具体的规则，优先级如下：

```text
用户 + 模型 > 用户 > 用户组 > 全局默认
```

示例：

- 已配置全局 `RPM=100`。
- 用户组 `premium` 配置 `RPM=200`。
- 用户 `alice` 配置 `RPM=50`。
- 用户 `alice` 使用模型 `gpt-test` 配置 `RPM=10`。

当 `alice` 请求 `gpt-test` 时使用 `RPM=10`；请求其他模型时使用 `RPM=50`。其他 `premium` 用户使用 `RPM=200`，其他用户使用全局规则。

### 覆盖而非叠加

四种范围是“选择最具体规则”的覆盖关系，并不是所有命中规则同时生效的叠加关系。每个请求仅会使用一条规则，且 RPM、TPM 都从这条规则读取。

这意味着全局默认规则不是不可突破的硬上限；管理员可以通过更具体的用户组、用户或用户+模型规则放宽或收紧限制。

| 已配置规则 | 请求对象 | 最终生效规则 | 结果 |
| --- | --- | --- | --- |
| 全局 `RPM=500` | 任意用户、任意模型 | 全局 | 最多 500 RPM |
| 全局 `RPM=500`；用户组 `premium` `RPM=300` | `premium` 用户 | 用户组 | 最多 300 RPM |
| 全局 `RPM=500`；用户组 `premium` `RPM=300` | 非 `premium` 用户 | 全局 | 最多 500 RPM |
| 全局 `RPM=500`；用户 `alice` `RPM=600` | `alice` 任意模型 | 用户 | 最多 600 RPM，不再同时受 500 RPM 限制 |
| 用户组 `premium` `RPM=300`；用户 `alice` `RPM=600` | 属于 `premium` 的 `alice` | 用户 | 最多 600 RPM |
| 用户 `alice` `RPM=600`；用户+模型 `alice/gpt-test` `RPM=100` | `alice` 请求 `gpt-test` | 用户+模型 | 最多 100 RPM |
| 用户 `alice` `RPM=600`；用户+模型 `alice/gpt-test` `RPM=100` | `alice` 请求其他模型 | 用户 | 最多 600 RPM |

TPM 的规则选择完全相同。例如：

| 已配置规则 | 请求对象 | 最终 TPM |
| --- | --- | --- |
| 全局 `TPM=100000`；用户组 `premium` `TPM=200000` | `premium` 用户 | `200000` |
| 全局 `TPM=100000`；用户 `alice` `TPM=50000` | `alice` | `50000` |
| 用户 `alice` `TPM=50000`；用户+模型 `alice/gpt-test` `TPM=10000` | `alice` 请求 `gpt-test` | `10000` |
| 用户 `alice` `TPM=50000`；用户+模型 `alice/gpt-test` `TPM=0` | `alice` 请求 `gpt-test` | 不限制 TPM；`0` 覆盖用户规则的 `50000` |

因此，若期望“全局限制永远不能被放宽”，当前实现不满足该语义，需要改为检查所有命中规则并取更严格的限制，而不是只选择最具体规则。

### RPM 与 TPM 同时配置的示例

一条生效规则中的 RPM 与 TPM 独立判断，任一维度超限都会返回 HTTP 429：

| 生效规则 | 最近 60 秒已使用 | 新请求预估 Token | 结果 | 记录的 `limit_type` |
| --- | --- | --- | --- | --- |
| `RPM=5`，`TPM=0` | 5 次请求 | 10 | 拒绝 | `rpm` |
| `RPM=0`，`TPM=3000` | 2 次请求，累计 2500 Token | 600 | 拒绝 | `tpm` |
| `RPM=5`，`TPM=3000` | 4 次请求，累计 2900 Token | 50 | 允许，第 5 次请求进入窗口 | 无 |
| `RPM=5`，`TPM=3000` | 4 次请求，累计 2900 Token | 200 | 拒绝 | `tpm` |
| `RPM=5`，`TPM=3000` | 5 次请求，累计 100 Token | 10 | 拒绝 | `rpm` |
| `RPM=0`，`TPM=0` | 任意 | 任意 | 允许 | 无 |

若 RPM 与 TPM 都会超限，当前检查顺序是先检查 RPM，因此日志中的 `limit_type` 为 `rpm`。

## 后端数据模型

新增 `UserRateLimit` 表，由 GORM 自动迁移：

| 字段 | 说明 |
| --- | --- |
| `scope` | `global`、`group`、`user`、`user_model` 之一 |
| `user_id` / `username` | 用户范围与展示信息；全局、用户组规则为零值 |
| `group_name` | 用户组规则的组名 |
| `model_name` | 用户 + 模型规则的模型名 |
| `rpm` | 每分钟请求上限，0 为不限 |
| `tpm` | 每分钟 Token 上限，0 为不限 |

规则使用 `(scope, user_id, group_name, model_name)` 唯一索引，保证同一范围只能存在一条规则。创建相同范围时更新既有规则；编辑时会检查与其他记录冲突。

相关文件：

- `model/user_rate_limit.go`
- `model/main.go`
- `controller/user_rate_limit.go`
- `router/api-router.go`

## 管理接口

接口均要求管理员权限：

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/api/user_rate_limit` | 分页列出规则，支持 `keyword` 搜索 |
| `POST` | `/api/user_rate_limit` | 创建或按范围更新规则 |
| `PUT` | `/api/user_rate_limit/:id` | 编辑规则 |
| `DELETE` | `/api/user_rate_limit/:id` | 删除规则 |

请求示例：

```json
{
	"scope": "user_model",
	"user_id": 1001,
	"model_name": "gpt-test",
	"rpm": 30,
	"tpm": 60000
}
```

用户组规则示例：

```json
{
	"scope": "group",
	"group_name": "premium",
	"rpm": 100,
	"tpm": 200000
}
```

## Relay 请求链路

限流接入点位于 `controller/relay.go`：

1. 解析并校验客户端请求。
2. 生成 `RelayInfo`，得到原始模型名称。
3. 估算本请求 Token 数量。
4. 按“用户 + 模型、用户、用户组、全局”顺序查找有效规则。
5. 在 60 秒滑动窗口中原子检查并预占本次请求与预估 Token。
6. 未通过时构造 OpenAI 兼容错误并返回 HTTP 429，不再执行后续计费和上游转发。
7. 通过时继续执行原有模型用量限制、计费与上游 Relay 流程。
8. 消费日志入口拿到实际 usage 后，用实际 `prompt_tokens + completion_tokens` 原子修正该请求在 TPM 窗口中的预占量；此步骤不依赖日志开关或日志落库成功。

用户组读取请求上下文中的 `ContextKeyUserGroup`，即用户账户所属组；不受 API Key 临时分组或渠道分组影响。

## 限流存储与并发安全

### Redis 已启用

使用单个 Lua 脚本完成以下原子操作：

1. 清理当前 key 中超过 60 秒的请求记录。
2. 检查当前 RPM。
3. 汇总窗口内 Token 并检查 TPM。
4. 将本次 `时间戳:请求ID:预估Token数` 写入窗口。
5. 设置 120 秒过期时间。

Lua 脚本在 Redis 中单线程执行，因此并发请求不能绕过 RPM/TPM 阈值，多实例部署共享同一限流状态。

日志结算时会按请求 ID 定位窗口记录，并通过第二个 Lua 脚本将预估 Token 替换为实际 Token。这个替换同样是原子操作，不影响同一用户的并发请求计数。

### Redis 未启用

使用进程内 map 与互斥锁保存每个规则、每个用户的滑动窗口记录，并以请求 ID 在日志结算时替换预估值。该模式在单实例下保持并发安全；多实例且未启用 Redis 时，实例间不能共享计数，因此生产集群应启用 Redis。

限流 key 格式：

```text
userRateLimit:{scope}:{ruleID}:{userID}
```

即使命中的是全局或用户组规则，计数也按用户隔离；“全局”和“用户组”表示规则适用范围，不表示该范围内所有用户共享一个总配额。

## 管理页面

页面实现位于：

- `web/default/src/features/system-settings/request-limits/user-rate-limit-section.tsx`
- `web/default/src/features/system-settings/request-limits/user-rate-limit-api.ts`
- `web/default/src/features/system-settings/security/section-registry.tsx`

页面提供单行快速配置表单：

- “应用范围”下拉选择全局默认、用户组、用户、用户与模型。
- 根据范围显示用户组、用户或模型选择器。
- 可直接填写 RPM、TPM。
- 下方表格集中展示、编辑和删除已有规则。
- 用户、用户组、模型选项分别读取既有管理 API，避免手动输入 ID 或名称。

所有新增文案已通过 `web/default/scripts/add-missing-keys.mjs` 写入六种前端语言，并执行 `bun run i18n:sync`。

## 错误响应

超过限制时返回 HTTP 429，错误消息为：

```text
请求频率或 token 使用量超过限制
```

错误带有 `skip retry` 标记，Relay 不会将其当作上游失败而重试其他渠道。

## 验证

已添加优先级回归测试：

- `model/user_rate_limit_test.go`

该测试验证在依次删除更具体规则后，生效规则按“用户+模型 -> 用户 -> 用户组 -> 全局”的顺序回退。

已执行验证：

```bash
go test ./model -run TestGetEffectiveUserRateLimitPrefersSpecificScope
./build.sh
```

两项均通过；`./build.sh` 包含 Go 后端与默认前端生产构建。
