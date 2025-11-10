package config

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/sirupsen/logrus"
)

// Data config
var Data config

type (
	config struct {
		Appname         string                    `toml:"appname"`
		Keyboards       map[string][]string       `toml:"keyboards"`
		TranslateWay    string                    `toml:"translate_way"`
		Translate       map[string]Translate      `toml:"translate"`
		ExplainTemplates ExplainTemplatesConfig   `toml:"explain_templates"`
	}

	Translate struct {
		Name  string `toml:"name" json:"name,omitempty"`
		AppID string `toml:"appID" json:"appID,omitempty"`
		Key   string `toml:"key" json:"key,omitempty"`
	}

	ExplainTemplatesConfig struct {
		DefaultTemplate string                    `toml:"default_template"`
		Templates       map[string]ExplainTemplate `toml:"templates"`
	}

	ExplainTemplate struct {
		Name        string `toml:"name" json:"name"`
		Description string `toml:"description" json:"description"`
		Template    string `toml:"template" json:"template"`
	}
)

// Init  config
func Init(projectName string) {
	filePath, _ := os.Getwd()
	b := strings.Index(filePath, projectName)
	configPath := filePath[:b+len(projectName)]

	configFile, err := os.Open(configPath + "/config.toml")
	if err != nil {
		logrus.WithError(err).Error("Open")
		os.Exit(1)
	}
	defer configFile.Close()

	fd, err := io.ReadAll(configFile)
	if err != nil {
		logrus.WithError(err).Error("ReadAll")
		os.Exit(1)
	}
	err = toml.Unmarshal(fd, &Data)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(Data)
}

func Save() {
	filePath := "./config.toml"
	data, err := toml.Marshal(&Data)
	if err != nil {
		logrus.WithError(err).Error("Marshal")
	}

	// 使用 os.Remove() 函数删除文件
	err = os.Remove(filePath)
	if err != nil {
		fmt.Println("删除文件时出错:", err)
		return
	}

	fmt.Println("文件已成功删除")

	// 打开文件，如果文件不存在则创建
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("无法创建文件:", err)
		return
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		fmt.Println("无法写入文件:", err)
		return
	}

	fmt.Println("数据写入成功!")
}
