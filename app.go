package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"handy-translate/config"
	"handy-translate/os_api/windows"
	"handy-translate/translate_service"
	"handy-translate/utils"
	"handy-translate/window/screenshot"
	"handy-translate/window/toolbar"
	"handy-translate/window/translate"
	"image/png"
	"log/slog"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/wailsapp/wails/v3/pkg/application"
)

var toolbarIsShowing bool = false // 标记工具栏是否已经显示

// 和js绑定的go方法集合
type AppInterface interface {
	Show(windowName string)
	Hide(windowName string)
	ToolBarShow(height float64)
	SetToolBarPinned(pinned bool)
	GetToolBarPinned() bool
}

// App is a service
type App struct{}

var currentToolbarMode = "translate" // 当前工具栏模式：translate/explain

// GetToolbarMode 获取工具栏模式
func GetToolbarMode() string {
	return currentToolbarMode
}

// SetToolbarMode 设置工具栏模式
func SetToolbarMode(mode string) {
	currentToolbarMode = mode
}

// MyFetch URl
func (a *App) MyFetch(URL string, content map[string]interface{}) interface{} {
	return utils.MyFetch(URL, content)
}

// Transalte 翻译逻辑
func (a *App) Transalte(queryText, fromLang, toLang string) string {
	app.Logger.Info("Transalte",
		slog.Any("queryText", queryText),
		slog.Any("toLang", toLang),
		slog.Any("fromLang", fromLang))

	res := processTranslate(queryText)
	return res
}

// TransalteStream 流式翻译逻辑（仅支持 DeepSeek）
func (a *App) TransalteStream(queryText, fromLang, toLang string) {
	app.Logger.Info("TransalteStream",
		slog.Any("queryText", queryText),
		slog.Any("toLang", toLang),
		slog.Any("fromLang", fromLang))

	translateWay := translate_service.GetTransalteWay(config.Data.TranslateWay)

	// 检查是否支持流式输出
	if streamTranslate, ok := translateWay.(translate_service.StreamTranslate); ok {
		// 支持流式输出
		slog.Info("使用流式翻译")
		err := streamTranslate.PostQueryStream(queryText, fromLang, toLang, func(chunk string) {
			// 每次收到数据块时发送事件到前端
			slog.Info("发送流式数据块", slog.String("chunk", chunk), slog.Int("length", len(chunk)))
			app.Event.Emit("result_stream", chunk)
		})

		if err != nil {
			slog.Error("PostQueryStream", slog.Any("err", err))
			// 发送错误事件
			app.Event.Emit("result_stream_error", err.Error())
		} else {
			// 发送完成事件
			app.Event.Emit("result_stream_done", "done")
		}
	} else {
		// 不支持流式输出，使用普通翻译
		res := processTranslate(queryText)
		app.Event.Emit("result", res)
	}
}

// ExplainStream 流式解释逻辑（仅支持 DeepSeek，支持模板选择）
func (a *App) ExplainStream(queryText, templateID string) {
	app.Logger.Info("ExplainStream",
		slog.Any("queryText", queryText),
		slog.Any("templateID", templateID))

	translateWay := translate_service.GetTransalteWay(config.Data.TranslateWay)

	// 检查是否支持流式输出
	if streamTranslate, ok := translateWay.(translate_service.StreamTranslate); ok {
		// 支持流式输出
		slog.Info("使用流式解释")
		err := streamTranslate.PostExplainStream(queryText, templateID, func(chunk string) {
			// 每次收到数据块时发送事件到前端
			slog.Info("发送流式解释数据块", slog.String("chunk", chunk), slog.Int("length", len(chunk)))
			app.Event.Emit("result_stream", chunk)
		})

		if err != nil {
			slog.Error("PostExplainStream", slog.Any("err", err))
			// 发送错误事件
			app.Event.Emit("result_stream_error", err.Error())
		} else {
			// 发送完成事件
			app.Event.Emit("result_stream_done", "done")
		}
	} else {
		// 不支持流式输出，使用普通解释
		res := processExplain(queryText, templateID)
		app.Event.Emit("result", res)
	}
}

// GetTransalteMap 获取所有翻译配置
func (a *App) GetTransalteMap() string {
	var translateList = config.Data.Translate
	bTranslate, err := json.Marshal(translateList)
	if err != nil {
		logrus.WithError(err).Error("Marshal")
	}
	return string(bTranslate)
}

