package translate

import (
	"fmt"
	"log/slog"
	"sync"

	"handy-translate/config"
)

// ProviderFactory 翻译提供者工厂函数类型。
type ProviderFactory func(cfg config.Translate) Provider

// Registry 翻译服务注册表（注册表模式）。
// 管理所有翻译提供者的注册和获取。
type Registry struct {
	mu        sync.RWMutex
	factories map[string]ProviderFactory
}

// NewRegistry 创建空的注册表。
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]ProviderFactory),
	}
}

// Register 注册一个翻译提供者工厂。
func (r *Registry) Register(name string, factory ProviderFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[name] = factory
	slog.Info("翻译提供者已注册", slog.String("provider", name))
}

// Get 根据名称和配置获取翻译提供者实例。
func (r *Registry) Get(name string, cfg config.Translate) (Provider, error) {
	r.mu.RLock()
	factory, ok := r.factories[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("翻译提供者未注册: %s", name)
	}

	return factory(cfg), nil
}

// GetFromConfig 从全局配置中获取指定名称的翻译提供者。
func (r *Registry) GetFromConfig(name string, configData *config.Config) (Provider, error) {
	if configData == nil {
		return nil, fmt.Errorf("翻译配置未初始化")
	}
	if name == "" {
		return nil, fmt.Errorf("未选择翻译提供者")
	}

	cfgEntry, exists := configData.Translate[name]
	if !exists {
		return nil, fmt.Errorf("翻译配置不存在: %s", name)
	}
	if cfgEntry.Key == "" {
		return nil, fmt.Errorf("翻译提供者 %s 尚未配置 API 密钥", name)
	}

	return r.Get(name, config.Translate{
		Name:    cfgEntry.Name,
		AppID:   cfgEntry.AppID,
		Key:     cfgEntry.Key,
		BaseURL: cfgEntry.BaseURL,
		Model:   cfgEntry.Model,
	})
}

// Names 返回所有已注册的提供者名称。
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}
