package history

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// 模式常量（避免依赖 UI 层包）
const explainMode = "explain"

// HistoryRecord 历史记录结构
type HistoryRecord struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"` // "translate" 或 "explain"
	SourceText string    `json:"source_text"`
	Result     string    `json:"result"`      // 仅翻译类型有值
	FromLang   string    `json:"from_lang"`   // 仅翻译类型
	ToLang     string    `json:"to_lang"`     // 仅翻译类型
	TemplateID string    `json:"template_id"` // 仅解释类型
	Timestamp  time.Time `json:"timestamp"`
}

// HistoryService 历史记录服务
type HistoryService struct {
	enabled     bool
	storagePath string
	mu          sync.Mutex // 互斥锁，保证并发写入安全
}

// NewHistoryService 创建历史记录服务实例（依赖注入）。
func NewHistoryService(enabled bool, storagePath string) *HistoryService {
	return &HistoryService{
		enabled:     enabled,
		storagePath: storagePath,
	}
}

// SaveTranslateRecord 保存翻译记录
func (h *HistoryService) SaveTranslateRecord(sourceText, result, fromLang, toLang string) {
	if !h.enabled {
		return
	}

	record := &HistoryRecord{
		ID:         uuid.New().String(),
		Type:       "translate",
		SourceText: sourceText,
		Result:     result,
		FromLang:   fromLang,
		ToLang:     toLang,
		Timestamp:  time.Now(),
	}

	date := record.Timestamp.Format("2006-01-02")
	filePath := filepath.Join(h.storagePath, "history", "translate", date+".json")

	h.appendToFile(filePath, record)
	slog.Debug("翻译历史记录已保存", slog.String("id", record.ID))
}

// SaveExplainRecord 保存解释记录（只保存源词语）
func (h *HistoryService) SaveExplainRecord(sourceText, result, templateID string) {
	if !h.enabled {
		return
	}

	record := &HistoryRecord{
		ID:         uuid.New().String(),
		Type:       explainMode,
		SourceText: sourceText,
		Result:     result,
		TemplateID: templateID,
		Timestamp:  time.Now(),
	}

	date := record.Timestamp.Format("2006-01-02")
	filePath := filepath.Join(h.storagePath, "history", explainMode, date+".json")

	h.appendToFile(filePath, record)
	slog.Debug("解释历史记录已保存", slog.String("id", record.ID), slog.String("word", sourceText))
}

// appendToFile 将记录追加到文件
func (h *HistoryService) appendToFile(filePath string, record *HistoryRecord) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 确保目录存在
	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		slog.Error("创建历史记录目录失败", slog.Any("error", err))
		return
	}

	// 读取现有记录
	var records []*HistoryRecord
	if _, err := os.Stat(filePath); err == nil {
		data, err := os.ReadFile(filePath)
		if err != nil {
			slog.Error("读取历史记录文件失败", slog.Any("error", err))
		} else {
			if err := json.Unmarshal(data, &records); err != nil {
				slog.Error("解析历史记录文件失败", slog.Any("error", err))
			}
		}
	}

	// 添加新记录
	records = append(records, record)

	// 写回文件
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		slog.Error("序列化历史记录失败", slog.Any("error", err))
		return
	}

	if err := writeFileAtomic(filePath, data, 0644); err != nil {
		slog.Error("写入历史记录文件失败", slog.Any("error", err))
	}
}

func writeFileAtomic(filePath string, data []byte, mode os.FileMode) error {
	tempFile, err := os.CreateTemp(filepath.Dir(filePath), ".history-*.tmp")
	if err != nil {
		return fmt.Errorf("create history temp file: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if err := tempFile.Chmod(mode); err != nil {
		tempFile.Close()
		return fmt.Errorf("set history file mode: %w", err)
	}
	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		return fmt.Errorf("write history temp file: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		return fmt.Errorf("sync history temp file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close history temp file: %w", err)
	}
	if err := os.Rename(tempPath, filePath); err != nil {
		return fmt.Errorf("replace history file: %w", err)
	}
	return nil
}
