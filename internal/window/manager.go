// Package window 提供统一的窗口管理（窗口管理器模式）。
package window

import (
	"log/slog"
	"runtime"
	"sync"

	"handy-translate/config"
	"handy-translate/internal/event"
	"handy-translate/os_api/windows"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ──────────────────────────────────────────────
// 窗口名称常量
// ──────────────────────────────────────────────

const (
	ToolbarWindowName    = "ToolBar"
	TranslateWindowName  = "Translate"
	ScreenshotWindowName = "Screenshot"
)

// ──────────────────────────────────────────────
// 模式常量
// ──────────────────────────────────────────────

const (
	ExplainMode   = "explain"
	TranslateMode = "translate"
)

// Manager 窗口管理器，统一管理所有窗口的显示/隐藏/定位。
type Manager struct {
	app        *application.App
	eventBus   *event.Bus
	Toolbar    *application.WebviewWindow
	Translate  *application.WebviewWindow
	Screenshot *application.WebviewWindow

	// 工具栏状态（通过 mutex 保护）
	mu                sync.RWMutex
	isPinned          bool
	QueryResultHeight int
	QueryResultWidth  int
	toolbarShowing    bool
	toolbarAnchorX    int
	toolbarAnchorY    int
	toolbarAnchorSet  bool
}

// NewManager 创建窗口管理器。
func NewManager(app *application.App, eventBus *event.Bus) *Manager {
	return &Manager{
		app:               app,
		eventBus:          eventBus,
		QueryResultHeight: 110,
		QueryResultWidth:  450,
	}
}

// Show 通过窗口名称显示窗口。
func (m *Manager) Show(windowName string) {
	win := m.GetWindow(windowName)
	if win == nil {
		slog.Error("Show: 窗口不存在", slog.String("windowName", windowName))
		return
	}
	win.Center()
	win.Show()
}

// Hide 通过窗口名称隐藏窗口。
func (m *Manager) Hide(windowName string) {
	win := m.GetWindow(windowName)
	if win == nil {
		slog.Error("Hide: 窗口不存在", slog.String("windowName", windowName))
		return
	}
	win.Hide()
	if windowName == ToolbarWindowName {
		m.ResetToolbarState()
	}
}

// GetWindow 通过名称获取窗口。
func (m *Manager) GetWindow(name string) *application.WebviewWindow {
	switch name {
	case ToolbarWindowName:
		return m.Toolbar
	case TranslateWindowName:
		return m.Translate
	case ScreenshotWindowName:
		return m.Screenshot
	default:
		return nil
	}
}

// GetPinned 获取工具栏固定状态（并发安全）。
func (m *Manager) GetPinned() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isPinned
}

// IsToolbarShowing 获取工具栏是否正在显示（并发安全）。
func (m *Manager) IsToolbarShowing() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.toolbarShowing
}

// SetToolbarShowing 设置工具栏显示状态（并发安全）。
func (m *Manager) SetToolbarShowing(showing bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.toolbarShowing = showing
}

// ResetToolbarState 重置工具栏状态（并发安全）。
func (m *Manager) ResetToolbarState() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.toolbarShowing = false
	m.toolbarAnchorSet = false
}

// SetPinned 设置工具栏固定状态（同时操作窗口置顶）。
func (m *Manager) SetPinned(pinned bool) {
	m.mu.Lock()
	m.isPinned = pinned
	m.mu.Unlock()

	if err := config.Update(func(data *config.Config) {
		data.ToolbarPinned = pinned
	}); err != nil {
		slog.Error("保存工具栏固定状态失败", slog.Any("error", err))
	}

	if m.Toolbar != nil {
		m.Toolbar.SetAlwaysOnTop(pinned)
		// 通过 EventBus 使用常量发射事件
		m.eventBus.EmitToolbarPinnedUpdated(pinned)
		slog.Info("工具栏置顶状态已更新", slog.Bool("pinned", pinned))
	}
}

