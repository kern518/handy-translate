package translate_service

import (
	"fmt"
	"testing"

	"handy-translate/config"
	"handy-translate/translate_service/baidu"
	"handy-translate/translate_service/youdao"

	"github.com/OwO-Network/gdeeplx"
)

func TestGetTranslateWay(t *testing.T) {
	result, err := gdeeplx.Translate("hello", "EN", "ZH", 0)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println(result)
}

func TestGetTranslateWayList(t *testing.T) {
	config.Init("handy-translate")
	v := GetTranslateWay(baidu.Way)
	s, err := v.PostQuery("app", "auto", "zh")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(s)
}

func TestTranslateYouDao(t *testing.T) {
	config.Init("handy-translate")
	v := GetTranslateWay(youdao.Way)
	s, err := v.PostQuery("test", "auto", "zh")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(s)
}
