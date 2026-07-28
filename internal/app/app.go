// Package app 提供应用核心结构（依赖注入模式）。
package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"handy-translate/config"
	"handy-translate/history"
	"handy-translate/internal/event"
	"handy-translate/internal/service"
	"handy-translate/internal/window"
	"handy-translate/os_api/windows"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Application 应用核心（依赖注入容器）。
// 所有依赖通过 NewApplication 注入，不使用全局变量。
type Application struct {
	WailsApp   *application.App
	Translator *service.Translator
	WindowMgr  *window.Manager
	EventBus   *event.Bus
	History    *history.HistoryService
	OCR        *service.OCRService
	State      *State

	// 查询取消机制：每次新查询会取消上一个正在进行的查询
	queryMu     sync.Mutex
	cancelQuery context.CancelFunc
	queryID     uint64
}

// NewApplication 组装应用所有依赖。
func NewApplication(
	wailsApp *application.App,
	translator *service.Translator,
	windowMgr *window.Manager,
	eventBus *event.Bus,
	historySvc *history.HistoryService,
	ocrSvc *service.OCRService,
) *Application {
	return &Application{
		WailsApp:   wailsApp,
		Translator: translator,
		WindowMgr:  windowMgr,
		EventBus:   eventBus,
		History:    historySvc,
		OCR:        ocrSvc,
		State:      NewState(),
	}
}

// RegisterEvents 注册所有事件监听。
func (a *Application) RegisterEvents() {
	// 翻译语言切换
	a.EventBus.On(event.TranslateLang, func(e *application.CustomEvent) {
		if dataSlice, ok := e.Data.([]interface{}); ok {
			if len(dataSlice) >= 2 {
				from := fmt.Sprintf("%v", dataSlice[0])
				to := fmt.Sprintf("%v", dataSlice[1])
				a.State.SetLangs(from, to)
				slog.Info("translateLang 已更新",
					slog.String("fromLang", from),
					slog.String("toLang", to))
			}
		}
	})

	// 工具栏模式切换
	a.EventBus.On(event.ToolbarMode, func(e *application.CustomEvent) {
		if mode, ok := e.Data.(string); ok {
			a.SetToolbarMode(mode)
			slog.Info("toolbarMode 已更新", slog.String("mode", mode))
			a.EventBus.EmitToolbarModeUpdated(mode)

			// 如果有当前查询文本，重新处理
			currentQuery := a.State.GetCurrentQuery()
			if currentQuery != "" {
				a.EventBus.EmitQuery(currentQuery)
				a.startCurrentQuery(currentQuery, mode)
			}
		}
	})
}

// SetToolbarMode 设置工具栏模式并保存配置。
func (a *Application) SetToolbarMode(mode string) {
	a.State.SetToolbarMode(mode)
	if err := config.Update(func(data *config.Config) {
		data.ToolbarMode = mode
	}); err != nil {
		slog.Error("Failed to save config", slog.String("error", err.Error()))
	}
}

// ProcessHook 监听处理鼠标/键盘事件。
func (a *Application) ProcessHook() {
	if runtime.GOOS == "windows" {
		go windows.WindowsHook()
	}

	for msg := range windows.HookChan {
		switch msg {
		case "mouse":
			a.handleMouseEvent()
		case "screenshot":
			a.HandleScreenshotEvent()
		default:
			slog.Error("processHook 未知消息", slog.String("msg", msg))
		}
	}
}

// ──────────────────────────────────────────────
// 内部方法
// ──────────────────────────────────────────────

func (a *Application) handleMouseEvent() {
	result, ok := a.WailsApp.Clipboard.Text()
	if !ok {
		slog.Error("Failed to get clipboard text")
		return
	}

	queryText := result
	fl, tl := a.State.GetLangs()
	slog.Debug("processHook",
		slog.String("fromLang", fl),
		slog.String("toLang", tl))
	// 当工具栏未固定时，重置并显示
	if !a.WindowMgr.GetPinned() {
		a.WindowMgr.ResetToolbarState()
		a.WindowMgr.ShowToolbarAtCursor(a.WindowMgr.QueryResultHeight)
	}

	if queryText != "" {
		a.State.SetCurrentQuery(queryText)
		a.EventBus.EmitQuery(queryText)

		mode := a.State.GetToolbarMode()
		a.startCurrentQuery(queryText, mode)
	}
}

// startCurrentQuery 立即取消旧查询并异步启动新查询，避免 Hook 消费循环
// 被网络请求占用。queryID 用于确保旧任务结束时不会清理新任务的取消函数。
func (a *Application) startCurrentQuery(queryText, mode string) {
	ctx, cancel, queryID := a.beginQuery()

	go func() {
		defer a.finishQuery(queryID, cancel)
		a.processCurrentQuery(ctx, queryText, mode)
	}()
}

