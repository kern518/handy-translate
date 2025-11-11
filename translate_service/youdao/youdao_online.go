package youdao

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"handy-translate/config"
)

const YouDaoOnlineWay = "youdao_online"

type YouDaoOnline struct {
	config.Translate
}

func (y *YouDaoOnline) PostQuery(query string) []string {
	url := "https://dict.youdao.com/suggest?num=2&ver=3.0&doctype=json&cache=false&le=en&q=" + url.QueryEscape(query) // 替换为你要请求的 URL

	// 发起 GET 请求
	response, err := http.Get(url)
	if err != nil {
		slog.Error("Get", slog.Any("err", err))
		return nil
	}
	defer response.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(response.Body)
	if err != nil {
		slog.Error("ReadAll", slog.Any("err", err))
		return nil
	}

	var tr YoudaoOnlineTranslate

	err = json.Unmarshal(body, &tr)
	if err != nil {
		slog.Error("Unmarshal", slog.Any("err", err))
		return nil
	}
	if len(tr.Data.Entries) > 0 {
		return []string{tr.Data.Entries[0].Explain}
	}
	return nil
}

type YoudaoOnlineTranslate struct {
	Result Result `json:"result"`
	Data   Data   `json:"data"`
}

type Result struct {
	Msg  string `json:"msg"`
	Code int    `json:"code"`
}

type Entries struct {
	Explain string `json:"explain"`
	Entry   string `json:"entry"`
}

type Data struct {
	Entries  []Entries `json:"entries"`
	Query    string    `json:"query"`
	Language string    `json:"language"`
	Type     string    `json:"type"`
}
