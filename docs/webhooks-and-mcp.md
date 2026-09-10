# Webhook 与 MCP 功能方案

> 状态：已交付首批实现（Webhook、MCP 查询/任务、电影文件审批）；未实现的扩展仍保留为设计目标。实际配置、工具清单与限制见 [集成使用说明](integrations.md)。范围限定为单进程、SQLite WAL 部署，不引入消息中间件或外部 Worker。

### 当前实现边界

- 已实现 A/B：作业 succeeded/failed 与 Outbox 同事务提交、独立投递队列、密钥轮换、网络策略、管理 UI。新增文件应用事件仅来自已具备恢复记录的自动化计划；旧写入入口不补发 applied 事件。
- 已实现 C：官方 Go SDK v1.6.1、`2025-11-25` 协议基线、按来源过滤的查询、Token 管理、限流和调用审计。
- 已实现 D：扫描、电影候选搜索、电影/TV Artwork 候选抓取、TV 季/集数据库元数据更新、任务取消和安全重试，均使用持久化作业。
- 已实现 E 的电影切片：NFO、Artwork 和逐文件重命名计划，Web 管理员审批、幂等入队、根目录约束、共享执行锁、逐项结果和启动核对。TV 批量文件审批、整体目录重命名及跨文件回滚未开放。
- OAuth、stdio、资源订阅、复杂事件过滤和额外查询工具仍为后续能力；测试现状和已知限制记录在使用说明中。

## 1. 目标与关键决策

Webhook 用于将已提交的业务结果通知外部系统；MCP 用于让受限客户端查询媒体库，并逐步开放后台任务和经过人工批准的文件操作。两者复用应用服务、权限检查与安全写入流程。

```text
Web UI / REST / MCP → 应用服务 → Jobs / Metadata / Library / Plans
                          ↓ 同一数据库事务
                    业务状态 + Outbox
                                ↓
                     Webhook fan-out → delivery worker → 外部接收方
```

| 决策 | 本方案约定 |
| --- | --- |
| 发布范围 | 首个闭环只包含任务成功/失败通知和 4 个 MCP 查询工具 |
| 投递语义 | 至少一次尝试，可能重复、乱序；有限重试后进入 dead，不承诺最终必达或 exactly-once |
| 事务边界 | 数据库业务状态与事件原子提交；文件系统操作不能与 SQLite 组成原子事务 |
| 调度方式 | 进程内独立、持久化的 Webhook 投递队列，隔离媒体作业容量 |
| MCP 认证 | 首期静态 API Token，仅面向支持自定义 Bearer Header 的客户端；标准 OAuth 互操作单独交付 |
| 文件批准 | preview 不签发可直接执行的凭证；管理员在 Web UI 审批具体计划后才允许 apply |
| 默认权限 | 只读、按来源限定；MCP 不开放配置管理和任意文件访问 |
| 首期排除 | 入站 Webhook、事件表达式、批量投递、MCP prompts、stdio、媒体删除、自动批准写入 |

入站 Webhook 本质上是另一套命令 API，有需求时先评估现有 REST/MCP，避免重复授权和业务逻辑。

## 2. 仓库基线与接入边界

实施前以 [架构](architecture.md)、[Phase 5A](phase-5a-operations.md)、[重命名](phase-5b-rename.md) 和 [TV NFO 计划](tv-nfo-write-plans.md) 为基线；同时遵循根目录两份 README 的单容器与文件安全约束。

实施前的代码基线（已在本轮改动中演进）：

