# MediaHelper

MediaHelper 是一个基于 Go 语言开发的高性能多媒体自动化管理系统，专门用于将下载器（qBittorrent 等）下载的电影、电视剧、动漫资源按照 Plex、Jellyfin、Emby 的规范进行元数据刮削、硬链接整理与重命名归档。

## 核心特性

- **高效轻量**：Go 语言原生开发，单二进制部署，内存占用低（仅数十 MB），适合 NAS 及微型服务器。
- **纯 Go 零 CGO 数据库存储 (SQLite + GORM)**：
  - 采用纯 Go 实现的 SQLite 驱动（`github.com/glebarez/sqlite`），彻底告别 CGO 与 GCC 编译器依赖，Windows/Linux 任意环境跨平台开箱即用。
  - 默认开启 WAL 预写日志模式与 5000ms 忙等待超时，保障并发安全。
  - 自动迁移创建 `TorrentRecord`（种子追踪）、`TaskHistory`（归档历史）、`MediaMapping`（剧名纠错映射）三张核心数据表。
- **全自动双模触发**：
  - **Webhook 秒级加速**：支持 qBittorrent 下载完成后调用外部命令或 API 实时触发整理。
  - **定时轮询底座保障**：后台 Worker 定时自动拉取已完成任务，掉线任务或重启后自动查漏补缺。
- **路径映射引擎**：完美解决 Docker / NAS 跨容器和跨系统路径映射难题。
- **智能文件名解析 (Parser)**：
  - 欧美 Scene/P2P 标准格式解析。
  - 国内 PT 种子中英混排自动剥离（如分离出纯中文片名提升 TMDB 命中率）。
  - 中文季集（“第二季”、“第05集”）与动漫连字符格式（` - 28 `、`[28]`）自适应识别。
- **TMDB 刮削与 NFO 规范生成 (Scraper)**：
  - 对接 TheMovieDatabase (TMDB)，抓取电影、电视剧、分集详情、演职人员及海报。
  - 支持 HTTP/SOCKS5 网络代理与自定义反代 API 域名加速。
  - 生成标准符合 Jellyfin / Plex / Emby / Kodi 规范的 `movie.nfo`、`tvshow.nfo` 及单集 NFO。
- **媒体库整理引擎 (Organizer)**：
  - 自动创建影视层级目录（电影：`{Title} ({Year})/{Title} ({Year}) - {Resolution}.mkv`，电视剧：`{Title} ({Year})/Season {Season:02d}/{Title} - S{Season:02d}E{Episode:02d} - {EpisodeTitle}.mkv`）。
  - 支持 **硬链接 (Hardlink, 默认且强烈推荐)**、复制 (Copy)、移动 (Move)、符号链接 (Symlink)。
  - 伴生字幕（`.zh.srt`、`.ass` 等）自动跟随重命名与硬链接。
  - 自动跨平台非法字符过滤（处理 Windows/Linux 中的 `:`、`?`、`"` 等特殊字符）。
- **生命周期与钩子驱动的插件机制 (Plugin System)**：
  - 零外部黑盒依赖，完备的生命周期管理（`Init`、`Start`、`Stop`）与 Panic 崩溃隔离保护。
  - 支持 `TorrentCompletedHook`、`PostScrapeHook`、`PostOrganizeHook` 等多阶段扩展。
  - 内置开箱即用插件：
    - **Jellyfin 媒体库刷新**：影视归档后自动通知 Jellyfin 刷新媒体库。
    - **Webhook 消息通知**：影视归档后向企业微信、钉钉、飞书或自定义 Webhook 发送卡片通知。

---

## 快速配置

编辑 `configs/config.yaml`：

