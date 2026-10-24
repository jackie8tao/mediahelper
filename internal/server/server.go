package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackie8tao/mediahelper/docs"
	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/downloader"
	"github.com/jackie8tao/mediahelper/internal/pathmapper"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
	"github.com/jackie8tao/mediahelper/internal/plugin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Server manages HTTP Web, REST APIs, and Webhook endpoints.
type Server struct {
	cfg          *config.Config
	engine       *pipeline.Engine
	compRegistry *pipeline.ComponentRegistry
	pluginMgr    *plugin.Manager
	pathMapper   *pathmapper.PathMapper
	router       *gin.Engine
	httpServer   *http.Server
}

// New creates a new Server instance.
func New(
	cfg *config.Config,
	engine *pipeline.Engine,
	compRegistry *pipeline.ComponentRegistry,
	pluginMgr *plugin.Manager,
) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	if compRegistry == nil {
		compRegistry = pipeline.NewComponentRegistry()
	}

	s := &Server{
		cfg:          cfg,
		engine:       engine,
		compRegistry: compRegistry,
		pluginMgr:    pluginMgr,
		pathMapper:   pathmapper.NewPathMapper(cfg.GetPathMappings()),
		router:       router,
	}

	s.setupRoutes()
	return s
}

// setupRoutes registers all HTTP API routes and middleware.
func (s *Server) setupRoutes() {
	s.router.GET("/health", s.handleHealth)
	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := s.router.Group("/api/v1")
	{
		v1.GET("/status", s.handleStatus)
		v1.GET("/history", s.handleHistory)
		v1.GET("/plugins", s.handlePlugins)
		v1.GET("/downloader/torrents", s.handleDownloaderTorrents)
		v1.POST("/sync", s.handleSync)
		v1.POST("/webhook/qbittorrent", s.handleQBWebhook)

		// Pipeline management endpoints
		v1.GET("/pipelines", s.handleListPipelines)
		v1.GET("/pipelines/:id", s.handleGetPipeline)
		v1.POST("/pipelines/:id/execute", s.handleExecutePipeline)
		v1.GET("/executions/:id", s.handleGetExecution)
	}
}

// Start launches the HTTP server listening on the configured address.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	slog.Info("HTTP Web & Webhook Server listening", "addr", addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// handleHealth returns service health and current unix timestamp.
// @Summary Service Health Check
// @Description Returns the service running status and current timestamp
// @Tags System
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /health [get]
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
	})
}

// handleStatus returns overall system status.
// @Summary System Status
// @Description Returns system configuration overview, components count, and library paths
// @Tags System
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/status [get]
func (s *Server) handleStatus(c *gin.Context) {
	var compCount int
	if s.compRegistry != nil {
		compCount = len(s.compRegistry.List())
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"app":               "MediaHelper",
			"version":           "1.0.0",
			"components_count":  compCount,
			"link_mode":         s.cfg.Library.LinkMode,
			"movie_library_dir": s.cfg.Library.MovieDir,
			"tv_library_dir":    s.cfg.Library.TVDir,
		},
	})
}

// handleHistory returns recent pipeline execution history.
// @Summary Query Execution History
// @Description Returns recent pipeline execution records
// @Tags Pipeline
// @Produce json
// @Param limit query int false "Max records limit (default: 50)"
// @Param offset query int false "Offset for pagination"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/history [get]
func (s *Server) handleHistory(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	if s.engine == nil || s.engine.Repo() == nil {
		c.JSON(http.StatusOK, gin.H{
			"code":  0,
			"total": 0,
			"data":  []any{},
		})
		return
	}

	executions, total, err := s.engine.Repo().ListExecutions(0, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":  500,
			"error": fmt.Sprintf("query history failed: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":  0,
		"total": total,
		"data":  executions,
	})
}

