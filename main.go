package main

import (
	"embed"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"

	"handy-translate/config"
	"handy-translate/history"
	internalApp "handy-translate/internal/app"
	"handy-translate/internal/event"
	"handy-translate/internal/service"
	"handy-translate/internal/translate"
	"handy-translate/internal/window"
	screenshotWin "handy-translate/window/screenshot"
	"handy-translate/window/toolbar"
	translateWin "handy-translate/window/translate"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed frontend/dist
var assets embed.FS

//go:embed frontend/public/appicon.png
var iconlogo []byte

var projectName = "handy-translate"

func main() {
	// ──────────────────────────────────────────
	// 1. 创建 Wails 应用（Binding 稍后注入）
	// ──────────────────────────────────────────
	// Binding 需要在 Application 组装完成后填充
	var binding internalApp.Binding

	wailsApp := application.New(application.Options{
		Name: projectName,
		Services: []application.Service{
			application.NewService(&binding),
		},
		Icon: iconlogo,
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.wails.handy-translate",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				slog.Info("Second instance launched", slog.Any("args", data.Args))
			},
			AdditionalData: map[string]string{
				"launchtime": time.Now().String(),
			},
		},
	})

	// ──────────────────────────────────────────
	// 2. 创建窗口
	// ──────────────────────────────────────────
	toolbar.NewWindow(wailsApp)
	translateWin.NewWindow(wailsApp)
	screenshotWin.NewWindow(wailsApp)

	// ──────────────────────────────────────────
	// 3. 初始化配置
	// ──────────────────────────────────────────
	config.Init(projectName)
	runtimeConfig := config.Snapshot()

	// ──────────────────────────────────────────
	// 4. 组装依赖（依赖注入）
	// ──────────────────────────────────────────

	// 事件总线（观察者模式）
	eventBus := event.NewBus(wailsApp)

	// 翻译提供者注册表（注册表模式 + 策略模式）
	providerRegistry := translate.NewRegistry()
	translate.RegisterAll(providerRegistry)

	// 历史记录服务
	historySvc := history.NewHistoryService(runtimeConfig.History.Enabled, runtimeConfig.History.StoragePath)

	// 翻译业务门面（门面模式）
	wordCache := service.NewWordCache("data/word_cache")
	translator := service.NewTranslator(providerRegistry, &config.Data, eventBus, historySvc, wordCache)

	// 窗口管理器
	windowMgr := window.NewManager(wailsApp, eventBus)
	windowMgr.Toolbar = toolbar.Window
	windowMgr.Translate = translateWin.Window
	windowMgr.Screenshot = screenshotWin.Window

	// OCR 服务
	ocrSvc := service.NewOCRService(resolveOCRExecutable())

	// 应用核心（依赖注入容器）
	app := internalApp.NewApplication(
		wailsApp,
		translator,
		windowMgr,
		eventBus,
		historySvc,
		ocrSvc,
	)

	// 将 Application 注入到 Binding（适配器模式）
	binding = *internalApp.NewBinding(app)

	// ──────────────────────────────────────────
	// 5. 注册事件（观察者模式）
	// ──────────────────────────────────────────
	app.RegisterEvents()

	// 从配置读取工具栏模式
	if runtimeConfig.ToolbarMode != "" {
		app.SetToolbarMode(runtimeConfig.ToolbarMode)
		slog.Info("从配置读取工具栏模式", slog.String("mode", runtimeConfig.ToolbarMode))
	} else {
		app.SetToolbarMode("translate")
	}

	// 从配置读取工具栏固定状态
	if runtimeConfig.ToolbarPinned {
		windowMgr.SetPinned(true)
		slog.Info("从配置读取工具栏固定状态", slog.Bool("pinned", true))
	}

	// ──────────────────────────────────────────
	// 6. 系统托盘
	// ──────────────────────────────────────────
	systemTray := wailsApp.SystemTray.New()
	myMenu := wailsApp.Menu.New()

	providerMenu := myMenu.AddSubmenu("翻译服务")
	type providerMenuItem struct {
		id   string
		name string
	}
	providerItems := make([]providerMenuItem, 0, len(runtimeConfig.Translate))
	for id, provider := range runtimeConfig.Translate {
		providerItems = append(providerItems, providerMenuItem{id: id, name: provider.Name})
	}
	sort.Slice(providerItems, func(i, j int) bool {
		return providerItems[i].name < providerItems[j].name
	})
	for _, provider := range providerItems {
		item := provider
		providerMenu.AddRadio(item.name, item.id == runtimeConfig.TranslateWay).OnClick(func(ctx *application.Context) {
			app.SwitchTranslateWay(item.id)
		})
	}

	myMenu.AddSeparator()

	myMenu.Add("截图").OnClick(func(ctx *application.Context) {
		app.HandleScreenshotEvent()
	})

	myMenu.Add("退出").OnClick(func(ctx *application.Context) {
		wailsApp.Quit()
	})

	systemTray.SetMenu(myMenu)
	systemTray.SetIcon(iconlogo)

	systemTray.OnClick(func() {
		toolbar.Window.Show()
	})

	// ──────────────────────────────────────────
	// 7. 启动 Hook 监听 + 运行应用
	// ──────────────────────────────────────────
	go app.ProcessHook()

	slog.Info("🚀 应用启动完成", slog.String("name", projectName))

	err := wailsApp.Run()
	if err != nil {
		panic(err)
	}
}

func resolveOCRExecutable() string {
	const executableName = "RapidOCR-json.exe"

	if executable, err := os.Executable(); err == nil {
		bundledPath := filepath.Join(filepath.Dir(executable), executableName)
		if _, statErr := os.Stat(bundledPath); statErr == nil {
			return bundledPath
		}
	} else {
		slog.Warn("无法获取程序目录，尝试从当前工作目录查找 OCR", slog.Any("error", err))
	}

	if workingDirectory, err := os.Getwd(); err == nil {
		developmentPath := filepath.Join(workingDirectory, executableName)
		if _, statErr := os.Stat(developmentPath); statErr == nil {
			return developmentPath
		}
	}

	// 保留发布目录语义，使后续启动错误日志包含预期的绝对路径。
	if executable, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(executable), executableName)
	}
	return executableName
}
