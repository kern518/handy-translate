package screenshot

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/draw"
	"image/png"
	"log/slog"
	"sync"

	"github.com/kbinani/screenshot"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

var ScreenshotImg *image.RGBA
var screenshotMu sync.RWMutex

var WindowName = "Screenshot"

var Window *application.WebviewWindow

// NewWindow 截图功能也可以提取成一个单独程序，设计screenshot，robotgo库的使用
func NewWindow(app *application.App) {
	Window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:           WindowName,
		InitialPosition: application.WindowCentered,
		Hidden:          true,
		KeyBindings: map[string]func(window application.Window){
			"escape": func(window application.Window) {
				window.Hide()
			},
			"F12": func(window application.Window) {
				if w, ok := window.(*application.WebviewWindow); ok {
					w.OpenDevTools()
				}
			},
		},
		BackgroundType: application.BackgroundTypeTransparent,
		URL:            "http://wails.localhost/screenshot.html",
	})

	Window.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
		app.Logger.Info("[Event] Window WindowClosing win2")
		e.Cancel()
		Window.Hide()
	})

}

func ScreenshotFullScreen() string {
	base64Image, _ := ScreenshotAt(0, 0)
	return base64Image
}

// ScreenshotAt 截取包含指定桌面坐标的显示器，并返回该显示器的桌面边界。
func ScreenshotAt(x, y int) (string, image.Rectangle) {
	displayCount := screenshot.NumActiveDisplays()
	if displayCount == 0 {
		slog.Error("未检测到可用显示器")
		return "", image.Rectangle{}
	}

	displayBounds := make([]image.Rectangle, 0, displayCount)
	for index := 0; index < displayCount; index++ {
		displayBounds = append(displayBounds, screenshot.GetDisplayBounds(index))
	}
	displayIndex := selectDisplayIndex(image.Pt(x, y), displayBounds)
	bounds := displayBounds[displayIndex]
	img, err := screenshot.CaptureRect(bounds)

	if err != nil {
		// 错误处理，输出错误信息并返回
		slog.Error("Error capturing screenshot", slog.Any("err", err))
		return "", image.Rectangle{}
	}
	screenshotMu.Lock()
	ScreenshotImg = img
	screenshotMu.Unlock()

	base64Image := encodeImageToBase64(img)
	if base64Image == "" {
		// 错误处理，未能生成Base64图像，返回
		slog.Error("Error encoding image to Base64")
		return "", image.Rectangle{}
	}
	return base64Image, bounds
}

func selectDisplayIndex(point image.Point, bounds []image.Rectangle) int {
	for index, displayBounds := range bounds {
		if point.In(displayBounds) {
			return index
		}
	}
	return 0
}

// CaptureSelectedScreen 截图功能
func CaptureSelectedScreen(startX, startY, endwidth, endheight int) image.Image {
	slog.Debug("CaptureSelectedScreen",
		slog.Int("startX", startX),
		slog.Int("startY", startY),
		slog.Int("endwidth", endwidth),
		slog.Int("endheight", endheight))

	screenshotMu.RLock()
	source := ScreenshotImg
	screenshotMu.RUnlock()
	if source == nil {
		bounds := screenshot.GetDisplayBounds(0)
		img, err := screenshot.CaptureRect(bounds)

		if err != nil {
			slog.Error("截图失败", slog.Any("error", err))
			return nil
		}
		screenshotMu.Lock()
		ScreenshotImg = img
		source = img
		screenshotMu.Unlock()
	}

	rect := image.Rect(startX, startY, endwidth, endheight).Intersect(source.Bounds())
	if rect.Empty() {
		slog.Warn("截图选区超出有效范围",
			slog.Any("selection", image.Rect(startX, startY, endwidth, endheight)),
			slog.Any("bounds", source.Bounds()))
		return nil
	}

	// 复制选区，避免返回的 SubImage 继续引用全屏截图的大块内存。
	cropped := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	draw.Draw(cropped, cropped.Bounds(), source, rect.Min, draw.Src)
	return cropped
}

// 将图像编码为Base64字符串
func encodeImageToBase64(img image.Image) string {
	// 创建一个缓冲区用于保存Base64编码的数据
	var imgBytes []byte
	buf := new(bytes.Buffer)
	err := png.Encode(buf, img)
	if err != nil {
		slog.Error("截图编码失败", slog.Any("error", err))
		return ""
	}

	imgBytes = buf.Bytes()

	// 使用base64编码图像数据
	base64Image := base64.StdEncoding.EncodeToString(imgBytes)

	return base64Image
}
