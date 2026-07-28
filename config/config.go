package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/pelletier/go-toml/v2"
)

// Data config
var Data Config

// configFilePath 存储配置文件的绝对路径，Init 和 Save 共用。
var configFilePath string
var saveMu sync.Mutex
var dataMu sync.RWMutex

type (
	Config struct {
		Appname          string                 `toml:"appname"`
		Keyboards        map[string][]string    `toml:"keyboards"`
		TranslateWay     string                 `toml:"translate_way"`
		Translate        map[string]Translate   `toml:"translate"`
		ExplainTemplates ExplainTemplatesConfig `toml:"explain_templates"`
		History          HistoryConfig          `toml:"history"`
		ToolbarMode      string                 `toml:"toolbar_mode"`
		ToolbarPinned    bool                   `toml:"toolbar_pinned"`
	}

	Translate struct {
		Name    string `toml:"name" json:"name,omitempty"`
		AppID   string `toml:"appID" json:"appID,omitempty"`
		Key     string `toml:"key" json:"key,omitempty"`
		BaseURL string `toml:"base_url" json:"base_url,omitempty"`
		Model   string `toml:"model" json:"model,omitempty"`
	}

	ExplainTemplatesConfig struct {
		DefaultTemplate string                     `toml:"default_template"`
		Templates       map[string]ExplainTemplate `toml:"templates"`
	}

	ExplainTemplate struct {
		Name        string `toml:"name" json:"name"`
		Description string `toml:"description" json:"description"`
		Template    string `toml:"template" json:"template"`
	}

	HistoryConfig struct {
		Enabled     bool   `toml:"enabled"`
		StoragePath string `toml:"storage_path"`
	}
)

// DefaultConfig 返回可安全启动的默认配置。所有凭据均为空，
// 首次启动时会将其写入 config.toml，避免空 map 导致后续空指针。
func DefaultConfig() Config {
	return Config{
		Appname:      "handy-translate",
		TranslateWay: "deepseek",
		ToolbarMode:  "translate",
		Keyboards: map[string][]string{
			"screenshot": {"alt", "shift", "q"},
			"toolBar":    {"center", "", ""},
		},
		Translate: map[string]Translate{
			"baidu":    {Name: "百度翻译"},
			"youdao":   {Name: "有道翻译"},
			"caiyun":   {Name: "彩云小译"},
			"deepseek": {Name: "DeepSeek", AppID: "deepseek"},
			"minimax": {
				Name:    "MiniMax",
				AppID:   "minimax",
				BaseURL: "https://api.minimaxi.com",
				Model:   "MiniMax-M2.7",
			},
			"google": {
				Name:    "Google Gemini",
				AppID:   "google",
				BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai",
				Model:   "gemini-2.0-flash",
			},
		},
		ExplainTemplates: ExplainTemplatesConfig{
			DefaultTemplate: "programmer",
			Templates: map[string]ExplainTemplate{
				"programmer": {
					Name:        "技术视角",
					Description: "适合解释编程、技术相关术语",
					Template:    "请从程序员视角简洁解释以下术语，说明核心原理和常见用途，控制在 3～5 句话：{{.text}}",
				},
				"academic": {
					Name:        "文学视角",
					Description: "适合解释文学词语",
					Template:    "请结合历史与思想背景，简洁解释以下词语的核心含义和现实意义，控制在 3～5 句话：{{.text}}",
				},
			},
		},
		History: HistoryConfig{
			Enabled:     true,
			StoragePath: "./data",
		},
	}
}

// Init 初始化配置。
func Init(projectName string) {
	defaults := DefaultConfig()
	dataMu.Lock()
	Data = defaults
	dataMu.Unlock()

	configDir := resolveConfigDir(projectName)
	configFilePath = filepath.Join(configDir, "config.toml")

	fd, err := os.ReadFile(configFilePath)
	if os.IsNotExist(err) {
		if saveErr := Save(); saveErr != nil {
			slog.Error("创建默认配置失败", slog.Any("error", saveErr))
		} else {
			slog.Warn("配置文件不存在，已创建安全的默认配置",
				slog.String("path", configFilePath))
		}
		return
	}
	if err != nil {
		slog.Error("读取配置文件失败，将使用默认配置", slog.Any("error", err))
		return
	}

	loaded := DefaultConfig()
	if err := toml.Unmarshal(fd, &loaded); err != nil {
		slog.Error("解析配置文件失败", slog.Any("error", err))
		return
	}

	normalizeConfig(&loaded)
	dataMu.Lock()
	Data = loaded
	dataMu.Unlock()

	slog.Info("配置已加载",
		slog.String("path", configFilePath),
		slog.String("translateWay", loaded.TranslateWay),
		slog.Int("providerCount", len(loaded.Translate)))
}

