package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var (
	// 命令行参数
	urls            []string
	urlFile         string
	threads         int
	screenshot      bool
	screenshotDelay time.Duration
	proxy           string
	timeout         int
	output          string
	headless        bool
	userAgent       string
	verbose         bool
	resumeFile      string
)

// 颜色输出
var (
	colorRed    = color.New(color.FgRed).SprintFunc()
	colorGreen  = color.New(color.FgGreen).SprintFunc()
	colorYellow = color.New(color.FgYellow).SprintFunc()
	colorBlue   = color.New(color.FgBlue).SprintFunc()
	colorCyan   = color.New(color.FgCyan).SprintFunc()
)

var rootCmd = &cobra.Command{
	Use:   "chrome-scraper",
	Short: "A Chrome-based website information scraper",
	Run:   runScraper,
}

func init() {
	rootCmd.Flags().StringSliceVarP(&urls, "url", "u", []string{}, "Target URL list, separated by commas")
	rootCmd.Flags().StringVarP(&urlFile, "file", "f", "", "Path to a file containing URLs")
	rootCmd.Flags().IntVarP(&threads, "threads", "t", 3, "Number of concurrent workers")
	rootCmd.Flags().BoolVarP(&screenshot, "screenshot", "s", false, "Capture a screenshot")
	rootCmd.Flags().DurationVar(&screenshotDelay, "delay", 0, "Delay after page load before collecting page data and taking a screenshot (for example: 500ms, 2s)")
	rootCmd.Flags().StringVarP(&proxy, "proxy", "p", "", "Proxy server address (for example: http://127.0.0.1:8080)")
	rootCmd.Flags().IntVar(&timeout, "timeout", 30, "Request timeout in seconds")
	rootCmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (supports .json and .txt)")
	rootCmd.Flags().BoolVar(&headless, "headless", true, "Run in headless mode")
	rootCmd.Flags().StringVar(&userAgent, "user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36", "Custom User-Agent string")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().StringVar(&resumeFile, "resume", "", "Resume state file path for interrupted runs")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", colorRed("Error:"), err)
		os.Exit(1)
	}
}

func runScraper(cmd *cobra.Command, args []string) {
	// 打印banner
	printBanner()

	// 收集所有URL
	allURLs, err := collectURLs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", colorRed("Error:"), err)
		os.Exit(1)
	}

	if len(allURLs) == 0 {
		fmt.Fprintf(os.Stderr, "%s please provide at least one URL\n", colorRed("Error:"))
		fmt.Println("Use -h to view help")
		os.Exit(1)
	}

	if screenshotDelay < 0 {
		fmt.Fprintf(os.Stderr, "%s screenshot delay must be greater than or equal to 0\n", colorRed("Error:"))
		os.Exit(1)
	}

	// 创建配置
	config := &Config{
		URLs:            allURLs,
		Threads:         threads,
		Screenshot:      screenshot,
		ScreenshotDelay: screenshotDelay,
		Proxy:           proxy,
		Timeout:         time.Duration(timeout) * time.Second,
		Output:          output,
		Headless:        headless,
		UserAgent:       userAgent,
		ResumeFile:      resumeFile,
	}

	// 显示配置信息
	// printConfig(config)  // 已禁用

	// 开始爬取
	results := performScraping(config)

	// 输出结果
	outputResults(results, config)

	// 显示统计信息
	printStatistics(results)
}

func printBanner() {
	banner := `
 ██████╗██╗  ██╗██████╗  ██████╗ ███╗   ███╗███████╗    ███████╗ ██████╗██████╗  █████╗ ██████╗ ███████╗██████╗
██╔════╝██║  ██║██╔══██╗██╔═══██╗████╗ ████║██╔════╝    ██╔════╝██╔════╝██╔══██╗██╔══██╗██╔══██╗██╔════╝██╔══██╗
██║     ███████║██████╔╝██║   ██║██╔████╔██║█████╗      ███████╗██║     ██████╔╝███████║██████╔╝█████╗  ██████╔╝
██║     ██╔══██║██╔══██╗██║   ██║██║╚██╔╝██║██╔══╝      ╚════██║██║     ██╔══██╗██╔══██║██╔═══╝ ██╔══╝  ██╔══██╗
╚██████╗██║  ██║██║  ██║╚██████╔╝██║ ╚═╝ ██║███████╗    ███████║╚██████╗██║  ██║██║  ██║██║     ███████╗██║  ██║
 ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝ ╚═╝     ╚═╝╚══════╝    ╚══════╝ ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝     ╚══════╝╚═╝  ╚═╝
`
	fmt.Println(colorCyan(banner))
	fmt.Println(colorYellow("                                    Chrome Website Scraper v1.0"))
	fmt.Println()
}

