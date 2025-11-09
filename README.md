# 概述

基于wails框架，结合Go+React，开发支持Windows的翻译小工具。当鼠标选中文本时，点击鼠标中键，弹出窗口渲染翻译结果。目前支持Windows平台，其他平台理论上也可以编译运行，但功能未开放完整。通过wails构建生成的包容量相较于Electron小，仅有10M左右。 本次版本开发使用的是wails[v3](https://v3alpha.wails.io/)版本, 欢迎您提出使用该项目的问题, 我会积极跟进的~

# 功能说明

- [X] 鼠标选中文字进行翻译
- [X] 通过配置文件自定义快捷键
- [X] 支持有道，百度，彩云，DeepSeek翻译源
- [X] 支持截图OCR翻译
- [X] 系统托盘

# 效果展示

- 点击**鼠标中键**选中文本弹出窗口
- 按压**CTRL+SHIFT+F**截图翻译弹出窗口

![示例视频](https://raw.githubusercontent.com/byzze/oss/main/handly-translate/effect.gif)

# 安装编译环境

[wails安装参考教程](https://v3alpha.wails.io/getting-started/installation/)

**v3处于alpha测试版本，不是稳定版本，使用过程中，需要多参考参考源码和文档**

```
wails3 show
```

# 配置翻译源

填写对应的翻译秘钥

**修改配置名**
`config.toml.bak -> config.toml`

**填写对应的api信息**

```toml
appname = "handy-translate"
translate_way = "baidu"

[keyboards] 
toolBar = ["center", "", ""] # 小窗口翻译快捷键， 表示鼠标中键
screenshot = ["ctrl", "shift",  "f"] # 截图快捷键

[translate]
[translate.baidu] # 秘钥文档：https://fanyi-api.baidu.com/api/trans/product/apidoc
name = "百度翻译"
appID = "20230823001790949"
key = "hTlcbpu7xxxxxxxxx"

[translate.youdao] # 秘钥文档：https://ai.youdao.com/DOCSIRMA/html/trans/api/wbfy/index.html
name = "有道翻译"
appID = "appKey"
key = "appSecret"

[translate.deepseek] # https://platform.deepseek.com/
name = 'DeepSeek'
appID = 'deepseek'
key = ''
```

# 开发调试

`wails3 build`

`wails3 dev`

# 构建程序

## 方式一

直接编译可执行文件, 存在dist文件

`wails package`

## 方式二

` wails3 build`

# 执行

- Windows双击生成文件 `handry-translate.exe`

<!-- # 下载文件

[windows安装文件](https://github.com/byzze/handy-translate/releases/download/v1.0.2/handy-translate-amd64-installer.exe) -->

# OCR models

实现截图ocr解析文件模型，该模型有点大，大约75M， 文件夹：models

# 参考用到的工具组件链接

- [robotgo](https://github.com/go-vgo/robotgo) 鼠标，键盘监听
- [wails v2](https://wails.io)
- [wails v3](https://v3alpha.wails.io/)
- [NEXTUI](https://nextui.org/) 前端UI组件
- [pot-desktop](https://github.com/pot-app/pot-desktop) rust开发的跨平台翻译工具
- [go-qoq](https://github.com/duolabmeng6/go-qoq) wails3开发的翻译工具
