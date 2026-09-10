# Webhook 与 MCP 集成使用说明

本轮实现基于 [功能方案](webhooks-and-mcp.md)，复用单进程 Go 服务、SQLite 和现有管理员 Session。新建配置后运行数据库迁移 `0017_integrations.sql`、`0018_automation.sql`；历史任务不会补发事件。

## 启用与配置

MCP 随应用提供 `/mcp` 入口，无 Token 时拒绝访问。设置页的「外部集成」区可创建 Token，并显式选择来源、权限和过期时间。Token 明文只展示一次；丢失后创建替代 Token 并撤销旧 Token。

Webhook 需要单独的主密钥文件：内容为 32 字节随机值的十六进制编码（64 个字符，可带末尾换行），权限为 `0600`，由应用运行用户读取。服务不自动创建或替换主密钥。

```sh
umask 077
openssl rand -hex 32 > webhook.key
```

将该文件只读挂载到容器，例如 `/config/webhook.key`，并设置 `MEDIAGRAP_WEBHOOK_KEY_FILE=/config/webhook.key`。备份必须同时保留数据库和此文件。丢失密钥或解密失败时不能继续发送已有 Webhook，不能通过生成新主密钥恢复旧签名 Secret。

| 环境变量 | 默认值 / 行为 |
| --- | --- |
| `MEDIAGRAP_WEBHOOK_KEY_FILE` | 未设置；管理页显示签名不可用，投递不启动 |
| `MEDIAGRAP_WEBHOOK_PAUSED` | `false`；设为 `true` 后停止领取投递，保留队列，需重启应用生效 |
| `MEDIAGRAP_WEBHOOK_ALLOW_HTTP` | `false`；设为 `true` 才允许白名单中的 HTTP 目标 |
| `MEDIAGRAP_WEBHOOK_ALLOWED_TARGETS` | 空；逗号分隔的精确 `host:port`，例如 `nas.example:8081` |
| `MEDIAGRAP_WEBHOOK_ALLOWED_CIDRS` | 空；允许的私网 CIDR，必须同时匹配上述目标白名单 |
| `MEDIAGRAP_MCP_ORIGINS` | 空；存在 Origin 的请求默认拒绝，可配置逗号分隔的精确受信 Origin |
| `MEDIAGRAP_INTEGRATION_BACKLOG_LIMIT` | `100000`；未分发事件与待投递记录总量达到阈值时拒绝新任务，最小 100 |

公网 HTTPS Webhook 默认可用；私网需要目标与网段同时授权。HTTP 还需要单独开关。Loopback、link-local、云元数据地址不会因白名单而放开。Webhook 不继承 Provider 代理、不跟随跳转；发送前重新解析 DNS，并将连接固定到校验后的地址。主密钥文件缺失时不影响媒体查询和已有应用功能。

浏览器管理 API 继续使用 Session + CSRF；API Token 只能用于 MCP，不能用于管理 Webhook、创建 Token 或批准计划。当前应用为单管理员模型；后续引入角色时还需在 Token 当前身份检查中增加角色权限交集。

## Webhook

设置页支持创建、编辑 URL/来源/事件、暂停、删除、测试、轮换签名 Secret、查看最近 100 次投递及重试失败记录。URL 不通过读取 API 返回；编辑时留空表示保留原目标。Secret 仅创建或轮换后展示一次。

已开放事件：

- `job.succeeded`、`job.failed`：已持久化的任务成功或最终失败；自动重试中的暂时失败不发送。
- `write_plan.applied`、`artwork_plan.applied`、`rename_plan.applied`：自动化计划全部完成、逐项记录已提交；partial、needs_review 不会产生 applied。
- `webhook.test`：管理员触发的独立测试投递，不进入普通订阅分发，也不递归产生作业事件。

需要在 Webhook 编辑表单显式勾选文件应用事件；默认订阅任务成功和失败。按来源筛选，空来源集合不表示全库。无来源归属的旧系统作业不进入来源订阅。

请求 body 在事件生成时冻结。验签使用：

