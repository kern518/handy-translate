package translate_service

import (
	"handy-translate/config"
	"handy-translate/translate_service/baidu"
	"handy-translate/translate_service/caiyun"
	"handy-translate/translate_service/deepseek"
	"handy-translate/translate_service/google"
	"handy-translate/translate_service/minimax"
	"handy-translate/translate_service/youdao"
)

type Translate interface {
	GetName() string
	PostQuery(query, sourceLang, targetLang string) ([]string, error)
}

// StreamTranslate 支持流式输出的翻译接口
type StreamTranslate interface {
	Translate
	PostQueryStream(query, sourceLang, targetLang string, callback func(chunk string)) error
	PostExplainStream(query, templateID string, callback func(chunk string)) error
}

// 翻译服务注册表：新增服务只需在 init() 中注册
var registry = map[string]func(config.Translate) Translate{}

func init() {
	registry[youdao.Way] = func(cfg config.Translate) Translate {
		return &youdao.Youdao{Translate: cfg}
	}
	registry[caiyun.Way] = func(cfg config.Translate) Translate {
		return &caiyun.Caiyun{Translate: cfg}
	}
	registry[baidu.Way] = func(cfg config.Translate) Translate {
		return &baidu.Baidu{Translate: cfg}
	}
	registry[deepseek.Way] = func(cfg config.Translate) Translate {
		return &deepseek.Deepseek{Translate: cfg}
	}
	registry[minimax.Way] = func(cfg config.Translate) Translate {
		return &minimax.Minimax{Translate: cfg}
	}
	registry[google.Way] = func(cfg config.Translate) Translate {
		return &google.Google{Translate: cfg}
	}
}

// GetTranslateWay 通过注册表获取翻译服务实例
func GetTranslateWay(way string) Translate {
	factory, ok := registry[way]
	if !ok {
		return nil
	}

	currentConfig := config.Snapshot()
	cfgEntry, exists := currentConfig.Translate[way]
	if !exists {
		return nil
	}

	return factory(config.Translate{
		Name:    cfgEntry.Name,
		AppID:   cfgEntry.AppID,
		Key:     cfgEntry.Key,
		BaseURL: cfgEntry.BaseURL,
		Model:   cfgEntry.Model,
	})
}