// SetTransalteWay 设置当前翻译服务
func (a *App) SetTransalteWay(translateWay string) {
	config.Data.TranslateWay = translateWay
	translate_service.SetQueryText("")
	config.Save()
	slog.Info("SetTransalteList", slog.Any("config.Data.Translate", config.Data.Translate))
}

// GetTransalteWay 获取当前翻译的服务
func (a *App) GetTransalteWay() string {
	return config.Data.TranslateWay
}

// GetExplainTemplates 获取所有解释模板
func (a *App) GetExplainTemplates() string {
	templates := make(map[string]map[string]interface{})

	// 如果配置为空，返回空结果
	if len(config.Data.ExplainTemplates.Templates) == 0 {
		return "{}"
	}

	// 构建返回数据
	for id, template := range config.Data.ExplainTemplates.Templates {
		templates[id] = map[string]interface{}{
			"id":          id,
			"name":        template.Name,
			"description": template.Description,
		}
	}

	result := map[string]interface{}{
		"default_template": config.Data.ExplainTemplates.DefaultTemplate,
		"templates":        templates,
	}

	b, err := json.Marshal(result)
	if err != nil {
		logrus.WithError(err).Error("Marshal ExplainTemplates")
		return "{}"
	}
	return string(b)
}

// SetDefaultExplainTemplate 设置默认解释模板
func (a *App) SetDefaultExplainTemplate(templateID string) {
	config.Data.ExplainTemplates.DefaultTemplate = templateID
	config.Save()
	slog.Info("SetDefaultExplainTemplate", slog.String("templateID", templateID))
}

// Show 通过名字控制窗口事件
func (a *App) Show(windowName string) {
	var win *application.WebviewWindow
	switch windowName {
	case screenshot.WindowName:
		win = screenshot.Window
	case translate.WindowName:
		win = translate.Window
	case toolbar.WindowName:
		win = toolbar.Window
	}

	// 检查窗口是否存在
	if win == nil {
		app.Logger.Error("Show: 窗口不存在", slog.String("windowName", windowName))
		return
	}

	win.Center()
	win.Show()
}

// Hide 通过名字控制窗口事件
func (a *App) Hide(windowName string) {
	var win *application.WebviewWindow
	switch windowName {
	case screenshot.WindowName:
		win = screenshot.Window
	case translate.WindowName:
		win = translate.Window
	case toolbar.WindowName:
		win = toolbar.Window
	}

	// 检查窗口是否存在
	if win == nil {
		app.Logger.Error("Hide: 窗口不存在", slog.String("windowName", windowName))
		return
	}

	win.Hide()
}

// ToolBarShow 显示工具弹窗，控制大小，布局, 前端调用，传递文本高度
func (a *App) ToolBarShow(height float64) {
	// 40 + 55 窗口空白区域+翻译的图标区域
	height = height + 35 + 54
	app.Logger.Info("ToolBarShow", slog.Float64("height", height), slog.Bool("isShowing", toolbarIsShowing))

	h := min(int(height), toolbar.QueryResultHeight+500)

	if h == 0 {
		h = toolbar.QueryResultHeight
	}
	toolbar.QueryResultHeight = h
	processToolbarShow()
}

// SetToolBarPinned 设置工具栏固定状态
func (a *App) SetToolBarPinned(pinned bool) {
	toolbar.IsPinned = pinned
	app.Logger.Info("SetToolBarPinned", slog.Bool("pinned", pinned))
}

// GetToolBarPinned 获取工具栏固定状态
func (a *App) GetToolBarPinned() bool {
	app.Logger.Info("GetToolBarPinned", slog.Bool("pinned", toolbar.IsPinned))
	return toolbar.IsPinned
}

