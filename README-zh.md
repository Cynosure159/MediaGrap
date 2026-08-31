# MediaGrap

[English](README.md) | **简体中文**

MediaGrap 是一个轻量、自托管的媒体信息刮削与媒体库管理工具。它面向 NAS、家庭服务器和其他 Docker 环境，通过响应式 Web UI 管理电影、电视剧等媒体的元数据、NFO 文件与图片资源。

项目目标是在保留 tinyMediaManager、MediaElch 一类工具核心能力的同时，降低服务端常驻资源占用，并提供更适合浏览器、移动设备和容器环境的操作体验。

> 当前状态：已完成 Phase 2 的电影刮削、安全 NFO 写入以及经预览确认的 TMDb 海报/背景图下载。Phase 3 已开始：可识别常见电视剧集命名并提供独立电视剧库视图；电视剧刮削与 NFO 写入将在后续切片完成。

## 核心目标

- 使用 Go 实现低资源占用、高并发的后端服务。
- 提供简洁、响应式的 Vue 3 Web UI。
- 支持移动端浏览器和 PWA 安装。
- 支持简体中文与英文界面切换，并持久化用户语言偏好。
- 使用单个 Docker 容器完成标准部署。
- 支持 linux/amd64 与 linux/arm64。
- 支持 HTTP、HTTPS、SOCKS5 代理及 `NO_PROXY` 规则。
- 扫描、刮削、图片下载和文件操作均支持后台任务、进度显示、取消与重试。
- 安全读取和写入 Kodi 兼容的 NFO 与图片文件。
- 所有重命名、移动和覆盖操作均先生成预览，并进行路径与冲突检查。

## 技术方案

| 模块 | 方案 |
| --- | --- |
| 后端 | Go 模块化单体 |
| 前端 | Vue 3、TypeScript、Vite |
| Web 应用 | 响应式布局、PWA、明暗主题 |
| 数据库 | SQLite WAL，保留未来适配 PostgreSQL 的边界 |
| API | 版本化 JSON REST API + SSE 实时任务事件 |
| 元数据 | 可扩展 Provider Adapter，首期支持 TMDb |
| 图片 | 首期支持 TMDb，可选 Fanart.tv |
| 媒体分析 | 可选调用 `ffprobe` |
| 部署 | 多阶段构建、非 root 单容器、前端资源嵌入 Go 二进制 |

选择 Go 而非 Java 服务端，主要是为了获得更低的空闲内存占用、简单的静态编译与跨平台构建能力。当前业务负载主要来自目录遍历、网络请求、XML/JSON 处理和文件写入，Go 能在实现复杂度与性能之间取得较好的平衡。

## 规划功能

### MVP：电影管理

- 管理员初始化、登录和安全会话。
- 添加已挂载到容器内的媒体目录。
- 全量与增量扫描、文件名解析、已有 NFO 和图片识别。
- TMDb 搜索、详情与图片刮削。
- 元数据候选结果选择、字段编辑与手动字段保护。
- Kodi 电影 NFO 读取和写入。
- 海报、背景图、Logo 等资源选择与下载。
- 持久化后台任务、进度、取消、失败重试和重启恢复。
- 文件变更预览、安全写入、备份策略和审计记录。
- Docker Compose 部署、健康检查和代理配置。

### 后续版本

- 电视剧、季度、剧集和缺集检测。
- 模板化重命名、条件表达式、冲突检查和 Dry Run。
- 重复媒体与缺失元数据筛选。
- CSV 导出、定时扫描、Webhook 和 Kodi JSON-RPC 同步。
- 演唱会与音乐 Sidecar 元数据。
- 更多元数据提供商、多用户、API Token、PostgreSQL 和外部 Worker。

## 架构概览

```text
桌面浏览器 / 移动端 PWA
           │
      HTTP JSON + SSE
           │
┌──────────────────────────────────────────┐
│              MediaGrap                  │
│                                          │
│ Web/API 认证 媒体库 元数据 文件任务 管理 │
│                  │                       │
│          应用服务与领域模型              │
│             │           │                │
│          SQLite     Adapter 接口          │
│                       │     │             │
│                    文件系统 Provider      │
└──────────────────────────────────────────┘
             │                │
      配置与数据卷       挂载的媒体目录
                              │
                    TMDb / Fanart.tv 等服务
```

