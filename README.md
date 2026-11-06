# MediaHelper

MediaHelper 是一个使用 Go 开发的媒体自动化工具，通过组件和流水线连接下载器、媒体识别、文件整理与元数据刮削，帮助管理电影、电视剧和动漫资源。

项目希望让用户从现成流程开始，逐步按需组合组件，并最终能够开发自己的扩展能力。目前已实现内置组件和流水线执行引擎，主要通过命令行工具、HTTP API 和代码级配置使用；面向普通用户的配置界面和独立第三方组件机制尚未完成。

## 当前能力

| 模块 | 已实现能力 |
| --- | --- |
| 下载器 | qBittorrent 接入，以及下载器相关流水线操作 |
| 媒体识别 | DeepSeek 识别媒体信息，可组合 TMDB 匹配器校验和补全结果 |
| 文件整理 | 本地媒体库归档，支持硬链接、复制、移动和符号链接 |
| 元数据刮削 | TMDB、豆瓣实现，支持获取影视资料、生成 NFO 和下载图片 |
| 流水线 | 顺序执行步骤、事件与 Cron 触发、条件过滤、参数表达式、步骤超时和失败后继续 |
| 执行记录 | SQLite 持久化流水线与步骤执行记录，查询执行历史和详情 |
| 使用入口 | HTTP API、qBittorrent Webhook、Swagger 和模块调试 CLI |

系统重启时会将遗留的未完成执行标记为中断；这不等于自动从失败步骤继续执行。

## 快速开始

### 1. 准备环境

使用满足 [go.mod](go.mod) 要求的 Go 工具链，当前声明为 `go 1.27.2`。运行涉及外部服务的功能时，还需要相应的服务地址、凭据和网络连接。

在项目根目录执行：

```bash
go mod download
mkdir -p bin data
go build -o bin/mediahelper ./cmd/mediahelper
go build -o bin/mdhp ./tools/mdhp
```

### 2. 配置并启动服务

首次使用时，将 `configs/config.example.json` 复制为 `configs/config.json`。已有配置时直接编辑，无需覆盖。

系统配置使用 JSON，只负责 HTTP 监听地址和数据库位置：

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 9090
  },
  "database": {
    "dsn": "data/mediahelper.db"
  }
}
```

```bash
./bin/mediahelper -config configs/config.json
```

启动后可访问：

- [健康检查](http://localhost:9090/health)
- [服务状态](http://localhost:9090/api/v1/status)
- [Swagger API 文档](http://localhost:9090/swagger/index.html)

**启动服务不代表已经完成媒体整理配置。** 组件配置从数据库读取；新数据库中的组件列表为空，即使系统创建了默认流水线，也需要先配置并注册它引用的组件。

### 3. 先通过 CLI 试用单项功能

`mdhp` 可以直接调用模块，无需先配置服务端流水线。以下示例使用 Bash / Zsh 语法，路径和凭据需要替换为实际值。

识别媒体目录，按需启用 TMDB 匹配：

```bash
./bin/mdhp parser parse \
  --apikey "$DEEPSEEK_API_KEY" \
  --path /downloads/Example.Show.S01 \
  --matcher tmdb \
  --tmdb-key "$TMDB_API_KEY"
```

将一个电影文件硬链接到媒体库：

```bash
./bin/mdhp organizer organize \
  --movie-dir /media/Movies \
  --link-mode hardlink \
  --media-type movie \
  --title '盗梦空间' \
  --year 2010 \
  --files '[{"file_path":"/downloads/Inception.2010.mkv"}]'
```

为已有媒体目录补全 TMDB 资料：

```bash
./bin/mdhp scraper scrape \
  --apikey "$TMDB_API_KEY" \
  --dir '/media/Movies/盗梦空间 (2010)' \
  --media-type movie \
  --tmdb-id 27205
```

整理命令会实际处理文件；刮削命令会写入 NFO、下载图片。硬链接要求源文件与目标目录位于同一文件系统。

查看完整参数：

```bash
./bin/mdhp --help
./bin/mdhp parser parse --help
./bin/mdhp organizer organize --help
./bin/mdhp scraper scrape --help
```

## 流水线如何工作

流水线由触发器和有序步骤组成。每个步骤引用一个已注册的组件，传入参数，并将输出提供给后续步骤。

```text
事件 / 定时任务 / 手动执行
          ↓
      条件检查
          ↓
   步骤 1 → 步骤 2 → 步骤 3
          ↓
   保存执行状态与步骤结果
