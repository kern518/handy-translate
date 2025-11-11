package translate

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

var WindowName = "Translate"

var Window *application.WebviewWindow

// NewWindow 截图功能也可以提取成一个单独程序，设计screenshot，robotgo库的使用
func NewWindow(app *application.App) {
	Window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     WindowName,
		Width:     500,
		Height:    500,
		Frameless: true,
		Hidden:    true,
		URL:       "http://wails.localhost/translate.html",
	})

	if Window == nil {
		app.Logger.Error("创建翻译窗口失败: Window 为 nil")
		return
	}

	Window.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
		app.Logger.Info("[Event] Window WindowClosing win2")
		e.Cancel()
		Window.Hide()
	})

	// 添加窗口显示事件日志，用于调试
	Window.OnWindowEvent(events.Common.WindowShow, func(e *application.WindowEvent) {
		app.Logger.Info("[Event] 翻译窗口已显示")
	})
}