func collectURLs() ([]string, error) {
	var allURLs []string

	// 从命令行参数获取URL
	allURLs = append(allURLs, urls...)

	// 从文件读取URL
	if urlFile != "" {
		fileURLs, err := readURLsFromFile(urlFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read URL file: %v", err)
		}
		allURLs = append(allURLs, fileURLs...)
	}

	// 去重和验证
	uniqueURLs := make(map[string]bool)
	var validURLs []string

	for _, url := range allURLs {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}

		if !uniqueURLs[url] {
			uniqueURLs[url] = true
			validURLs = append(validURLs, url)
		}
	}

	return validURLs, nil
}

func readURLsFromFile(filename string) ([]string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var urls []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			urls = append(urls, line)
		}
	}

	return urls, nil
}

func printConfig(config *Config) {
	fmt.Printf("%s Scrape Configuration\n", colorBlue("[Config]"))
	fmt.Printf("  Targets: %s\n", colorGreen(fmt.Sprintf("%d", len(config.URLs))))
	fmt.Printf("  Threads: %s\n", colorGreen(fmt.Sprintf("%d", config.Threads)))
	fmt.Printf("  Screenshots: %s\n", colorGreen(fmt.Sprintf("%t", config.Screenshot)))
	fmt.Printf("  Timeout: %s\n", colorGreen(fmt.Sprintf("%v", config.Timeout)))
	if config.Proxy != "" {
		fmt.Printf("  Proxy: %s\n", colorGreen(config.Proxy))
	}
	if config.Output != "" {
		fmt.Printf("  Output: %s\n", colorGreen(config.Output))
	}
	fmt.Println()
}

func performScraping(config *Config) []*ScrapeResult {
	// 加载断点续爬状态
	state := loadResumeState(config.ResumeFile)

	// 过滤已完成的URL
	var pendingURLs []string
	for _, u := range config.URLs {
		if !state.Completed[u] {
			pendingURLs = append(pendingURLs, u)
		}
	}

	skipped := len(config.URLs) - len(pendingURLs)
	if skipped > 0 {
		fmt.Printf("%s skipped completed: %d, remaining: %d\n", colorYellow("[Resume]"), skipped, len(pendingURLs))
	}

	proxyStatus := "off"
	if config.Proxy != "" {
		proxyStatus = "on"
	}
	fmt.Printf("%s starting scrape...    screenshots: %s        delay: %s        threads: %s        proxy: %s\n",
		colorBlue("[Start]"),
		colorYellow(fmt.Sprintf("%t", config.Screenshot)),
		colorYellow(config.ScreenshotDelay.String()),
		colorYellow(fmt.Sprintf("%d", config.Threads)),
		colorYellow(proxyStatus),
	)

	if len(pendingURLs) == 0 {
		fmt.Printf("%s all URLs have already been scraped\n", colorGreen("[Done]"))
		return []*ScrapeResult{}
	}

	// 创建进度条
	bar := progressbar.NewOptions(len(pendingURLs),
		progressbar.OptionSetDescription("Scrape progress"),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "█",
			SaucerPadding: "░",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWidth(50),
	)

	// 创建爬虫实例
	scraper, err := NewScraper(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to create scraper: %v\n", colorRed("Error:"), err)
		os.Exit(1)
	}
	defer scraper.Close()

	// 创建工作队列和结果收集
	urlChan := make(chan string, len(pendingURLs))
	resultChan := make(chan *ScrapeResult, len(pendingURLs))
	var wg sync.WaitGroup
	var stateMu sync.Mutex
	var outputMu sync.Mutex

	// 启动工作协程
	for i := 0; i < config.Threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for u := range urlChan {
				result := scraper.ScrapeURL(u)

				if config.Screenshot {
					if err := saveScreenshot(result); err != nil {
						if result.Error != "" {
							result.Error = fmt.Sprintf("%s; screenshot save failed: %v", result.Error, err)
						} else {
							result.Error = fmt.Sprintf("failed to save screenshot: %v", err)
						}
					}
				}

				// 记录完成状态
				if config.ResumeFile != "" && result.Error == "" {
					stateMu.Lock()
					state.Completed[u] = true
					saveResumeState(config.ResumeFile, state)
					stateMu.Unlock()
				}

				resultChan <- result
				printScanLog(bar, result, &outputMu, verbose)
			}
		}()
	}

	// 发送URL到队列
	for _, u := range pendingURLs {
		urlChan <- u
	}
	close(urlChan)

	// 等待所有工作完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	var results []*ScrapeResult
	for result := range resultChan {
		results = append(results, result)
	}

	bar.Finish()
	fmt.Println()

	// 全部完成后清理状态文件
	if config.ResumeFile != "" && len(state.Completed) == len(config.URLs) {
		os.Remove(config.ResumeFile)
		fmt.Printf("%s all tasks completed, resume state file removed\n", colorGreen("[Done]"))
	}

	return results
}

