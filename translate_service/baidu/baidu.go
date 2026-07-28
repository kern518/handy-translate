package baidu

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"handy-translate/config"
	"handy-translate/utils/httpclient"
)

const Way = "baidu"

type Baidu struct {
	config.Translate
}

const (
	endpoint = "https://api.fanyi.baidu.com"
	path     = "/api/trans/vip/translate"
)

func (b *Baidu) GetName() string {
	return Way
}

type translateResult struct {
	From        string        `json:"from"`
	To          string        `json:"to"`
	TransResult []TransResult `json:"trans_result"`
	ErrorCode   string        `json:"error_code"`
	ErrorMsg    string        `json:"error_msg"`
}
type TransResult struct {
	Dst string `json:"dst"`
	Src string `json:"src"`
}

func (b *Baidu) PostQuery(query, fromLang, toLang string) ([]string, error) {
	return b.PostQueryContext(context.Background(), query, fromLang, toLang)
}

func (b *Baidu) PostQueryContext(ctx context.Context, query, fromLang, toLang string) ([]string, error) {
	slog.Debug("PostQuery", slog.String("query", query), slog.String("fromLang", fromLang), slog.String("toLang", toLang))
	uri := endpoint + path
	appKey := b.Key
	appID := b.AppID

	// Generate salt and sign
	salt := strconv.Itoa(rand.Intn(32768) + 32768)
	sign := makeMD5(appID + query + salt + appKey)

	// Build request
	form := url.Values{}
	form.Add("appid", appID)
	form.Add("q", query)
	form.Add("from", fromLang)
	form.Add("to", toLang)
	// form.Add("from", "en")
	// form.Add("to", "zh")
	form.Add("salt", salt)
	form.Add("sign", sign)

	// Send request
	client := httpclient.GetDefaultClient()
	req, err := http.NewRequestWithContext(ctx, "POST", uri, strings.NewReader(form.Encode()))
	if err != nil {
		slog.Error("Error creating request:", slog.Any("err", err))
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Error sending request:", slog.Any("err", err))
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("baidu API returned status %d", resp.StatusCode)
	}

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Error reading response:", slog.Any("err", err))
		return nil, err
	}

	var result translateResult
	if err := json.Unmarshal(body, &result); err != nil {
		slog.Error("Error:", slog.Any("err", err))
		return nil, err
	}
	if result.ErrorCode != "" {
		return nil, fmt.Errorf("baidu API error %s: %s", result.ErrorCode, result.ErrorMsg)
	}

	prettyResult, _ := json.MarshalIndent(result, "", "    ")
	slog.Debug("PostQuery 响应", slog.String("result", string(prettyResult)))

	if len(result.TransResult) > 0 {
		if result.TransResult[0].Dst == result.TransResult[0].Src {
			return nil, nil
		}
		var res []string
		for _, v := range result.TransResult {
			res = append(res, v.Dst)
		}
		return res, nil
	}
	return nil, fmt.Errorf("baidu API returned no translation")
}

func makeMD5(s string) string {
	hash := md5.New()
	hash.Write([]byte(s))
	return fmt.Sprintf("%x", hash.Sum(nil))
}
