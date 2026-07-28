package windows

import (
	"fmt"
	"log/slog"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"handy-translate/config"
)

// 因为windows下的robotgo鼠标获取文本内容有些瑕疵，故这里用windows原生api增强
const (
	WH_MOUSE_LL    = 14
	WM_MBUTTONDOWN = 0x0207
	WM_MBUTTONUP   = 0x0208
)

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	setWindowsHookExW   = user32.NewProc("SetWindowsHookExW")
	callNextHookEx      = user32.NewProc("CallNextHookEx")
	unhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	getMessageW         = user32.NewProc("GetMessageW")
	getClipboardSeq     = user32.NewProc("GetClipboardSequenceNumber")
	keybdEventProc      = user32.NewProc("keybd_event") // 键盘事件函数

	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	getModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	rtlMoveMemory    = kernel32.NewProc("RtlMoveMemory")
)

const (
	KEYEVENTF_KEYUP = 0x0002
	VK_CONTROL      = 0x11
	VK_C            = 0x43
)

const (
	WH_KEYBOARD_LL = 13
	WM_KEYDOWN     = 0x0100
	WM_KEYUP       = 0x0101
	WM_SYSKEYDOWN  = 0x0104
	WM_SYSKEYUP    = 0x0105

	VK_LCONTROL = 0xA2
	VK_RCONTROL = 0xA3
	VK_LSHIFT   = 0xA0
	VK_RSHIFT   = 0xA1
	VK_LMENU    = 0xA4 // Alt
	VK_RMENU    = 0xA5
	VK_LWIN     = 0x5B
	VK_RWIN     = 0x5C
)

type KBDLLHOOKSTRUCT struct {
	vkCode      uint32
	scanCode    uint32
	flags       uint32
	time        uint32
	dwExtraInfo uintptr
}

type POINT struct {
	X, Y int32
}

var (
	hMouseHook    uintptr
	hKeyboardHook uintptr
)

// HookChan channle
var HookChan = make(chan string, 10)

type hotkey struct {
	modifiers []string
	key       uint32
	label     string
}

var (
	pressedKeys            = make(map[uint32]bool)
	screenshotHotkey       hotkey
	screenshotHotkeyActive bool
	mouseWorkerOnce        sync.Once
	mouseCopyRequests      = make(chan struct{}, 1)
)

// MSG Windows 消息结构体（供 GetMessageW 使用）。
type MSG struct {
	HWND    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

// LowLevelMouseProc 代用windows api 才能做到选中文字，鼠标事件触发前执行模拟ctrl + c 操作
func LowLevelMouseProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
	if nCode >= 0 && isMiddleMouseMessage(wParam) {
		if wParam == WM_MBUTTONDOWN {
			select {
			case mouseCopyRequests <- struct{}{}:
			default:
				slog.Debug("鼠标取词请求仍在处理中，忽略重复事件")
			}
			// 中键是取词触发键，不再把按下事件交给前台编辑器。
			// 否则编辑器会先取消文本选区，随后 Ctrl+C 复制的就会是整行。
		} else {
			// 成对吞掉抬起事件，避免前台程序收到不完整的中键序列。
		}
		return 1
	}
	r1, _, _ := callNextHookEx.Call(uintptr(nCode), wParam, lParam)
	return r1
}

func isMiddleMouseMessage(message uintptr) bool {
	return message == WM_MBUTTONDOWN || message == WM_MBUTTONUP
}

func WindowsHook() {
	startMouseCopyWorker()

	currentConfig := config.Snapshot()
	screenshotHotkey = parseHotkey(
		currentConfig.Keyboards["screenshot"],
		[]string{"alt", "shift", "q"},
	)
	slog.Info("截图快捷键已注册", slog.String("hotkey", screenshotHotkey.label))

	// 启动键盘钩子
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		hMod, _, _ := getModuleHandleW.Call(0)

		var err error
		hKeyboardHook, _, err = setWindowsHookExW.Call(
			uintptr(WH_KEYBOARD_LL),
			syscall.NewCallback(onKeyboard),
			hMod,
			0,
		)
		if hKeyboardHook == 0 {
			slog.Error("键盘钩子安装失败", slog.Any("error", err))
			return
		}
		defer unhookWindowsHookEx.Call(hKeyboardHook)

		slog.Info("键盘钩子已安装")

		var msg MSG
		// 必须在同一个线程中处理消息循环
		for {
			ret, _, messageErr := getMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if ret == 0 {
				break
			}
			if ret == ^uintptr(0) {
				slog.Error("键盘钩子消息循环失败", slog.Any("error", messageErr))
				break
			}
		}
	}()

	// 启动鼠标钩子（主线程中）
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hMod, _, _ := getModuleHandleW.Call(0)
	var mouseHookErr error
	hMouseHook, _, mouseHookErr = setWindowsHookExW.Call(
		uintptr(WH_MOUSE_LL),
		syscall.NewCallback(LowLevelMouseProc),
		hMod,
		0,
	)
	if hMouseHook == 0 {
		slog.Error("鼠标钩子安装失败", slog.Any("error", mouseHookErr))
		return
	}
	defer unhookWindowsHookEx.Call(hMouseHook)

	var msg MSG
	// 监听消息
	for {
		ret, _, messageErr := getMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if ret == 0 {
			break
		}
		if ret == ^uintptr(0) {
			slog.Error("鼠标钩子消息循环失败", slog.Any("error", messageErr))
			break
		}
	}
}