生产版本将 Vue 前端资源嵌入 Go 二进制，以一个进程提供 Web UI、API 和后台任务。项目初期采用模块化单体，避免微服务给家庭服务器带来的部署和资源开销；任务与 Provider 边界会保持清晰，以便未来按需拆分。

## 文件安全原则

媒体文件属于不可轻易恢复的用户数据，MediaGrap 将文件安全放在功能数量之前：

1. 扫描只建立索引，不自动触发刮削或修改文件。
2. 文件写入和重命名必须先生成不可变更的操作计划。
3. 执行前重新验证路径、权限、源目录边界和目标冲突。
4. NFO 使用同目录临时文件写入，再通过原子重命名替换。
5. 对已有 NFO 或图片资源的替换必须经过各自的写入预览与显式确认；软链接和非普通文件禁止替换。
6. 跨文件系统移动采用复制、校验、再删除源文件的流程。
7. 所有变更保留不含敏感信息的审计记录。

## Docker 部署形态

计划中的标准目录约定：

| 容器路径 | 用途 | 权限 |
| --- | --- | --- |
| `/config` | SQLite、配置、密钥、任务与审计数据 | 读写 |
| `/cache` | Provider 缓存、缩略图和临时下载 | 读写，可丢弃 |
| `/media/...` | 用户显式挂载的媒体库 | 可只读或读写 |

正式镜像将以非 root 用户运行，并提供 amd64、arm64 构建。媒体目录由管理员通过 Docker 挂载，Web UI 不会提供任意宿主机目录浏览能力。

## 开发路线

1. 搭建 Go、Vue PWA、SQLite migration、基础测试和容器构建。
2. 完成登录、媒体源管理、持久化任务和只读电影扫描。
3. 接入 TMDb、元数据编辑、Kodi NFO 与安全写入，发布 MVP。
4. 增加电视剧、季度和剧集管理。
5. 增加重命名、导出、定时任务与 Kodi 同步。
6. 根据需求扩展音乐、演唱会、Provider 和多用户能力。

详细里程碑参见[开发路线图](docs/roadmap.md)。

## 设计文档

- [产品范围与验收标准](docs/product-scope.md)
- [系统架构设计](docs/architecture.md)
- [技术选型与决策](docs/technology-decisions.md)
- [Docker 与部署设计](docs/deployment-design.md)
- [开发路线图](docs/roadmap.md)
- [MediaElch 参考分析](docs/mediaelch-reference.md)
- [Phase 0 基础实现](docs/phase-0-foundation.md)
- [Phase 1 只读媒体发现](docs/phase-1-discovery.md)
- [Phase 2 刮削与安全写入](docs/phase-2-scrape-and-safe-write.md)
- [Phase 3 电视剧发现](docs/phase-3-tv-discovery.md)
- [设置说明](docs/settings.md)
- [ADR 0001：基础技术栈](docs/adr/0001-foundation-stack.md)

## MediaElch 参考边界

MediaGrap 会参考 MediaElch 的公开功能、工作流、Kodi NFO 兼容方式和模块划分，但不会直接复制其 LGPL-3.0 源代码、测试、图标、翻译或其他受保护资源。

Provider 接入将优先使用官方 API，并分别核对 API Key、速率限制、缓存、署名和品牌展示要求。对于 Kodi NFO，将依据公开格式文档和独立编写的测试样例实现兼容。

## 项目文档约定

除仓库入口 `README.md`、`README-zh.md` 和协作指令 `AGENTS.md` 外，所有面向开发者和用户的项目文档都放在 `docs/` 目录中。公共行为、配置、部署方式或架构发生变化时，应同步更新对应文档。

## 许可证

项目的公开许可证尚未确定。在正式发布或引入第三方代码前，需要完成许可证选择和依赖合规审查。