// handlePlugins returns loaded plugin information.
// @Summary List Plugins
// @Description Returns information on all registered plugins and their enabled status
// @Tags Plugins
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/plugins [get]
func (s *Server) handlePlugins(c *gin.Context) {
	var list []interface{}
	if s.pluginMgr != nil {
		for _, info := range s.pluginMgr.ListPlugins() {
			list = append(list, info)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"code":  0,
		"total": len(list),
		"data":  list,
	})
}

// handleDownloaderTorrents returns active torrent tasks from registered downloaders.
// @Summary Query Downloader Tasks
// @Description Retrieves current torrent tasks from registered downloader components
// @Tags Downloader
// @Produce json
// @Param filter query string false "State filter (all, downloading, completed, paused)" default(all)
// @Param category query string false "Category filter"
// @Param downloader query string false "Specific downloader component ID (defaults to 'downloader')"
// @Success 200 {object} map[string]interface{}
// @Failure 503 {object} map[string]interface{}
// @Router /api/v1/downloader/torrents [get]
func (s *Server) handleDownloaderTorrents(c *gin.Context) {
	dlID := c.DefaultQuery("downloader", "downloader")
	comp, ok := s.compRegistry.Get(dlID)
	if !ok {
		// Fallback check for any registered downloader component
		for _, regComp := range s.compRegistry.List() {
			if strings.HasPrefix(regComp.ID(), "downloader") {
				comp = regComp
				ok = true
				break
			}
		}
	}

	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":  503,
			"error": "no downloader component is registered or enabled",
		})
		return
	}

	dlComp, ok := comp.(*downloader.DownloaderComponent)
	if !ok || dlComp.Downloader() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":  503,
			"error": "component is not an active downloader client",
		})
		return
	}

	filter := c.DefaultQuery("filter", "all")
	category := c.Query("category")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	torrents, err := dlComp.Downloader().GetAllTorrents(ctx, filter, category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":  500,
			"error": fmt.Sprintf("fetch torrents failed: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":       0,
		"total":      len(torrents),
		"downloader": dlComp.ID(),
		"filter":     filter,
		"category":   category,
		"data":       torrents,
	})
}

// handleSync triggers an immediate full scan and organization of completed torrents.
// @Summary Manually Trigger Full Sync
// @Description Scans downloaders for completed untagged torrents and triggers pipeline execution
// @Tags Pipeline
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/sync [post]
func (s *Server) handleSync(c *gin.Context) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		slog.Info("Starting manual sync across registered downloaders...")

		// Scan all registered downloaders
		var dlComps []*downloader.DownloaderComponent
		for _, comp := range s.compRegistry.List() {
			if d, ok := comp.(*downloader.DownloaderComponent); ok && d.Downloader() != nil {
				dlComps = append(dlComps, d)
			}
		}

		totalTriggered := 0
		for _, d := range dlComps {
			dlCfg := d.Config()
			torrents, err := d.Downloader().GetCompletedTorrents(ctx, dlCfg.Categories)
			if err != nil {
				slog.Warn("Failed to fetch completed torrents for downloader", "id", d.ID(), "error", err)
				continue
			}

			successTag := dlCfg.SuccessTag
			if successTag == "" {
				successTag = "MediaHelper:Done"
			}

			for _, t := range torrents {
				// Check if already tagged
				alreadyDone := false
				for _, tag := range t.Tags {
					if strings.EqualFold(strings.TrimSpace(tag), successTag) {
						alreadyDone = true
						break
					}
				}
				if alreadyDone {
					continue
				}

				payload := map[string]any{
					"torrent":      t,
					"name":         t.Name,
					"hash":         t.Hash,
					"content_path": t.ContentPath,
					"save_path":    t.SavePath,
					"category":     t.Category,
					"downloader":   d.ID(),
				}

				if s.engine != nil {
					_, err := s.engine.TriggerEvent(ctx, "sync:manual", payload)
					if err != nil {
						slog.Error("Trigger sync pipeline failed", "torrent", t.Name, "error", err)
					} else {
						totalTriggered++
					}
				}
			}
		}

		slog.Info("Manual sync triggered finished", "triggered_count", totalTriggered)
	}()

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "Manual sync triggered successfully in background",
	})
}

