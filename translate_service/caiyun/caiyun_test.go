package caiyun

import (
	"fmt"
	"handy-translate/config"
	"os"
	"testing"
)

func TestTranslate(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("set RUN_INTEGRATION_TESTS=1 to call the real Caiyun API")
	}
	config.Init("handy-translate")
	currentConfig := config.Snapshot()
	// source := []string{"Lingocloud is the best translation service.", "彩云小译は最高の翻訳サービスです"}
	// target := Translate(source, "auto2zh")

	// fmt.Println(target)
	source := `hello`
	var caiyun = &Caiyun{
		Translate: config.Translate{
			Key: currentConfig.Translate[Way].Key,
		},
	}
	target, _ := caiyun.PostQuery(source, "", "")

	fmt.Println(target)
}
