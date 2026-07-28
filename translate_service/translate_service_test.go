package translate_service

import (
	"fmt"
	"os"
	"testing"

	"handy-translate/config"
	"handy-translate/translate_service/baidu"
	"handy-translate/translate_service/youdao"

	"github.com/OwO-Network/gdeeplx"
)

func requireIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("set RUN_INTEGRATION_TESTS=1 to call real translation services")
	}
}

func TestGetTranslateWay(t *testing.T) {
	requireIntegration(t)
	result, err := gdeeplx.Translate("hello", "EN", "ZH", 0)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println(result)
}

func TestGetTranslateWayList(t *testing.T) {
	requireIntegration(t)
	config.Init("handy-translate")
	v := GetTranslateWay(baidu.Way)
	if v == nil {
		t.Fatal("Baidu provider is not configured")
	}
	s, err := v.PostQuery("app", "auto", "zh")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(s)
}

func TestTranslateYouDao(t *testing.T) {
	requireIntegration(t)
	config.Init("handy-translate")
	v := GetTranslateWay(youdao.Way)
	if v == nil {
		t.Fatal("Youdao provider is not configured")
	}
	s, err := v.PostQuery("test", "auto", "zh")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(s)
}
