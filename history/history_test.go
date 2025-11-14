package history

import (
	"os"
	"testing"
	"time"
)

func TestSaveTranslateRecord(t *testing.T) {
	// 创建一个启用的历史记录服务实例进行测试
	service := &HistoryService{
		enabled:     true,
		storagePath: "./test_data",
	}

	// 测试保存翻译记录
	service.SaveTranslateRecord("Hello world", "你好世界", "en", "zh")

	// 检查文件是否创建
	date := time.Now().Format("2006-01-02")
	filePath := "./test_data/history/translate/" + date + ".json"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("翻译历史记录文件未创建: %s", filePath)
	}

	// 清理测试文件
	defer func() {
		os.RemoveAll("./test_data")
	}()
}

func TestSaveExplainRecord(t *testing.T) {
	// 创建一个启用的历史记录服务实例进行测试
	service := &HistoryService{
		enabled:     true,
		storagePath: "./test_data",
	}

	// 测试保存解释记录
	service.SaveExplainRecord("machine learning", "机器学习是人工智能的一个分支...", "template1")

	// 检查文件是否创建
	date := time.Now().Format("2006-01-02")
	filePath := "./test_data/history/explain/" + date + ".json"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("解释历史记录文件未创建: %s", filePath)
	}

	// 清理测试文件
	defer func() {
		os.RemoveAll("./test_data")
	}()
}

func TestDisabledHistoryService(t *testing.T) {
	// 创建一个禁用的历史记录服务实例
	service := &HistoryService{
		enabled:     false,
		storagePath: "./test_data",
	}

	// 测试禁用状态下不保存记录
	service.SaveTranslateRecord("Hello", "你好", "en", "zh")

	// 检查文件是否未创建
	date := time.Now().Format("2006-01-02")
	filePath := "./test_data/history/translate/" + date + ".json"
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("历史记录功能禁用时不应该创建文件: %s", filePath)
	}
}
