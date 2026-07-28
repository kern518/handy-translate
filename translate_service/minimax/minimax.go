package minimax

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"handy-translate/config"
)

const (
	Way              = "minimax"
	DefaultBaseURL   = "https://api.minimaxi.com"
	DefaultModel     = "MiniMax-M1"
	TranslatePrompts = `You are a professional translator.
Please translate the following text accurately and naturally.
Keep the original meaning, tone, and formatting.
Do not explain or add anything else.`
)

var (
	initOnce     sync.Once
	httpClient   *http.Client
	streamClient *http.Client
)

type Minimax struct {
	config.Translate
}

// MiniMax 原生 API 请求/响应结构体

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content,omitempty"`
	Name    string `json:"name,omitempty"`
}

type ChatCompletionRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream,omitempty"`
}

type Choice struct {
	Index        int     `json:"index"`
	FinishReason string  `json:"finish_reason"`
	Message      Message `json:"message"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatCompletionResponse struct {
	ID       string    `json:"id"`
	Object   string    `json:"object"`
	Created  int64     `json:"created"`
	Model    string    `json:"model"`
	Choices  []Choice  `json:"choices"`
	Usage    Usage     `json:"usage"`
	BaseResp *BaseResp `json:"base_resp,omitempty"`
}

type BaseResp struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

// SSE 流式响应结构体
type StreamChoice struct {
	Index int         `json:"index"`
	Delta StreamDelta `json:"delta"`
}

type StreamDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

type StreamResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
}

func (c *Minimax) GetName() string {
	return Way
}

func (c *Minimax) getBaseURL() string {
	if c.BaseURL != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	return DefaultBaseURL
}

func (c *Minimax) getModel() string {
	if c.Model != "" {
		return c.Model
	}
	return DefaultModel
}

// initClients 统一初始化 HTTP clients（只执行一次）
func initClients() {
	initOnce.Do(func() {
		// 非流式请求：有全局超时
		httpClient = &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     90 * time.Second,
			},
		}
		// 流式请求：不设全局超时，通过 context 控制
		streamClient = &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     90 * time.Second,
			},
		}
	})
}

// buildTranslateMessages 构建翻译请求的 messages
func buildTranslateMessages(systemPrompt, query, fromLang, toLang string) []Message {
	return []Message{
		{
			Role:    "system",
			Name:    "Translator",
			Content: fmt.Sprintf("%s\nTranslate from %s to %s.", systemPrompt, fromLang, toLang),
		},
		{
			Role:    "user",
			Content: query,
			Name:    "用户",
		},
	}
}

// PostQuery 非流式翻译
func (c *Minimax) PostQuery(query, fromLang, toLang string) ([]string, error) {
	return c.PostQueryContext(context.Background(), query, fromLang, toLang)
}

func (c *Minimax) PostQueryContext(ctx context.Context, query, fromLang, toLang string) ([]string, error) {
	initClients()

	reqBody := ChatCompletionRequest{
		Model:    c.getModel(),
		Messages: buildTranslateMessages(TranslatePrompts, query, fromLang, toLang),
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.getBaseURL() + "/v1/text/chatcompletion_v2"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Key)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned unexpected status code: %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// 检查业务错误
	if chatResp.BaseResp != nil && chatResp.BaseResp.StatusCode != 0 {
		return nil, fmt.Errorf("API error: code=%d msg=%s", chatResp.BaseResp.StatusCode, chatResp.BaseResp.StatusMsg)
	}

	// 提取回复内容
	var results []string
	for _, choice := range chatResp.Choices {
		if choice.Message.Content != "" {
			results = append(results, choice.Message.Content)
		}
	}

	if len(results) == 0 {
		return []string{""}, nil
	}

	return results, nil
}

// PostQueryStream 流式翻译
func (c *Minimax) PostQueryStream(query, fromLang, toLang string, callback func(chunk string)) error {
	return c.PostQueryStreamContext(context.Background(), query, fromLang, toLang, callback)
}

// PostQueryStreamContext 流式翻译，并在调用方取消 ctx 时终止 HTTP 请求。
func (c *Minimax) PostQueryStreamContext(ctx context.Context, query, fromLang, toLang string, callback func(chunk string)) error {
	reqBody := ChatCompletionRequest{
		Model:    c.getModel(),
		Messages: buildTranslateMessages(TranslatePrompts, query, fromLang, toLang),
		Stream:   true,
	}

	return c.doStreamRequest(ctx, reqBody, callback)
}

// PostExplainStream 流式术语解释
func (c *Minimax) PostExplainStream(query, templateID string, callback func(chunk string)) error {
	return c.PostExplainStreamContext(context.Background(), query, templateID, callback)
}

// PostExplainStreamContext 流式解释，并在调用方取消 ctx 时终止 HTTP 请求。
func (c *Minimax) PostExplainStreamContext(ctx context.Context, query, templateID string, callback func(chunk string)) error {
	var prompt string

	if templateID == "" {
		// 没有模板 ID 时，直接使用 query 作为完整提示词（如 QueryWord 场景）
		prompt = query
	} else {
		// 获取模板内容
		templateStr := c.getTemplate(templateID)
		if templateStr == "" {
			return fmt.Errorf("template not found: %s", templateID)
		}
		// 替换模板中的占位符
		prompt = strings.ReplaceAll(templateStr, "{{.text}}", query)
	}

	reqBody := ChatCompletionRequest{
		Model: c.getModel(),
		Messages: []Message{
			{
				Role:    "system",
				Name:    "Explainer",
				Content: "你是一个知识渊博的助手。",
			},
			{
				Role:    "user",
				Content: prompt,
				Name:    "用户",
			},
		},
		Stream: true,
	}

	return c.doStreamRequest(ctx, reqBody, callback)
}

// doStreamRequest 执行流式请求（SSE）
func (c *Minimax) doStreamRequest(ctx context.Context, reqBody ChatCompletionRequest, callback func(chunk string)) error {
	initClients()

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := c.getBaseURL() + "/v1/text/chatcompletion_v2"

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Key)

	resp, err := streamClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned unexpected status code: %d: %s", resp.StatusCode, string(body))
	}

	// 解析 SSE 流
	return parseSSEStream(resp.Body, callback)
}

// parseSSEStream 使用 bufio.Scanner 逐行解析 SSE 流
// 保证每行数据完整，避免因原始 Read() 的随机边界导致 JSON 截断和乱序
func parseSSEStream(reader io.Reader, callback func(chunk string)) error {
	scanner := bufio.NewScanner(reader)
	// 设置足够大的 buffer 以处理大 JSON 行（默认 64KB，最大 1MB）
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行（SSE 事件之间的分隔符）
		if line == "" {
			continue
		}

		// 只处理 data: 开头的行
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		// 提取 data: 后面的内容（兼容 "data: " 和 "data:" 两种格式）
		eventData := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

		// 检查是否是结束信号
		if eventData == "[DONE]" {
			return nil
		}

		// 空 data 跳过
		if eventData == "" {
			continue
		}

		var streamResp StreamResponse
		if err := json.Unmarshal([]byte(eventData), &streamResp); err != nil {
			slog.Warn("SSE 事件 JSON 解析失败",
				slog.String("data", truncateForLog(eventData, 200)),
				slog.String("error", err.Error()))
			continue
		}

		// 按 index 顺序提取 delta.content，保证顺序
		for _, choice := range streamResp.Choices {
			if choice.Delta.Content != "" {
				callback(choice.Delta.Content)
			}
		}
	}

	// 检查 Scanner 错误（网络中断、buffer 不够等）
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read SSE stream: %w", err)
	}

	return nil
}

// truncateForLog 截断长文本用于日志显示
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// getTemplate 获取提示词模板，委托到共用模板查找逻辑。
func (c *Minimax) getTemplate(templateID string) string {
	currentConfig := config.Snapshot()
	return config.FindTemplate(&currentConfig.ExplainTemplates, templateID)
}