// ShowToolbarAtCursor 在鼠标位置附近显示工具栏。
// 定位策略：按优先级尝试 4 个方向（右下→右上→左下→左上），
// 选择第一个能完全容纳弹窗的方向；若全部放不下，则选最大空间方向并缩高。
func (m *Manager) ShowToolbarAtCursor(height int) {
	w := m.Toolbar
	if w == nil {
		return
	}

	m.mu.Lock()
	h := min(height, m.QueryResultHeight+600)
	if h == 0 {
		h = m.QueryResultHeight
	}
	m.QueryResultHeight = h
	showing := m.toolbarShowing
	isPinned := m.isPinned
	anchorX := m.toolbarAnchorX
	anchorY := m.toolbarAnchorY
	anchorSet := m.toolbarAnchorSet
	m.mu.Unlock()

	ww := w.Width() // 窗口宽度
	w.SetSize(ww, h)
	w.SetAlwaysOnTop(isPinned)

	// 固定窗口由用户决定位置，内容变化时只调整大小。
	if showing && isPinned {
		return
	}

	xval := anchorX
	yval := anchorY
	if !showing || !anchorSet {
		if runtime.GOOS != "windows" {
			slog.Error("仅支持Windows平台", slog.String("platform", runtime.GOOS))
			return
		}
		pos := windows.GetCursorPos()
		xval, yval = int(pos.X), int(pos.Y)

		m.mu.Lock()
		m.toolbarAnchorX = xval
		m.toolbarAnchorY = yval
		m.toolbarAnchorSet = true
		m.mu.Unlock()
	}

	// 获取光标所在屏幕的工作区域（排除任务栏）
	// 注意：w.GetScreen() 返回的是窗口所在屏幕。多屏幕环境下，光标可能在
	// 另一个屏幕上，所以先将窗口移动到光标位置，确保 GetScreen 返回正确的屏幕。
	// 窗口首次显示时需要这样做；后续内容扩高时窗口已经位于正确的屏幕，
	// 不再临时移动到光标位置，避免可见窗口发生闪烁。
	if !showing {
		w.SetPosition(xval, yval)
	}
	sc, err := w.GetScreen()
	if err != nil || sc == nil {
		slog.Error("ShowToolbarAtCursor: 获取屏幕工作区失败",
			slog.Any("error", err))
		return
	}
	workArea := sc.WorkArea

	const gap = 10   // 窗口与光标之间的间距
	const minH = 120 // 最小允许高度

	// 工作区边界
	waLeft := workArea.X
	waRight := workArea.X + workArea.Width
	waTop := workArea.Y
	waBottom := workArea.Y + workArea.Height

	// ── 4 个候选方向 ──
	type candidate struct {
		name string // 方向名称（发送给前端）
		posX int
		posY int
		fitH int // 该方向可用的最大高度
		fitW int // 该方向可用的最大宽度
	}

	candidates := []candidate{
		{ // 右下（默认）
			name: "right-bottom",
			posX: xval + gap,
			posY: yval + gap,
			fitW: waRight - (xval + gap),
			fitH: waBottom - (yval + gap),
		},
		{ // 右上
			name: "right-top",
			posX: xval + gap,
			posY: yval - h - gap,
			fitW: waRight - (xval + gap),
			fitH: yval - gap - waTop,
		},
		{ // 左下
			name: "left-bottom",
			posX: xval - ww - gap,
			posY: yval + gap,
			fitW: xval - gap - waLeft,
			fitH: waBottom - (yval + gap),
		},
		{ // 左上
			name: "left-top",
			posX: xval - ww - gap,
			posY: yval - h - gap,
			fitW: xval - gap - waLeft,
			fitH: yval - gap - waTop,
		},
	}

	// 选择第一个能完全容纳弹窗的方向
	chosen := -1
	for i, c := range candidates {
		if c.fitW >= ww && c.fitH >= h {
			chosen = i
			break
		}
	}

	// 如果四个方向都放不下，选择可用空间（面积）最大的方向
	if chosen < 0 {
		bestArea := 0
		for i, c := range candidates {
			availW := min(c.fitW, ww)
			availH := min(c.fitH, h)
			if availW < 0 {
				availW = 0
			}
			if availH < 0 {
				availH = 0
			}
			area := availW * availH
			if area > bestArea {
				bestArea = area
				chosen = i
			}
		}
		if chosen < 0 {
			chosen = 0 // fallback
		}

		// 动态缩小高度以适应可用空间
		availH := candidates[chosen].fitH
		if availH < h && availH >= minH {
			h = availH
			w.SetSize(ww, h)
		} else if availH < minH {
			h = minH
			w.SetSize(ww, h)
		}
	}

	c := candidates[chosen]
	posX := c.posX
	posY := c.posY

	// 如果高度被缩小了，需要重新计算上方方向的 posY
	if chosen == 1 || chosen == 3 { // 上方方向
		posY = yval - h - gap
	}

	// 边界钳制（保护）
	if posX+ww > waRight {
		posX = waRight - ww
	}
	if posX < waLeft {
		posX = waLeft
	}
	if posY+h > waBottom {
		posY = waBottom - h
	}
	if posY < waTop {
		posY = waTop
	}

	slog.Debug("ShowToolbarAtCursor 定位",
		slog.String("direction", c.name),
		slog.Int("cursorX", xval), slog.Int("cursorY", yval),
		slog.Int("posX", posX), slog.Int("posY", posY),
		slog.Int("winW", ww), slog.Int("winH", h),
		slog.Int("workAreaX", workArea.X), slog.Int("workAreaY", workArea.Y),
		slog.Int("workAreaW", workArea.Width), slog.Int("workAreaH", workArea.Height))

	w.SetPosition(posX, posY)

	// 通知前端滑入方向
	if m.eventBus != nil {
		m.eventBus.EmitToolbarSlideDirection(c.name)
	}

	// 窗口已经显示时，尺寸变化后重新定位即可，不重复调用显示 API。
	if showing {
		return
	}

	// 显示窗口
	if runtime.GOOS == "windows" {
		win := windows.FindWindow(ToolbarWindowName)
		if win != nil {
			win.ShowForWindows()
		} else {
			m.Toolbar.Show()
		}
	} else {
		m.Toolbar.Show()
	}

	m.mu.Lock()
	m.toolbarShowing = true
	m.mu.Unlock()
}
