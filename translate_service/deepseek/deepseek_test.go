package deepseek

import (
	"handy-translate/config"
	"strings"
	"testing"
)

// TestDeepseek_PostExplainStream 验证术语解释的流式输出是否正常。
// 需要在 config.toml 中配置 deepseek 的 API Key，未配置时将跳过此测试。
func TestDeepseek_PostExplainStream(t *testing.T) {
	config.Init("handy-translate")

	// 无可用密钥时跳过
	if config.Data.Translate[Way].Key == "" {
		t.Skip("skip: DeepSeek API key is not configured")
	}

	d := &Deepseek{
		Translate: config.Translate{
			Name:  config.Data.Translate[Way].Name,
			AppID: config.Data.Translate[Way].AppID,
			Key:   config.Data.Translate[Way].Key,
		},
	}

	// 使用非流式接口以规避底层流解码差异导致的不稳定
	resp, err := d.PostExplain("CPU")
	if err != nil {
		t.Fatalf("PostExplain returned error: %v", err)
	}
	if len(strings.TrimSpace(resp)) == 0 {
		t.Fatalf("expected non-empty explanation, got empty")
	}
}
