package baidu

import (
	"fmt"
	"handy-translate/config"
	"os"
	"testing"
)

func TestBaidu_PostQuery(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("set RUN_INTEGRATION_TESTS=1 to call the real Baidu API")
	}
	config.Init("handy-translate")
	currentConfig := config.Snapshot()
	source := `hello`
	var baidu = &Baidu{
		Translate: config.Translate{
			Key:   currentConfig.Translate[Way].Key,
			AppID: currentConfig.Translate[Way].AppID,
		},
	}
	target, err := baidu.PostQuery(source, "auto", "zh")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(err)
	fmt.Println(target)
}