// QBWebhookPayload defines the request body received from qBittorrent completion hooks.
type QBWebhookPayload struct {
	Hash        string `json:"hash" form:"hash"`
	Name        string `json:"name" form:"name"`
	Category    string `json:"category" form:"category"`
	ContentPath string `json:"content_path" form:"content_path"`
	SavePath    string `json:"save_path" form:"save_path"`
}

// handleQBWebhook receives real-time download completion events and dispatches pipeline triggers.
// @Summary qBittorrent Download Completion Webhook
// @Description Webhook invoked by qBittorrent upon torrent completion to trigger pipeline execution
// @Tags Webhook
// @Accept json,x-www-form-urlencoded
// @Produce json
// @Param payload body QBWebhookPayload false "Torrent metadata payload"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/webhook/qbittorrent [post]
func (s *Server) handleQBWebhook(c *gin.Context) {
	var payload QBWebhookPayload
	if err := c.ShouldBind(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":  400,
			"error": fmt.Sprintf("invalid payload: %v", err),
		})
		return
	}

	if payload.Name == "" && payload.Hash == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":  400,
			"error": "both name and hash cannot be empty",
		})
		return
	}

	eventData := map[string]any{
		"hash":         payload.Hash,
		"name":         payload.Name,
		"category":     payload.Category,
		"content_path": payload.ContentPath,
		"save_path":    payload.SavePath,
	}

	slog.Info("Received qBittorrent webhook event, dispatching pipeline trigger...", "hash", payload.Hash, "name", payload.Name)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if s.engine != nil {
			_, err := s.engine.TriggerEvent(ctx, "webhook:qbittorrent", eventData)
			if err != nil {
				slog.Error("Webhook pipeline trigger failed", "hash", payload.Hash, "error", err)
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "Webhook event accepted",
		"data": gin.H{
			"hash": payload.Hash,
			"name": payload.Name,
		},
	})
}

// handleListPipelines returns all defined workflows.
// @Summary List Defined Pipelines
// @Description Returns list of all configured pipelines
// @Tags Pipeline
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/pipelines [get]
func (s *Server) handleListPipelines(c *gin.Context) {
	if s.engine == nil || s.engine.Repo() == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "total": 0, "data": []any{}})
		return
	}

	pipelines, err := s.engine.Repo().ListPipelines(false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":  0,
		"total": len(pipelines),
		"data":  pipelines,
	})
}

// handleGetPipeline retrieves a specific pipeline workflow definition by ID.
// @Summary Get Pipeline by ID
// @Description Returns detailed configuration of a pipeline
// @Tags Pipeline
// @Produce json
// @Param id path int true "Pipeline ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/pipelines/{id} [get]
func (s *Server) handleGetPipeline(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "error": "invalid pipeline ID"})
		return
	}

	p, err := s.engine.Repo().GetPipeline(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "error": "pipeline not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": p})
}

// handleExecutePipeline manually triggers a specific pipeline by ID.
// @Summary Manually Trigger Pipeline
// @Description Manually executes a pipeline workflow by ID
// @Tags Pipeline
// @Produce json
// @Param id path int true "Pipeline ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/pipelines/{id}/execute [post]
func (s *Server) handleExecutePipeline(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "error": "invalid pipeline ID"})
		return
	}

	var payload map[string]any
	_ = c.ShouldBindJSON(&payload)

	exec, err := s.engine.ExecutePipeline(context.Background(), uint(id), "manual", payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "Pipeline execution started",
		"data": exec,
	})
}

// handleGetExecution returns execution details and step logs by execution ID.
// @Summary Get Execution Details
// @Description Returns execution status, timestamps, and step logs by execution ID
// @Tags Pipeline
// @Produce json
// @Param id path string true "Execution ID (UUID)"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/executions/{id} [get]
func (s *Server) handleGetExecution(c *gin.Context) {
	execID := c.Param("id")
	exec, err := s.engine.Repo().GetExecution(execID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "error": "execution not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": exec})
}
