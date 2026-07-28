// screenshot.go 完整的截图+OCR+翻译流程。
package app

import (
	"image/png"
	"log/slog"
	"os"
	"runtime"

	"handy-translate/os_api/windows"
	"handy-translate/window/screenshot"
)

// HandleCaptureSelectedScreen 截取选中区域 → OCR → 翻译。
func (a *Application) HandleCaptureSelectedScreen(startX, startY, width, height float64) {
	croppedImg := screenshot.CaptureSelectedScreen(int(startX), int(startY), int(width), int(height))
	if croppedImg == nil {
		return
	}

	tempFile, err := os.CreateTemp("", "handy-translate-ocr-*.png")
	if err != nil {
		slog.Error("创建 OCR 临时文件失败", slog.Any("err", err))
		return
	}
	filename := tempFile.Name()
	defer os.Remove(filename)

	if err := png.Encode(tempFile, croppedImg); err != nil {
		tempFile.Close()
		slog.Error("写入 OCR 临时文件失败", slog.Any("err", err))
		return
	}
	if err := tempFile.Close(); err != nil {
		slog.Error("关闭 OCR 临时文件失败", slog.Any("err", err))
		return
	}

	// OCR 解析文本
	queryText := a.OCR.Recognize(filename)
	slog.Info("OCR 识别完成", slog.Int("textLen", len(queryText)))
	if queryText == "" {
		a.EventBus.EmitStreamError("未识别到文字，请重新选择截图区域")
		return
	}

	a.State.SetCurrentQuery(queryText)
	a.WindowMgr.ResetToolbarState()

	// 发送查询文本
	a.EventBus.EmitQuery(queryText)

	// 复用统一查询流程，使截图查询同样能够取消前一个后台请求。
	mode := a.State.GetToolbarMode()
	a.startCurrentQuery(queryText, mode)
}

// HandleScreenshotEvent 处理截图快捷键事件。
func (a *Application) HandleScreenshotEvent() {
	x, y := 0, 0
	if runtime.GOOS == "windows" {
		cursor := windows.GetCursorPos()
		x, y = int(cursor.X), int(cursor.Y)
	}

	base64Image, bounds := screenshot.ScreenshotAt(x, y)
	if base64Image == "" {
		a.EventBus.EmitStreamError("截图失败，请重试")
		return
	}

	if a.WindowMgr.Screenshot != nil && !bounds.Empty() {
		a.WindowMgr.Screenshot.SetPosition(bounds.Min.X, bounds.Min.Y)
		a.WindowMgr.Screenshot.SetSize(bounds.Dx(), bounds.Dy())
	}
	a.EventBus.EmitScreenshotBase64(base64Image)
}
