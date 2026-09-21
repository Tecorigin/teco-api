# Redis 滑动时间窗口实现 RPM 及 TPM 限制

## 1. 适用范围与目标

当前环境已启用 Redis，因此 RPM/TPM 限制使用 Redis 作为唯一的窗口状态存储。所有应用实例访问同一个 Redis 后，可以共享限流状态，避免多实例分别计数导致用户绕过限额。

实现目标：

1. 对每个用户的生效 RPM/TPM 规则执行最近 60 秒的滑动窗口检查。
2. RPM 统计窗口内请求条数。
3. TPM 统计窗口内 Token 总数。
4. 请求进入时使用预估 Token 原子预占，避免并发请求绕过 TPM。
5. 请求完成时，用使用日志同源的实际 usage 替换预估值。
6. 任何命中都在上游渠道选择与转发前直接返回 HTTP 429。

相关代码：

- `model/user_rate_limit.go`：Redis Lua 脚本、规则选择、预占和结算。
- `controller/relay.go`：Relay 请求前检查与 429 阻断日志。
- `model/log.go`：消费日志入口中的实际 Token 结算。

## 2. 规则选择与窗口隔离

一个请求只使用最具体的一条规则：

```text
用户 + 模型 > 用户 > 用户组 > 全局默认
```

这里的“全局默认”是规则的适用范围，不代表所有用户共享一个全局桶。无论规则来自哪个范围，窗口计数都按用户隔离。

Redis key 格式：

```text
userRateLimit:{scope}:{ruleID}:{userID}
```

示例：

```text
userRateLimit:global:1:1001
userRateLimit:group:3:1001
userRateLimit:user:8:1001
userRateLimit:user_model:12:1001
```

含义：

- `scope`：实际命中的规则范围，取值为 `global`、`group`、`user`、`user_model`。
- `ruleID`：数据库 `user_rate_limits.id`，规则修改、删除并新建后会使用新 key，不会继承旧规则的窗口计数。
- `userID`：请求认证得到的用户 ID；同一个全局或用户组规则也为不同用户分别建桶。

## 3. Redis 数据结构

使用 Redis `LIST` 保存窗口内的请求记录，列表从左到右按写入顺序排列：

```text
时间戳毫秒:请求ID:Token数
```

例子：

```text
1726905600000:req-a:852
1726905612000:req-b:900
1726905625000:req-c:780
```

字段含义：

| 字段 | 作用 |
| --- | --- |
| 时间戳 | 判断记录是否仍属于最近 60 秒窗口 |
| 请求 ID | 请求完成时定位预占记录，并将预估 Token 替换成实际 Token |
| Token 数 | 请求开始时为预估值；请求结束后改为实际 `prompt_tokens + completion_tokens` |

列表严格按追加时间排列，因此过期记录总在左侧，可以通过 `LPOP` 逐条清理。

## 4. 滑动窗口定义

当前窗口为请求执行时刻向前推 60 秒：

$$
	ext{cutoff} = \text{now}_{ms} - 60000
$$

每次新请求到来时，移除时间戳不晚于 `cutoff` 的记录；剩余记录就是最近 60 秒的有效窗口。

这不是自然分钟窗口。例如：

- 用户在 `12:00:50` 到 `12:00:55` 发出 5 个请求，且 `RPM=5`。
- 第 6 个请求在 `12:00:56` 被拒绝。
- 到 `12:01:00` 时，这 5 个请求仍然在最近 60 秒内，不会立刻恢复额度。
- 最早一条请求到 `12:01:50` 离开窗口后，才恢复一个请求额度。

这样可避免自然分钟方式在 `12:00:59` 和 `12:01:00` 两侧产生瞬时双倍流量。

## 5. 请求前原子预占

Relay 在解析请求、生成 `RelayInfo`、计算预估 Token 后调用限流检查。Redis 使用 `EVAL` 执行一段 Lua 脚本；脚本在 Redis 内单线程运行，清理、检查和写入不可被其他请求插入。

输入参数：