func printScanLog(bar *progressbar.ProgressBar, result *ScrapeResult, mu *sync.Mutex, verbose bool) {
	mu.Lock()
	defer mu.Unlock()

	target := result.TargetURL
	if target == "" {
		target = "-"
	}

	title := strings.TrimSpace(strings.ReplaceAll(result.Title, "\n", " "))
	if title == "" {
		title = "-"
	}

	statusCode := "-"
	if result.StatusCode > 0 {
		statusCode = fmt.Sprintf("%d", result.StatusCode)
	}

	if result.Error != "" {
		title = result.Error
	}

	if bar != nil {
		_ = bar.Clear()
	}

	line := fmt.Sprintf("%s | %s | %s", target, statusCode, title)
	switch {
	case result.Error != "":
		fmt.Printf("[+] %s\n", colorRed(line))
	case result.StatusCode >= 400:
		fmt.Printf("[+] %s\n", colorYellow(line))
	default:
		fmt.Printf("[+] %s\n", line)
	}

	if verbose && result.Error == "" {
		fmt.Printf("       final_url=%s js_links=%d fingerprints=%d\n", result.FinalURL, len(result.JSLinks), len(result.Fingerprint))
	}

	if bar != nil {
		_ = bar.Add(1)
	}
}

// loadResumeState 加载断点续爬状态
func loadResumeState(path string) *ResumeState {
	state := &ResumeState{Completed: make(map[string]bool)}
	if path == "" {
		return state
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return state
	}
	if err := json.Unmarshal(data, state); err != nil {
		return state
	}
	return state
}

// saveResumeState 保存断点续爬状态
func saveResumeState(path string, state *ResumeState) {
	data, err := json.Marshal(state)
	if err != nil {
		return
	}
	ioutil.WriteFile(path, data, 0644)
}

func outputResults(results []*ScrapeResult, config *Config) {
	if config.Output == "" {
		return
	}

	// 文件输出
	ext := strings.ToLower(filepath.Ext(config.Output))
	switch ext {
	case ".json":
		outputJSON(results, config.Output)
	case ".txt":
		outputText(results, config.Output)
	default:
		fmt.Printf("%s unsupported output format, using JSON instead\n", colorYellow("Warning:"))
		outputJSON(results, config.Output+".json")
	}
}

func printConsoleResults(results []*ScrapeResult) {
	fmt.Printf("%s Scrape Results\n", colorBlue("[Results]"))
	fmt.Println(strings.Repeat("=", 80))

	for i, result := range results {
		fmt.Printf("\n%s %d. %s\n", colorCyan("[Site]"), i+1, result.TargetURL)

		if result.Error != "" {
			fmt.Printf("   %s %s\n", colorRed("Error:"), result.Error)
			continue
		}

		fmt.Printf("   %s %s\n", colorGreen("Title:"), result.Title)
		if result.StatusCode > 0 {
			fmt.Printf("   %s %d\n", colorGreen("Status Code:"), result.StatusCode)
		}
		fmt.Printf("   %s %s\n", colorGreen("Duration:"), result.Duration)

		if len(result.JSLinks) > 0 {
			fmt.Printf("   %s\n", colorGreen("JS Links:"))
			for _, js := range result.JSLinks {
				fmt.Printf("     • %s\n", js)
			}
		}

		if result.FinalURL != "" && result.FinalURL != result.TargetURL {
			fmt.Printf("   %s %s\n", colorGreen("Final URL:"), result.FinalURL)
		}

		if len(result.Fingerprint) > 0 {
			fmt.Printf("   %s %s\n", colorGreen("Fingerprint:"), strings.Join(result.Fingerprint, ", "))
		}

		if result.FaviconIcon != "" {
			fmt.Printf("   %s %s  (hash: %d)\n", colorGreen("Favicon:"), result.FaviconIcon, result.Favicon)
		}

		if result.ScreenshotPath != "" {
			fmt.Printf("   %s saved\n", colorGreen("Screenshot:"))
		}
	}
}

func outputJSON(results []*ScrapeResult, filename string) {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Printf("%s failed to serialize JSON: %v\n", colorRed("Error:"), err)
		return
	}

	err = ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		fmt.Printf("%s failed to write file: %v\n", colorRed("Error:"), err)
		return
	}

	fmt.Printf("%s results saved to: %s\n", colorGreen("[Done]"), filename)
}