```text
HMAC-SHA256(secret 的 UTF-8 字节, timestamp + "." + raw_body)
X-MediaGrap-Signature: sha256=<十六进制签名>
```

这里的 Secret 是设置页展示的字符串，不需要 Base64 解码；主密钥文件与接收端验签 Secret 是两种不同密钥。接收方必须验证时间窗口、使用常量时间比较，并从已验签 body 读取 event ID 进行持久化幂等处理。每次重试的时间戳和签名会更新，body/event ID/delivery ID 保持稳定。轮换后下一次尝试使用新 key ID，接收方短期同时保留新旧密钥。

首次发送加最多 5 次自动重试。网络错误、408、429、5xx 进入退避队列；3xx、其他 4xx、证书及地址策略失败进入 dead。429/503 接受 `Retry-After`，最长等待 24 小时。全局并发 2，同一 endpoint 同时只发送一个请求。有限重试不保证最终送达，也不保证顺序。

成功/取消历史保留 7 天，dead 保留 30 天；管理员可在对应保留期内手动开启新一轮重试。删除/暂停 endpoint 取消尚未发送的记录；已发出的请求无法撤回。目标变更不修改已排队 delivery 的目标快照。

`GET /api/v1/integrations/status` 返回签名是否可用、发送暂停状态、Worker 心跳、未分发/待投递/dead 数量。Outbox 80% 容量告警，达到阈值暂停新任务接收；运行任务仍可提交终态。阈值按记录数计，不是严格磁盘字节配额，仍需监控数据卷剩余空间。

## MCP 客户端

传输为 Streamable HTTP，固定协议版本 `2025-11-25`，使用官方 Go SDK `v1.6.1`。已用该版本 SDK 客户端完成真实 HTTP 初始化、工具发现、调用和撤销测试。未承诺未经验证的桌面客户端兼容性；客户端必须能自行设置 Bearer Header，当前未实现 OAuth 自动发现授权。

客户端连接参数：

```text
URL: https://你的服务地址/mcp
Authorization: Bearer <设置页创建的 API Token>
```

非本机环境通过 HTTPS 或可信 TLS 反向代理访问。客户端按协议先 initialize；后续 POST 携带 `MCP-Protocol-Version: 2025-11-25`。独立 GET SSE 流返回 405，任务状态通过查询工具读取。

每 Token 每分钟最多 60 次请求，全局并发 4，请求上限 64 KiB、响应上限 256 KiB、单次请求超时 10 秒。列表默认 25、最大 100，使用 `afterId`/`nextAfterId` 分页。调用审计保留 7 天，不保存完整输入、输出或凭证。

### 工具与权限

| 工具 | 必需权限 | 行为 |
| --- | --- | --- |
| `list_media`、`get_media` | `media:read` | 来源过滤后的数据库记录，不含宿主路径或图片 URL |
| `list_jobs`、`get_job` | `jobs:read` | 来源过滤后的任务状态，不含 payload 或内部错误文本 |
| `get_job_result` | `jobs:read` | 当前 Token 创建的任务结果，例如规范化电影候选 |
| `list_artwork_candidates` | `metadata:read` | 读取已缓存电影 Artwork 候选（最多 100），参数 `id` 为媒体 ID |
| `scan_source` | `jobs:write` | 增量扫描，只更新索引 |
| `search_media_candidates`、`scrape_media_candidates` | `jobs:write metadata:write` | 搜索并持久化规范化电影候选到作业结果，不选择或应用候选 |
| `scrape_artwork_candidates` | `jobs:write metadata:write` | 刷新已匹配电影的图片候选 |
| `scrape_tv_artwork_candidates` | `jobs:write metadata:write` | 刷新 TV 剧级或季级图片候选 |
| `scrape_tv_season`、`scrape_tv_episode` | `jobs:write metadata:write` | 更新已匹配剧的数据库元数据，不写 NFO |
| `cancel_job`、`retry_job` | `jobs:write` | 仅操作当前 Token 创建的任务；文件任务不能直接重试 |
| `preview_write_plan`、`preview_artwork_plan`、`preview_rename_plan`、`get_plan` | `plans:preview media:read` | 电影文件计划创建/查询，不自动批准 |
| `apply_write_plan`、`apply_artwork_plan`、`apply_rename_plan` | `plans:apply` | 精确匹配且已通过网页审批的计划入队 |