func resolveConfigDir(projectName string) string {
	if cwd, err := os.Getwd(); err == nil {
		for dir := cwd; ; dir = filepath.Dir(dir) {
			if filepath.Base(dir) == projectName || fileExists(filepath.Join(dir, "config.toml.bak")) {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
		}
	}

	if executable, err := os.Executable(); err == nil {
		return filepath.Dir(executable)
	}
	return "."
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func normalizeConfig(data *Config) {
	defaults := DefaultConfig()
	if data.Appname == "" {
		data.Appname = defaults.Appname
	}
	if data.TranslateWay == "" {
		data.TranslateWay = defaults.TranslateWay
	}
	if data.ToolbarMode != "translate" && data.ToolbarMode != "explain" {
		data.ToolbarMode = defaults.ToolbarMode
	}
	if data.Keyboards == nil {
		data.Keyboards = defaults.Keyboards
	}
	if len(data.Keyboards["screenshot"]) == 0 {
		data.Keyboards["screenshot"] = defaults.Keyboards["screenshot"]
	}
	if data.Translate == nil {
		data.Translate = make(map[string]Translate)
	}
	for name, defaultProvider := range defaults.Translate {
		if _, exists := data.Translate[name]; !exists {
			data.Translate[name] = defaultProvider
		}
	}
	if data.ExplainTemplates.Templates == nil {
		data.ExplainTemplates = defaults.ExplainTemplates
	}
	if data.History.StoragePath == "" {
		data.History.StoragePath = defaults.History.StoragePath
	}
}

// normalize 修复全局配置中的缺失字段。仅供包内兼容和测试使用。
func normalize() {
	dataMu.Lock()
	defer dataMu.Unlock()
	normalizeConfig(&Data)
}

// Snapshot 返回配置的深拷贝，调用方可安全读取其中的 map 和 slice。
func Snapshot() Config {
	dataMu.RLock()
	defer dataMu.RUnlock()
	return cloneConfig(Data)
}

// Update 在同一把锁下修改配置并保存一致的快照。
func Update(mutator func(*Config)) error {
	if mutator == nil {
		return fmt.Errorf("config mutator is nil")
	}

	dataMu.Lock()
	defer dataMu.Unlock()
	mutator(&Data)
	snapshot := cloneConfig(Data)

	return saveSnapshot(snapshot)
}

// Save 保存配置到文件（原子写入）。
func Save() error {
	return saveSnapshot(Snapshot())
}

func saveSnapshot(snapshot Config) error {
	saveMu.Lock()
	defer saveMu.Unlock()

	if configFilePath == "" {
		configFilePath = "./config.toml"
	}

	data, err := toml.Marshal(&snapshot)
	if err != nil {
		slog.Error("Marshal config failed", slog.Any("error", err))
		return fmt.Errorf("marshal config: %w", err)
	}

	// 使用原子写入: 先写临时文件，然后重命名
	// 这样可以避免写入失败导致配置文件损坏
	tempFilePath := configFilePath + ".tmp"

	// 创建临时文件
	file, err := os.OpenFile(tempFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		slog.Error("Create temp config file failed", slog.Any("error", err))
		return fmt.Errorf("create temp file: %w", err)
	}

	// 写入数据
	_, err = file.Write(data)
	if err != nil {
		file.Close()
		os.Remove(tempFilePath) // 清理临时文件
		slog.Error("Write config failed", slog.Any("error", err))
		return fmt.Errorf("write config: %w", err)
	}

	// 确保数据写入磁盘
	if err := file.Sync(); err != nil {
		file.Close()
		os.Remove(tempFilePath)
		slog.Error("Sync config failed", slog.Any("error", err))
		return fmt.Errorf("sync config: %w", err)
	}

	file.Close()

	// 原子性地重命名临时文件为目标文件
	if err := os.Rename(tempFilePath, configFilePath); err != nil {
		os.Remove(tempFilePath) // 清理临时文件
		slog.Error("Rename config file failed", slog.Any("error", err))
		return fmt.Errorf("rename config file: %w", err)
	}

	slog.Debug("Config saved successfully")
	return nil
}

func cloneConfig(source Config) Config {
	cloned := source

	if source.Keyboards != nil {
		cloned.Keyboards = make(map[string][]string, len(source.Keyboards))
		for name, keys := range source.Keyboards {
			cloned.Keyboards[name] = append([]string(nil), keys...)
		}
	}

	if source.Translate != nil {
		cloned.Translate = make(map[string]Translate, len(source.Translate))
		for name, provider := range source.Translate {
			cloned.Translate[name] = provider
		}
	}

	if source.ExplainTemplates.Templates != nil {
		cloned.ExplainTemplates.Templates = make(
			map[string]ExplainTemplate,
			len(source.ExplainTemplates.Templates),
		)
		for name, template := range source.ExplainTemplates.Templates {
			cloned.ExplainTemplates.Templates[name] = template
		}
	}

	return cloned
}