```yaml
server:
  host: "0.0.0.0"
  port: 9090

# SQLite 数据库存储路径 (纯 Go 驱动，无需安装 GCC)
database:
  dsn: "data/mediahelper.db"

qbittorrent:
  enabled: true
  url: "http://127.0.0.1:8080"
  username: "admin"
  password: "adminadmin"
  categories:
    - "movie"
    - "tv"
    - "anime"
  auto_tag: true
  success_tag: "MediaHelper:Done"
  fail_tag: "MediaHelper:Error"
  poll_interval: 30
  path_mappings:
    - remote: "/downloads"
      local: "D:/Downloads"

tmdb:
  api_key: "your_tmdb_api_key_here"
  base_url: "https://api.themoviedb.org/3"
  image_base_url: "https://image.tmdb.org/t/p/original"
  language: "zh-CN"
  proxy: ""
  generate_nfo: true
  download_images: true

library:
  movie_dir: "D:/Media/Movies"
  tv_dir: "D:/Media/TV"
  anime_dir: "D:/Media/Anime"
  link_mode: "hardlink"
  min_file_size: 52428800

plugins:
  enabled: true
  items:
    jellyfin_refresh:
      enabled: false
      server_url: "http://127.0.0.1:8096"
      api_key: "your_jellyfin_api_key"

    webhook_notify:
      enabled: false
      webhook_url: "https://your-webhook-endpoint.com/api"
```

---

## qBittorrent 实时 Webhook 配置

在 qBittorrent WebUI 中打开 `设置 (Tools/Options) -> 下载 (Downloads)`：
勾选 **“种子完成时运行外部程序” (Run external program on torrent completion)**，填入命令：

### Windows 环境：
```cmd
curl -X POST http://127.0.0.1:9090/api/v1/webhook/qbittorrent -H "Content-Type: application/json" -d "{\"hash\":\"%I\", \"name\":\"%N\", \"category\":\"%L\", \"content_path\":\"%F\", \"save_path\":\"%D\"}"
```

### Linux / Docker 环境：
```bash
curl -X POST http://mediahelper:9090/api/v1/webhook/qbittorrent \
  -H "Content-Type: application/json" \
  -d '{"hash":"%I", "name":"%N", "category":"%L", "content_path":"%F", "save_path":"%D"}'
```

---

## API 接口与 Swagger / Postman 联调

本项目支持自动生成标准 OpenAPI / Swagger 规范文档，并可直接无缝导入 Postman。

### 1. 生成 Swagger 文档 (VS Code Task)
在 VS Code 中按 `Ctrl + Shift + P` -> 输入 `Tasks: Run Task` -> 选择 **`Generate Swagger Docs`**（或按快捷键调用），即可自动生成文档到 `docs/` 目录：
- `docs/swagger.json` (OpenAPI 2.0 JSON)
- `docs/swagger.yaml` (OpenAPI 2.0 YAML)
- `docs/docs.go` (Go 嵌入代码)

### 2. 导入到 Postman 进行调试
有以下两种最便捷的导入方式：
* **方式 A (文件导入 - 离线可用)**：
  打开 Postman -> 点击左上角 **Import** -> 将项目目录中的 `docs/swagger.json` 直接拖入窗口，Postman 将自动解析生成包含所有 API 端点的完整 Collection。
* **方式 B (在线实时同步)**：
  启动 MediaHelper 后，在 Postman 中点击 **Import** -> 切换至 **Link** -> 输入 `http://localhost:9090/swagger/doc.json` 即可直接拉取同步。

### 3. 在线 Swagger UI
服务启动后，直接在浏览器中打开：
👉 **http://localhost:9090/swagger/index.html** 即可查看交互式 API 文档并在线发起请求测试。

---

## API 接口列表

| 路径 | 方法 | 说明 |
| :--- | :--- | :--- |
| `/health` | `GET` | 服务探活接口 |
| `/api/v1/status` | `GET` | 查询下载器连接状态、库路径、TMDB 启用状态 |
| `/api/v1/downloader/torrents` | `GET` | 查询下载器当前所有种子任务 (支持 `?filter=all/downloading/completed` 及 `?category=xxx`) |
| `/api/v1/history` | `GET` | 查询最近整理历史与状态报告 |
| `/api/v1/plugins` | `GET` | 查询所有已加载插件、启用状态及其挂载的钩子清单 |
| `/api/v1/sync` | `POST` | 手动触发全量扫描未整理任务 |
| `/api/v1/webhook/qbittorrent` | `POST` | qBittorrent 任务完成实时通知接口 |

---

## 运行与编译

```bash
# 运行全部单元测试
go test -v ./...

# 编译为独立执行文件 (纯 Go 无 CGO 依赖，支持全平台交叉编译)
go build -o mediahelper.exe ./cmd/mediahelper

# 启动服务
./mediahelper.exe -config configs/config.yaml
```
