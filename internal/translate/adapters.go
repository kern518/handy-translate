// Package translate 提供翻译服务提供者的适配器，
// 将旧的 translate_service 下的实现适配到新的 Provider/StreamProvider 接口。
package translate

import (
	"context"

	"handy-translate/config"
	"handy-translate/translate_service/baidu"
	"handy-translate/translate_service/caiyun"
	"handy-translate/translate_service/deepseek"
	"handy-translate/translate_service/google"
	"handy-translate/translate_service/minimax"
	"handy-translate/translate_service/youdao"
)

// ──────────────────────────────────────────────
// 适配器：将旧接口包装为新 Provider 接口
// ──────────────────────────────────────────────

// baidu 适配器
type baiduAdapter struct{ inner *baidu.Baidu }

func (a *baiduAdapter) Name() string { return baidu.Way }
func (a *baiduAdapter) Translate(ctx context.Context, req TranslateRequest) ([]string, error) {
	return a.inner.PostQueryContext(ctx, req.Text, req.SourceLang, req.TargetLang)
}

// caiyun 适配器
type caiyunAdapter struct{ inner *caiyun.Caiyun }

func (a *caiyunAdapter) Name() string { return caiyun.Way }
func (a *caiyunAdapter) Translate(ctx context.Context, req TranslateRequest) ([]string, error) {
	return a.inner.PostQueryContext(ctx, req.Text, req.SourceLang, req.TargetLang)
}

// youdao 适配器
type youdaoAdapter struct{ inner *youdao.Youdao }

func (a *youdaoAdapter) Name() string { return youdao.Way }
func (a *youdaoAdapter) Translate(ctx context.Context, req TranslateRequest) ([]string, error) {
	return a.inner.PostQueryContext(ctx, req.Text, req.SourceLang, req.TargetLang)
}

// google 适配器（支持流式）
type googleAdapter struct{ inner *google.Google }

func (a *googleAdapter) Name() string { return google.Way }
func (a *googleAdapter) Translate(ctx context.Context, req TranslateRequest) ([]string, error) {
	return a.inner.PostQueryContext(ctx, req.Text, req.SourceLang, req.TargetLang)
}
func (a *googleAdapter) TranslateStream(ctx context.Context, req TranslateRequest, onChunk func(string)) error {
	return a.inner.PostQueryStreamContext(ctx, req.Text, req.SourceLang, req.TargetLang, onChunk)
}
func (a *googleAdapter) ExplainStream(ctx context.Context, text, templateID string, onChunk func(string)) error {
	return a.inner.PostExplainStreamContext(ctx, text, templateID, onChunk)
}

// deepseek 适配器（支持流式）
type deepseekAdapter struct{ inner *deepseek.Deepseek }

func (a *deepseekAdapter) Name() string { return deepseek.Way }
func (a *deepseekAdapter) Translate(ctx context.Context, req TranslateRequest) ([]string, error) {
	return a.inner.PostQueryContext(ctx, req.Text, req.SourceLang, req.TargetLang)
}
func (a *deepseekAdapter) TranslateStream(ctx context.Context, req TranslateRequest, onChunk func(string)) error {
	return a.inner.PostQueryStreamContext(ctx, req.Text, req.SourceLang, req.TargetLang, onChunk)
}
func (a *deepseekAdapter) ExplainStream(ctx context.Context, text, templateID string, onChunk func(string)) error {
	return a.inner.PostExplainStreamContext(ctx, text, templateID, onChunk)
}

// minimax 适配器（支持流式）
type minimaxAdapter struct{ inner *minimax.Minimax }

func (a *minimaxAdapter) Name() string { return minimax.Way }
func (a *minimaxAdapter) Translate(ctx context.Context, req TranslateRequest) ([]string, error) {
	return a.inner.PostQueryContext(ctx, req.Text, req.SourceLang, req.TargetLang)
}
func (a *minimaxAdapter) TranslateStream(ctx context.Context, req TranslateRequest, onChunk func(string)) error {
	return a.inner.PostQueryStreamContext(ctx, req.Text, req.SourceLang, req.TargetLang, onChunk)
}
func (a *minimaxAdapter) ExplainStream(ctx context.Context, text, templateID string, onChunk func(string)) error {
	return a.inner.PostExplainStreamContext(ctx, text, templateID, onChunk)
}

// ──────────────────────────────────────────────
// RegisterAll 注册所有翻译提供者到 Registry
// ──────────────────────────────────────────────

// RegisterAll 将所有已知的翻译服务注册到 Registry。
func RegisterAll(r *Registry) {
	r.Register(baidu.Way, func(cfg config.Translate) Provider {
		return &baiduAdapter{inner: &baidu.Baidu{Translate: cfg}}
	})
	r.Register(caiyun.Way, func(cfg config.Translate) Provider {
		return &caiyunAdapter{inner: &caiyun.Caiyun{Translate: cfg}}
	})
	r.Register(youdao.Way, func(cfg config.Translate) Provider {
		return &youdaoAdapter{inner: &youdao.Youdao{Translate: cfg}}
	})
	r.Register(google.Way, func(cfg config.Translate) Provider {
		return &googleAdapter{inner: &google.Google{Translate: cfg}}
	})
	r.Register(deepseek.Way, func(cfg config.Translate) Provider {
		return &deepseekAdapter{inner: &deepseek.Deepseek{Translate: cfg}}
	})
	r.Register(minimax.Way, func(cfg config.Translate) Provider {
		return &minimaxAdapter{inner: &minimax.Minimax{Translate: cfg}}
	})
}

// compile-time interface checks
var _ Provider = (*baiduAdapter)(nil)
var _ Provider = (*caiyunAdapter)(nil)
var _ Provider = (*youdaoAdapter)(nil)
var _ StreamProvider = (*googleAdapter)(nil)
var _ StreamProvider = (*deepseekAdapter)(nil)
var _ StreamProvider = (*minimaxAdapter)(nil)
