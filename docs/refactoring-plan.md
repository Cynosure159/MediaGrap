# MediaGrap 架构重构与代码优化计划

> 本文档针对 MediaGrap 当前架构与代码现状，制定全面的重构与优化方案，旨在提升代码简洁性、消除冗余、增强复用性并提高系统的可维护性与可扩展性。

---

## 1. 架构现状与痛点诊断

### 1.1 后端现状 (Go Monolith)

1. **HTTP 路由与 Handler 组织缺乏领域聚焦**：
   - 目前 `internal/httpapi` 存在以开发阶段命名的历史文件（`phase_one.go`、`phase_two.go`），职责边界混杂。
   - 路由派发通过手动的 `urlSegments()` 字符串切分与 switch 逻辑实现，未充分利用 Go 1.22+ 原生模式匹配路由（`r.PathValue`）。
   - 请求解码、参数校验、业务编排与响应封装逻辑混杂在 Handler 中。
2. **`metadata.Service` 单一职责过重 (God Service)**：
   - `metadata.Service` 同时承担了：SQLite 元数据 CRUD、Kodi NFO 文件读写编排、HTTP 海报/背景图下载与防 SSRF 校验、文件写入预检（WritePlan/ArtworkPlan）、原子重命名与回滚。
   - 违反单一职责原则（SRP），阻碍了后续扩展电视剧 NFO 写入、批量重命名等特性。
3. **任务系统（Job Runner）硬编码耦合**：
   - 扫描工作器（Worker）硬编码在 `library.Service` 中通过 `time.Ticker` 轮询 SQLite 执行，缺乏通用的 Job 执行器抽象。后续的批量刮削、图片缓存、文件重命名任务无法直接复用。
4. **数据访问与事务管理分散**：
   - SQL 查询直接分散在各个 Service 方法内，缺乏统一的 Repository 层抽象和轻量级事务管理。

### 1.2 前端现状 (Vue 3 + TypeScript)

1. **超大组件单文件膨胀 (Monolithic Component)**：
   - `MovieInspector.vue` 单文件达到 **1,352 行**，将 5 大工作区 Tab（Overview、Artwork、Cast、NFO Raw、File Audit）、候选刮削弹窗、Diff 预览弹窗、硬编码 Mock 数据以及数百行样式集中在单个组件内，维护成本高。
   - `TVShowInspector.vue` 达到 **474 行**，同样包含大量未拆分的子面板与内联逻辑。
2. **单行压缩式代码残留**：
   - `web/src/api/library.ts` 与 `web/src/composables/useLibrary.ts` 部分逻辑采用紧凑单行写法，破坏了代码可读性与 Git 审查体验。
3. **状态与 Composable 粗粒度耦合**：
   - `useLibrary` 在每次调用 `refresh()` 时同时并发请求 `sources`、`media`、`tvShows`、`jobs`，缺乏细粒度的按需加载与缓存机制。
4. **前端重复拼接 XML 逻辑**：
   - 在 `MovieInspector.vue` 中通过 Vue 计算属性自行拼接 Kodi XML 字符串，与后端 Kodi NFO 序列化契约存在重复与不一致风险。

---

## 2. 重构目标与核心设计原则

- **单一职责原则 (SRP)**：分离路由接入层、领域服务层、文件安全引擎层与外部适配器。
- **两阶段安全写协议 (Plan & Apply)**：将文件写操作统一抽象为独立引擎，确保路径白名单校验、防路径穿透、临时文件写入与原子重命名。
- **组件原子化与组合式拆分**：每个 UI 组件专注单一视图/工作区，单文件代码量控制在 150~300 行以内。
- **对齐设计规范**：严格遵循 `docs/architecture.md` 与 Stitch 原型设计系统，利用统一的 CSS 变量与设计令牌。

---

## 3. 后端分层重构方案

### 3.1 目标包结构设计 (对齐架构规范)