func processToolbarShow() {
	height := toolbar.QueryResultHeight
	w := toolbar.Window
	w.SetSize(w.Width(), height)

	// 如果窗口已经显示，只调整大小，不改变位置
	if toolbarIsShowing {
		slog.Info("工具栏已显示，仅调整大小", slog.Int("height", height))
		// 窗口已经显示，不需要重新定位
		return
	}

	xval := 0
	yval := 0

	if runtime.GOOS == "windows" {
		pos := windows.GetCursorPos()
		xval, yval = int(pos.X), int(pos.Y) // 处理获取坐标不正确，采用windows原生api
	} else {
		slog.Error("仅支持Windows平台", slog.String("platform", runtime.GOOS))
		return
	}

	sc, _ := w.GetScreen()
	// 计算屏幕任务多出的高度，防止弹出框超出屏幕外面
	c := int(float64(sc.Size.Height) * 0.1)

	// 计算左边对应的窗体高度是否超出屏幕外，超出则需要重新计算y轴坐标，防止弹出框超出屏幕外面
	if yval+height+c >= sc.Size.Height {
		gap := yval + height + c - sc.Size.Height
		slog.Info("窗口初始定位（超出屏幕）", slog.Int("gap", gap), slog.Int("x", xval+10), slog.Int("y", yval-gap))
		w.SetPosition(xval+10, yval-gap)
	} else {
		slog.Info("窗口初始定位（正常）", slog.Int("x", xval+10), slog.Int("y", yval+10))
		w.SetPosition(xval+10, yval+10)
	}

	// 显示窗口
	if runtime.GOOS == "windows" {
		// Windows 平台：尝试使用原生 API 显示窗口（更可靠）
		win := windows.FindWindow(toolbar.WindowName)
		if win != nil {
			slog.Info("使用 Windows 原生 API 显示工具栏")
			win.ShowForWindows()
		} else {
			// 找不到窗口时使用 Wails API 作为后备
			slog.Warn("无法通过 FindWindow 找到工具栏，使用 Wails Show() 方法")
			toolbar.Window.Show()
		}
	} else {
		// 非 Windows 平台：使用 Wails API
		toolbar.Window.Show()
	}

	// 标记窗口已显示
	toolbarIsShowing = true
	slog.Info("工具栏已显示并标记")
}

// ResetToolbarState 重置工具栏状态（在窗口隐藏时调用）
func ResetToolbarState() {
	toolbarIsShowing = false
	slog.Info("工具栏状态已重置")
}

// CaptureSelectedScreen 截取选中的区域
func (a *App) CaptureSelectedScreen(startX, startY, width, height float64) {

	croppedImg := screenshot.CaptureSelectedScreen(int(startX), int(startY), int(width), int(height))
	if croppedImg == nil {
		return
	}

	var buf bytes.Buffer
	err := png.Encode(&buf, croppedImg)
	if err != nil {
		slog.Error("png.Encode", slog.Any("err", err))
		return
	}

	filename := "screenshot.png" // 保存的文件名
	base64String := base64.StdEncoding.EncodeToString(buf.Bytes())

	err = saveBase64Image(base64String, filename)
	if err != nil {
		slog.Error("saveBase64Image", slog.Any("err", err))
	}

	// OCR解析文本
	queryText := ExecOCR(".\\RapidOCR-json.exe", filename)

	// 重置工具栏状态，准备新的翻译
	ResetToolbarState()

	// 检查是否使用了流式翻译
	translateWay := translate_service.GetTransalteWay(config.Data.TranslateWay)

	// 无论是流式还是普通翻译，都先发送 query 事件让前端准备
	sendQueryText(queryText)

	if _, ok := translateWay.(translate_service.StreamTranslate); ok {
		// 流式翻译：开始流式翻译（会发送 result_stream 事件）
		translateRes := processTranslate(queryText)
		slog.Info("截图OCR流式翻译完成，结果长度，模式", slog.Int("len", len(translateRes)), slog.String("mode", GetToolbarMode()))
	} else {
		// 普通翻译：翻译后发送完整结果
		translateRes := processTranslate(queryText)
		sendResult(translateRes, "")
	}
}

// 翻译处理
func processTranslate(queryText string) string {
	translateWay := translate_service.GetTransalteWay(config.Data.TranslateWay)

	// 检查是否支持流式输出
	if streamTranslate, ok := translateWay.(translate_service.StreamTranslate); ok {
		// 支持流式输出
		slog.Info("使用流式翻译")
		var streamResult string
		err := streamTranslate.PostQueryStream(queryText, fromLang, toLang, func(chunk string) {
			streamResult += chunk
			// 每次收到数据块时发送事件到前端
			slog.Info("发送流式数据块", slog.String("chunk", chunk), slog.Int("length", len(chunk)))
			app.Event.Emit("result_stream", chunk)
		})

		if err != nil {
			slog.Error("PostQueryStream", slog.Any("err", err))
			app.Event.Emit("result_stream_error", err.Error())
			return ""
		}

		// 发送完成事件
		app.Event.Emit("result_stream_done", "done")

		app.Logger.Info("流式翻译完成",
			slog.String("result", streamResult),
			slog.String("translateWay", translateWay.GetName()))

		return streamResult
	}

	// 不支持流式，使用普通翻译
	result, err := translateWay.PostQuery(queryText, fromLang, toLang)
	if err != nil {
		slog.Error("PostQuery", slog.Any("err", err))
	}

	app.Logger.Info("Transalte",
		slog.Any("result", result),
		slog.Any("translateWay", translateWay.GetName()))

	translateRes := strings.Join(result, "\n")

	return translateRes
}

