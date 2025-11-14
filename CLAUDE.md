# CLAUDE.md

此文件为 Claude Code (claude.ai/code) 在此代码库中工作时提供指导。

## 项目概述

Handy-translate 是一个基于 Wails v3 (Go + React) 构建的翻译工具。它通过鼠标手势和键盘快捷键提供便捷的翻译体验。应用程序支持多个翻译服务，并包含截图翻译的 OCR 功能。

**主要功能：**
- 鼠标中键翻译选中文本
- 截图 OCR 翻译
- 支持多个翻译提供商（百度、有道、彩云、DeepSeek）
- 流式翻译支持（DeepSeek）
- 系统托盘集成
- 专注于 Windows 平台支持

## 开发命令

### Wails v3 命令
```bash
# 开发模式，支持热重载
wails3 dev

# 构建应用程序
wails3 build

# 打包应用程序（创建安装程序）
wails package

# 显示 Wails 环境信息
wails3 show
```

### 测试
```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./translate_service/baidu
go test ./translate_service/deepseek
```

### 前端开发
前端是位于 `frontend/` 的 React 应用程序：
```bash
cd frontend
npm install
npm run dev
npm run build
```

## 架构概述

### Go 后端结构

**主应用程序 (`main.go`)**
- 初始化带有嵌入前端资源的 Wails 应用程序
- 设置三个窗口：工具栏、翻译和截图
- 配置带菜单选项的系统托盘
- 处理语言和模式更改的全局事件监听器

**核心服务 (`app.go`)**
- 通过 Wails 导出到前端的主服务
- 处理翻译逻辑（常规和流式）
- 管理窗口显示/隐藏操作
- 处理 OCR 和截图功能
- 协调不同翻译服务之间的交互

**翻译服务 (`translate_service/`)**
- 支持多个提供商的基于接口的架构
- `Translate` 接口用于基本翻译
- `StreamTranslate` 接口用于流式翻译（DeepSeek）
- 独立提供商：百度、有道、彩云、DeepSeek

**配置 (`config/`)**
- 基于 TOML 的配置系统
- 支持多个翻译服务配置
- 键盘快捷键自定义
- DeepSeek 的解释模板系统

**平台特定代码**
- 专注于 Windows 的实现，位于 `os_api/windows/`
- 通过 `pchook/` 进行全局鼠标/键盘钩子
- `window/` 包中的窗口管理

### 前端结构

**React 应用程序**
- 使用 Vite 和现代 React 模式构建
- 使用 NextUI 组件库
- 三个主要 HTML 入口点：`toolbar.html`、`translate.html`、`screenshot.html`
- 用于状态管理和 Wails 通信的自定义钩子

**主要前端组件**
- 工具栏窗口：快速翻译弹出窗口
- 翻译窗口：完整翻译界面
- 截图窗口：OCR 翻译界面
- 不同翻译提供商的服务层

## 配置

### 主配置文件 (`config.toml`)
```toml
appname = "handy-translate"
translate_way = "baidu"

[keyboards]
toolBar = ["center", "", ""]  # 工具栏鼠标中键
screenshot = ["ctrl", "shift", "f"]  # 截图快捷键

[translate.baidu]
name = "百度翻译"
appID = "your_app_id"
key = "your_key"

[translate.deepseek]
name = 'DeepSeek'
appID = 'deepseek'
key = 'your_deepseek_key'

[explain_templates]
default_template = "template1"
[explain_templates.templates.template1]
name = "模板名称"
description = "模板描述"
template = "实际模板内容"
```

### 重要配置说明
- `config.toml.bak` 是模板文件
- 重命名为 `config.toml` 并填入您的 API 密钥
- 每个翻译服务需要不同的 API 凭证
- DeepSeek 支持流式翻译和解释模板

## 关键依赖

### Go 后端
- `github.com/wailsapp/wails/v3` - 应用程序框架
- `github.com/go-vgo/robotgo` - 鼠标/键盘自动化
- `github.com/kbinani/screenshot` - 屏幕截图
- `github.com/tmc/langchaingo` - AI/LLM 集成
- `github.com/pelletier/go-toml/v2` - 配置解析

### 前端
- React with Vite
- NextUI 组件库
- Tailwind CSS 用于样式

## OCR 集成

应用程序使用外部 OCR 工具：
- `RapidOCR-json.exe` 用于从截图中提取文本
- 位于项目根目录
- `models/` 目录中的 75MB 模型文件

## 翻译提供商

### 支持的服务
1. **百度翻译** - 需要 appID 和 key
2. **有道翻译** - 需要 appID 和 appSecret
3. **彩云翻译** - 需要 appID 和 key
4. **DeepSeek** - 支持流式翻译和解释模板

### 流式翻译
只有 DeepSeek 支持流式翻译，提供：
- 实时翻译结果
- 使用自定义模板的解释功能
- 长文本的更好用户体验

## 平台支持

目前专注于 Windows，具有：
- 用于窗口管理的原生 Windows API 集成
- 全局热键支持
- 系统托盘功能
- 平台特定的鼠标/键盘钩子

其他平台可能可以编译但缺乏完整功能支持。

## 开发说明

- Wails v3 处于 alpha 阶段 - API 可能会更改
- 前端资源嵌入到 Go 二进制文件中
- 具有适当处理的单实例应用程序
- Go 后端和 React 前端之间的事件驱动架构
- 用于调试翻译问题的广泛日志记录