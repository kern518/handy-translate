package utils

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"handy-translate/utils/httpclient"
)

func DoGet(url string, header map[string][]string, paramsMap map[string][]string, expectContentType string) []byte {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client := httpclient.GetDefaultClient()
	params := neturl.Values{}
	for k, v := range paramsMap {
		params[k] = v
	}
	parseUrl, _ := neturl.Parse(url)
	parseUrl.RawQuery = params.Encode()

	req, _ := http.NewRequestWithContext(ctx, "GET", parseUrl.String(), nil)
	for k, v := range header {
		for hv := range v {
			req.Header.Add(k, v[hv])
		}
	}
	res, err := client.Do(req)
	if err != nil {
		slog.Error("request failed:", slog.Any("err", err))
		return nil
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	contentType := res.Header.Get("Content-Type")
	if !strings.Contains(contentType, expectContentType) {
		slog.Error("contentType not match", slog.String("contentType", contentType), slog.String("expectContentType", expectContentType))
		return nil
	}
	return body
}

func DoPost(url string, header map[string][]string, bodyMap map[string][]string, expectContentType string) []byte {
	body, err := DoPostContext(context.Background(), url, header, bodyMap, expectContentType)
	if err != nil {
		slog.Error("request failed", slog.Any("err", err))
		return nil
	}
	return body
}

func DoPostContext(
	parent context.Context,
	url string,
	header map[string][]string,
	bodyMap map[string][]string,
	expectContentType string,
) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	defer cancel()

	client := httpclient.GetDefaultClient()
	params := neturl.Values{}
	for k, v := range bodyMap {
		for pv := range v {
			params.Add(k, v[pv])
		}
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	for k, v := range header {
		for hv := range v {
			req.Header.Add(k, v[hv])
		}
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return nil, fmt.Errorf("API returned status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	contentType := res.Header.Get("Content-Type")
	if !strings.Contains(contentType, expectContentType) {
		return nil, fmt.Errorf("content type %q does not contain %q", contentType, expectContentType)
	}
	return body, nil
}