func outputText(results []*ScrapeResult, filename string) {
	var output strings.Builder

	output.WriteString("Chrome Scraper Results\n")
	output.WriteString(strings.Repeat("=", 50) + "\n\n")

	for i, result := range results {
		output.WriteString(fmt.Sprintf("%d. %s\n", i+1, result.TargetURL))

		if result.Error != "" {
			output.WriteString(fmt.Sprintf("   Error: %s\n", result.Error))
			output.WriteString("\n")
			continue
		}

		output.WriteString(fmt.Sprintf("   Title: %s\n", result.Title))
		if result.StatusCode > 0 {
			output.WriteString(fmt.Sprintf("   Status Code: %d\n", result.StatusCode))
		}
		output.WriteString(fmt.Sprintf("   Duration: %s\n", result.Duration))

		if len(result.JSLinks) > 0 {
			output.WriteString("   JS Links:\n")
			for _, js := range result.JSLinks {
				output.WriteString(fmt.Sprintf("     • %s\n", js))
			}
		}

		if result.FinalURL != "" && result.FinalURL != result.TargetURL {
			output.WriteString(fmt.Sprintf("   Final URL: %s\n", result.FinalURL))
		}

		if len(result.Fingerprint) > 0 {
			output.WriteString(fmt.Sprintf("   Fingerprint: %s\n", strings.Join(result.Fingerprint, ", ")))
		}

		if result.FaviconIcon != "" {
			output.WriteString(fmt.Sprintf("   Favicon: %s (hash: %d)\n", result.FaviconIcon, result.Favicon))
		}

		if result.ScreenshotPath != "" {
			output.WriteString("   Screenshot: saved\n")
		}

		output.WriteString("\n")
	}

	err := ioutil.WriteFile(filename, []byte(output.String()), 0644)
	if err != nil {
		fmt.Printf("%s failed to write file: %v\n", colorRed("Error:"), err)
		return
	}

	fmt.Printf("%s results saved to: %s\n", colorGreen("[Done]"), filename)
}

func printStatistics(results []*ScrapeResult) {
	successful := 0
	failed := 0
	totalDuration := time.Duration(0)

	for _, result := range results {
		if result.Error == "" {
			successful++
		} else {
			failed++
		}
		totalDuration += result.Duration
	}

	fmt.Printf("%s total: %d  success: %s  failed: %s  total time: %s  average: %s\n",
		colorBlue("[Stats]"),
		len(results),
		colorGreen(fmt.Sprintf("%d", successful)),
		colorRed(fmt.Sprintf("%d", failed)),
		colorYellow(totalDuration.Round(time.Millisecond).String()),
		colorYellow(func() string {
			if len(results) == 0 {
				return "0s"
			}
			return (totalDuration / time.Duration(len(results))).Round(time.Millisecond).String()
		}()),
	)
}

func saveScreenshot(result *ScrapeResult) error {
	const screenshotDir = "screenshots"

	if result == nil || result.Screenshot == "" {
		return nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to determine working directory: %v", err)
	}

	outputDir := filepath.Join(wd, screenshotDir)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create screenshot directory: %v", err)
	}

	data, err := base64.StdEncoding.DecodeString(result.Screenshot)
	if err != nil {
		return fmt.Errorf("failed to decode screenshot for %s: %v", result.TargetURL, err)
	}

	filename := fmt.Sprintf("%s.png", screenshotFileBase(result))
	fullPath := filepath.Join(outputDir, filename)
	if err := ioutil.WriteFile(fullPath, data, 0644); err != nil {
		return fmt.Errorf("failed to save screenshot for %s: %v", result.TargetURL, err)
	}

	result.ScreenshotPath = fullPath
	result.Screenshot = ""
	return nil
}

func screenshotFileBase(result *ScrapeResult) string {
	source := result.TargetURL
	if source == "" {
		return "screenshot"
	}

	source = strings.ReplaceAll(source, ".", "-")
	replacer := strings.NewReplacer(
		"://", "-",
		"/", "-",
		"\\", "-",
		":", "-",
		"?", "-",
		"&", "-",
		"=", "-",
		"#", "-",
		"%", "-",
		" ", "-",
	)
	source = replacer.Replace(source)
	for strings.Contains(source, "--") {
		source = strings.ReplaceAll(source, "--", "-")
	}
	source = strings.Trim(source, ".-_")
	if source == "" {
		return "screenshot"
	}
	if len(source) > 80 {
		source = source[:80]
	}
	return source
}
