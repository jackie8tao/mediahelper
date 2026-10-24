package main

// @title MediaHelper API
// @version 1.0
// @description MediaHelper Automated Media Management System REST API Documentation
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
	"github.com/jackie8tao/mediahelper/internal/parser"
	"github.com/jackie8tao/mediahelper/internal/pathmapper"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
	"github.com/jackie8tao/mediahelper/internal/plugin"
	"github.com/jackie8tao/mediahelper/internal/scraper"
	"github.com/jackie8tao/mediahelper/internal/server"
	"github.com/jackie8tao/mediahelper/plugins/jellyfin"
	"github.com/jackie8tao/mediahelper/plugins/notify"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	// Initialize structured logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("Starting MediaHelper...", "config", *configPath)

	// 1. Load system configuration from file
	sysCfg, err := config.LoadSystemConfigFromFile(*configPath)
	if err != nil {
		slog.Warn("Failed to load config file, using default configuration", "error", err)
		sysCfg = config.DefaultSystemConfig()
	}

	// 2. Initialize SQLite database storage (auto-migration and WAL mode enabled)
	db, err := database.InitDB(sysCfg.Database.DSN)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1) // Critical error, cannot continue without database
	}

	// 3. Load component configuration from database
	compCfg, err := config.LoadComponentConfigFromDB(db)
	if err != nil {
		slog.Warn("Failed to load component config, using default configuration", "error", err)
		compCfg = config.DefaultComponentConfig()
	}

	// 4. Initialize pipeline component registry and register components
	compRegistry := pipeline.NewComponentRegistry()
	if _, err := pathmapper.RegisterComponent(compCfg, compRegistry); err != nil {
		slog.Error("Failed to register path mapper component", "error", err)
	}
	if _, err := downloader.RegisterComponents(compCfg, compRegistry); err != nil {
		slog.Error("Failed to register downloader components", "error", err)
	}
	if _, err := parser.RegisterComponents(compCfg, compRegistry); err != nil {
		slog.Error("Failed to register parser components", "error", err)
	}
	if _, err := scraper.RegisterComponents(compCfg, compRegistry); err != nil {
		slog.Error("Failed to register scraper components", "error", err)
	}
	if _, err := organizer.RegisterComponents(compCfg, compRegistry); err != nil {
		slog.Error("Failed to register organizer components", "error", err)
	}

	// 5. Initialize plugin manager and load plugins
	pluginMgr := plugin.NewManager()
	_ = pluginMgr.Register(jellyfin.New())
	_ = pluginMgr.Register(notify.New())

	initPluginCtx, cancelInitPlugin := context.WithTimeout(context.Background(), 10*time.Second)
	if err := pluginMgr.InitAll(initPluginCtx, compCfg.Plugins); err != nil {
		slog.Error("Failed to initialize plugins", "error", err)
	}
	cancelInitPlugin()
	_ = pluginMgr.StartAll(context.Background())

	// 6. Start pipeline engine (seed default pipelines, self-healing recovery, and cron scheduling)
	pipelineEngine := pipeline.NewEngine(db, compRegistry)
	if err := pipeline.SeedDefaultPipelines(context.Background(), pipelineEngine.Repo()); err != nil {
		slog.Warn("Failed to seed default pipelines", "error", err)
	}
	if err := pipelineEngine.Start(context.Background()); err != nil {
		slog.Error("Failed to start pipeline engine", "error", err)
	}

	// 7. Start HTTP API & Webhook web server
	srv := server.New(sysCfg, pipelineEngine, compRegistry, pluginMgr)
	go func() {
		if err := srv.Start(); err != nil {
			slog.Error("HTTP Server failed to start", "error", err)
		}
	}()

	slog.Info("MediaHelper running successfully", "port", sysCfg.Server.Port)

	// 8. Listen for system termination signals for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down MediaHelper gracefully...")

	// Stop pipeline engine
	if pipelineEngine != nil {
		pipelineEngine.Stop()
	}

	// Stop HTTP server
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP Server forced to shutdown", "error", err)
	}

	// Stop all plugins
	_ = pluginMgr.StopAll(shutdownCtx)

	slog.Info("MediaHelper exited cleanly")
}
