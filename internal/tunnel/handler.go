package tunnel

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"diagnostic-client/internal/config"
	"diagnostic-client/internal/db"
	"diagnostic-client/pkg/models"
)

type MessageType string

const (
	TypeMetrics MessageType = "metrics"
	TypeLogList MessageType = "log_list"
	TypeLogData MessageType = "log_data"
)

type Message struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// FileCache maintains an in-memory cache of the current file state
type FileCache struct {
	files map[string]models.FileNode
	count int
	mutex sync.RWMutex
}

type Handler struct {
	cfg             *config.Config
	db              *db.DB
	networkStreamCh chan []models.NetworkPacket
	logStreamCh     chan models.LogEntry
	fileUpdateCh    chan models.FileNode
	fileCache       *FileCache

	// Network packet batching
	batchMutex    sync.Mutex
	networkBatch  []models.NetworkPacket
	lastBatchTime time.Time

	// Shutdown coordination
	shutdownOnce sync.Once
	shutdownCh   chan struct{}
}

func NewHandler(cfg *config.Config, db *db.DB) *Handler {
	h := &Handler{
		cfg:             cfg,
		db:              db,
		networkStreamCh: make(chan []models.NetworkPacket, cfg.NetworkBufferSize),
		logStreamCh:     make(chan models.LogEntry, cfg.LogBufferSize),
		fileUpdateCh:    make(chan models.FileNode, 2000),
		networkBatch:    make([]models.NetworkPacket, 0, cfg.BatchSize),
		lastBatchTime:   time.Now(),
		shutdownCh:      make(chan struct{}),
		fileCache: &FileCache{
			files: make(map[string]models.FileNode),
		},
	}

	go h.initializeFileCache()
	go h.periodicNetworkFlush()

	return h
}

func (h *Handler) HandleConnection(ctx context.Context, conn net.Conn) {
	log.Printf("[TUNNEL] New agent connection from %s", conn.RemoteAddr())
	defer conn.Close()

	decoder := json.NewDecoder(conn)

	for {
		select {
		case <-ctx.Done():
			return
		case <-h.shutdownCh:
			return
		default:
			var msg Message
			if err := decoder.Decode(&msg); err != nil {
				if ctx.Err() == nil {
					log.Printf("[TUNNEL] Error decoding message: %v", err)
				}
				return
			}

			if err := h.processMessage(ctx, msg); err != nil {
				log.Printf("[TUNNEL] Error processing message: %v", err)
			}
		}
	}
}