| 参数 | 含义 |
| --- | --- |
| `KEYS[1]` | 当前用户、当前规则对应的窗口 key |
| `ARGV[1]` | 应用服务器当前毫秒时间戳 |
| `ARGV[2]` | 当前规则 RPM，0 表示不限制 |
| `ARGV[3]` | 当前规则 TPM，0 表示不限制 |
| `ARGV[4]` | 本请求预估 Token |
| `ARGV[5]` | 本请求 request ID |

脚本执行顺序：

1. 计算 `cutoff = now - 60000`。
2. 读取列表第一个元素；若已过期，循环 `LPOP`，直至列表为空或第一个元素仍有效。
3. `LLEN` 获取窗口请求数。
4. 若 `rpm > 0` 且 `count >= rpm`，返回 `2`，拒绝原因是 `rpm`。
5. 若 `tpm > 0`，使用 `LRANGE` 读取有效窗口记录，解析末尾 Token 并累加。
6. 若 `totalTokens + estimatedTokens > tpm`，返回 `3`，拒绝原因是 `tpm`。
7. 使用 `RPUSH` 写入 `timestamp:requestID:estimatedTokens`。
8. 设置 `PEXPIRE 120000`，返回 `1`，允许请求。

返回码约定：

| Lua 返回值 | Go 层结果 | Relay 行为 |
| --- | --- | --- |
| `1` | 允许且返回预占句柄 | 继续计费、选渠道、转发 |
| `2` | `limit_type=rpm` | 返回 429，写入 RPM 阻断日志 |
| `3` | `limit_type=tpm` | 返回 429，写入 TPM 阻断日志 |

RPM 先于 TPM 检查。因此两个维度同时超限时，返回和记录的原因是 `rpm`。

## 6. TPM 预估值与实际值结算

### 6.1 为什么需要预占

上游模型的实际 usage 只有在响应完成后才能得到。若在请求开始时不预占，多个并发请求可以同时看到剩余额度充足并全部放行，最终超过 TPM。

因此请求开始时先使用：

```go
service.EstimateRequestToken(c, meta, relayInfo)
```

计算得到的预估值写入窗口。此步骤是准入控制，必须在上游调用前完成。

### 6.2 实际 usage 修正

请求成功或完成后，现有消费日志链路会生成：

```text
actualTokens = prompt_tokens + completion_tokens
```

`RecordConsumeLog` 在日志开关和日志落库之前，读取当前请求的预占句柄，调用第二段 Redis Lua 脚本：

1. `LRANGE` 读取该 key 的记录。
2. 根据 request ID 找到本请求记录。
3. 用 `LSET` 将 `时间戳:requestID:预估Token` 改为 `时间戳:requestID:实际Token`。
4. 找不到记录时不报错，例如记录已在 60 秒后过期。

这个操作同样使用 `EVAL`，所以替换与其他请求的预占不会发生竞态。

### 6.3 上游不返回 usage 的情况

若上游协议未返回可靠 usage，既有消费日志逻辑会回退到可用的预估输入 Token。此时窗口最终保留回退值，而不是虚构的“实际值”。

## 7. 429 响应与错误日志

当 RPM 或 TPM 被拒绝：

1. 不会选择渠道，不会向上游发起请求，也不会产生扣费。
2. Relay 返回 OpenAI 兼容 HTTP 429。
3. 错误日志写入一条本地阻断记录，日志类型为 `LogTypeError`。

本地 RPM/TPM 限流属于管理员显式配置的审计事件，会直接调用 `RecordErrorLog` 写入 `logs` 表，不受 `ERROR_LOG_ENABLED` 影响。即使该环境变量为 `false`，也会持久化 `type=5` 的错误日志。

阻断日志内容示例：

```text
RPM 限制，已在本地直接拦截并返回 429
```

或：

```text
TPM 限制，已在本地直接拦截并返回 429
```

日志 `other` 中包含：

```json
{
	"limit_blocked": true,
	"limit_type": "rpm",
	"rate_limit_scope": "user_model",
	"rpm_limit": 5,
	"tpm_limit": 3000,
	"estimated_tokens": 852
}
```

