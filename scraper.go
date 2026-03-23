package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/projectdiscovery/wappalyzergo"
	"github.com/spaolacci/murmur3"
)

// Scraper Chrome爬虫结构
type Scraper struct {
	config     *Config
	browser    *rod.Browser
	launcher   *launcher.Launcher
	wappalyzer *wappalyzer.Wappalyze
	mu         sync.Mutex
}

// NewScraper 创建新的爬虫实例
func NewScraper(cfg *Config) (*Scraper, error) {
	wapp, err := wappalyzer.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize wappalyzer: %v", err)
	}

	l := launcher.New().
		Headless(cfg.Headless).
		Set("disable-blink-features", "AutomationControlled").
		Set("disable-web-security").
		Set("disable-features", "VizDisplayCompositor").
		UserDataDir("")

	if cfg.Proxy != "" {
		l = l.Proxy(cfg.Proxy)
	}

	controlURL, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %v", err)
	}

	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect browser: %v", err)
	}

	return &Scraper{
		config:     cfg,
		browser:    browser,
		launcher:   l,
		wappalyzer: wapp,
	}, nil
}

// Close 关闭爬虫
func (s *Scraper) Close() {
	if s.browser != nil {
		s.browser.Close()
	}
	if s.launcher != nil {
		s.launcher.Cleanup()
	}
}

// ScrapeURL 爬取单个URL
func (s *Scraper) ScrapeURL(targetURL string) *ScrapeResult {
	start := time.Now()
	result := &ScrapeResult{
		TargetURL:   targetURL,
		FinalURL:    targetURL,
		JSLinks:     []string{},
		Fingerprint: []string{},
	}

	validURL, err := validateURL(targetURL)
	if err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}
	result.TargetURL = validURL
	result.FinalURL = validURL

	page, err := s.browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		result.Error = fmt.Sprintf("failed to create page: %v", err)
		result.Duration = time.Since(start)
		return result
	}
	defer page.Close()

	if s.config.UserAgent != "" {
		_ = page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
			UserAgent: s.config.UserAgent,
		})
	}

	// 拦截响应头
	headers := make(map[string][]string)
	var headerMu sync.Mutex
	go page.EachEvent(func(e *proto.NetworkResponseReceived) {
		if e.Response != nil {
			headerMu.Lock()
			if e.Type == proto.NetworkResourceTypeDocument {
				result.StatusCode = e.Response.Status
			}
			for k, v := range e.Response.Headers {
				headers[strings.ToLower(k)] = []string{fmt.Sprintf("%v", v)}
			}
			headerMu.Unlock()
		}
	})()

	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
	defer cancel()

	if err = page.Context(ctx).Navigate(validURL); err != nil {
		result.Error = fmt.Sprintf("navigation failed: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	if err = page.Context(ctx).WaitLoad(); err != nil {
		result.Error = fmt.Sprintf("page load timed out: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	if s.config.ScreenshotDelay > 0 {
		time.Sleep(s.config.ScreenshotDelay)
	}

	// 获取最终URL（处理重定向）
	if finalURL := s.getFinalURL(page); finalURL != "" {
		result.FinalURL = finalURL
	}

	result.Title = s.getTitle(page)
	result.JSLinks = s.getJSLinks(page, result.FinalURL)

	// 获取HTML用于指纹识别
	html, _ := page.HTML()
	headerMu.Lock()
	result.Fingerprint = s.detectFingerprint(html, headers)
	headerMu.Unlock()

	faviconIcon := s.getFaviconURL(page, result.FinalURL)
	result.FaviconIcon = faviconIcon
	result.Favicon = s.getFaviconHash(faviconIcon, s.config.Proxy)

	if s.config.Screenshot {
		q := 80
		if data, err := page.Screenshot(true, &proto.PageCaptureScreenshot{
			Format:  proto.PageCaptureScreenshotFormatPng,
			Quality: &q,
		}); err == nil {
			result.Screenshot = base64.StdEncoding.EncodeToString(data)
		}
	}

	result.Duration = time.Since(start)
	return result
}

// getFinalURL 获取页面最终URL
func (s *Scraper) getFinalURL(page *rod.Page) string {
	res, err := page.Eval(`() => window.location.href`)
	if err != nil {
		return ""
	}
	return res.Value.Str()
}

// getTitle 获取页面标题
func (s *Scraper) getTitle(page *rod.Page) string {
	res, err := page.Eval(`() => document.title`)
	if err != nil {
		return ""
	}
	return res.Value.Str()
}

// getJSLinks 获取JS链接
func (s *Scraper) getJSLinks(page *rod.Page, baseURL string) []string {
	var jsLinks []string
	scripts, err := page.Eval(`() => Array.from(document.querySelectorAll('script[src]')).map(s => s.src)`)
	if err != nil {
		return jsLinks
	}
	for _, script := range scripts.Value.Arr() {
		if src := script.Str(); src != "" {
			if link := resolveURL(src, baseURL); link != "" {
				jsLinks = append(jsLinks, link)
			}
		}
	}
	return jsLinks
}

// detectFingerprint 使用wappalyzergo进行指纹识别
func (s *Scraper) detectFingerprint(html string, headers map[string][]string) []string {
	seen := make(map[string]bool)
	var result []string

	fingerprints := s.wappalyzer.Fingerprint(headers, []byte(html))
	for tech := range fingerprints {
		if !seen[tech] {
			seen[tech] = true
			result = append(result, tech)
		}
	}
	return result
}

// getFaviconURL 获取favicon链接
func (s *Scraper) getFaviconURL(page *rod.Page, baseURL string) string {
	selectors := []string{
		`link[rel="icon"]`,
		`link[rel="shortcut icon"]`,
		`link[rel="apple-touch-icon"]`,
		`link[rel="apple-touch-icon-precomposed"]`,
	}
	for _, sel := range selectors {
		res, err := page.Eval(fmt.Sprintf(`() => { const l = document.querySelector('%s'); return l ? l.href : ''; }`, sel))
		if err == nil {
			if v := res.Value.Str(); v != "" {
				return resolveURL(v, baseURL)
			}
		}
	}
	parsedURL, err := url.Parse(baseURL)
	if err == nil {
		return fmt.Sprintf("%s://%s/favicon.ico", parsedURL.Scheme, parsedURL.Host)
	}
	return ""
}

// getFaviconHash 下载favicon并计算Shodan风格murmur3哈希
func (s *Scraper) getFaviconHash(faviconURL string, proxy string) int32 {
	if faviconURL == "" {
		return 0
	}
	client := &http.Client{Timeout: 10 * time.Second}
	if proxy != "" {
		if proxyURL, err := url.Parse(proxy); err == nil {
			client.Transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
		}
	}
	resp, err := client.Get(faviconURL)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil || len(data) == 0 {
		return 0
	}

	// Shodan favicon hash: murmur3(base64 with 76-char line wrapping)
	encoded := base64.StdEncoding.EncodeToString(data)
	var lined strings.Builder
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		lined.WriteString(encoded[i:end])
		lined.WriteByte('\n')
	}
	h := murmur3.New32()
	h.Write([]byte(lined.String()))
	return int32(h.Sum32())
}

// resolveURL 将相对URL解析为绝对URL
func resolveURL(href, baseURL string) string {
	if href == "" {
		return ""
	}
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return href
	}
	rel, err := url.Parse(href)
	if err != nil {
		return href
	}
	return base.ResolveReference(rel).String()
}
