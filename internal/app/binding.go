// binding.go 提供 Wails 前端绑定方法（适配器模式）。
package app

import (
	"log/slog"

	"handy-translate/config"
	"handy-translate/utils"
)

// Binding 是绑定到 Wails 前端的服务（适配器模式）。
// 将内部 Application 的方法适配为前端可调用的格式。
type Binding struct {
	app *Application
}

// NewBinding 创建前端绑定适配器。
func NewBinding(app *Application) *Binding {
	return &Binding{app: app}
}

// ──────────────────────────────────────────────
// 前端绑定方法（供 Wails 调用）
// ──────────────────────────────────────────────

// MyFetch 封装 HTTP 请求（解决前端跨域问题）。
func (b *Binding) MyFetch(URL string, content map[string]interface{}) interface{} {
	return utils.MyFetch(URL, content)
}

// Translate 翻译接口。
func (b *Binding) Translate(queryText, fromLang, toLang string) string {
	slog.Info("🔍 Translate 开始",
		slog.String("text_preview", TruncateText(queryText, 50)),
		slog.String("from", fromLang),
		slog.String("to", toLang))

	ctx, cancel, queryID := b.app.beginQuery()
	defer b.app.finishQuery(queryID, cancel)
	return b.app.Translator.Translate(ctx, queryText, fromLang, toLang)
}

// TranslateMeanings 翻译释义接口。
func (b *Binding) TranslateMeanings(queryText, fromLang, toLang string) string {
	slog.Info("🔍 TranslateMeanings 开始",
		slog.String("text_preview", TruncateText(queryText, 50)),
		slog.String("from", fromLang),
		slog.String("to", toLang))

	ctx, cancel, queryID := b.app.beginQuery()
	defer b.app.finishQuery(queryID, cancel)
	return b.app.Translator.TranslateMeanings(ctx, queryText, fromLang, toLang)
}

// TranslateStream 流式翻译接口。
func (b *Binding) TranslateStream(queryText, fromLang, toLang string) {
	slog.Info("🔍 TranslateStream 开始",
		slog.String("text_preview", TruncateText(queryText, 50)),
		slog.String("from", fromLang),
		slog.String("to", toLang))

	ctx, cancel, queryID := b.app.beginQuery()
	defer b.app.finishQuery(queryID, cancel)
	b.app.Translator.TranslateStream(ctx, queryText, fromLang, toLang)
}

// GetTranslateMap 获取所有翻译配置。
func (b *Binding) GetTranslateMap() string {
	return GetTranslateMapJSON()
}

// SetTranslateWay 设置当前翻译服务。
func (b *Binding) SetTranslateWay(translateWay string) {
	b.app.SwitchTranslateWay(translateWay)
}

// GetTranslateWay 获取当前翻译服务。
func (b *Binding) GetTranslateWay() string {
	return config.Snapshot().TranslateWay
}

// GetExplainTemplates 获取所有解释模板。
func (b *Binding) GetExplainTemplates() string {
	return GetExplainTemplatesJSON()
}

// SetDefaultExplainTemplate 设置默认解释模板。
func (b *Binding) SetDefaultExplainTemplate(templateID string) {
	SetDefaultExplainTemplate(templateID)
}

// GetToolbarMode 获取工具栏模式。
func (b *Binding) GetToolbarMode() string {
	return b.app.State.GetToolbarMode()
}

// Show 通过名字显示窗口。
func (b *Binding) Show(windowName string) {
	b.app.WindowMgr.Show(windowName)
}

// Hide 通过名字隐藏窗口。
func (b *Binding) Hide(windowName string) {
	b.app.WindowMgr.Hide(windowName)
}

// ToolBarShow 显示工具栏弹窗。
func (b *Binding) ToolBarShow(height float64) {
	h := int(height) + 56 // CardHeader(~46px) + padding(~10px)，与前端 calc(100vh - 42px) 对齐
	slog.Debug("ToolBarShow",
		slog.Float64("height", height),
		slog.Bool("isShowing", b.app.WindowMgr.IsToolbarShowing()))
	b.app.WindowMgr.ShowToolbarAtCursor(h)
}

// SetToolBarPinned 设置工具栏固定状态（同时操作窗口置顶）。
func (b *Binding) SetToolBarPinned(pinned bool) {
	b.app.WindowMgr.SetPinned(pinned)
	slog.Info("SetToolBarPinned", slog.Bool("pinned", pinned))
}

// GetToolBarPinned 获取工具栏固定状态。
func (b *Binding) GetToolBarPinned() bool {
	return b.app.WindowMgr.GetPinned()
}

// QueryWord 单词查询：使用 LLM 一次返回音标、词性、释义（中英）、例句（中英）。
// 结果通过 word_query_result 事件发送（避免 Wails RPC 长时间阻塞）。
func (b *Binding) QueryWord(word string) {
	slog.Info("📖 QueryWord 开始", slog.String("word", word))
	ctx, cancel, queryID := b.app.beginQuery()
	go func() {
		defer b.app.finishQuery(queryID, cancel)
		b.app.Translator.QueryWord(ctx, word)
	}()
}

// CaptureSelectedScreen 截取选中区域 → OCR → 翻译。
func (b *Binding) CaptureSelectedScreen(startX, startY, width, height float64) {
	b.app.HandleCaptureSelectedScreen(startX, startY, width, height)
}
