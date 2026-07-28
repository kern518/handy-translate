package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

func TestInitCreatesSafeDefaultConfig(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	projectDir := t.TempDir()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})

	Init(filepath.Base(projectDir))

	if _, err := os.Stat(filepath.Join(projectDir, "config.toml")); err != nil {
		t.Fatalf("default config was not created: %v", err)
	}
	if Data.TranslateWay == "" || Data.Translate[Data.TranslateWay].Name == "" {
		t.Fatal("default translation provider is incomplete")
	}
	if Data.Translate[Data.TranslateWay].Key != "" {
		t.Fatal("default config must not contain an API key")
	}
	if len(Data.Keyboards["screenshot"]) == 0 {
		t.Fatal("default screenshot hotkey is missing")
	}
}

func TestNormalizeRepairsMissingMaps(t *testing.T) {
	Data = Config{}
	normalize()

	if Data.Translate == nil || Data.Keyboards == nil {
		t.Fatal("normalize did not initialise maps")
	}
	if _, ok := Data.Translate[Data.TranslateWay]; !ok {
		t.Fatal("normalize did not add the selected provider")
	}
}

func TestConcurrentUpdatesPersistLatestSnapshot(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	projectDir := t.TempDir()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})
	Init(filepath.Base(projectDir))

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := Update(func(data *Config) {
				data.ToolbarPinned = i%2 == 0
				data.Appname = filepath.Base(projectDir) + string(rune('A'+i))
			}); err != nil {
				t.Errorf("Update() error = %v", err)
			}
		}()
	}
	wg.Wait()

	raw, err := os.ReadFile(filepath.Join(projectDir, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	var persisted Config
	if err := toml.Unmarshal(raw, &persisted); err != nil {
		t.Fatal(err)
	}

	current := Snapshot()
	if persisted.Appname != current.Appname ||
		persisted.ToolbarPinned != current.ToolbarPinned {
		t.Fatalf("persisted config does not match latest snapshot: persisted=%+v current=%+v",
			persisted, current)
	}
}
