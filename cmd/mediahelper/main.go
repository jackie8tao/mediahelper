package main

// @title MediaHelper API
// @version 1.0
// @description MediaHelper 多媒体自动化管理系统 REST API 文档
// @termsOfService https://github.com/jackie8tao/mediahelper

// @contact.name Jackie Tao
// @contact.url https://github.com/jackie8tao/mediahelper

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:9090
// @BasePath /

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/database"
	"github.com/jackie8tao/mediahelper/internal/downloader"
	_ "github.com/jackie8tao/mediahelper/internal/downloader/aria2"
	_ "github.com/jackie8tao/mediahelper/internal/downloader/qbittorrent"
	_ "github.com/jackie8tao/mediahelper/internal/downloader/transmission"
	"github.com/jackie8tao/mediahelper/internal/organizer"
	"github.com/jackie8tao/mediahelper/internal/pathmapper"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
	"github.com/jackie8tao/mediahelper/internal/plugin"
	"github.com/jackie8tao/mediahelper/internal/scraper"
	"github.com/jackie8tao/mediahelper/internal/scraper/tmdb"
	"github.com/jackie8tao/mediahelper/internal/server"
	"github.com/jackie8tao/mediahelper/internal/service"
	"github.com/jackie8tao/mediahelper/plugins/jellyfin"
	"github.com/jackie8tao/mediahelper/plugins/notify"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	// 初始化结构化日志
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("Starting MediaHelper...", "config", *configPath)

	// 1. 加载本地引导配置 (用于获取数据库连接 DSN 等参数)
	fileCfg, err := config.LoadConfig(*configPath)
	if err != nil {
		slog.Warn("Failed to load config file, using default configuration", "error", err)
		fileCfg = config.DefaultConfig()
	}

	// 2. 初始化 SQLite 数据库存储 (自动建表与 WAL 模式开启)
	db, err := database.InitDB(fileCfg.Database.DSN)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
	}

	// 3. 优先从数据库载入最新系统配置 (若数据库无记录则自动将本地配置初始化入库)
	cfg := fileCfg
	if db != nil {
		dbCfg, err := config.LoadOrInit(db, fileCfg)
		if err != nil {
			slog.Warn("Failed to load config from database, using file configuration", "error", err)
		} else {
			cfg = dbCfg
			slog.Info("System configuration loaded successfully from database")
		}
	}

	// 4. 初始化流水线组件注册中心并载入组件
	compRegistry := pipeline.NewComponentRegistry()
	if _, err := pathmapper.RegisterComponent(cfg, compRegistry); err != nil {
		slog.Error("Failed to register path mapper component", "error", err)
	}

	dlComps, err := downloader.RegisterComponents(cfg, compRegistry)
	if err != nil {
		slog.Error("Failed to register downloader components", "error", err)
	}

	var dl downloader.Downloader
	if len(dlComps) > 0 {
		dl = dlComps[0].Downloader()
		// 异步尝试连通性校验，避免阻塞主服务启动
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := dl.TestConnection(ctx); err != nil {
				slog.Warn("Downloader connection test failed (will retry when running)", "error", err)
			}
		}()
	}

	// 3. 初始化 TMDB 刮削器
	var scraperSvc scraper.Scraper
	if cfg.TMDB.APIKey != "" {
		tmdbCli, err := tmdb.NewClient(cfg.TMDB)
		if err != nil {
			slog.Error("Failed to initialize TMDB client", "error", err)
		} else {
			scraperSvc = scraper.NewService(cfg.TMDB, tmdbCli)
			slog.Info("TMDB scraper service initialized successfully", "language", cfg.TMDB.Language)
		}
	} else {
		slog.Warn("TMDB api_key is empty, metadata scraping is disabled")
	}

	// 4. 初始化文件整理引擎
	org := organizer.NewOrganizer(*cfg, scraperSvc)

	// 5. 初始化插件管理器并载入插件
	pluginMgr := plugin.NewManager()
	_ = pluginMgr.Register(jellyfin.New())
	_ = pluginMgr.Register(notify.New())

	initPluginCtx, cancelInitPlugin := context.WithTimeout(context.Background(), 10*time.Second)
	if err := pluginMgr.InitAll(initPluginCtx, cfg.Plugins); err != nil {
		slog.Error("Failed to initialize plugins", "error", err)
	}
	cancelInitPlugin()
	_ = pluginMgr.StartAll(context.Background())

	// 6. 初始化核心流水线协调器
	pipelineSvc := service.NewPipeline(cfg, dl, scraperSvc, org, pluginMgr)

	// 7. 启动流水线引擎 (负责自愈与 Cron 调度)
	var pipelineEngine *pipeline.Engine
	if db != nil {
		pipelineEngine = pipeline.NewEngine(db, compRegistry)
		if err := pipelineEngine.Start(context.Background()); err != nil {
			slog.Error("Failed to start pipeline engine", "error", err)
		}
	}

	// 8. 启动后台定时同步 Worker
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	defer cancelWorker()

	if dl != nil && cfg.QBittorrent.Enabled {
		go pipelineSvc.StartBackgroundWorker(workerCtx)
	}

	// 9. 启动 HTTP API & Webhook Web 服务
	srv := server.New(cfg, pipelineSvc)
	go func() {
		if err := srv.Start(); err != nil {
			slog.Error("HTTP Server failed to start", "error", err)
		}
	}()

	slog.Info("MediaHelper running successfully", "port", cfg.Server.Port)

	// 10. 监听系统信号优雅关停
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down MediaHelper gracefully...")

	// 停止流水线引擎
	if pipelineEngine != nil {
		pipelineEngine.Stop()
	}

	// 取消后台 Worker
	cancelWorker()

	// 停止 HTTP 服务
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP Server forced to shutdown", "error", err)
	}

	// 停止所有插件
	_ = pluginMgr.StopAll(shutdownCtx)

	slog.Info("MediaHelper exited cleanly")
}