- `internal/jobs/jobs.go` 原有状态更新与 `recordEvent` 分开执行；本轮为成功/最终失败新增终态事务，同时提交状态、SSE 记录和外部事件。普通进度消息仍沿用现有 SSE 记录机制。
- `internal/library/service.go` 暴露媒体、来源、任务查询与任务入口；对外 DTO 必须另行投影，不能直接返回含 `RootPath` 的来源对象。
- `internal/library/rename.go` 已有计划及作业处理器；MCP 开放写入前仍需完成计划不可变性、批准绑定、重复提交和故障恢复检查。
- `internal/httpapi/` 负责现有 HTTP 入口，`internal/app/` 负责服务装配与生命周期；MCP 不通过 HTTP 回调自身 REST API。
- `job_events` 继续承担现有 SSE 协议，Outbox 承担外部业务事件。不得直接转发 SSE 消息、进度文本或内部错误字符串。

Webhook/MCP 对外开放是新增边界，不能把架构文档中的目标能力当作现成实现。任何写入能力未满足本文验收条件时，不注册对应工具。

## 3. 事件契约

### 3.1 分批开放事件

| 批次 | 事件 | 精确定义 |
| --- | --- | --- |
| MVP | `job.succeeded`、`job.failed` | 作业提交成功或最终失败状态；自动重试中的暂时错误不发 failed |
| 后续 | `job.queued`、`job.started`、`job.cancelled`、`job.interrupted` | 已提交的状态转换；cancelled 表示取消已记录，不代表文件操作已回滚 |
| 后续 | `metadata.updated`、`media.created`、`media.missing` | 对应业务变化已持久化；重复扫描相同状态不重复产生事件 |
| 写入能力成熟后 | `write_plan.applied`、`artwork_plan.applied`、`rename_plan.applied` | 全部操作成功且结果已记录；partial/failed/cancelled 不发 applied |

首期不发送逐文件扫描、作业进度和连接测试事件。Webhook 自身的投递、测试、失败及维护任务不进入以上业务事件集，避免失败通知递归放大。

### 3.2 Envelope

```json
{
  "id": "evt_123",
  "schemaVersion": 1,
  "type": "job.succeeded",
  "occurredAt": "2026-09-09T12:00:00Z",
  "aggregate": { "type": "job", "id": "42", "version": 3 },
  "data": { "jobId": 42, "kind": "scan", "sourceId": 7, "state": "succeeded" }
}
```

- ID 全局唯一；aggregate version 为该对象已提交事件的单调版本，可识别乱序。作业重试后再次完成产生新事件，不按 `jobId + eventType` 永久去重。
- Envelope 和 data 使用固定 DTO 白名单；不包含凭证、完整路径、Provider 原始响应、作业原始 payload 或未脱敏错误文本。
- 字段新增保持兼容；删除字段、改变含义或类型需要新 schemaVersion。接收方应忽略未知字段。
- 单事件上限建议 64 KiB；大结果仅放摘要、计数及对象 ID，授权客户端另行查询详情。
- 事件一经写入不可修改；重投保持同一 ID 和完全一致的 body 字节。

## 4. Outbox、一致性与故障恢复

### 4.1 提交与分发

1. 应用服务在短事务内执行受预期状态约束的业务更新，递增事件版本，并插入 `system_events`。Outbox 写入失败则整个事务回滚；只有实际状态变化才生成事件。
2. 发布器按未分发索引批量读取事件，在事务内匹配当前启用且创建时间不晚于事件的订阅，创建 delivery，并设置 `dispatched_at`。
3. `(endpoint_id, event_id)` 唯一约束防止重复 fan-out。无订阅的事件同样标记已分发；新增订阅不回补历史。
4. `dispatched_at` 仅表示已完成分发，与 HTTP 投递成功无关。投递状态由 delivery 独立维护。
5. Worker 通过条件更新领取到期 delivery，写入 lease；事务外发送 HTTP，再按 lease owner 回写结果。进程退出后，过期 lease 可重新领取。

首期明确采用“分发时的订阅配置”，不承诺事件发生时的订阅快照。delivery 保存目标配置版本；修改 URL 或过滤规则只影响后续 fan-out，已排队记录保持原目标。暂停/删除立即阻止后续领取，队列记录转为 cancelled；已经发出的请求无法撤回。恢复启用后只接收新分发记录，取消的投递须显式重试。

