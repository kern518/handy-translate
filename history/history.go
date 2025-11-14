package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	"handy-translate/config"

	"github.com/google/uuid"
)

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
}

// NewHistoryService 创建历史记录服务实例
func NewHistoryService() *HistoryService {
	return &HistoryService{
		enabled:     config.Data.History.Enabled,
		storagePath: config.Data.History.StoragePath,
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
	filePath := path.Join(h.storagePath, "history", "translate", date+".json")

	h.appendToFile(filePath, record)
	fmt.Printf("翻译历史记录已保存，ID: %s\n", record.ID)
}

// SaveExplainRecord 保存解释记录（只保存源词语）
func (h *HistoryService) SaveExplainRecord(sourceText, result, templateID string) {
	if !h.enabled {
		return
	}

	record := &HistoryRecord{
		ID:         uuid.New().String(),
		Type:       "explain",
		SourceText: sourceText,
		Result:     result, // 解释类型不保存结果
		TemplateID: templateID,
		Timestamp:  time.Now(),
	}

	date := record.Timestamp.Format("2006-01-02")
	filePath := path.Join(h.storagePath, "history", "explain", date+".json")

	h.appendToFile(filePath, record)
	fmt.Printf("解释历史记录已保存，ID: %s，词语: %s\n", record.ID, sourceText)
}

// appendToFile 将记录追加到文件
func (h *HistoryService) appendToFile(filePath string, record *HistoryRecord) {
	// 确保目录存在
	err := os.MkdirAll(path.Dir(filePath), 0755)
	if err != nil {
		fmt.Printf("创建历史记录目录失败: %v\n", err)
		return
	}

	// 读取现有记录
	var records []*HistoryRecord
	if _, err := os.Stat(filePath); err == nil {
		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("读取历史记录文件失败: %v\n", err)
		} else {
			if err := json.Unmarshal(data, &records); err != nil {
				fmt.Printf("解析历史记录文件失败: %v\n", err)
			}
		}
	}

	// 添加新记录
	records = append(records, record)

	// 写回文件
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		fmt.Printf("序列化历史记录失败: %v\n", err)
		return
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		fmt.Printf("写入历史记录文件失败: %v\n", err)
	}
}

// 全局历史记录服务实例
var GlobalHistoryService *HistoryService
