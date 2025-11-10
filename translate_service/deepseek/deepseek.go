package deepseek

import (
	"context"
	"log"
	"sync"

	"handy-translate/config"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/prompts"
)

const Way = "deepseek"

var (
	once sync.Once
	llm  *openai.LLM
)

type Deepseek struct {
	config.Translate
}

type TranslationPayload struct {
	Source    []string `json:"source"`
	TransType string   `json:"trans_type"`
	RequestID string   `json:"request_id"`
	Detect    bool     `json:"detect"`
}

type TranslationResponse struct {
	Target []string `json:"target"`
}

func (c *Deepseek) GetName() string {
	return Way
}

func (c *Deepseek) GetLLM() *openai.LLM {
	once.Do(func() {
		var err error
		llm, err = openai.New(
			openai.WithToken(config.Data.Translate[Way].Key),
			openai.WithModel("deepseek-chat"),
			openai.WithBaseURL("https://api.deepseek.com"),
		)
		if err != nil {
			log.Fatal(err)
		}
	})
	return llm
}

func (c *Deepseek) PostQuery(query, fromLang, toLang string) ([]string, error) {
	// Initialize the OpenAI client with Deepseek model

	// // 定义模板
	// promptTemplate := prompts.NewPromptTemplate(
	// 	"You are a professional translator.\n"+
	// 		"Please translate the following text accurately and naturally.\n"+
	// 		"Keep the original meaning, tone, and formatting.\n"+
	// 		"Do not explain or add anything else.\n\n"+
	// 		"If the text is Chinese, translate to English.\n"+
	// 		"If the text is English, translate to Chinese.\n\n"+
	// 		"Text:\n{{.text}}",
	// 	[]string{"text"},
	// )

	// // 构建输入
	// promptValue, err := promptTemplate.Format(map[string]any{
	// 	"text": query,
	// })
	// if err != nil {
	// 	panic(err)
	// }

	// // 调用 LLM
	// resp, err := llms.GenerateFromSinglePrompt(context.Background(), c.GetLLM(), promptValue)
	// if err != nil {
	// 	panic(err)
	// }

	// slog.Info(resp)

	// return []string{resp, ""}, nil
	return []string{"", ""}, nil
}

// PostExplain 非流式术语解释，便于测试与一次性获取完整结果
func (c *Deepseek) PostExplain(query string) (string, error) {
	promptTemplate := prompts.NewPromptTemplate(
		"你是一名技术术语专家。\n"+
			"请用简洁、清晰的中文解释以下技术术语。\n"+
			"要求：\n"+
			"1. 简要说明它是什么及核心原理\n"+
			"2. 概述主要用途或应用场景\n"+
			"3. 控制在 3~5 句话内，让人能快速理解\n\n"+
			"术语：\n{{.text}}",
		[]string{"text"},
	)

	promptValue, err := promptTemplate.Format(map[string]any{
		"text": query,
	})
	if err != nil {
		return "", err
	}

	// 非流式一次性生成
	resp, err := llms.GenerateFromSinglePrompt(context.Background(), c.GetLLM(), promptValue)
	if err != nil {
		return "", err
	}
	return resp, nil
}

// PostQueryStream 流式翻译
func (c *Deepseek) PostQueryStream(query, fromLang, toLang string, callback func(chunk string)) error {
	// 定义模板
	promptTemplate := prompts.NewPromptTemplate(
		"You are a professional translator.\n"+
			"Please translate the following text accurately and naturally.\n"+
			"Keep the original meaning, tone, and formatting.\n"+
			"Do not explain or add anything else.\n\n"+
			"If the text is Chinese, translate to English.\n"+
			"If the text is English, translate to Chinese.\n\n"+
			"Text:\n{{.text}}",
		[]string{"text"},
	)

	// 构建输入
	promptValue, err := promptTemplate.Format(map[string]any{
		"text": query,
	})
	if err != nil {
		return err
	}

	// 流式调用 LLM
	ctx := context.Background()
	_, err = c.GetLLM().GenerateContent(ctx, []llms.MessageContent{
		{
			Parts: []llms.ContentPart{
				llms.TextPart(promptValue),
			},
			Role: llms.ChatMessageTypeHuman,
		},
	}, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		// 每次接收到数据块时调用回调函数
		if len(chunk) > 0 {
			callback(string(chunk))
		}
		return nil
	}))

	return err
}

// PostExplainStream 流式术语解释（支持模板选择）
func (c *Deepseek) PostExplainStream(query, templateID string, callback func(chunk string)) error {
	// 获取模板内容
	templateStr := c.getTemplate(templateID)

	// 定义术语解释模板
	promptTemplate := prompts.NewPromptTemplate(
		templateStr,
		[]string{"text"},
	)

	// 构建输入
	promptValue, err := promptTemplate.Format(map[string]any{
		"text": query,
	})
	if err != nil {
		return err
	}

	// 流式调用 LLM
	ctx := context.Background()
	_, err = c.GetLLM().GenerateContent(ctx, []llms.MessageContent{
		{
			Parts: []llms.ContentPart{
				llms.TextPart(promptValue),
			},
			Role: llms.ChatMessageTypeHuman,
		},
	}, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		// 每次接收到数据块时调用回调函数
		if len(chunk) > 0 {
			callback(string(chunk))
		}
		return nil
	}))

	return err
}

// getTemplate 获取提示词模板，支持从配置中读取
func (c *Deepseek) getTemplate(templateID string) string {
	// 如果配置为空，使用硬编码的默认模板
	if len(config.Data.ExplainTemplates.Templates) == 0 {
		return ""
	}

	// 如果 templateID 为空，使用默认模板
	if templateID == "" {
		templateID = config.Data.ExplainTemplates.DefaultTemplate
	}

	// 如果默认模板也为空，使用第一个可用模板
	if templateID == "" {
		for id := range config.Data.ExplainTemplates.Templates {
			templateID = id
			break
		}
	}

	// 从配置中获取模板
	if template, exists := config.Data.ExplainTemplates.Templates[templateID]; exists {
		return template.Template
	}

	// 如果指定的模板不存在，尝试使用默认模板
	if config.Data.ExplainTemplates.DefaultTemplate != "" {
		if template, exists := config.Data.ExplainTemplates.Templates[config.Data.ExplainTemplates.DefaultTemplate]; exists {
			return template.Template
		}
	}

	// 如果都找不到，使用第一个可用模板
	for _, template := range config.Data.ExplainTemplates.Templates {
		return template.Template
	}

	// 最后的回退：使用硬编码的默认模板
	return ""
}
