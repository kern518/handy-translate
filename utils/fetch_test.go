package utils

import (
	"fmt"
	"os"
	"testing"
)

func TestMyFetch(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("set RUN_INTEGRATION_TESTS=1 to perform a real HTTP request")
	}
	res := MyFetch(`https://fanyi.baidu.com/langdetect`,
		map[string]interface{}{
			"method": "POST",
			"headers": map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			},
			"body": "query=apple",
		})
	fmt.Println(res)
}