func (a *Application) beginQuery() (context.Context, context.CancelFunc, uint64) {
	a.queryMu.Lock()
	defer a.queryMu.Unlock()

	if a.cancelQuery != nil {
		a.cancelQuery()
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.cancelQuery = cancel
	a.queryID++
	return ctx, cancel, a.queryID
}

func (a *Application) finishQuery(queryID uint64, cancel context.CancelFunc) {
	cancel()
	a.queryMu.Lock()
	if a.queryID == queryID {
		a.cancelQuery = nil
	}
	a.queryMu.Unlock()
}

func (a *Application) processCurrentQuery(ctx context.Context, queryText, mode string) {
	slog.Info("处理查询", slog.String("mode", mode), slog.Int("textLen", len(queryText)))
	fl, tl := a.State.GetLangs()

	switch mode {
	case window.ExplainMode:
		currentConfig := config.Snapshot()
		templateID := currentConfig.ExplainTemplates.DefaultTemplate
		if templateID == "" {
			for id := range currentConfig.ExplainTemplates.Templates {
				templateID = id
				break
			}
		}
		if templateID == "" {
			slog.Error("解释模式但未找到模板")
			a.EventBus.EmitStreamError("未找到解释模板")
		} else {
			result := a.Translator.Explain(ctx, queryText, templateID)
			slog.Info("解释完成", slog.Int("len", len(result)))
		}
	default:
		if isWord(queryText) {
			slog.Info("单词查询（两阶段）", slog.String("word", queryText))

			// 阶段 1: 如果缓存未命中，先做快速翻译（~1秒可见）
			if _, cached := a.Translator.WordCacheGet(queryText); !cached {
				result := a.Translator.Translate(ctx, queryText, fl, tl)
				slog.Info("阶段1 翻译完成", slog.Int("len", len(result)))
			}

			// 检查上下文是否已被取消
			if ctx.Err() != nil {
				slog.Debug("查询已取消，跳过阶段2")
				return
			}

			// 阶段 2: 查询完整词典（缓存命中秒出 / LLM ~5秒）
			a.Translator.QueryWord(ctx, queryText)
			return
		}
		result := a.Translator.Translate(ctx, queryText, fl, tl)
		slog.Info("翻译完成", slog.Int("len", len(result)))
	}
}

// ──────────────────────────────────────────────
// 辅助方法
// ──────────────────────────────────────────────

var wordRegex = regexp.MustCompile(`^[a-zA-Z]([a-zA-Z'-]*[a-zA-Z])?$`)

// isWord 判断是否为单个英文单词（支持连字符和撇号）。
func isWord(text string) bool {
	trimmed := strings.TrimSpace(text)
	return len(trimmed) <= 30 && wordRegex.MatchString(trimmed)
}

// TruncateText 截断文本用于日志显示。
func TruncateText(text string, maxLen int) string {
	runes := []rune(text)
	if len(runes) <= maxLen {
		return text
	}
	return string(runes[:maxLen]) + "..."
}

// GetTranslateMapJSON 获取所有翻译配置的 JSON 字符串。
func GetTranslateMapJSON() string {
	// 前端只需要服务名称来渲染选择列表。绝不能把 API Key、
	// AppID、BaseURL 等完整凭据暴露给 WebView JavaScript。
	type publicProvider struct {
		Name string `json:"name"`
	}

	currentConfig := config.Snapshot()
	providers := make(map[string]publicProvider, len(currentConfig.Translate))
	for id, provider := range currentConfig.Translate {
		providers[id] = publicProvider{Name: provider.Name}
	}

	b, err := json.Marshal(providers)
	if err != nil {
		slog.Error("Marshal", slog.Any("error", err))
		return "{}"
	}
	return string(b)
}

// GetExplainTemplatesJSON 获取所有解释模板的 JSON 字符串。
func GetExplainTemplatesJSON() string {
	currentConfig := config.Snapshot()
	if len(currentConfig.ExplainTemplates.Templates) == 0 {
		return "{}"
	}

	templates := make(map[string]map[string]interface{})
	for id, tmpl := range currentConfig.ExplainTemplates.Templates {
		templates[id] = map[string]interface{}{
			"id":          id,
			"name":        tmpl.Name,
			"description": tmpl.Description,
		}
	}

	result := map[string]interface{}{
		"default_template": currentConfig.ExplainTemplates.DefaultTemplate,
		"templates":        templates,
	}

	b, err := json.Marshal(result)
	if err != nil {
		slog.Error("Marshal ExplainTemplates", slog.Any("error", err))
		return "{}"
	}
	return string(b)
}

// SetTranslateWay 设置当前翻译服务，返回是否切换成功。
func SetTranslateWay(way string) bool {
	currentConfig := config.Snapshot()
	if _, ok := currentConfig.Translate[way]; !ok {
		slog.Warn("忽略未知的翻译服务", slog.String("provider", way))
		return false
	}

	if err := config.Update(func(data *config.Config) {
		data.TranslateWay = way
	}); err != nil {
		slog.Error("Failed to save config", slog.String("error", err.Error()))
		return false
	}
	slog.Info("翻译服务已切换", slog.String("provider", way))
	return true
}

// SwitchTranslateWay 切换翻译服务，并立即用新服务重新处理当前文本。
func (a *Application) SwitchTranslateWay(way string) bool {
	if !SetTranslateWay(way) {
		return false
	}

	currentQuery := a.State.GetCurrentQuery()
	if currentQuery != "" {
		a.EventBus.EmitQuery(currentQuery)
		a.startCurrentQuery(currentQuery, a.State.GetToolbarMode())
	}
	return true
}

// SetDefaultExplainTemplate 设置默认解释模板。
func SetDefaultExplainTemplate(templateID string) {
	if err := config.Update(func(data *config.Config) {
		data.ExplainTemplates.DefaultTemplate = templateID
	}); err != nil {
		slog.Error("Failed to save config", slog.String("error", err.Error()))
	}
}