### 4.2 必须覆盖的崩溃窗口

| 崩溃位置 | 恢复行为 |
| --- | --- |
| 业务提交前 | 状态与事件一起回滚 |
| 业务提交后、分发前 | 重启继续扫描 Outbox |
| 分发事务中 | delivery 与 dispatched_at 一起回滚或提交 |
| 接收方已处理、发送方未记成功 | lease 过期后重发相同事件；接收方负责业务幂等 |
| 文件已修改、数据库结果未提交 | 标记需核对，不盲目重做文件操作；核对计划逐项结果后再提交最终状态与事件 |

文件操作执行过程中持久化操作意图、前置条件及逐项结果，恢复时区分已完成、未执行和无法判定。无法判定进入人工处理，不伪造 applied，也不承诺跨文件回滚。安全写入相关事件必须等该恢复闭环验收后再开放。

## 5. Webhook 持久化模型

以下为逻辑字段，实际 SQL 遵循现有迁移和 Repository 风格。

| 表 | 主要字段与约束 |
| --- | --- |
| `system_events` | id、event_type、schema_version、aggregate_type/id/version、payload_bytes、occurred_at、dispatched_at；聚合版本唯一约束 |
| `webhook_endpoints` | id、name、url、enabled、event_types_json、source_ids_json、config_version、active_key_id、created_at、updated_at、deleted_at |
| `webhook_signing_keys` | id、endpoint_id、encrypted_secret、encryption_key_version、created_at、retired_at |
| `webhook_deliveries` | id、endpoint_id、event_id、target_config_snapshot、state、attempt_count、retry_generation、next_attempt_at、lease_owner、lease_expires_at、last_status、last_error_code、created_at、delivered_at；endpoint/event 唯一 |
| `webhook_attempts` | delivery_id、attempt_number、key_id、started_at、duration_ms、status_code、error_code；delivery/attempt 唯一 |

为未分发事件、`(state, next_attempt_at)` 和过期 lease 建索引；分页读取，不全表加载。body 从不可变事件记录读取，避免每次尝试重复存储；被 delivery 引用的事件不得提前清理。

API Token 可只存哈希，HMAC 签名密钥则必须可恢复：使用独立主密钥进行认证加密，主密钥放在受限配置文件或挂载 secret 中，不放在同一数据库。备份恢复必须同时恢复主密钥；缺失或解密失败时停止投递并报告安全错误，不自动生成替代密钥。

轮换生成新 key ID，下一次尝试使用当前有效密钥；接收方在过渡期接受新旧密钥，至少保留旧密钥至最长在途超时与验签窗口结束。轮换不更改事件 body/ID。密钥明文仅创建/轮换响应展示一次，普通 GET 永不返回。

## 6. Webhook 请求、安全与调度

### 6.1 签名与接收约定

```http
POST /configured-endpoint
Content-Type: application/json
User-Agent: MediaGrap-Webhook/1
X-MediaGrap-Event: job.succeeded
X-MediaGrap-Event-ID: evt_123
X-MediaGrap-Delivery-ID: dlv_456
X-MediaGrap-Timestamp: 1788955200
X-MediaGrap-Key-ID: key_789
X-MediaGrap-Signature: sha256=<hex>
```

签名为 `HMAC-SHA256(secret, timestamp + "." + raw_body)`，按实际发送的 UTF-8 原始字节计算。每次尝试刷新秒级时间戳和签名，事件时间保持不变。

接收方先验证时间差（建议 ±5 分钟）和签名，常量时间比较；从已验证 body 读取事件 ID，校验其与 header 一致，再以事件 ID 去重。不要重新序列化 JSON 后验签。建议将去重记录与本地业务效果原子提交，或先持久化入站任务再返回 2xx。去重保留期至少覆盖允许重投的事件保留期；手动重试同样可能产生重复。

### 6.2 状态机和重试

