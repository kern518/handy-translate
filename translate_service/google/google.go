// Package google 提供 Google Gemini 大模型翻译服务。
// 使用 API Key 认证，直接调用 Gemini OpenAI 兼容 REST API。
package google

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
	Way            = "google"
	DefaultBaseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
	DefaultModel   = "gemini-2.5-flash"

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

// Google Gemini 大模型翻译服务。
type Google struct {
	config.Translate
}

// ──────────────────────────────────────────────
// OpenAI 兼容 API 结构体
// ──────────────────────────────────────────────

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream,omitempty"`
}

type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

type StreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content,omitempty"`
		} `json:"delta"`
	} `json:"choices"`
}

// ──────────────────────────────────────────────
// 公开方法
// ──────────────────────────────────────────────

func (g *Google) GetName() string {
	return Way
}

// PostQuery 非流式翻译。
func (g *Google) PostQuery(query, fromLang, toLang string) ([]string, error) {
	return g.PostQueryContext(context.Background(), query, fromLang, toLang)
}

func (g *Google) PostQueryContext(ctx context.Context, query, fromLang, toLang string) ([]string, error) {
	initClients()

	reqBody := ChatRequest{
		Model:    g.getModel(),
		Messages: buildTranslateMessages(TranslatePrompts, query, fromLang, toLang),
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := g.getBaseURL() + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.Key)

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
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("API error (code=%d): %s", chatResp.Error.Code, chatResp.Error.Message)
	}

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

// PostQueryStream 流式翻译。
func (g *Google) PostQueryStream(query, fromLang, toLang string, callback func(chunk string)) error {
	return g.PostQueryStreamContext(context.Background(), query, fromLang, toLang, callback)
}

// PostQueryStreamContext 流式翻译，并在调用方取消 ctx 时终止 HTTP 请求。
func (g *Google) PostQueryStreamContext(ctx context.Context, query, fromLang, toLang string, callback func(chunk string)) error {
	reqBody := ChatRequest{
		Model:    g.getModel(),
		Messages: buildTranslateMessages(TranslatePrompts, query, fromLang, toLang),
		Stream:   true,
	}
	return g.doStreamRequest(ctx, reqBody, callback)
}

// PostExplainStream 流式术语解释（支持模板选择）。
func (g *Google) PostExplainStream(query, templateID string, callback func(chunk string)) error {
	return g.PostExplainStreamContext(context.Background(), query, templateID, callback)
}

// PostExplainStreamContext 流式解释，并在调用方取消 ctx 时终止 HTTP 请求。
func (g *Google) PostExplainStreamContext(ctx context.Context, query, templateID string, callback func(chunk string)) error {
	var prompt string

	if templateID == "" {
		// 没有模板 ID 时，直接使用 query 作为完整提示词（如 QueryWord 场景）
		prompt = query
	} else {
		// 获取模板内容
		templateStr := g.getTemplate(templateID)
		if templateStr == "" {
			return fmt.Errorf("template not found: %s", templateID)
		}
		// 替换模板中的占位符
		prompt = strings.ReplaceAll(templateStr, "{{.text}}", query)
	}

	reqBody := ChatRequest{
		Model: g.getModel(),
		Messages: []Message{
			{
				Role:    "system",
				Content: "你是一个知识渊博的助手。",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: true,
	}

	return g.doStreamRequest(ctx, reqBody, callback)
}

// ──────────────────────────────────────────────
// 内部方法
// ──────────────────────────────────────────────

func (g *Google) getBaseURL() string {
	if g.BaseURL != "" {
		return strings.TrimRight(g.BaseURL, "/")
	}
	return DefaultBaseURL
}

func (g *Google) getModel() string {
	if g.Model != "" {
		return g.Model
	}
	return DefaultModel
}

// getTemplate 获取提示词模板，委托到共用模板查找逻辑。
func (g *Google) getTemplate(templateID string) string {
	currentConfig := config.Snapshot()
	return config.FindTemplate(&currentConfig.ExplainTemplates, templateID)
}

func buildTranslateMessages(systemPrompt, query, fromLang, toLang string) []Message {
	return []Message{
		{
			Role:    "system",
			Content: fmt.Sprintf("%s\nTranslate from %s to %s.", systemPrompt, fromLang, toLang),
		},
		{
			Role:    "user",
			Content: query,
		},
	}
}

func initClients() {
	initOnce.Do(func() {
		httpClient = &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     90 * time.Second,
			},
		}
		streamClient = &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     90 * time.Second,
			},
		}
	})
}

// doStreamRequest 执行流式请求（SSE）。
func (g *Google) doStreamRequest(ctx context.Context, reqBody ChatRequest, callback func(chunk string)) error {
	initClients()

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := g.getBaseURL() + "/chat/completions"

	slog.Debug("Google Gemini 请求",
		slog.String("url", url),
		slog.String("model", reqBody.Model))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.Key)

	resp, err := streamClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析 SSE 流
	return parseSSEStream(resp.Body, callback)
}

// parseSSEStream 逐行解析 SSE 流。
func parseSSEStream(reader io.Reader, callback func(chunk string)) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data:") {
			continue
		}

		eventData := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

		if eventData == "[DONE]" {
			return nil
		}

		if eventData == "" {
			continue
		}

		var streamResp StreamResponse
		if err := json.Unmarshal([]byte(eventData), &streamResp); err != nil {
			slog.Warn("Google Gemini SSE JSON 解析失败",
				slog.String("data", truncateForLog(eventData, 200)),
				slog.String("error", err.Error()))
			continue
		}

		for _, choice := range streamResp.Choices {
			if choice.Delta.Content != "" {
				callback(choice.Delta.Content)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read SSE stream: %w", err)
	}

	return nil
}

func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
