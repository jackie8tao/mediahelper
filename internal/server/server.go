package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackie8tao/mediahelper/docs"
	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/downloader"
	"github.com/jackie8tao/mediahelper/internal/service"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Server HTTP Web 与 Webhook 服务管理器
type Server struct {
	cfg        *config.Config
	pipeline   *service.Pipeline
	pathMapper *downloader.PathMapper
	router     *gin.Engine
	httpServer *http.Server
}

// New 创建 Server 实例
func New(cfg *config.Config, pipeline *service.Pipeline) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	s := &Server{
		cfg:        cfg,
		pipeline:   pipeline,
		pathMapper: downloader.NewPathMapper(cfg.QBittorrent.PathMappings),
		router:     router,
	}

	s.setupRoutes()
	return s
}

// setupRoutes 注册路由
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
	}
}

// Start 启动 HTTP 服务监听
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

// Shutdown 优雅停止 HTTP 服务
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// handleHealth 探活接口
// @Summary 服务探活接口
// @Description 检测系统运行状态及时间戳
// @Tags 系统基础
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /health [get]
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
	})
}

// handleStatus 获取系统运行状态
// @Summary 获取系统运行状态
// @Description 查询系统配置状态、下载器地址、媒体库路径等
// @Tags 系统基础
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/status [get]
func (s *Server) handleStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"app":               "MediaHelper",
			"version":           "1.0.0",
			"qbittorrent_url":   s.cfg.QBittorrent.URL,
			"tmdb_enabled":      s.cfg.TMDB.APIKey != "",
			"link_mode":         s.cfg.Library.LinkMode,
			"movie_library_dir": s.cfg.Library.MovieDir,
			"tv_library_dir":    s.cfg.Library.TVDir,
		},
	})
}

// handleHistory 查询历史记录
// @Summary 查询归档执行历史
// @Description 获取最近的影视整理历史记录
// @Tags 归档历史
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/history [get]
func (s *Server) handleHistory(c *gin.Context) {
	records := s.pipeline.History().GetAll()
	c.JSON(http.StatusOK, gin.H{
		"code":  0,
		"total": len(records),
		"data":  records,
	})
}

// handlePlugins 获取所有插件信息及启用状态
// @Summary 查询插件列表
// @Description 获取系统当前加载的所有插件信息及启用状态
// @Tags 插件管理
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/plugins [get]
func (s *Server) handlePlugins(c *gin.Context) {
	var list []interface{}
	if pm := s.pipeline.PluginManager(); pm != nil {
		for _, info := range pm.ListPlugins() {
			list = append(list, info)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"code":  0,
		"total": len(list),
		"data":  list,
	})
}

// handleDownloaderTorrents 查询下载器当前所有种子任务
// @Summary 查询下载器任务列表
// @Description 获取下载器 (如 qBittorrent) 当前的所有种子任务信息
// @Tags 下载器
// @Produce json
// @Param filter query string false "任务状态过滤 (all, downloading, completed, paused, active, inactive)" default(all)
// @Param category query string false "按分类过滤 (如 movie, tv, anime)"
// @Success 200 {object} map[string]interface{}
// @Failure 503 {object} map[string]interface{}
// @Router /api/v1/downloader/torrents [get]
func (s *Server) handleDownloaderTorrents(c *gin.Context) {
	dl := s.pipeline.Downloader()
	if dl == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":  503,
			"error": "downloader is not initialized or disabled",
		})
		return
	}

	filter := c.DefaultQuery("filter", "all")
	category := c.Query("category")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	torrents, err := dl.GetAllTorrents(ctx, filter, category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":  500,
			"error": fmt.Sprintf("fetch torrents error: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":     0,
		"total":    len(torrents),
		"filter":   filter,
		"category": category,
		"data":     torrents,
	})
}

// handleSync 手动触发全量扫描同步
// @Summary 手动触发全量同步
// @Description 在后台立即触发一次未整理任务扫描与补漏整理
// @Tags 任务调度
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/sync [post]
func (s *Server) handleSync(c *gin.Context) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		count, err := s.pipeline.SyncOnce(ctx)
		if err != nil {
			slog.Error("Manual sync failed", "error", err)
		} else {
			slog.Info("Manual sync completed", "processed", count)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "Manual sync triggered successfully in background",
	})
}

// QBWebhookPayload qBittorrent 触发 Payload
type QBWebhookPayload struct {
	Hash        string `json:"hash" form:"hash"`
	Name        string `json:"name" form:"name"`
	Category    string `json:"category" form:"category"`
	ContentPath string `json:"content_path" form:"content_path"`
	SavePath    string `json:"save_path" form:"save_path"`
}

// handleQBWebhook 处理 qBittorrent 完成时调用的 Webhook 接口
// @Summary qBittorrent 任务完成实时通知 Webhook
// @Description 供 qBittorrent 下载完成后调用外部命令或 API 实时触发整理
// @Tags Webhook
// @Accept json,x-www-form-urlencoded
// @Produce json
// @Param payload body QBWebhookPayload false "种子信息 JSON 载荷"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/webhook/qbittorrent [post]
func (s *Server) handleQBWebhook(c *gin.Context) {
	var payload QBWebhookPayload

	// 支持 JSON 或 Form 表单传递
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

	// 路径映射转换
	contentPath := s.pathMapper.ToLocal(payload.ContentPath)
	savePath := s.pathMapper.ToLocal(payload.SavePath)

	torrent := downloader.Torrent{
		Hash:        payload.Hash,
		Name:        payload.Name,
		Category:    payload.Category,
		ContentPath: contentPath,
		SavePath:    savePath,
		IsCompleted: true,
	}

	slog.Info("Received qBittorrent webhook event", "hash", torrent.Hash, "name", torrent.Name)

	// 异步调用流水线处理，秒级响应给 qBittorrent 避免超时
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		_, _ = s.pipeline.ProcessTorrent(ctx, torrent)
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