```text
queued → delivering → succeeded
                    → retry_wait → delivering
                    → dead
queued / retry_wait → cancelled
过期 delivering lease → retry_wait
```

- 首次发送加最多 5 次自动重试，共最多 6 次；退避基准为 1 分钟、5 分钟、15 分钟、1 小时、6 小时，加随机抖动。
- 2xx 为成功；网络暂时故障、超时、408、429、5xx 可重试；其他 4xx、3xx、证书验证失败、URL 安全校验失败进入 dead。
- 对 429/503 解析有效 `Retry-After`，取其与本地退避的较晚时间，并限制最长等待为 24 小时；无效值使用本地退避。
- 每次请求总超时 10 秒，连接超时建议 3 秒；响应读取上限 8 KiB，仅记录状态、耗时和标准错误码，不保存正文。
- 全局并发默认 2、每 endpoint 并发 1；不保证事件顺序，因为失败重试可能晚于后续事件。
- 手动重试只允许 dead/cancelled，且 endpoint 当前启用、事件仍保留；保留 event/delivery ID，递增 retry_generation 并开启新的尝试预算，总尝试编号不重置。并发重试通过条件更新合并。
- 错误日志使用 endpoint/event/delivery ID、尝试次数、HTTP 状态和耗时，不记录完整 URL、响应或凭证。

### 6.3 SSRF 和内网使用

- 默认仅 HTTPS、公网目标；拒绝 URL userinfo、fragment 和非 HTTP(S) scheme。URL 可能含接收端凭证，管理响应仅显示脱敏形式，日志不输出。
- 创建、修改、测试和每次投递均执行同一 URL 策略。解析全部 A/AAAA，拒绝不允许的地址；拨号固定到已验证 IP，TLS 仍验证原域名，防 DNS rebinding。
- 禁止自动重定向；覆盖 IPv4、IPv6、映射地址、loopback、link-local、组播、未指定地址及云元数据地址。
- NAS 内网服务通过部署级显式 host/CIDR/port 白名单开放；HTTP 也需单独允许。Web 管理员不能通过 endpoint 参数解除网络策略，云元数据与 link-local 始终拒绝。
- Webhook 默认直连，不自动继承 Provider 代理。后续若支持代理，必须证明最终目的地址仍受同等策略约束，不能仅校验代理地址。

## 7. Webhook 管理与运行维护

规划 REST API：

```text
GET/POST    /api/v1/webhooks
GET/PATCH   /api/v1/webhooks/{id}
DELETE      /api/v1/webhooks/{id}
POST        /api/v1/webhooks/{id}/rotate-secret
POST        /api/v1/webhooks/{id}/test
GET         /api/v1/webhooks/{id}/deliveries
POST        /api/v1/webhooks/{id}/deliveries/{deliveryId}/retry
```

管理 API 使用现有管理员 Session，写操作继续校验 CSRF；出站投递只使用 HMAC，MCP 只使用 API Token。不得把“不复用浏览器 Session”错误应用到浏览器管理页。

- 测试发送走相同安全检查、签名与队列，使用独立 `webhook.test` 合成事件，不伪造业务成功；返回 delivery ID。
- 删除采用软删除并取消待投递记录，历史按保留策略清理。修改带配置版本，拒绝过期覆盖；重试校验 delivery 确实属于指定 endpoint。
- UI 放在设置页的集成区，遵循 [UI/UX 规范](ui-ux-design.md) 和 `stitch-prototypes/settings-console.html`；提供启停、轮换、测试、失败原因和重试，不新增主导航。
- 建议默认限制 20 个 endpoint；成功/取消历史保留 7 天，dead 保留 30 天，过期后不再允许重试。未分发/待投递记录不自动删除。
- 对积压量、最老未分发时间、dead 数、数据库增长和 Worker 心跳提供运维可见性；不以 event ID 等高基数字段作为指标标签。
- Outbox 超过软阈值告警；达到配置的硬容量预算时暂停接收新的扫描/抓取作业，继续处理已运行作业的终态，禁止静默丢事件。磁盘写满仍可能阻断状态提交，应报告不健康并在恢复后核对作业。
- 停用 Webhook 发送不删除队列，恢复后继续；应用关闭停止领取新 delivery，限时等待在途请求，剩余工作依赖 lease 恢复。

