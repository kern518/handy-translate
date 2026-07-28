package windows

import (
	"log/slog"

	"github.com/lxn/win"
)

// Window Windows os 窗口
type Window struct {
	Name string
	HWND win.HWND
}

// ShowForWindows windows 窗口下的弹窗， 因为wails的弹窗无法和通过鼠标有效弹出，这里采用windows原生api
func (w *Window) ShowForWindows() {
	if w == nil {
		slog.Error("ShowForWindows: 窗口对象为 nil")
		return
	}

	if w.HWND == 0 {
		slog.Error("ShowForWindows: 无效的窗口句柄",
			slog.String("windowName", w.Name))
		return
	}

	hwnd := w.HWND

	// ShowWindow 的返回值表示调用前窗口是否可见，而不是本次调用是否成功。
	// 因此先显示窗口，再通过 IsWindowVisible 检查最终状态。
	win.ShowWindow(hwnd, win.SW_SHOW)
	if !win.IsWindowVisible(hwnd) {
		slog.Warn("ShowForWindows: 窗口显示后仍不可见",
			slog.String("windowName", w.Name))
		return
	}

	// 窗口可见后再尝试切到前台。Windows 的前台焦点保护可能拒绝该请求，
	// 但不影响窗口已经显示，因此只记录为调试信息。
	if !win.SetForegroundWindow(hwnd) {
		slog.Debug("ShowForWindows: Windows 拒绝切换前台窗口",
			slog.String("windowName", w.Name))
	}

	slog.Debug("ShowForWindows: 成功显示窗口",
		slog.String("windowName", w.Name))
}