当前没有 shell、任意路径/URL 读写、MCP approve 工具，也未发布资源订阅、prompts 或 stdio。所有工具执行时检查权限，发现列表隐藏名称不能代替权限检查。

任务参数按工具使用 `sourceId`、`mediaId` 或 `showId`，TV 可附带 `season`、`episode`。`idempotencyKey` 长度为 8–128；同 Token、同工具、同 key 的相同参数返回原 job ID，不同参数返回冲突。每个 Token 同时最多一个 queued/running 自动化任务。

取消是协作式的：已经完成的文件原语不会被撤销。Token 撤销后，尚未执行的任务不会开始业务操作，运行任务约每秒检查撤销并请求取消。恢复权限也不能复用已消费的文件批准。

## 文件计划与审批

电影文件预览工具接受 `mediaId` 和 `idempotencyKey`。NFO 使用当前已保存的电影元数据；Artwork 还需 `selections: [{kind, candidateId}]`；重命名还需 `pattern`。最多 100 个操作，序列化计划上限 96 KiB，以确保 MCP 结果可完整预览。

1. preview 返回 `planId` 对应的 `id`、固定 `version: 1`、`digest`、来源内相对路径、覆盖标记、NFO 内容或图片候选 ID，以及 `/settings?approval=...` 审批链接。预览本身可以保存元数据/计划，不写媒体文件。
2. 管理员在设置页「自动化文件审批」查看操作明细、NFO 内容或图片预览，勾选已检查后批准。审批记录绑定 Token、计划摘要与管理员；有效期 5 分钟，整个预览有效期 24 小时。
3. 客户端调用对应 apply，携带 `planId`、`version`、`digest`、`idempotencyKey`。审批消费与入队在同一事务中进行，响应只返回 job ID。
4. 执行前检查当前身份、来源根目录身份、元数据摘要、文件指纹和冲突。应用各入口共用执行锁；通过 `os.Root` 限定目录边界，并拒绝符号链接。
5. 写入先在目标目录暂存并同步，再发布。重命名按预览逐文件执行，保留未选文件及空目录；同设备使用不覆盖目标的 link/unlink，跨设备采用流式复制、内容哈希校验、发布后删除源文件。
6. 每项记录 intent/staged/publishing/done 与审计。启动核对已发布结果，匹配才补全数据库；无法判断标记 `needs_review`。不会自动删除一个仍然存在的源文件或重新发起文件变更。

计划文件内容、目标和批准后不能编辑；当前固定版本为 1，重新预览使用新幂等键。计划保留永久应用状态以阻止同版本重复写入。部分完成、冲突或失败之后须先核对已有结果，再创建剩余操作的新计划；暂存文件可能保留用于故障排查，不会静默清理不确定结果。

当前文件审批仅面向电影；TV 批量文件计划和整体目录重命名尚未开放。NAS 不支持 hard link 或必要的目录同步时，操作会失败并留下可检查记录，不回退为覆盖目标的非安全移动。跨文件全局回滚/恢复备份不在当前能力内。

## 验证与限制

后端测试覆盖终态 Outbox 回滚、取消竞争、fan-out 事务、lease 重发、签名、轮换、重试分类、DNS/私网策略、MCP 来源越权、Token 撤销、人工审批/CSRF、并发 apply、外部文件变化、跨设备复制路径、NFO/Artwork/重命名及发布后数据库失败恢复。跨设备路径通过 EXDEV 故障注入测试，仍需在目标 NAS 文件系统做部署验收。

前端提供组件交互测试和生产构建。当前环境无可连接浏览器，未完成真实桌面/移动端视觉验收。测试环境的 Node Web Storage 与 jsdom 冲突时，运行：

```sh
NODE_OPTIONS=--no-experimental-webstorage npm --prefix web run test
```

仓库尚无单独 lint 脚本；使用 Go vet、Go race tests、Vue 类型检查、组件测试和生产构建作为本轮质量检查。