// 解释处理（支持模板选择）
func processExplain(queryText, templateID string) string {
	translateWay := translate_service.GetTransalteWay(config.Data.TranslateWay)

	// 检查是否支持流式输出
	if streamTranslate, ok := translateWay.(translate_service.StreamTranslate); ok {
		// 支持流式输出
		slog.Info("使用流式解释")
		var streamResult string
		err := streamTranslate.PostExplainStream(queryText, templateID, func(chunk string) {
			streamResult += chunk
			// 每次收到数据块时发送事件到前端
			slog.Info("发送流式解释数据块", slog.String("chunk", chunk), slog.Int("length", len(chunk)))
			app.Event.Emit("result_stream", chunk)
		})

		if err != nil {
			slog.Error("PostExplainStream", slog.Any("err", err))
			app.Event.Emit("result_stream_error", err.Error())
			return ""
		}

		// 发送完成事件
		app.Event.Emit("result_stream_done", "done")

		app.Logger.Info("流式解释完成",
			slog.String("result", streamResult),
			slog.String("translateWay", translateWay.GetName()))

		return streamResult
	}

	// 不支持流式
	slog.Error("PostExplainStream", slog.String("err", "不支持流式解释"))
	return ""
}

func sendQueryText(queryText string) {
	app.Event.Emit("query", queryText)
}

func sendResult(result, explains string) {
	app.Event.Emit("result", result)
	app.Event.Emit("explains", explains)
}

// 监听处理鼠标事件
func processHook() {
	// TODO 工厂设计模式
	if runtime.GOOS == "windows" {
		go windows.WindowsHook()
	}

	for msg := range windows.HookChan {
		switch msg {
		case "mouse":
			result, ok := app.Clipboard.Text()
			if !ok {
				app.Logger.Error("Failed to get clipboard text")
			}

			queryText := result

			app.Logger.Info("processHook GetQueryText",
				slog.String("queryText", queryText),
				slog.String("fromLang", fromLang),
				slog.String("toLang", toLang))

			// 当工具栏已固定时，不重置或重新定位窗口，直接在现有窗口渲染数据
			if !toolbar.IsPinned {
				ResetToolbarState()
				processToolbarShow()
			}

			if queryText != translate_service.GetQueryText() && queryText != "" {
				translate_service.SetQueryText(queryText)

				// 检查是否使用了流式翻译
				translateWay := translate_service.GetTransalteWay(config.Data.TranslateWay)

				// 无论是流式还是普通翻译，都先发送 query 事件让前端准备
				sendQueryText(queryText)

				// 根据工具栏模式选择翻译或解释
				mode := GetToolbarMode()
				if mode == "explain" {
					// 解释模式：由前端根据选中的模板主动调用 ExplainStream
					// 这里不直接调用，让前端收到 query 事件后主动调用
					slog.Info("解释模式，等待前端调用 ExplainStream")
				} else {
					// 翻译模式（默认）
					if _, ok := translateWay.(translate_service.StreamTranslate); ok {
						// 流式翻译：开始流式翻译（会发送 result_stream 事件）
						translateRes := processTranslate(queryText)
						slog.Info("流式翻译完成，结果长度", slog.Int("len", len(translateRes)))
					} else {
						// 普通翻译：翻译后发送完整结果
						translateRes := processTranslate(queryText)
						sendResult(translateRes, "")
					}
				}
			}
		case "screenshot":
			screenshot.ScreenshotFullScreen()
			base64Image := screenshot.ScreenshotFullScreen()
			app.Event.Emit("screenshotBase64", base64Image)
		default:
			app.Logger.Error("processHook", slog.String("msg", msg))
		}

	}
}