`limit_type` 为 `rpm` 或 `tpm`，可用来区分具体拒绝原因。

限流发生在渠道选择之前，因此阻断日志的 `channel_id` 可能为 `0`，这是正常现象。

## 8. 过期策略与内存成本

每次成功预占后执行：

```text
PEXPIRE key 120000
```

含义：用户在最后一次成功请求后的 120 秒内未再请求，整个窗口 key 会自动删除。TTL 比窗口长 60 秒，用于保证窗口状态在完整 60 秒内存在，同时避免长期闲置 key 堆积。

容量近似由窗口内请求数决定：

$$
	ext{记录数} \le \text{生效 RPM}
$$

记录包含 request ID，因此高 RPM 配置会增加 Redis LIST 长度。TPM 检查当前会对窗口执行 `LRANGE` 并逐条累加，单次检查复杂度为：

$$
O(n)
$$

其中 $n$ 是当前 60 秒窗口中的请求数。对于普通用户级 RPM 限制该成本可接受；不建议将单个用户的 RPM 配置得过高，例如几十万以上。若需要极高吞吐，应改用聚合计数或分桶时间窗口方案。

## 9. Redis 故障语义

若 Redis 调用或 Lua 执行失败，Go 层返回限流检查错误，Relay 以内部错误结束请求；不会在 Redis 不可用时静默放行受限请求。这是 fail-closed 策略，用于防止 Redis 短暂故障使高频请求绕过配置的限额。

运维上应监控：

- Redis 连通性与命令超时。
- Redis CPU，Lua 脚本执行时间和慢日志。
- `used_memory`、key 数与过期 key 清理情况。
- 应用日志中的 `rate limit check failed`、`rate limit settlement failed`。

## 10. 多实例与时间语义

多个应用实例通过同一个 Redis key 竞争同一个窗口；Redis Lua 保证同一 key 的检查和写入串行，因此不存在多实例并发放行的漏洞。

当前预占脚本传入的是应用服务器 `time.Now().UnixMilli()`，不是 Redis `TIME`。部署要求所有应用实例使用 NTP 保持时钟同步；若实例间时钟偏差明显，窗口清理可能提前或滞后。生产环境建议监控时间同步状态。

## 11. 排障与检查命令

### 11.1 查找限流 key

生产环境避免使用阻塞式 `KEYS`，使用 `SCAN`：

```bash
redis-cli --scan --pattern 'userRateLimit:*'
```

### 11.2 查看某个用户的窗口

```bash
redis-cli LRANGE 'userRateLimit:user:8:1001' 0 -1
redis-cli PTTL 'userRateLimit:user:8:1001'
```

示例输出：

```text
1726905600000:req-a:852
1726905612000:req-b:900
```

### 11.3 清理某用户窗口用于测试

```bash
redis-cli DEL 'userRateLimit:user:8:1001'
```

删除窗口只会重置当前 60 秒的 RPM/TPM 计数，不会删除数据库中的规则配置。

### 11.4 脚本测试

仓库提供：

```bash
./test_rate_limit.sh rpm
./test_rate_limit.sh tpm
```

测试 TPM 时应将该规则的 RPM 设置为 `0`，避免 RPM 先触发 429。TPM 模式会连续发送请求直到窗口累计超过 TPM。

## 12. 已知边界与后续优化

1. 当前 TPM 预占使用估算值，在请求完成前可能与最终实际 usage 存在偏差；结算后会修正，但并发中的短暂窗口仍以预估值为准。
2. 非正常中断且没有进入消费日志的请求会保留预估值，最长影响到窗口记录过期。
3. 当前列表记录格式以 `:` 分隔，request ID 需保持不含 `:`；项目生成的 request ID 满足此要求。
4. 规则更新或新建后若规则 ID 变化，会自然使用新 key；旧 key 会在 TTL 到期后自动清理。
5. 当前只有“最具体规则覆盖”语义；如果需要全局和用户规则同时生效，需要调整为对所有命中规则逐一预占和结算。
