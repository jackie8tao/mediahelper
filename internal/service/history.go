package service

import (
	"sync"
	"time"
)

// HistoryRecord 归档历史记录
type HistoryRecord struct {
	Hash        string    `json:"hash"`
	TorrentName string    `json:"torrent_name"`
	Title       string    `json:"title"`
	Year        int       `json:"year"`
	MediaType   string    `json:"media_type"`
	Status      string    `json:"status"` // success / failed / skipped
	TargetDir   string    `json:"target_dir,omitempty"`
	FileCount   int       `json:"file_count"`
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// HistoryManager 历史记录管理器 (线程安全环形/限制长度缓存)
type HistoryManager struct {
	mu      sync.RWMutex
	records []HistoryRecord
	maxSize int
}

// NewHistoryManager 创建历史记录管理器
func NewHistoryManager(maxSize int) *HistoryManager {
	if maxSize <= 0 {
		maxSize = 100
	}
	return &HistoryManager{
		records: make([]HistoryRecord, 0, maxSize),
		maxSize: maxSize,
	}
}

// Add 追加一条记录
func (h *HistoryManager) Add(rec HistoryRecord) {
	h.mu.Lock()
	defer h.mu.Unlock()

	rec.CreatedAt = time.Now()
	// 最新记录插入头部
	h.records = append([]HistoryRecord{rec}, h.records...)
	if len(h.records) > h.maxSize {
		h.records = h.records[:h.maxSize]
	}
}

// GetAll 获取当前所有记录副本
func (h *HistoryManager) GetAll() []HistoryRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]HistoryRecord, len(h.records))
	copy(result, h.records)
	return result
}
