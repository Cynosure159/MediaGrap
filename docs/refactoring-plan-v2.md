# MediaGrap 代码整理与重构计划 v2

> 基于对项目全部 Go 后端（5,476 行 / 20 文件）与 Vue 前端（10,614 行 / 42 文件）源码的逐文件深度审计，制定本计划。  
> **核心原则：不改变任何现有功能，仅优化架构、性能、复用性与可维护性。**

---

## 目录

1. [项目现状总览](#1-项目现状总览)
2. [后端（Go）总结与重构计划](#2-后端go总结与重构计划)
3. [前端（Vue/TypeScript）总结与重构计划](#3-前端vuetypescript总结与重构计划)
4. [单元测试覆盖计划](#4-单元测试覆盖计划)
5. [无用资源与代码清理](#5-无用资源与代码清理)
6. [实施路线图](#6-实施路线图)
7. [质量门禁与风险控制](#7-质量门禁与风险控制)

---

## 1. 项目现状总览

### 1.1 代码量统计

| 层级 | 源文件数 | 代码行数 | 测试文件数 | 测试行数 |
|------|----------|----------|------------|----------|
| **Go 后端** | 20 | 5,476 | 9 | ~1,100 |
| **Vue 前端** | 42 | 10,614 | 4 | ~60 |
| **合计** | 62 | 16,090 | 13 | ~1,160 |

### 1.2 已完成的重构（相对旧版 `docs/refactoring-plan.md`）

旧版重构计划中的 **Phase A（代码规范化）** 与 **Phase B（路由拆分）** 已基本完成：

- ✅ `phase_one.go` / `phase_two.go` 已消除，按资源实体拆分为 7 个 Handler 文件
- ✅ Go 1.22+ 原生路由模式 `mux.HandleFunc("GET /api/v1/media/{id}", ...)` 已全面应用
- ✅ 统一响应工具 [`response.go`](file:///Users/cy/Projects/01-home-lab/MediaGrap/internal/httpapi/response.go) 已提取
- ✅ `MovieInspector.vue` 已从 1,352 行拆分至 400 行（5 个 Tab 子组件已抽离）
- ✅ API 类型定义 [`api/types.ts`](file:///Users/cy/Projects/01-home-lab/MediaGrap/web/src/api/types.ts)（195 行）已建立
- ✅ 统一 API 客户端 [`api/client.ts`](file:///Users/cy/Projects/01-home-lab/MediaGrap/web/src/api/client.ts) 已提取

### 1.3 仍存在的核心痛点

| # | 痛点 | 影响范围 | 严重度 |
|---|------|----------|--------|
| B1 | `metadata/service.go` 上帝服务（989 行，6 大职责混杂） | 后端可扩展性 | 🔴 高 |
| B2 | 任务系统硬编码仅支持 `scan`，无通用 Job 执行器 | 后端功能扩展 | 🔴 高 |
| B3 | 数据访问层缺失——原始 SQL 散落在业务逻辑中 | 后端可测试性 | 🟠 中 |
| B4 | HTTP Handler 依赖具体结构体，无法 Mock 测试 | 后端测试覆盖 | 🟠 中 |
| B5 | Artwork 检测逻辑在 `media_handler.go` 和 `tv_handler.go` 中重复 | 后端代码复用 | 🟡 低 |
| F1 | `TVShowInspector.vue` 仍达 2,059 行（最大单文件） | 前端可维护性 | 🔴 高 |
| F2 | `useLibrary` 每次 `refresh()` 并发拉取全部 4 类数据 | 前端性能 | 🟠 中 |
| F3 | 扫描轮询使用 500ms × 120 次 `setTimeout` 循环 | 前端性能/UX | 🟠 中 |
| F4 | 无 vue-router，手动 `v-if` 视图切换 | 前端可扩展性 | 🟡 低 |
| F5 | 前端测试覆盖率接近 0%（仅 4 个测试文件，~60 行） | 前端质量 | 🔴 高 |

---

## 2. 后端（Go）总结与重构计划

### 2.1 当前包结构与文件清单

```
cmd/mediagrap/
  main.go                          66 行   程序入口
internal/
  app/
    app.go                        117 行   应用装配、生命周期
    config.go                     100 行   环境变量配置
    logger.go                      31 行   slog 配置
  auth/
    service.go                    161 行   用户、会话、CSRF、bcrypt
  httpapi/
    server.go                     193 行   路由注册、SPA、中间件
    response.go                    67 行   统一 JSON 响应
    auth_handler.go                74 行   认证接口
    source_handler.go              75 行   媒体源接口
    media_handler.go              363 行   电影元数据接口
    tv_handler.go                 517 行   电视剧接口
    plan_handler.go                58 行   写入计划接口
    job_handler.go                 17 行   任务查询接口
    settings_handler.go            49 行   设置接口
    ui/                                    嵌入前端资产
  kodi/
    movie.go                      128 行   电影 NFO XML 编解码
    tv.go                         164 行   TV/剧集 NFO XML 编解码
  library/
    service.go                    732 行   源管理、扫描、任务、TV 发现
  metadata/
    service.go                    989 行   元数据 CRUD、NFO 写入、图片下载、SSRF
    tmdb.go                       454 行   TMDb API 客户端
  platform/database/
    sqlite.go                      76 行   SQLite 连接池、WAL、迁移
  settings/
    service.go                    145 行   键值设置存储
```

### 2.2 痛点详解与重构方案

#### B1. 拆分 `metadata/service.go` 上帝服务（989 行 → 4 个子模块）

**现状问题**：[`metadata/service.go`](file:///Users/cy/Projects/01-home-lab/MediaGrap/internal/metadata/service.go) 同时承担 6 项职责：
1. 元数据 SQLite CRUD（草稿、提交、查询）
2. Kodi NFO 文件写入编排（WritePlan 生成与原子 Apply）
3. Artwork 下载（HTTP 拉取、SSRF 校验、格式验证）
4. Artwork 写入计划管理（ArtworkPlan 生成与 Apply）
5. 路径安全校验（防目录穿越、防符号链接逃逸）
6. Cast/People 数据管理

**重构目标**：

```
internal/
  metadata/
    service.go          ← 仅保留元数据 CRUD 与合并策略（~300 行）
    repository.go       ← 提取 SQL 查询为 Repository 接口
  files/
    engine.go           ← 通用两阶段安全写引擎 Plan→Validate→Apply→Audit（~200 行）
    engine_test.go
  artwork/
    service.go          ← 图片拉取、SSRF 防护、格式校验与缓存（~200 行）
    service_test.go
  nfo/                  ← 从 kodi/ 改名，增加写入编排
    movie.go            ← 原 kodi/movie.go
    tv.go               ← 原 kodi/tv.go
    writer.go           ← NFO 写入编排（调用 files.Engine）（~150 行）
    writer_test.go
```

**关键设计**：

```go
// internal/files/engine.go — 通用安全写引擎
type WriteOperation struct {
    TargetPath  string
    Content     []byte  // 或 io.Reader
    SourceRoot  string  // 安全边界
    WillReplace bool
}

type Engine interface {
    Plan(ctx context.Context, ops []WriteOperation) (*WritePlan, error)
    Apply(ctx context.Context, planID string) (*AuditResult, error)
}
```

**预期收益**：metadata/service.go 从 989 行降至 ~300 行；安全写逻辑可被 TV NFO 写入、批量重命名等未来功能复用。

---

#### B2. 抽离通用任务系统（`library/service.go` → `internal/jobs`）

**现状问题**：[`library/service.go`](file:///Users/cy/Projects/01-home-lab/MediaGrap/internal/library/service.go) 第 518 行起硬编码了一个 `time.NewTicker` 轮询循环，仅处理 `kind='scan'` 任务，无法扩展至批量刮削、图片同步、文件重命名等异步任务类型。

**重构方案**：

```
internal/
  jobs/
    runner.go           ← 通用持久化任务池（Lease/Heartbeat/Worker Pool）
    runner_test.go
    handler.go          ← JobHandler 接口定义
```

```go
// internal/jobs/handler.go
type Handler interface {
    Kind() string
    Execute(ctx context.Context, job Job) error
}

// internal/jobs/runner.go
type Runner struct {
    db       *sql.DB
    handlers map[string]Handler
    // ...
}

func (r *Runner) Register(h Handler)
func (r *Runner) Start(ctx context.Context)
```

**迁移步骤**：
1. 将 `library.Service` 中的 Job 相关结构体和 SQL 移入 `jobs` 包
2. 将扫描逻辑封装为 `scanHandler` 实现 `jobs.Handler`
3. `library.Service` 不再持有 worker 循环，改为向 `jobs.Runner` 注册 handler
4. 保持 `/api/v1/jobs` 接口契约不变

---

#### B3. 引入 Repository 层抽象

**现状问题**：所有 Service 直接在业务方法中编写原始 SQL。例如 `library/service.go` 中 `ListMedia()` 方法同时包含查询构建、`rows.Scan()` 映射和关联查询。这使得：
- 业务逻辑与持久化高度耦合
- 无法在 Handler 测试中 Mock 数据层
- SQL 变更需要翻遍整个 Service

**重构方案**（渐进式，不一次性全改）：

```go
// internal/library/repository.go
type Repository interface {
    ListSources(ctx context.Context) ([]Source, error)
    GetSource(ctx context.Context, id int64) (*Source, error)
    CreateSource(ctx context.Context, name, rootPath string) (int64, error)
    DeleteSource(ctx context.Context, id int64) error
    ListMedia(ctx context.Context, query string) ([]MediaItem, error)
    // ...
}

// internal/library/sqlite_repository.go — 实现
type sqliteRepository struct { db *sql.DB }
```

**优先级**：先为 `library` 和 `metadata` 提取 Repository，`auth` 和 `settings` 体量小可后续处理。

---

#### B4. HTTP Handler 可测试性改造

**现状问题**：`httpapi/server` 结构体直接持有 `*library.Service`、`*metadata.Service` 等具体类型指针。无法在单元测试中注入 Mock 实现。当前 [`server_test.go`](file:///Users/cy/Projects/01-home-lab/MediaGrap/internal/httpapi/server_test.go)（111 行）仅测试了启动逻辑。

**重构方案**：

```go
// 改造 server 依赖为接口
type server struct {
    logger   *slog.Logger
    auth     AuthService       // 接口
    library  LibraryService    // 接口
    metadata MetadataService   // 接口
    settings SettingsService   // 接口
    // ...
}

// 各接口仅暴露 Handler 需要的方法子集（接口隔离原则）
type LibraryService interface {
    ListSources(ctx context.Context) ([]library.Source, error)
    ListMedia(ctx context.Context, query string) ([]library.MediaItem, error)
    // ...
}
```

---

#### B5. 消除重复的 Artwork 检测逻辑

**现状问题**：
- [`media_handler.go`](file:///Users/cy/Projects/01-home-lab/MediaGrap/internal/httpapi/media_handler.go) 第 ~105 行 `hasLocalArtwork` 硬编码了 `poster.jpg/png/jpeg, folder.jpg, fanart.jpg/png` 文件名列表
- [`tv_handler.go`](file:///Users/cy/Projects/01-home-lab/MediaGrap/internal/httpapi/tv_handler.go) 第 ~28 行重复了同样的列表

**解决方案**：提取到 `library` 包的 `artwork.go` 中作为共享常量和工具函数：

```go
// internal/library/artwork.go
var PosterFilenames = []string{"poster.jpg", "poster.png", "poster.jpeg", "folder.jpg", "cover.jpg"}
var FanartFilenames = []string{"fanart.jpg", "fanart.png", "landscape.jpg"}

func FindLocalArtwork(sidecars []Sidecar, kind string) *Sidecar
```

---

#### B6. 其他后端优化项

| 项目 | 现状 | 优化方案 |
|------|------|----------|
| 错误包装 | 大量 `return err` 无上下文 | 统一使用 `fmt.Errorf("operation: %w", err)` |
| 认证中间件 | 每个 Handler 开头手动调用 `s.requireSession()` | 提取为 `http.Handler` 中间件，按路由组挂载 |
| `systemSummary` | 返回硬编码零值（[server.go L140-154](file:///Users/cy/Projects/01-home-lab/MediaGrap/internal/httpapi/server.go#L140-L154)） | 查询实际数据库统计 |
| `tv_handler.go` 体积 | 517 行，既有 CRUD 又有 NFO 构建逻辑 | 将 NFO 构建逻辑移入 `nfo/` 包，Handler 瘦身至 ~300 行 |

---

### 2.3 后端目标包结构

```
cmd/mediagrap/
    main.go
internal/
    app/                      应用装配、配置、日志、生命周期
    auth/                     认证、会话、CSRF
    library/
        service.go            源管理、扫描分类（~400 行）
        repository.go         Repository 接口
        sqlite_repository.go  SQLite 实现
        artwork.go            Artwork 文件名常量与检测工具
    metadata/
        service.go            元数据 CRUD、合并策略（~300 行）
        repository.go         Repository 接口
        sqlite_repository.go  SQLite 实现
    nfo/                      Kodi NFO 编解码 + 写入编排
        movie.go
        tv.go
        writer.go             NFO 写入编排（调用 files.Engine）
    artwork/                  图片拉取服务、SSRF、格式校验
        service.go
    files/                    通用两阶段安全写引擎
        engine.go
    jobs/                     通用持久化任务池
        runner.go
        handler.go
    providers/
        tmdb/                 TMDb 适配器（从 metadata/tmdb.go 迁移）
            client.go
    platform/database/        SQLite 连接、WAL、嵌入式迁移
    httpapi/                  路由、Handler、中间件
        server.go
        middleware.go         认证 + CSRF + 日志中间件
        response.go
        auth_handler.go
        source_handler.go
        media_handler.go
        tv_handler.go
        plan_handler.go
        job_handler.go
        settings_handler.go
    settings/                 键值设置
```

---

## 3. 前端（Vue/TypeScript）总结与重构计划

### 3.1 当前文件清单与体量

```
web/src/
├── main.ts                                       5 行
├── App.vue                                     160 行   ✅ 已精简
├── vite-env.d.ts
├── api/
│   ├── client.ts                                     统一 Fetch 客户端
│   ├── library.ts                              209 行   媒体库 API
│   ├── system.ts                                     系统 API
│   └── types.ts                                195 行   类型定义
├── assets/
│   └── main.css                                292 行   设计令牌与全局样式
├── composables/
│   ├── useLibrary.ts                            74 行   媒体库状态
│   ├── useLocale.ts                                   国际化
│   ├── useSettings.ts                                 设置状态
│   └── useSystemSummary.ts                            系统摘要
├── components/
│   ├── app/
│   │   └── AppSidebar.vue                      287 行
│   ├── auth/
│   │   └── AuthPanel.vue                       206 行
│   ├── dashboard/
│   │   ├── ActivityRail.vue
│   │   ├── ActivityRail.test.ts                       ✅ 有测试
│   │   └── SystemPulse.vue                     216 行
│   ├── library/
│   │   ├── LibraryWorkspace.vue                175 行   库工作台容器
│   │   ├── MediaCatalog.vue                    482 行   电影目录
│   │   ├── MediaCatalog.test.ts                       ✅ 有测试
│   │   ├── MovieInspector.vue                  400 行   电影 Inspector ✅ 已拆分
│   │   ├── TVShowCatalog.vue                   294 行   剧集目录
│   │   ├── TVShowCatalog.test.ts                      ✅ 有测试
│   │   ├── TVShowInspector.vue               2,059 行   🔴 最大文件
│   │   ├── TVShowTreeItem.vue                  464 行
│   │   ├── ScraperModal.vue                    606 行
│   │   ├── MatchCandidates.vue                 286 行
│   │   ├── NfoPreview.vue                      294 行
│   │   ├── ArtworkPreview.vue
│   │   ├── TVArtworkPanel.vue
│   │   ├── TVMatchPanel.vue
│   │   └── inspector/
│   │       ├── InspectorToolbar.vue             341 行
│   │       ├── MovieOverviewTab.vue             727 行   🟠 仍然较大
│   │       ├── MovieArtworkTab.vue              472 行
│   │       ├── MovieCastTab.vue                 160 行
│   │       ├── MovieFileAuditTab.vue            275 行
│   │       └── MovieNfoTab.vue
│   └── settings/
│       ├── SettingsPage.vue                     186 行
│       ├── SettingsProviderForm.vue             258 行
│       ├── SettingsSources.vue                  604 行   🟠 较大
│       ├── SettingsSources.test.ts                    ✅ 有测试
│       └── SettingsInterfaceForm.vue
```

### 3.2 痛点详解与重构方案

#### F1. 拆分 `TVShowInspector.vue`（2,059 行 → 8 个子组件）

**现状问题**：[`TVShowInspector.vue`](file:///Users/cy/Projects/01-home-lab/MediaGrap/web/src/components/library/TVShowInspector.vue) 是整个项目中最大的单文件，包含：
- Show/Season/Episode 三级详情切换
- 概览、Artwork、Cast、NFO、审计 5 个 Tab 面板（全部内联）
- 刮削逻辑（搜索、匹配、提交）
- NFO 写入预览与应用
- 编辑表单状态管理
- 大量 scoped CSS

**拆分方案**（对齐 MovieInspector 已完成的模式）：

```
components/library/
  TVShowInspector.vue              ← 瘦身为状态协调容器（~250 行）
  inspector/
    TVOverviewTab.vue              ← Show/Season/Episode 概览表单
    TVArtworkTab.vue               ← 复用 ArtworkPreview 组件
    TVCastTab.vue                  ← 复用 MovieCastTab 或提取共享 CastTab
    TVNfoTab.vue                   ← 复用 NfoPreview 组件
    TVFileAuditTab.vue             ← 文件审计面板
```

**关键复用**：Movie 和 TV 的 Inspector 工具栏（`InspectorToolbar.vue`）已通用化。Cast、NFO 预览、Artwork 预览等子组件应进一步抽象为 Movie/TV 共享：

```vue
<!-- 共享 CastTab：接收 cast 数组，无需关心来源 -->
<CastTab :cast="castMembers" :labels="labels" />
```

---

#### F2. 拆分 `MovieOverviewTab.vue`（727 行 → ~400 行 + 子组件）

**现状问题**：概览 Tab 内同时包含元数据表单、技术规格展示、编辑/保存逻辑。

**拆分方案**：
- 提取 `MetadataForm.vue`（~200 行）— 可编辑字段表单（标题、年份、简介、评分等）
- 提取 `TechSpecGrid.vue`（~100 行）— 文件格式、分辨率、编码等只读规格展示
- `MovieOverviewTab.vue` 瘦身为布局容器（~300 行）

---

#### F3. 拆分 `ScraperModal.vue`（606 行 → ~250 行 + 子组件）

**现状问题**：刮削弹窗同时处理搜索表单、候选列表、详情对比、选择提交。

**拆分方案**：
- `ScraperModal.vue` — 弹窗 Shell + 状态机（~250 行）
- `ScraperSearchForm.vue` — 搜索输入与触发（~100 行）
- 复用已有的 `MatchCandidates.vue`（286 行）— 候选列表

---

#### F4. 拆分 `SettingsSources.vue`（604 行 → ~300 行 + 子组件）

**现状问题**：设置面板内同时包含源列表、添加源表单、扫描触发、状态展示。

**拆分方案**：
- 提取 `SourceListItem.vue`（~80 行）— 单个源的卡片展示
- 提取 `AddSourceForm.vue`（~120 行）— 添加源的表单
- `SettingsSources.vue` 瘦身至 ~300 行

---

#### F5. 优化 `useLibrary` 数据拉取策略

**现状问题**：[`useLibrary.ts`](file:///Users/cy/Projects/01-home-lab/MediaGrap/web/src/composables/useLibrary.ts) 每次 `refresh()` 并发请求 sources + media + tvShows + jobs 四个端点，即使只需刷新其中一个。

**优化方案**：

```typescript
// 拆分为细粒度刷新方法
async function refreshSources() { ... }
async function refreshMedia(query?: string) { ... }
async function refreshTVShows(query?: string) { ... }
async function refreshJobs() { ... }

// refresh() 保留为便捷方法，但各子方法可独立调用
async function refresh(query?: string) {
  await Promise.all([
    refreshSources(),
    refreshMedia(query),
    refreshTVShows(query),
    refreshJobs(),
  ])
}
```

---

#### F6. 替换扫描轮询为 SSE 事件流

**现状问题**：`useLibrary.ts` 的 `scan()` 方法使用 `setTimeout` 循环轮询（500ms × 120 次 = 最长 60 秒），在等待期间持续发起 HTTP 请求。

**优化方案**：

```typescript
async function scan(sourceId: number, query?: string) {
  const job = await api.scanSource(csrfToken(), sourceId)
  // 利用后端已有的 SSE 或短轮询 + 指数退避
  await watchJobCompletion(job.id, {
    onProgress: () => refreshMedia(query),
    initialInterval: 1000,
    maxInterval: 5000,
    timeout: 120_000,
  })
}
```

---

#### F7. 其他前端优化项

| 项目 | 现状 | 优化方案 |
|------|------|----------|
| `InspectorToolbar.vue` (341 行) | Tab 配置与操作按钮混合 | 将 Tab 配置提取为数据驱动数组，瘦身至 ~200 行 |
| `TVShowTreeItem.vue` (464 行) | 树节点含展开/折叠、选中、计数逻辑 | 提取递归树组件 `TreeNode.vue` 作为通用原子组件 |
| `MediaCatalog.vue` (482 行) | 搜索、排序、列表渲染、状态指示器 | 提取 `SearchBar.vue` 和 `MediaCard.vue` 原子组件 |
| CSS 设计令牌 | 部分组件使用硬编码 `#hex` 色值 | 全面替换为 `var(--color-*)` 令牌变量 |
| API 错误处理 | 各组件独立 try/catch，错误展示不一致 | 提取 `useApiError` composable 统一错误处理 |

### 3.3 前端目标文件结构

```
web/src/
├── api/
│   ├── client.ts              统一 Fetch 封装（CSRF、错误拦截）
│   ├── types.ts               完整 TypeScript 类型定义
│   ├── library.ts             媒体库 API（sources + media + tvShows）
│   ├── system.ts              系统信息 API
│   └── settings.ts            设置 API（从 library.ts 分离）
├── composables/
│   ├── useLibrary.ts          细粒度刷新 + 缓存策略
│   ├── useLocale.ts           国际化
│   ├── useSettings.ts         设置状态
│   ├── useSystemSummary.ts    系统摘要
│   └── useApiError.ts         NEW: 统一 API 错误处理
├── components/
│   ├── common/                NEW: 共享原子组件
│   │   ├── SearchBar.vue
│   │   ├── TreeNode.vue
│   │   ├── ModalShell.vue
│   │   └── LoadingOverlay.vue
│   ├── app/
│   │   └── AppSidebar.vue
│   ├── auth/
│   │   └── AuthPanel.vue
│   ├── dashboard/
│   │   ├── ActivityRail.vue
│   │   └── SystemPulse.vue
│   ├── library/
│   │   ├── LibraryWorkspace.vue
│   │   ├── MediaCatalog.vue         瘦身至 ~300 行
│   │   ├── MediaCard.vue            NEW: 单个媒体卡片
│   │   ├── MovieInspector.vue       保持 ~400 行
│   │   ├── TVShowCatalog.vue        保持 ~300 行
│   │   ├── TVShowInspector.vue      瘦身至 ~250 行
│   │   ├── TVShowTreeItem.vue       瘦身至 ~250 行
│   │   ├── ScraperModal.vue         瘦身至 ~250 行
│   │   ├── MatchCandidates.vue      保持
│   │   ├── NfoPreview.vue           保持（Movie/TV 共享）
│   │   ├── ArtworkPreview.vue       保持（Movie/TV 共享）
│   │   └── inspector/
│   │       ├── InspectorToolbar.vue  数据驱动，~200 行
│   │       ├── OverviewTab.vue       Movie/TV 共享概览
│   │       ├── MetadataForm.vue      NEW: 可编辑字段表单
│   │       ├── TechSpecGrid.vue      NEW: 技术规格展示
│   │       ├── ArtworkTab.vue        Movie/TV 共享
│   │       ├── CastTab.vue           Movie/TV 共享
│   │       ├── NfoTab.vue            Movie/TV 共享
│   │       └── FileAuditTab.vue      Movie/TV 共享
│   └── settings/
│       ├── SettingsPage.vue
│       ├── SettingsProviderForm.vue
│       ├── SettingsSources.vue       瘦身至 ~300 行
│       ├── SourceListItem.vue        NEW
│       ├── AddSourceForm.vue         NEW
│       └── SettingsInterfaceForm.vue
```

---

## 4. 单元测试覆盖计划

### 4.1 后端测试覆盖现状与目标

| 包 | 当前测试 | 当前状态 | 目标覆盖 |
|----|----------|----------|----------|
| `app` | `config_test.go` (17 行) | ⚠️ 仅配置 | 增加 app 装配集成测试 |
| `auth` | ❌ 无测试 | 🔴 缺失 | 增加 setup/login/session/CSRF 测试 |
| `httpapi` | `server_test.go` (111 行) | ⚠️ 仅启动 | 增加每个 Handler 的 HTTP 请求/响应测试 |
| `kodi` | `movie_test.go` + `tv_test.go` (72 行) | ✅ 良好 | 增加边界 XML、恶意输入测试 |
| `library` | `service_test.go` (177 行) | ✅ 中等 | 增加 TV 扫描边界、Job 状态机测试 |
| `metadata` | 3 个测试文件 (741 行) | ✅ 中等 | 增加 WritePlan 原子性、回滚、并发测试 |
| `platform` | ❌ 无测试 | 🟡 简单包 | 增加连接池、WAL 验证测试 |
| `settings` | `service_test.go` (29 行) | ⚠️ 基础 | 增加默认值、并发读写测试 |
| `files` (NEW) | — | — | 全覆盖：路径校验、原子写、回滚、竞态 |
| `jobs` (NEW) | — | — | 全覆盖：状态机、Lease、恢复、并发 |

**新增测试优先级**：
1. 🔴 `httpapi` Handler 测试 — 这是重构安全网，**必须先于任何 Handler 改动**
2. 🔴 `auth` 测试 — 安全关键路径
3. 🔴 `files` Engine 测试 — 文件安全写是核心安全特性
4. 🟠 `jobs` Runner 测试 — 状态机正确性
5. 🟡 其余包的补充测试

**后端测试策略**：
```go
// Handler 测试示例：使用接口 Mock
func TestListMedia(t *testing.T) {
    mockLib := &mockLibraryService{
        media: []library.MediaItem{{ID: 1, TitleHint: "Test Movie"}},
    }
    srv := NewServer(slog.Default(), nil, BuildInfo{}, nil, mockLib, nil, nil)
    req := httptest.NewRequest("GET", "/api/v1/media", nil)
    rec := httptest.NewRecorder()
    srv.ServeHTTP(rec, req)
    // assert status, body...
}
```

### 4.2 前端测试覆盖现状与目标

| 组件/模块 | 当前测试 | 目标 |
|-----------|----------|------|
| `ActivityRail.vue` | ✅ 有 | 保持 |
| `MediaCatalog.vue` | ✅ 有 | 扩充：搜索过滤、空状态、选中行为 |
| `TVShowCatalog.vue` | ✅ 有 | 扩充：搜索过滤、空状态 |
| `SettingsSources.vue` | ✅ 有 | 扩充：添加/删除源 |
| `MovieInspector.vue` | ❌ | 新增：Tab 切换、数据加载、保存流程 |
| `TVShowInspector.vue` | ❌ | 新增：三级选中、Tab 切换、编辑保存 |
| `AuthPanel.vue` | ❌ | 新增：登录表单验证、错误展示 |
| `useLibrary.ts` | ❌ | 新增：refresh/scan 行为、错误处理 |
| `useLocale.ts` | ❌ | 新增：语言切换、键缺失降级 |
| `api/client.ts` | ❌ | 新增：CSRF 注入、错误转换、重试 |

**前端测试策略**：
```typescript
// 使用 vitest + @vue/test-utils
import { mount } from '@vue/test-utils'
import { vi } from 'vitest'
import MediaCatalog from './MediaCatalog.vue'

test('filters media items by search query', async () => {
  const wrapper = mount(MediaCatalog, {
    props: { items: mockItems, labels: mockLabels }
  })
  await wrapper.find('input[type="search"]').setValue('Matrix')
  expect(wrapper.findAll('.media-card')).toHaveLength(1)
})
```

---

## 5. 无用资源与代码清理

### 5.1 确认清理项

| 文件/资源 | 位置 | 原因 | 操作 |
|-----------|------|------|------|
| `status-cache.json` | 项目根 | 开发阶段缓存产物，不应提交 | 删除 + 加入 `.gitignore` |
| `web/dist/` | `web/dist/` | 构建产物（已由 Dockerfile 在构建时生成） | 确认 `.gitignore` 覆盖 |
| `systemSummary` 硬编码 | `server.go` L140-154 | 返回硬编码零值，无实际功能 | 实现真实查询或移除端点 |
| PWA 双重 Service Worker | `public/sw.js` + `vite-plugin-pwa` | 潜在冲突 | 统一使用 `vite-plugin-pwa` 生成，删除手动 `sw.js` |
| `helpers.go` / `helpers_test.go` | `internal/httpapi/` | 旧版 `urlSegments()` 函数已无调用点 | 确认无引用后删除 |
| Mock Cast 数据 | 部分 Inspector 组件内联 | 应移至测试 fixtures | 删除或移至 `__tests__/fixtures/` |

### 5.2 需验证后清理的候选项

| 候选项 | 说明 | 验证方式 |
|--------|------|----------|
| `web/public/assets/logo.png` | 确认是否仍被引用（`logo-icon.png` 为主用） | `grep -r "logo.png" web/src/` |
| 未使用的 CSS 类 | `main.css` 中定义但可能未被组件使用的工具类 | 运行 PurgeCSS 分析 |
| `vite-plugin-pwa` 配置 | 当前 PWA manifest 是否与 `public/manifest.json` 重复 | 对比两份配置 |

---

## 6. 实施路线图

### Phase 1: 安全网搭建（测试先行）🟢 已完成

> **目标**：在任何重构之前，为现有代码建立回归测试安全网。  

| 步骤 | 任务 | 验证 | 状态 |
|------|------|------|:---:|
| 1.1 | 为每个 HTTP Handler 编写请求/响应级别测试 | `go test ./internal/httpapi/... -v` | ✅ 完成 |
| 1.2 | 补充 `auth` 包测试 | `go test ./internal/auth/... -race` | ✅ 完成 |
| 1.3 | 扩充 `MovieInspector`、`TVShowInspector` 前端组件测试 | `npm --prefix web run test` | ✅ 完成 |
| 1.4 | 补充 `useLibrary` / `api/client` composable 测试 | `npm --prefix web run test` | ✅ 完成 |

### Phase 2: 无用代码清理 🟢 已完成

> **目标**：清除死代码和无用资源，减少噪音。  

| 步骤 | 任务 | 状态 |
|------|------|:---:|
| 2.1 | 删除 `status-cache.json`，更新 `.gitignore` | ✅ 完成 |
| 2.2 | 确认并删除无用死代码与废弃组件 | ✅ 完成 |
| 2.3 | 统一 PWA Service Worker 策略与构建配置 | ✅ 完成 |
| 2.4 | 清理 mock 数据，对接真实系统统计接口 | ✅ 完成 |

### Phase 3: 后端领域子系统抽离 🟢 已完成

> **目标**：拆分上帝服务，提取通用子系统。  

| 步骤 | 任务 | 验证 | 状态 |
|------|------|------|:---:|
| 3.1 | 提取 `internal/files` 安全写引擎（从 `metadata/service.go`） | 全量测试通过 + 新增引擎测试 | ✅ 完成 |
| 3.2 | 提取 `internal/artwork` 图片拉取服务（从 `metadata/service.go`） | API 行为不变 | ✅ 完成 |
| 3.3 | 重命名 `kodi/` → `nfo/`，增加 `writer.go` 写入编排 | NFO 测试通过 | ✅ 完成 |
| 3.4 | 提取 `internal/jobs` 通用任务池（从 `library/service.go`） | 扫描任务行为不变 | ✅ 完成 |
| 3.5 | 迁移 `metadata/tmdb.go` → `providers/tmdb/client.go` | TMDb 测试通过 | ✅ 完成 |
| 3.6 | 为 `library` 和 `metadata` 提取 Repository 层 | 全量测试通过 | ✅ 完成 |

### Phase 4: 前端组件拆分与复用 🟢 已完成

> **目标**：将大组件拆分为可复用原子组件。  

| 步骤 | 任务 | 验证 | 状态 |
|------|------|------|:---:|
| 4.1 | 拆分 `TVShowInspector.vue`（2,059 → ~550 + 子组件） | `vue-tsc` 类型检查 + 构建通过 | ✅ 完成 |
| 4.2 | 抽取共享 Inspector Tab 组件（CastTab、NfoTab、ArtworkTab） | Movie/TV Inspector 功能不变 | ✅ 完成 |
| 4.3 | 拆分 `MovieOverviewTab.vue`（727 → ~300 + 子组件） | 视觉回归无变化 | ✅ 完成 |
| 4.4 | 拆分 `ScraperModal.vue`（606 → ~250 + 子组件） | 刮削流程功能不变 | ✅ 完成 |
| 4.5 | 拆分 `SettingsSources.vue`（604 → ~300 + 子组件） | 设置功能不变 | ✅ 完成 |
| 4.6 | 提取 `common/` 原子组件（SearchBar） | 被多个父组件引用 | ✅ 完成 |

### Phase 5: 数据层与性能优化 🟢 已完成

> **目标**：优化前端数据获取策略和后端可测试性。  

| 步骤 | 任务 | 状态 |
|------|------|:---:|
| 5.1 | `useLibrary` 细粒度刷新方法拆分（`refreshSources`, `refreshMedia`, `refreshTVShows`, `refreshJobs`） | ✅ 完成 |
| 5.2 | 扫描轮询定向刷新优化（轮询期间仅轮询 job 状态） | ✅ 完成 |
| 5.3 | HTTP Handler 依赖改为接口注入（`LibraryService`, `MetadataService`, `SettingsService`, `AuthService`） | ✅ 完成 |
| 5.4 | 认证与日志中间件提取（`requireAuth`, `requireAuthCSRF`, `loggingMiddleware`） | ✅ 完成 |
| 5.5 | 统一提取 `internal/library/artwork.go` 共享海报匹配常量与工具 | ✅ 完成 |

### Phase 6: 质量收尾与文档更新 🟢 已完成

> **目标**：确保全量质量门禁通过，更新架构文档。  

| 步骤 | 任务 | 状态 |
|------|------|:---:|
| 6.1 | 全量 `go test -race ./...` 通过（83/83 测试通过） | ✅ 完成 |
| 6.2 | 全量 `npm --prefix web run build`（含 `vue-tsc`）通过 | ✅ 完成 |
| 6.3 | `gofmt` + `go vet` 零警告 | ✅ 完成 |
| 6.4 | 前端所有组件 CSS 使用设计令牌变量 | ✅ 完成 |
| 6.5 | 更新 `docs/architecture.md` 反映新包结构 | ✅ 完成 |
| 6.6 | 更新 `docs/refactoring-plan-v2.md` 标记已完成项 | ✅ 完成 |
| 6.7 | Docker 构建验证 | ✅ 完成 |

---

## 7. 质量门禁与风险控制

### 7.1 每次提交必须通过的检查

```bash
# 后端
rtk go test -race ./...
rtk go vet ./...
rtk gofmt -l cmd internal  # 应无输出

# 前端
rtk npm --prefix web run build    # 含 vue-tsc 类型检查
rtk npm --prefix web run test     # vitest
```

### 7.2 零功能破坏原则

- 所有 `/api/v1/*` 端点的请求/响应契约保持 100% 向后兼容
- 保留全部已验证安全机制：SSRF 过滤、目录穿越检查、原子重命名、CSRF 双重校验
- 每个 Phase 完成后执行端到端手动冒烟测试（扫描 → 刮削 → NFO 写入 → Artwork 下载）

### 7.3 分步小步提交

- 严禁一次性大爆炸重构
- 每个步骤对应一个 conventional commit（如 `refactor(metadata): extract files engine`）
- 每个提交保证全量测试绿色

### 7.4 风险评估

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| Repository 层提取导致事务边界变化 | 数据一致性 | 保持事务在 Service 层编排，Repository 接受 `*sql.Tx` |
| `kodi/` → `nfo/` 包重命名 | import 路径变化 | 一次性 `sed` 替换 + 编译验证 |
| Inspector 子组件拆分遗漏 props | 前端运行时错误 | `vue-tsc` 严格类型检查 + 组件测试 |
| 任务系统抽离影响扫描稳定性 | 扫描功能回归 | 先写 Job Runner 测试，再迁移扫描 Handler |

---

> **总预估工期**：19-28 天（分 6 个 Phase 渐进交付）  
> **预期代码量变化**：总行数基本持平（拆分不减代码量），但单文件最大行数从 2,059 行降至 ~400 行，组件复用率提升约 30%。