```text
cmd/mediagrap/
    main.go                   # 程序启动入口
internal/
    app/                      # 应用装配、配置读取、生命周期与优雅停机
    auth/                     # 认证、用户会话、密码校验与 CSRF
    library/                  # 媒体源配置、文件系统扫描器、音视频文件识别
    metadata/                 # 标准元数据模型 (Canonical Record)、字段锁定与合并策略
    nfo/                      # Kodi 兼容 XML 编解码器 (Movie / TV / Episode)
    artwork/                  # 剧照/海报拉取、SSRF 防护、格式校验与缓存
    files/                    # 文件安全写引擎 (Plan -> Validate -> Atomic Apply -> Audit)
    jobs/                     # 持久化任务队列 (Job State Machine, Leases, Worker Pool)
    providers/                # 刮削器适配器契约与实现
        provider.go           # 统一 Provider 接口定义
        tmdb/                 # TMDb API 客户端 (包含限流、代理、重试)
    platform/
        database/             # SQLite 连接池、WAL 配置、嵌入式迁移
    httpapi/                  # RESTful API 路由与 Handler (Go 1.22+ 模式)
        server.go             # 统一路由注册与中间件挂载
        middleware.go         # 认证、CSRF、请求日志与 Panic 恢复中间件
        response.go           # 统一 JSON 编码、错误格式封装
        auth_handler.go       # 认证与安装引导接口
        source_handler.go     # 媒体源管理与扫描接口
        media_handler.go      # 电影元数据、候选匹配与草稿接口
        tv_handler.go         # 电视剧、季、剧集管理接口
        plan_handler.go       # NFO / Artwork 写入计划与执行接口
        job_handler.go        # 任务查询与进度流接口
        settings_handler.go   # 系统设置与网络代理测试接口
```

### 3.2 关键子系统重构设计

#### A. HTTP API 层现代化 (Go 1.22+ Native Routing)
- **消除 `phase_*.go`**：按资源实体（Resource-Oriented）拆分为独立的 Handler 文件。
- **原生路径参数解析**：
  ```go
  // 改造后: Go 1.22 原生路由
  mux.HandleFunc("GET /api/v1/media/{id}", app.getMediaDetail)
  mux.HandleFunc("GET /api/v1/media/{id}/candidates", app.getMediaCandidates)
  mux.HandleFunc("POST /api/v1/media/{id}/select", app.selectMediaCandidate)
  mux.HandleFunc("POST /api/v1/write-plans/{id}/apply", app.applyWritePlan)
  ```
- **统一响应与错误结构** (`response.go`)：
  ```go
  func respondJSON(w http.ResponseWriter, status int, data any)
  func respondError(w http.ResponseWriter, status int, code, message string)
  func decodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, bool)
  ```

#### B. 抽离通用安全文件写引擎 (`internal/files`)
- 抽象统一的两阶段写引擎，服务于：电影 NFO 写入、电视剧 NFO 写入、海报下载保存、未来媒体文件重命名。
- 职责：
  1. **Plan 阶段**：路径校验、媒体根目录隔离（防越权/防软链接逃逸）、哈希比对、冲突与覆盖检测。
  2. **Apply 阶段**：源锁争用、临时文件（`*.tmp`）安全写入、`fsync` 刷盘、原子重命名（`os.Rename`）、审计记录（`audit_events`）。

#### C. 抽象通用任务系统 (`internal/jobs`)
- 将 `library.Service` 中的后台轮询解耦为通用的持久化任务池。
- 支持注册不同类型的任务处理器：`JobHandler("scan", ...)`、`JobHandler("scrape_batch", ...)`、`JobHandler("artwork_sync", ...)`。

---

## 4. 前端组件化与工程化重构方案

### 4.1 目录结构与模块拆分

```text
web/src/
├── api/
│   ├── client.ts             # 统一 Fetch 封装 (CSRF 注入、错误拦截)
│   ├── auth.ts               # 认证与 Session API
│   ├── media.ts              # 电影与元数据 API
│   ├── tv.ts                 # 电视剧与单集 API
│   ├── sources.ts            # 媒体源与扫描 API
│   ├── plans.ts              # NFO 与 Artwork 计划/执行 API
│   ├── jobs.ts               # 任务状态 API
│   └── settings.ts           # 设置与系统信息 API
├── composables/
│   ├── useLocale.ts          # 国际化与翻译字典
│   ├── useMediaList.ts       # 电影列表筛选与分页
│   ├── useTVShowList.ts      # 剧集列表与季展开
│   ├── useMetadataEditor.ts  # 元数据草稿编辑、字段锁定与变更追踪
│   ├── useWritePlanner.ts    # NFO / Artwork 写入预检与执行
│   ├── useJobMonitor.ts      # 任务轮询与进度状态
│   └── useSettings.ts        # 系统设置管理
├── components/
│   ├── common/               # 基础原子组件 (SpecPill, ModalShell, TabNav, LoadingState)
│   ├── app/                  # 布局骨架 (AppSidebar, ActivityRail, AppHeader)
│   ├── auth/                 # 登录与引导 (AuthPanel)
│   ├── library/
│   │   ├── MediaCatalog.vue          # 电影目录列表 (左栏)
│   │   ├── TVShowCatalog.vue         # 剧集目录列表 (左栏)
│   │   ├── MovieInspector.vue        # 电影主工作台容器 (中/右栏)
│   │   ├── TVShowInspector.vue       # 剧集主工作台容器 (中/右栏)
│   │   ├── inspector/                # Inspector 拆解子面板
│   │   │   ├── InspectorToolbar.vue  # 顶部工作区 Tab 与快速操作栏
│   │   │   ├── OverviewTab.vue       # 概览与元数据表单
│   │   │   ├── ArtworkTab.vue        # 封面与背景海报工作区
│   │   │   ├── CastTab.vue           # 演职员表
│   │   │   ├── NfoTab.vue            # Kodi XML 源码预览与高亮
│   │   │   └── FileAuditTab.vue      # 物理文件属性与 sidecar 审查
│   │   └── modals/
│   │       ├── MatchCandidatesModal.vue # 刮削候选匹配对话框
│   │       └── NfoDiffPreviewModal.vue  # NFO 差异写入预览对话框
│   └── settings/                     # 设置面板各子组件
```

