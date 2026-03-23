package main

import (
	"fmt"
	"net/url"
	"time"
)

// ScrapeResult 爬取结果结构
type ScrapeResult struct {
	TargetURL      string        `json:"target_url"`
	FinalURL       string        `json:"final_url"`
	StatusCode     int           `json:"status_code,omitempty"`
	Title          string        `json:"title"`
	JSLinks        []string      `json:"js_links"`
	Fingerprint    []string      `json:"fingerprint"`
	FaviconIcon    string        `json:"favicon_icon"`
	Favicon        int32         `json:"favicon"`
	Screenshot     string        `json:"-"`
	ScreenshotPath string        `json:"-"`
	Error          string        `json:"error,omitempty"`
	Duration       time.Duration `json:"-"`
}

// Config 配置结构
type Config struct {
	URLs            []string
	Threads         int
	Screenshot      bool
	ScreenshotDelay time.Duration
	Proxy           string
	Timeout         time.Duration
	Output          string
	Headless        bool
	UserAgent       string
	ResumeFile      string
}

// ResumeState 断点续爬状态
type ResumeState struct {
	Completed map[string]bool `json:"completed"`
}

// validateURL 验证URL格式
func validateURL(rawURL string) (string, error) {
	if rawURL == "" {
		return "", fmt.Errorf("URL cannot be empty")
	}
	if !hasProtocol(rawURL) {
		rawURL = "http://" + rawURL
	}
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %v", err)
	}
	if parsedURL.Host == "" {
		return "", fmt.Errorf("URL must include a host")
	}
	return parsedURL.String(), nil
}

// hasProtocol 检查URL是否包含协议
func hasProtocol(rawURL string) bool {
	return len(rawURL) > 7 && (rawURL[:7] == "http://" || (len(rawURL) > 8 && rawURL[:8] == "https://"))
}
