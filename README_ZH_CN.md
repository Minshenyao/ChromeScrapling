# Chrome Scraper

[English README](./README.md)

一个基于Go语言和Rod库开发的高性能Chrome网站信息爬取工具。

## 功能特性

- 🌐 **网页标题获取** - 提取网页标题信息
- 📜 **JavaScript链接提取** - 自动发现并提取所有JS文件链接
- 🔍 **网站指纹识别** - 识别常见的Web技术栈和框架
- 🎨 **Favicon获取** - 提取网站图标链接
- 📸 **网页截图** - 可选的全页面截图功能
- ⚡ **多线程并发** - 支持自定义并发数量
- 🔒 **代理支持** - 支持HTTP/HTTPS代理
- 📊 **多种输出格式** - 支持JSON、TXT格式输出
- 🎯 **批量处理** - 支持从文件读取URL列表

## 安装

### 从源码编译

```bash
git clone <repository-url>
cd ChromeScrapling
go mod tidy
go build -trimpath -ldflags="-s -w" -o chrome-scraper
```

### 直接运行

```bash
go run . -u https://example.com
```

## 使用方法

### 基本用法

```bash
# 爬取单个网站
./chrome-scraper -u https://example.com

# 爬取多个网站
./chrome-scraper -u https://example.com,https://google.com

# 从文件读取URL列表
./chrome-scraper -f urls.txt

# 启用截图功能
./chrome-scraper -u https://example.com -s

# 页面加载后延迟2秒再采集title、JS和截图
./chrome-scraper -u https://example.com --delay 2s -s

# 使用代理
./chrome-scraper -u https://example.com --proxy http://127.0.0.1:8080

# 设置并发线程数
./chrome-scraper -u https://example.com -t 10

# 输出到JSON文件
./chrome-scraper -u https://example.com -o results.json

# 详细输出模式
./chrome-scraper -u https://example.com -v
```

### 命令行参数

| 参数 | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| `--url` | `-u` | - | 要爬取的URL列表，用逗号分隔 |
| `--file` | `-f` | - | 包含URL列表的文件路径 |
| `--threads` | `-t` | 3 | 并发线程数 |
| `--screenshot` | `-s` | false | 是否截图 |
| `--delay` | - | 0 | 页面加载完成后延迟多久再采集 title、JS、指纹、favicon 和截图，支持 `500ms`、`2s`、`1m` |
| `--proxy` | `-p` | - | 代理服务器地址 |
| `--timeout` | - | 30 | 请求超时时间(秒) |
| `--output` | `-o` | - | 输出文件路径 |
| `--headless` | - | true | 是否使用无头模式 |
| `--user-agent` | - | Mozilla/5.0... | 自定义User-Agent |
| `--verbose` | `-v` | false | 详细输出模式 |
| `--resume` | - | - | 断点续爬状态文件路径，中断后可继续上次任务 |

### URL文件格式

创建一个文本文件，每行一个URL：

```
https://example.com
https://google.com
https://github.com
# 这是注释，会被忽略
https://stackoverflow.com
```

## 输出格式

### 控制台输出

```
[Start] starting scrape...    screenshots: false        delay: 0s        threads: 3        proxy: off
[+] https://google.com | 200 | Google
Scrape progress 100% [██████████████████████████████████████████████████] (1/1, 14 it/min)
[Stats] total: 1  success: 1  failed: 0  total time: 3.554s  average: 3.554s
```

### JSON输出

```json
[
  {
    "target_url": "https://google.com",
    "final_url": "https://www.google.com/",
    "status_code": 200,
    "title": "Google",
    "js_links": null,
    "fingerprint": [
      "HTTP/3"
    ],
    "favicon_icon": "https://www.google.com/favicon.ico",
    "favicon": 708578229
  }
]
```

### 截图输出

启用 `-s` 后，截图会在每个目标处理完成后实时保存到当前目录下的 `screenshots/` 目录，不会等全部任务结束后再统一写入。文件名默认基于目标URL生成，例如：

```text
screenshots/https-google-com.png
```

## 指纹识别

工具内置了常见Web技术的指纹识别规则：

- **前端框架**: jQuery, Bootstrap, Vue.js, React, Angular
- **后端技术**: WordPress, Laravel, Django, Express
- **Web服务器**: Nginx, Apache, IIS
- **CDN服务**: Cloudflare
- **编程语言**: PHP

## 性能优化

- 使用Rod库，基于Chrome DevTools Protocol，性能优异
- 支持多线程并发，可根据系统性能调整
- 智能超时控制，避免长时间等待
- 内存优化的截图处理

## 注意事项

1. **系统要求**: 需要系统安装Chrome或Chromium浏览器
2. **网络环境**: 确保目标网站可访问
3. **并发控制**: 建议根据系统性能和目标网站承受能力调整线程数
4. **代理设置**: 支持HTTP/HTTPS代理，格式为 `http://host:port`
5. **截图功能**: 启用截图会增加处理时间；截图会实时保存到 `screenshots/`
6. **延迟采集**: `--delay` 会在页面加载完成后统一等待一段时间，再获取 title、JS、指纹和截图

## 故障排除

### 常见问题

1. **浏览器启动失败**
   - 确保系统已安装Chrome/Chromium
   - 检查系统权限设置

2. **网络连接超时**
   - 检查网络连接
   - 调整超时时间参数
   - 考虑使用代理

3. **内存使用过高**
   - 减少并发线程数
   - 关闭截图功能
   - 分批处理大量URL

## 开发

### 项目结构

```
ChromeScrapling/
├── main.go          # 主程序和CLI接口
├── scraper.go       # 核心爬虫逻辑
├── types.go         # 数据结构定义
├── go.mod           # Go模块文件
└── README.md        # 说明文档
```

### 扩展指纹识别

在 `types.go` 中的 `FingerprintRules` 数组添加新规则：

```go
{"技术名称", "检测类型", "匹配模式", "位置"}
```

检测类型支持：
- `header`: HTTP响应头检测
- `body`: 页面内容检测
- `script`: JavaScript文件检测
- `meta`: Meta标签检测

## 许可证

MIT License

## 贡献

欢迎提交Issue和Pull Request！