### 4.2 关键组件重构设计 (`MovieInspector.vue` 拆解)

`MovieInspector.vue` 重构后仅作为状态协调容器（~180 行）：

```vue
<script setup lang="ts">
import InspectorToolbar from './inspector/InspectorToolbar.vue'
import OverviewTab from './inspector/OverviewTab.vue'
import ArtworkTab from './inspector/ArtworkTab.vue'
import CastTab from './inspector/CastTab.vue'
import NfoTab from './inspector/NfoTab.vue'
import FileAuditTab from './inspector/FileAuditTab.vue'
import MatchCandidatesModal from './modals/MatchCandidatesModal.vue'
import NfoDiffPreviewModal from './modals/NfoDiffPreviewModal.vue'
import { useMetadataEditor } from '@/composables/useMetadataEditor'

// 仅负责组合 Composable 与调度子组件
</script>

<template>
  <main class="inspector-workspace">
    <InspectorToolbar :active-tab="activeTab" @select-tab="activeTab = $event" @save="handleSave" @scrape="openScrapeModal" />
    <div class="tab-content">
      <OverviewTab v-show="activeTab === 'overview'" v-model="draft" />
      <ArtworkTab v-show="activeTab === 'artwork'" :poster-url="draft.posterUrl" :backdrop-url="draft.backdropUrl" />
      <CastTab v-show="activeTab === 'cast'" :cast="castList" />
      <NfoTab v-show="activeTab === 'nfo'" :content="nfoXmlContent" />
      <FileAuditTab v-show="activeTab === 'files'" :item="detail?.item" />
    </div>
    <!-- 独立弹窗 -->
    <MatchCandidatesModal v-if="showCandidates" @select="handleSelectCandidate" @close="showCandidates = false" />
    <NfoDiffPreviewModal v-if="showNfoPreview" :plan="writePlan" @apply="handleApplyPlan" @close="showNfoPreview = false" />
  </main>
</template>
```

---

## 5. 实施路线图与阶段规划

- **Phase A: 代码规范化与清理** (展开压缩单行代码、统一 API Client 与后端响应样板)
- **Phase B: 后端路由与 Handler 拆分** (消除 `phase_*.go`、升级 Go 1.22 原生路由 `PathValue`、测试全绿)
- **Phase C: 前端 Inspector 解耦** (拆分 `MovieInspector` 为 5 个 Tab 子面板、抽离弹窗、拆解 Composables)
- **Phase D: 领域子系统抽离** (抽离 `internal/files` 安全写引擎与 `internal/artwork` 海报下载管理)
- **Phase E: 质量门禁与端到端回归** (全量测试、编译、静态分析与端到端验证)

---

## 6. 质量门禁与风险控制

1. **测试驱动安全验证**：
   - 每次重构提交必须保证 `rtk go test ./...` 100% 通过（当前已有 19 个测试套件）。
   - 每次前端重构必须通过 `rtk npm --prefix web run build`（包含 `vue-tsc` 类型检查）。
2. **零功能破坏原则 (Behavior Preservation)**：
   - 保证 `/api/v1/*` 现有 API 契约保持 100% 向后兼容。
   - 保留所有已验证的安全机制：防 SSRF 过滤、目录逃逸检查、原子重命名与 CSRF 双重校验。
3. **分步小步提交 (Small Reviewable Commits)**：
   - 严禁一次性大爆炸重构，按照阶段规划逐项交付与验证。
