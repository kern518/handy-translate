package translate_service

import (
	"sync"

	"handy-translate/config"
	"handy-translate/translate_service/baidu"
	"handy-translate/translate_service/caiyun"
	"handy-translate/translate_service/deepseek"
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

func GetTranslateWay(way string) Translate {
	var t Translate
	switch way {
	case youdao.Way:
		t = &youdao.Youdao{
			Translate: config.Translate{
				Name:  config.Data.Translate[way].Name,
				AppID: config.Data.Translate[way].AppID,
				Key:   config.Data.Translate[way].Key,
			},
		}
	case caiyun.Way:
		t = &caiyun.Caiyun{
			Translate: config.Translate{
				Name:  config.Data.Translate[way].Name,
				AppID: config.Data.Translate[way].AppID,
				Key:   config.Data.Translate[way].Key,
			},
		}
	case baidu.Way:
		t = &baidu.Baidu{
			Translate: config.Translate{
				Name:  config.Data.Translate[way].Name,
				AppID: config.Data.Translate[way].AppID,
				Key:   config.Data.Translate[way].Key,
			},
		}
	case deepseek.Way:
		t = &deepseek.Deepseek{
			Translate: config.Translate{
				Name:  config.Data.Translate[way].Name,
				AppID: config.Data.Translate[way].AppID,
				Key:   config.Data.Translate[way].Key,
			},
		}
	}

	return t
}

var queryText string

var lk sync.RWMutex

// SetQueryText
func SetQueryText(value string) {
	lk.Lock()
	queryText = value
	lk.Unlock()
}

// GetQueryText
func GetQueryText() string {
	lk.RLock()
	defer lk.RUnlock()
	return queryText
}