// startMouseCopyWorker 将模拟复制和等待剪贴板更新移出低级钩子回调。
// Windows 要求低级钩子快速返回，否则会造成全局输入卡顿甚至移除钩子。
func startMouseCopyWorker() {
	mouseWorkerOnce.Do(func() {
		go func() {
			for range mouseCopyRequests {
				previousSequence := clipboardSequenceNumber()
				PressCtrlC()
				if !waitForClipboardChange(previousSequence, 500*time.Millisecond, 10*time.Millisecond, clipboardSequenceNumber) {
					slog.Warn("复制选中文本超时，忽略本次取词")
					continue
				}
				select {
				case HookChan <- "mouse":
				default:
					slog.Warn("鼠标取词事件队列已满，忽略本次事件")
				}
			}
		}()
	})
}

func clipboardSequenceNumber() uint32 {
	sequence, _, _ := getClipboardSeq.Call()
	return uint32(sequence)
}

func waitForClipboardChange(
	previous uint32,
	timeout time.Duration,
	interval time.Duration,
	current func() uint32,
) bool {
	if current == nil {
		return false
	}
	if interval <= 0 {
		interval = 10 * time.Millisecond
	}

	deadline := time.Now().Add(timeout)
	for {
		if current() != previous {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(interval)
	}
}

func onKeyboard(nCode int, wParam, lParam uintptr) uintptr {
	if nCode >= 0 {
		// lParam 是由 Windows 提供的临时结构指针。通过系统内存复制 API
		// 将其复制到 Go 管理的值，避免把 uintptr 长期转换为 unsafe.Pointer。
		var kbd KBDLLHOOKSTRUCT
		rtlMoveMemory.Call(
			uintptr(unsafe.Pointer(&kbd)),
			lParam,
			unsafe.Sizeof(kbd),
		)
		switch wParam {
		case WM_KEYDOWN, WM_SYSKEYDOWN:
			pressedKeys[kbd.vkCode] = true
			if !screenshotHotkeyActive && screenshotHotkey.matches(pressedKeys) {
				screenshotHotkeyActive = true
				slog.Debug("截图快捷键匹配成功",
					slog.String("hotkey", screenshotHotkey.label))
				select {
				case HookChan <- "screenshot":
				default:
					slog.Warn("截图事件队列已满，忽略本次快捷键")
				}
			}
		case WM_KEYUP, WM_SYSKEYUP:
			pressedKeys[kbd.vkCode] = false
			if kbd.vkCode == screenshotHotkey.key {
				screenshotHotkeyActive = false
			}
		}
	}
	ret, _, _ := callNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
	return ret
}

func parseHotkey(values, fallback []string) hotkey {
	spec, err := buildHotkey(values)
	if err == nil {
		return spec
	}

	if len(values) > 0 {
		slog.Warn("截图快捷键配置无效，使用默认值",
			slog.Any("value", values),
			slog.String("error", err.Error()))
	}
	spec, _ = buildHotkey(fallback)
	return spec
}

func buildHotkey(values []string) (hotkey, error) {
	var spec hotkey
	var labels []string

	for _, raw := range values {
		token := strings.ToLower(strings.TrimSpace(raw))
		if token == "" {
			continue
		}

		switch token {
		case "ctrl", "control":
			spec.modifiers = append(spec.modifiers, "ctrl")
			labels = append(labels, "Ctrl")
		case "shift":
			spec.modifiers = append(spec.modifiers, "shift")
			labels = append(labels, "Shift")
		case "alt":
			spec.modifiers = append(spec.modifiers, "alt")
			labels = append(labels, "Alt")
		case "win", "windows", "meta":
			spec.modifiers = append(spec.modifiers, "win")
			labels = append(labels, "Win")
		default:
			if spec.key != 0 {
				return hotkey{}, fmt.Errorf("只能配置一个普通按键")
			}
			key, label, err := virtualKey(token)
			if err != nil {
				return hotkey{}, err
			}
			spec.key = key
			labels = append(labels, label)
		}
	}

	if spec.key == 0 {
		return hotkey{}, fmt.Errorf("缺少普通按键")
	}
	spec.label = strings.Join(labels, "+")
	return spec, nil
}

func virtualKey(token string) (uint32, string, error) {
	if len(token) == 1 {
		ch := token[0]
		switch {
		case ch >= 'a' && ch <= 'z':
			return uint32(ch - 'a' + 'A'), strings.ToUpper(token), nil
		case ch >= '0' && ch <= '9':
			return uint32(ch), token, nil
		}
	}

	if strings.HasPrefix(token, "f") {
		number, err := strconv.Atoi(strings.TrimPrefix(token, "f"))
		if err == nil && number >= 1 && number <= 12 {
			return uint32(0x70 + number - 1), strings.ToUpper(token), nil
		}
	}

	switch token {
	case "space":
		return 0x20, "Space", nil
	case "enter":
		return 0x0D, "Enter", nil
	case "escape", "esc":
		return 0x1B, "Esc", nil
	}

	return 0, "", fmt.Errorf("不支持的按键 %q", token)
}

func (h hotkey) matches(keys map[uint32]bool) bool {
	if h.key == 0 || !keys[h.key] {
		return false
	}

	for _, modifier := range h.modifiers {
		switch modifier {
		case "ctrl":
			if !keys[VK_LCONTROL] && !keys[VK_RCONTROL] {
				return false
			}
		case "shift":
			if !keys[VK_LSHIFT] && !keys[VK_RSHIFT] {
				return false
			}
		case "alt":
			if !keys[VK_LMENU] && !keys[VK_RMENU] {
				return false
			}
		case "win":
			if !keys[VK_LWIN] && !keys[VK_RWIN] {
				return false
			}
		}
	}

	return true
}