func (h *Handler) processMessage(ctx context.Context, msg Message) error {
	switch msg.Type {
	case TypeMetrics:
		return h.handleMetrics(ctx, msg.Payload)
	case TypeLogList:
		return h.handleFileList(ctx, msg.Payload)
	case TypeLogData:
		return h.handleLogData(ctx, msg.Payload)
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

// initializeFileCache loads the initial file state from the database
func (h *Handler) initializeFileCache() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	files, err := h.db.GetAllFiles(ctx)
	if err != nil {
		log.Printf("[TUNNEL] Error initializing file cache: %v", err)
		return
	}

	h.fileCache.mutex.Lock()
	defer h.fileCache.mutex.Unlock()

	for _, file := range files {
		h.fileCache.files[file.Path] = file
	}
	h.fileCache.count = len(files)

	log.Printf("[TUNNEL] Initialized file cache with %d files", len(files))
}

// handleFileList processes incoming file lists efficiently
func (h *Handler) handleFileList(ctx context.Context, payload json.RawMessage) error {
	var newFiles []models.FileNode
	if err := json.Unmarshal(payload, &newFiles); err != nil {
		return fmt.Errorf("unmarshal file list: %w", err)
	}

	changes := h.detectFileChanges(newFiles)
	if changes.isEmpty() {
		return nil
	}

	if err := h.applyFileChanges(ctx, changes); err != nil {
		return fmt.Errorf("apply file changes: %w", err)
	}

	h.notifyFileChanges(changes)
	return nil
}

type fileChanges struct {
	added   []models.FileNode
	updated []models.FileNode
	deleted []string
}

func (fc *fileChanges) isEmpty() bool {
	return len(fc.added) == 0 && len(fc.updated) == 0 && len(fc.deleted) == 0
}

func (h *Handler) detectFileChanges(newFiles []models.FileNode) *fileChanges {
	changes := &fileChanges{
		added:   make([]models.FileNode, 0),
		updated: make([]models.FileNode, 0),
		deleted: make([]string, 0),
	}

	// Create map of new files
	newFileMap := make(map[string]models.FileNode, len(newFiles))
	for _, file := range newFiles {
		newFileMap[file.Path] = file
	}

	// Find updates and deletions
	h.fileCache.mutex.RLock()
	for path, existingFile := range h.fileCache.files {
		if newFile, exists := newFileMap[path]; exists {
			if isFileChanged(existingFile, newFile) {
				changes.updated = append(changes.updated, newFile)
			}
			delete(newFileMap, path)
		} else {
			changes.deleted = append(changes.deleted, path)
		}
	}
	h.fileCache.mutex.RUnlock()

	// Remaining files are new
	for _, file := range newFileMap {
		changes.added = append(changes.added, file)
	}

	return changes
}

func (h *Handler) applyFileChanges(ctx context.Context, changes *fileChanges) error {
	if len(changes.deleted) > 0 {
		if err := h.db.DeleteFiles(ctx, changes.deleted); err != nil {
			return fmt.Errorf("delete files: %w", err)
		}
	}

	if len(changes.added) > 0 {
		if err := h.db.SaveFiles(ctx, changes.added); err != nil {
			return fmt.Errorf("save new files: %w", err)
		}
	}

	if len(changes.updated) > 0 {
		if err := h.db.UpdateFiles(ctx, changes.updated); err != nil {
			return fmt.Errorf("update files: %w", err)
		}
	}

	// Update cache
	h.updateFileCache(changes)

	log.Printf("[TUNNEL] File changes processed: +%d -%d ~%d",
		len(changes.added), len(changes.deleted), len(changes.updated))

	return nil
}

func (h *Handler) updateFileCache(changes *fileChanges) {
	h.fileCache.mutex.Lock()
	defer h.fileCache.mutex.Unlock()

	// Apply deletions
	for _, path := range changes.deleted {
		delete(h.fileCache.files, path)
	}

	// Apply additions and updates
	for _, file := range append(changes.added, changes.updated...) {
		h.fileCache.files[file.Path] = file
	}

	h.fileCache.count = len(h.fileCache.files)
}

func (h *Handler) notifyFileChanges(changes *fileChanges) {
	// Notify about new and updated files
	for _, file := range append(changes.added, changes.updated...) {
		select {
		case h.fileUpdateCh <- file:
		default:
			// Skip notification if channel is full
		}
	}
}

// handleMetrics processes network metrics
func (h *Handler) handleMetrics(ctx context.Context, payload json.RawMessage) error {
	var metrics struct {
		Timestamp string                 `json:"timestamp"`
		Packets   []models.NetworkPacket `json:"packets"`
	}
	if err := json.Unmarshal(payload, &metrics); err != nil {
		return fmt.Errorf("unmarshal metrics: %w", err)
	}

	h.batchMutex.Lock()
	h.networkBatch = append(h.networkBatch, metrics.Packets...)
	currentSize := len(h.networkBatch)
	h.batchMutex.Unlock()

	if currentSize >= h.cfg.BatchSize {
		return h.flushNetworkBatch(ctx)
	}
	return nil
}

// handleLogData processes log entries
func (h *Handler) handleLogData(ctx context.Context, payload json.RawMessage) error {
	var logs []models.LogEntry
	if err := json.Unmarshal(payload, &logs); err != nil {
		return fmt.Errorf("unmarshal logs: %w", err)
	}

	if err := h.db.SaveLogs(ctx, logs); err != nil {
		return fmt.Errorf("save logs: %w", err)
	}

	// Stream logs to subscribers
	for _, entry := range logs {
		select {
		case h.logStreamCh <- entry:
		default:
			// Skip if channel is full
		}
	}

	return nil
}

// periodicNetworkFlush ensures network batches are flushed periodically
func (h *Handler) periodicNetworkFlush() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.shutdownCh:
			return
		case <-ticker.C:
			if err := h.flushNetworkBatch(context.Background()); err != nil {
				log.Printf("[TUNNEL] Error flushing network batch: %v", err)
			}
		}
	}
}

func (h *Handler) flushNetworkBatch(ctx context.Context) error {
	h.batchMutex.Lock()
	if len(h.networkBatch) == 0 {
		h.batchMutex.Unlock()
		return nil
	}

	batch := h.networkBatch
	h.networkBatch = make([]models.NetworkPacket, 0, h.cfg.BatchSize)
	h.lastBatchTime = time.Now()
	h.batchMutex.Unlock()

	// Save to database
	if err := h.db.SaveNetworkPackets(ctx, batch); err != nil {
		// Preserve undelivered data for the next flush attempt.
		h.batchMutex.Lock()
		h.networkBatch = append(batch, h.networkBatch...)
		h.batchMutex.Unlock()
		return fmt.Errorf("save network batch: %w", err)
	}

	// Stream to subscribers
	select {
	case h.networkStreamCh <- batch:
	default:
		log.Printf("[TUNNEL] Network stream channel full, dropped %d packets", len(batch))
	}

	return nil
}

// Helper functions
func isFileChanged(a, b models.FileNode) bool {
	return a.ModTime != b.ModTime ||
		a.Size != b.Size ||
		a.IsDirectory != b.IsDirectory ||
		a.IsGzipped != b.IsGzipped
}

// Channel accessors
func (h *Handler) NetworkStream() <-chan []models.NetworkPacket {
	return h.networkStreamCh
}

func (h *Handler) LogStream() <-chan models.LogEntry {
	return h.logStreamCh
}

func (h *Handler) FileUpdates() <-chan models.FileNode {
	return h.fileUpdateCh
}

// Close handles graceful shutdown
func (h *Handler) Close() {
	h.shutdownOnce.Do(func() {
		close(h.shutdownCh)
		_ = h.flushNetworkBatch(context.Background())

		// Producers may still be finishing requests while listeners shut down.
		// Leave event channels open; the WebSocket fan-out has its own stop signal.
	})
}
