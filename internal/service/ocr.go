// Package service 提供 OCR 服务。
package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const defaultOCRTimeout = 30 * time.Second

// OCRService OCR 文字识别服务。
type OCRService struct {
	executablePath string
	timeout        time.Duration
}

// NewOCRService 创建 OCR 服务实例。
func NewOCRService(executablePath string) *OCRService {
	return &OCRService{
		executablePath: executablePath,
		timeout:        defaultOCRTimeout,
	}
}

// OCRResult OCR 识别结果结构。
type OCRResult struct {
	Code int `json:"code"`
	Data []struct {
		Box   [4][2]int `json:"box"`
		Score float64   `json:"score"`
		Text  string    `json:"text"`
	} `json:"data"`
}

// Recognize 对图片执行 OCR 识别，返回识别到的文本。
func (o *OCRService) Recognize(imagePath string) string {
	timeout := o.timeout
	if timeout <= 0 {
		timeout = defaultOCRTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, o.executablePath, "--image="+imagePath)

	var outputBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &stderrBuffer

	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000,
		}
	}

	if err := cmd.Start(); err != nil {
		slog.Error("OCR 启动失败", slog.Any("error", err))
		return ""
	}

	if err := cmd.Wait(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			slog.Error("OCR 执行超时", slog.Duration("timeout", timeout))
			return ""
		}
		stderrStr := stderrBuffer.String()
		if stderrStr != "" {
			slog.Error("OCR 执行失败", slog.Any("error", err), slog.String("stderr", stderrStr))
		} else {
			slog.Error("OCR 执行失败", slog.Any("error", err))
		}
		return ""
	}

	output := outputBuffer.Bytes()
	startIndex := strings.Index(string(output), "{")
	if startIndex == -1 {
		slog.Error("无法找到 JSON 数据起始位置")
		return ""
	}

	jsonStr := output[startIndex:]
	var result OCRResult
	if err := json.Unmarshal(jsonStr, &result); err != nil {
		slog.Error("解析 OCR 结果失败", slog.Any("error", err))
		return ""
	}

	var texts []string
	for _, item := range result.Data {
		texts = append(texts, item.Text)
	}

	return strings.Join(texts, "\n")
}

// SaveBase64Image 将 Base64 编码的图片保存到文件。
func SaveBase64Image(base64String, filename string) error {
	data, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		return fmt.Errorf("decode base64: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	if _, err = file.Write(data); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}
