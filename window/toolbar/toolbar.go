package toolbar

import (
	"runtime"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

var WindowName = "ToolBar"

var Window *application.WebviewWindow

var IsPinned = false            // 窗口固定状态
var QueryResultHeight int = 100 // 高度
var QueryResultWidth int = 450  // 宽度

var toolWindowStyleApplied sync.Once // 确保只应用一次样式

// NewWindow 截图功能也可以提取成一个单独程序，设计screenshot，robotgo库的使用
func NewWindow(app *application.App) {
	Window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:             WindowName,
		Width:             QueryResultWidth,
		Height:            QueryResultHeight,
		AlwaysOnTop:       false,
		Hidden:            true,
		DisableResize:     false,
		Frameless:         true,
		InitialPosition:   application.WindowCentered,
		EnableDragAndDrop: true,
		URL:               "http://wails.localhost/index.html",
		BackgroundType:    application.BackgroundTypeTranslucent, // 半透明背景，支持圆角
	})

	Window.SetMaxSize(QueryResultWidth+100, QueryResultHeight+500)

	// 处理窗口显示事件 - 在首次显示时应用 Windows 样式
	Window.OnWindowEvent(events.Common.WindowShow, func(e *application.WindowEvent) {
		if runtime.GOOS == "windows" {
			toolWindowStyleApplied.Do(func() {
				// 延迟一点，确保窗口完全创建并可以被 FindWindow 找到
				go func() {
					time.Sleep(100 * time.Millisecond)
					setupWindowsToolWindowStyle()
				}()
			})
		}
	})

	// 处理窗口关闭事件
	Window.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
		app.Quit()
	})

	// 处理失去焦点事件
	Window.OnWindowEvent(events.Common.WindowLostFocus, func(e *application.WindowEvent) {
		// 只有在未固定状态下才隐藏窗口
		if !IsPinned {
			// 延迟隐藏，避免与 WebView2 焦点管理冲突
			go func() {
				time.Sleep(100 * time.Millisecond)
				// 再次检查固定状态和窗口状态
				if !IsPinned && Window != nil {
					application.InvokeSync(func() {
						Window.Hide()
					})
				}
			}()
		}
	})
}
