package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackie8tao/mediahelper/docs"
	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
	"github.com/jackie8tao/mediahelper/internal/plugin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Server manages HTTP Web, REST APIs, and Webhook endpoints.
type Server struct {
	cfg          *config.SystemConfig
	engine       *pipeline.Engine
	compRegistry *pipeline.ComponentRegistry
	pluginMgr    *plugin.Manager
	router       *gin.Engine
	httpServer   *http.Server
}

// New creates a new Server instance.
func New(cfg *config.SystemConfig, engine *pipeline.Engine, compRegistry *pipeline.ComponentRegistry, pluginMgr *plugin.Manager) *Server {
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
			"app":              "MediaHelper",
			"version":          "1.0.0",
			"components_count": compCount,
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