## 8. MCP 传输与兼容性

路径为 `/mcp`，与现有 `/api/v1/jobs/events` 分离。首期建议锁定 MCP `2025-11-25` 作为兼容基线，而不是笼统宣称支持最新协议；SDK 版本和目标客户端组合在实施时固定并记录。

按该版本的 [Streamable HTTP 规范](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports)，实现初始化、能力协商、POST 和协议版本校验；若不提供独立 SSE 流，GET 返回 405。首期优先无状态模式、普通 JSON 响应，不启用资源订阅和服务端推送。有 Origin 时严格校验，非法 Origin 返回 403；无 Origin 的非浏览器客户端仍须认证。反向代理须保留认证和协议头。

静态 Token 模式只服务可配置 Bearer Header 的客户端，不能称为完整 MCP OAuth 支持。[MCP 授权规范](https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization) 描述 OAuth 及受保护资源元数据发现；面向需要自动授权的客户端时，单独实现该流程，验收前不承诺兼容。非本机连接通过 HTTPS 或受信 TLS 反向代理提供。

不同协议修订不得混用生命周期规则；新增版本支持必须有独立协议测试和明确版本列表。工具的 schema、结构化结果和错误行为遵循锁定版本的 [Tools 规范](https://modelcontextprotocol.io/specification/2025-11-25/server/tools)。

## 9. MCP 权限与工具契约

### 9.1 Token 与授权

使用通用 `api_tokens` 服务，首期 audience 限定为 `mcp`，不自动赋予 REST API 权限。

```text
id、name、owner_user_id、token_prefix、token_hash、audience、scopes_json、
source_ids_json、expires_at、last_used_at、revoked_at、created_at
```

Token 使用密码学安全随机值（至少 256 bit），仅存哈希，明文创建时返回一次；支持过期、撤销和替换轮换。每次调用检查 owner 有效性及当前权限，实际权限为用户权限、Token scopes 和来源集合的交集。首期不支持部分来源可见的操作时直接拒绝，不能退回全库访问。

默认 `media:read metadata:read jobs:read`，来源范围由管理员显式选择；空来源集合不代表全库。禁止 MCP Token 管理其他 Token、审批自身请求或修改安全设置。

### 9.2 工具矩阵

| 阶段 | 工具 | Scope / 约束 |
| --- | --- | --- |
| 只读 MVP | `list_media`、`get_media` | `media:read`；仅数据库查询、脱敏 DTO |
| 只读 MVP | `get_job`、`list_jobs` | `jobs:read`；按来源和可见性过滤，不返回 payload/原始错误 |
| 查询增强 | `list_sources`、`get_tv_show` | `media:read`；不返回来源根路径 |
| 查询增强 | `list_artwork_candidates` | `metadata:read`；仅返回已保存候选 |
| 查询增强 | `get_audit_entries`、`get_system_summary` | 独立 `audit:read`、`system:read`；不默认授予 |
| 后台任务 | `scan_source` | `jobs:write`；仅扫描索引，不自动刮削或写媒体 |
| 后台任务 | `search_media_candidates`、`scrape_media_candidates`、TV 刮削工具 | `jobs:write` + `metadata:write`；外部请求/持久化候选不能列为纯查询 |
| 后台任务 | `cancel_job`、`retry_job` | `jobs:write` 且具有原任务所需权限；文件任务不允许通用 retry 绕过批准 |
| 文件预览 | `preview_write_plan`、`preview_artwork_plan`、`preview_rename_plan` | `plans:preview` + 资源读取权限；允许保存计划，不执行媒体写入 |
| 文件应用 | `apply_write_plan`、`apply_artwork_plan`、`apply_rename_plan` | `plans:apply` + 有效人工批准 + 执行前重验 |

所有读取（含 resources）按相同权限执行，不能只在 `tools/list` 隐藏名称。越权 ID 不泄露对象存在性；无来源归属的系统作业/审计记录默认拒绝普通 Token 访问。

### 9.3 输入、结果与资源边界

- 工具仅接受领域 ID、枚举和受限查询；拒绝任意 SQL、shell、宿主路径、下载 URL。Provider 通过候选 ID 定位。
- 列表默认 25、最多 100 条，稳定排序并分页；对现有无分页查询补齐服务边界后再暴露。初始建议请求上限 64 KiB、结果上限 256 KiB，超过则分页或返回明确错误，不生成无限结果。
- 任务提交返回 `{ "jobId": 42, "state": "queued" }`。所有有副作用的提交要求 idempotencyKey，绑定 Token、工具和规范化参数；相同 key/相同参数返回原结果，不同参数返回冲突。
- JSON-RPC 请求 ID 不作为业务幂等键。幂等记录至少保留至关联任务终态后 7 天；计划还须永久约束同一版本不可重复应用。
- 工具业务错误使用 `isError` 和稳定错误码，例如 `PLAN_CONFLICT`、`APPROVAL_REQUIRED`、`RATE_LIMITED`；协议错误交给 SDK。返回 request ID，不返回堆栈或原始异常。
- 元数据、剧情与 NFO 内容属于不可信数据；不得将其中指令当作操作授权。工具 annotations 只是提示，不能替代服务端权限检查。
- 初始建议每 Token 60 次/分钟、全局查询并发 4、每 Token 同时运行写任务最多 1 个；超限快速返回，查询超时 10 秒。
- 请求取消只取消尚在执行的查询；已持久化任务由 `cancel_job` 显式取消，客户端断线不等于撤销业务任务。

资源待查询工具稳定后增加：`mediagrap://media/{id}`、`mediagrap://jobs/{id}`、`mediagrap://sources/{id}`。资源不是本地文件 URI，且不触发隐式 Provider 请求或扫描。

## 10. 文件操作批准协议

仅由 preview 自动返回一个 confirmationToken，随后让同一模型调用 apply，无法证明用户已批准。因此默认采用服务端记录的人工审批：

1. preview 持久化不可变计划，返回 planId、version、digest、来源内相对目标路径、变更/覆盖摘要、冲突、恢复边界、过期时间，以及站内审批链接；链接不含凭证。
2. 管理员在已认证 Web UI 查看同一份计划，通过 Session + CSRF 批准，记录 approver、发起 Token、计划摘要和作用范围。MCP 不提供 approve 工具。
3. apply 携带 planId、version、digest 和 idempotencyKey。服务端查询审批记录，核验有效期（建议批准后 5 分钟）、当前 Token/owner 权限、来源和计划状态。
4. 一个数据库事务内消费批准、建立幂等结果并创建 durable job；网络重发返回同一 job ID。不同 Token、版本或参数不能复用批准。
5. Worker 执行前再次校验撤销状态、路径边界、符号链接、文件类型、版本指纹和冲突，获取必要锁；前置条件变化使批准失效，必须重新 preview/批准。
6. 继续复用临时文件、原子替换、跨设备移动校验与逐项审计；部分完成如实返回 partial。失败/中断后的再次执行需要针对剩余操作重新预览批准。

批准过期时间控制入队；入队后的计划内容保持冻结，执行仍检查权限和前置条件。Token 撤销后阻止未开始任务，运行中的任务请求协作取消并记录已经发生的效果，不承诺回滚。

未来若增加受限无人值守策略，必须单独设计授权来源、操作类型与覆盖范围，不能用默认 `plans:apply` 推导自动批准。

## 11. 审计、模块与迁移

MCP 记录 Token ID、owner、tool/resource 名称、白名单参数摘要、结果码、request/job/plan ID 和耗时；不存完整请求或结果。查询审计也要有保留期和容量限制。认证失败只记录安全原因和受信来源信息，不记录传入 Token。写任务继承 actor 关联到执行审计和业务事件。

建议模块：

```text
internal/events/       事件模型、事务内写入接口、分发
internal/webhooks/     endpoint 服务、投递状态机、签名、URL 策略
internal/tokens/       Token 生命周期与 principal/scopes
internal/mcp/          transport、schema、工具与资源适配
internal/httpapi/      管理 API、人工批准入口
internal/jobs/         复用媒体任务，补齐事务状态转换
internal/app/          装配、Worker 生命周期、配置
internal/platform/database/migrations/   新表和索引
```

事务接口允许业务 Repository 与 event writer 使用同一事务；领域层不依赖 Webhook HTTP 实现。Webhook 单独持久化调度但复用日志、时钟、网络校验基础组件，不塞进媒体单 Worker 或再引入一套外部队列。

迁移依次覆盖事件/版本、Webhook、Token，写能力阶段再增加审批与幂等记录。迁移编号按实施时仓库最大编号分配，不提前硬编码；历史任务不补发事件。迁移、失败恢复和已有数据库升级必须有测试。

## 12. 分阶段交付与验收

| 阶段 | 可交付结果 | 放行条件 |
| --- | --- | --- |
| A：事件基础 | job succeeded/failed 同事务 Outbox、分发、事件 schema | 所有相关终态路径接入；回滚/重复转换/重启测试通过 |
| B：Webhook MVP | 单 endpoint 起步，再开放 CRUD、签名、重试、最小管理 UI | 崩溃重复、验签、SSRF、暂停/轮换、容量可观测性通过 |
| C：MCP 只读 MVP | Token 管理、4 个工具、分页、审计、客户端配置说明 | 协议基线、来源隔离、撤销、速率/大小限制通过 |
| D：受控任务 | 扫描/刮削、任务查询和取消、幂等 | 不隐式写媒体文件；重复提交和取消语义通过 |
| E：文件能力 | preview、Web 审批、apply、逐项恢复 | 人工审批不可伪造，故障恢复及文件安全全套通过 |
| 后续 | OAuth、更多 resources、过滤表达式、进度通知、stdio | 有明确客户端/产品需求后独立排期 |

A/B 与 C 只共享基础授权/脱敏能力，不强制只读 MCP 依赖完整事件系统；避免先建设庞大公共框架。Webhook 管理 UI 只在 B 交付一次，后续不重复列为 MVP。

最低验证集：

- **事务与恢复**：注入 Outbox 插入失败、fan-out 中断、HTTP 成功后进程退出、lease 竞争，验证无已提交事件静默丢失且重复可识别；包含终态取消竞争和重试后再次完成。
- **Webhook**：固定签名向量、body 单字节变更、过期时间戳、轮换；模拟 2xx/3xx/4xx/429/5xx/超时，验证预算、Retry-After、手动重试和 endpoint 删除。
- **网络安全**：IPv4/IPv6、DNS rebinding、重定向、内网白名单、代理绕过、云元数据地址；测试与正式投递使用同一策略。
- **MCP**：初始化/版本/Origin、无效 Token、撤销、越权 tool/resource/ID、输入输出上限、分页；至少验证一个真实目标客户端，记录版本和配置方式。
- **文件批准**：自批、跨 Token/跨计划复用、过期、权限撤销、并发 apply、文件被外部修改、软链接逃逸、部分完成及重启；同一版本不能重复写入。
- **隐私与运行**：日志/API/事件快照中无凭证与绝对根路径；慢 endpoint 不阻塞媒体作业，积压清理不删除待投递事件。

各代码阶段运行相关 Go 格式化、静态检查、单元/集成与竞态测试；有 UI 改动时执行前端 lint、类型检查、单测和生产构建。未开放的后续能力仍须通过各自验收，不以首批测试结果代替。
