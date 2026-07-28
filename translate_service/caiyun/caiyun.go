package caiyun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"handy-translate/config"
	"handy-translate/utils/httpclient"
)

// https://docs.caiyunapp.com/blog/2021/12/30/hello-world
const Way = "caiyun"

type Caiyun struct {
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

func (c *Caiyun) GetName() string {
	return Way
}

func (c *Caiyun) PostQuery(query, fromLang, toLang string) ([]string, error) {
	return c.PostQueryContext(context.Background(), query, fromLang, toLang)
}

func (c *Caiyun) PostQueryContext(ctx context.Context, query, fromLang, toLang string) ([]string, error) {
	url := "https://api.interpreter.caiyunai.com/v1/translator"
	if c.BaseURL != "" {
		url = strings.TrimRight(c.BaseURL, "/")
	}

	// WARNING, this token is a test token for new developers,
	// and it should be replaced by your token
	token := c.Key

	transType := fmt.Sprintf("%s2%s", fromLang, toLang)
	payload := TranslationPayload{
		Source: strings.Split(query, ","),
		// TransType: "auto2zh",
		TransType: transType,
		RequestID: "demo",
		Detect:    true,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-authorization", "token "+token)

	client := httpclient.GetDefaultClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("caiyun API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	slog.Debug("彩云翻译响应", slog.Int("body_length", len(respBody)))
	var translationResponse TranslationResponse
	err = json.Unmarshal(respBody, &translationResponse)

	if err != nil {
		return nil, err
	}

	return translationResponse.Target, nil
}