```

没有任何流水线时，服务会创建默认媒体整理流程，实际执行顺序为：

```text
识别媒体 → 整理文件 → 刮削资料 → 给下载任务添加完成标签
```

默认流程引用 `parser`、`organizer`、`scraper`、`downloader` 四个组件 ID，并监听 `webhook:qbittorrent` 和 `sync:manual` 事件。`sync:manual` 是内部事件名，当前没有对应的 `/api/v1/sync` HTTP 路由。

默认流程的刮削和标签步骤允许失败后继续，因此判断整理结果时应同时查看步骤详情。

### 参数与条件

步骤参数可以引用触发事件及前序步骤的输出，例如：

```json
{
  "id": "organizer",
  "name": "整理媒体文件",
  "component": "organizer",
  "params": {
    "action": "organize",
    "media_type": "{{ steps.parser.media_type }}",
    "title": "{{ steps.parser.title }}",
    "year": "{{ steps.parser.year }}",
    "files": "{{ steps.parser.files }}"
  }
}
```

这是步骤定义片段，不是可直接提交的 HTTP 请求。`steps.parser` 指前面 ID 为 `parser` 的步骤输出，`event` 指本次触发携带的数据。独立表达式保留结果类型，例如 `files` 可以传递文件列表。

步骤还支持 `when` 条件、`continue_on_error` 和 `timeout_seconds`。条件用于决定步骤是否执行，目前不应将其理解为完整的可视化分支或并行编排能力。

## 配置方式

当前配置分为三部分：

| 配置 | 存储位置 | 用途 |
| --- | --- | --- |
| 系统配置 | `configs/config.json` | 监听地址、端口、SQLite 路径 |
| 组件配置 | 数据库 | 下载器、解析器、刮削器和整理器实例及其参数 |
| 流水线定义 | 数据库 | 触发器、条件、步骤和参数映射 |

组件配置由 `downloaders`、`parsers`、`scrapers`、`organizers` 四个部分组成，每个实例包含 `enabled`、`id`、`type`、`name`、`description` 和服务特定的 `detail`。

当前尚未提供组件配置和流水线创建、编辑的 HTTP 接口。开发者可通过 [配置模块](internal/config/compcfg.go) 的保存函数及 [流水线 Repository](internal/pipeline/storage.go) 写入定义；主程序在启动时加载并注册组件，修改组件配置后需重启。CLI 中传入的参数不会自动保存为服务端组件配置。

## HTTP API

以下为当前服务注册的路由：

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET | `/health` | 健康检查 |
| GET | `/api/v1/status` | 服务信息及已注册组件数量 |
| GET | `/api/v1/history` | 分页查询执行历史，支持 `limit`、`offset` |
| POST | `/api/v1/webhook/qbittorrent` | 接收下载完成事件 |
| GET | `/api/v1/pipelines` | 查询流水线列表 |
| GET | `/api/v1/pipelines/:id` | 查询流水线定义 |
| POST | `/api/v1/pipelines/:id/execute` | 手动执行流水线，请求 JSON 作为触发数据 |
| GET | `/api/v1/executions/:id` | 查询执行详情及步骤记录 |

### qBittorrent 下载完成通知

完成组件配置后，可在 qBittorrent 的“下载完成时运行外部程序”中配置以下命令（Linux 环境示例）：

```bash
curl -X POST http://mediahelper:9090/api/v1/webhook/qbittorrent -H 'Content-Type: application/json' -d '{"hash":"%I","name":"%N","category":"%L","content_path":"%F","save_path":"%D"}'
```

将 `mediahelper:9090` 替换为 qBittorrent 所在环境可以访问的服务地址。事件中的文件路径也必须能被 MediaHelper 正确访问，部署时应核对容器挂载和路径配置。

Webhook 返回接收成功仅表示事件已被接收，最终结果需通过执行历史查询。

## 开发与扩展

### 当前扩展方式

流水线组件实现以下 Go 接口，并注册到 `ComponentRegistry`：

```go
type Component interface {
    ID() string
    Name() string
    Description() string
    Execute(ctx context.Context, sCtx *StepContext) (map[string]any, error)
}
```

输入参数通过 `StepContext` 获取，执行结果以 `map[string]any` 返回。项目还提供 `NewFuncComponent`，便于用函数构造组件。

目前扩展属于源码级集成：需要在项目中实现或接入组件，加入注册逻辑并重新构建。相关包位于 Go 的 `internal` 目录下，尚未作为独立公共 SDK 发布。现有工厂注册机制也不等于运行时安装第三方插件。

### 后续方向

以下是拟演进方向，尚未作为完整功能提供：

- **降低首次使用门槛**：提供场景模板、简化配置、整理预览与识别纠错。
- **开放流程配置**：提供创建、编辑、导入和导出入口，支持用户组合已有能力。
- **明确组件契约**：声明输入输出结构、配置项和版本，尽早发现参数不匹配。
- **开放外部扩展**：探索通过外部进程 JSON 协议或 HTTP 接入用户编写的动作，减少对核心源码的修改。
- **完善执行交互**：逐步支持人工确认、针对失败步骤的恢复，以及更直观的业务状态。

目标是让普通用户使用模板、进阶用户编排流程、开发者增加能力，共用同一套执行引擎。

## 项目结构

```text
cmd/mediahelper/       服务启动入口
configs/               系统配置示例
internal/config/       系统与组件配置
internal/database/     SQLite 初始化与数据模型
internal/downloader/   下载器组件及实现
internal/parser/       媒体识别与匹配
internal/organizer/    文件整理
internal/scraper/      元数据刮削与 NFO 生成
internal/pipeline/     组件注册、表达式、调度与执行引擎
internal/server/       HTTP API 和 Webhook
tools/mdhp/           模块调试 CLI
tools/qbt/            qBittorrent 命令行工具
tools/mdgen/          辅助生成工具
docs/                  Swagger 文档
```

## 验证与测试

```bash
go test ./...
go vet ./...
```

涉及第三方服务的测试可能需要凭据和网络环境，请结合具体测试文件准备配置。生成 Swagger 文档时，可以使用仓库中的 VS Code 任务；API 的实际可用范围以 [路由实现](internal/server/server.go) 为准。
