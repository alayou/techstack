package goproxycn

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client 是 goproxy.cn 服务的客户端
type Client struct {
	BaseURL    string
	HttpClient *http.Client
}

// options 配置选项
type options struct {
	baseUrl string
}

// Option 函数类型，用于配置 Client
type Option func(*options)

// WithProxy 设置代理地址
func WithProxy(baseUrl string) Option {
	return func(o *options) {
		o.baseUrl = baseUrl
	}
}

// NewClient 创建新的 goproxy.cn 客户端
// 默认 baseUrl 为 https://goproxy.cn
func NewClient(opts ...Option) *Client {
	var cfg = &options{
		baseUrl: "https://goproxy.cn",
	}
	for _, opt := range opts {
		opt(cfg)
	}
	client := Client{
		BaseURL:    cfg.baseUrl,
		HttpClient: &http.Client{Timeout: 10 * time.Second},
	}
	return &client
}

// ModuleHostCount 模块主机版本数量统计
type ModuleHostCount struct {
	ModuleHost         string `json:"module_host"`
	ModuleVersionCount int64  `json:"module_version_count"`
}

// SummaryResponse 服务摘要信息响应
// 包含缓存大小、模块版本总数、模块主机数量以及TOP10模块主机列表
type SummaryResponse struct {
	CacherSize         int64             `json:"cacher_size"`          // 缓存总大小（字节）
	ModuleVersionCount int64             `json:"module_version_count"` // 模块版本总数
	ModuleHostCount    int64             `json:"module_host_count"`    // 模块主机数量
	Top10ModuleHosts   []ModuleHostCount `json:"top_10_module_hosts"`  // TOP 10 模块主机列表
}

// TrendItem 趋势项目
type TrendItem struct {
	ModulePath    string `json:"module_path"`    // 模块路径
	DownloadCount int64  `json:"download_count"` // 下载次数
}

// ModuleStatsDay 模块单日统计
type ModuleStatsDay struct {
	Date          string `json:"date"`           // 日期，格式：2006-01-02T15:04:05Z
	DownloadCount int64  `json:"download_count"` // 下载次数
}

// ModuleVersionCount 模块版本下载统计
type ModuleVersionCount struct {
	ModuleVersion string `json:"module_version"` // 模块版本号
	DownloadCount int64  `json:"download_count"` // 下载次数
}

// ModuleStatsResponse 模块统计信息响应
// 包含总下载次数、最近30天每日下载统计以及TOP10版本下载统计
type ModuleStatsResponse struct {
	DownloadCount      int64                `json:"download_count"`         // 总下载次数
	Last30Days         []ModuleStatsDay     `json:"last_30_days"`           // 最近30天每日下载统计
	Top10ModuleVersion []ModuleVersionCount `json:"top_10_module_versions"` // TOP 10 模块版本下载统计
}

// GetStatsSummary 获取服务摘要信息
// 返回服务中所有模块版本的总大小和总数统计信息
//
// API: GET /stats/summary
// 示例: goproxy.cn/stats/summary
//
// 返回值:
//   - SummaryResponse: 包含缓存大小、模块版本总数、模块主机数量及TOP10主机列表
func (c *Client) GetStatsSummary(ctx context.Context) (SummaryResponse, error) {
	path := fmt.Sprintf("%s/stats/summary", c.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return SummaryResponse{}, err
	}
	res, err := c.HttpClient.Do(req)
	if err != nil {
		return SummaryResponse{}, err
	}
	defer res.Body.Close()

	var out SummaryResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return SummaryResponse{}, err
	}
	return out, nil
}

// TrendType 趋势类型
type TrendType string

const (
	TrendLatest     TrendType = "latest"       // 最新趋势
	TrendLast7Days  TrendType = "last-7-days"  // 最近7天趋势
	TrendLast30Days TrendType = "last-30-days" // 最近30天趋势
)

// GetStatsTrends 获取模块趋势统计
// 返回最近一段时间内最活跃的TOP 1000模块列表
//
// API: GET /stats/trends/<trend>
// 示例: goproxy.cn/stats/trends/latest
//
// 参数:
//   - ctx: 上下文
//   - trend: 趋势类型，支持 latest、last-7-days、last-30-days
//
// 返回值:
//   - []TrendItem: 模块路径和下载次数列表
func (c *Client) GetStatsTrends(ctx context.Context, trend TrendType) ([]TrendItem, error) {
	path := fmt.Sprintf("%s/stats/trends/%s", c.BaseURL, trend)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return []TrendItem{}, err
	}
	res, err := c.HttpClient.Do(req)
	if err != nil {
		return []TrendItem{}, err
	}
	defer res.Body.Close()

	var out []TrendItem
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return []TrendItem{}, err
	}
	return out, nil
}

// GetStatsModule 获取指定模块(版本)的统计信息
// 返回模块的总下载次数、最近30天每日下载统计以及各版本下载统计
//
// API: GET /stats/<module-path>[@<module-version>]
// 示例: goproxy.cn/stats/golang.org/x/text
// 示例: goproxy.cn/stats/golang.org/x/text@v0.3.2
//
// 参数:
//   - ctx: 上下文
//   - modulePath: 模块路径，如 golang.org/x/text
//   - moduleVersion: 模块版本（可选），如 v0.3.2，不填则返回模块所有版本的统计
//
// 返回值:
//   - ModuleStatsResponse: 包含总下载次数、最近30天统计及TOP10版本统计
func (c *Client) GetStatsModule(ctx context.Context, modulePath string, moduleVersion string) (ModuleStatsResponse, error) {
	var path string
	if moduleVersion != "" {
		path = fmt.Sprintf("%s/stats/%s@%s", c.BaseURL, modulePath, moduleVersion)
	} else {
		path = fmt.Sprintf("%s/stats/%s", c.BaseURL, modulePath)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return ModuleStatsResponse{}, err
	}
	res, err := c.HttpClient.Do(req)
	if err != nil {
		return ModuleStatsResponse{}, err
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return ModuleStatsResponse{}, err
	}

	var out ModuleStatsResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return ModuleStatsResponse{}, err
	}
	return out, nil
}

// DefaultClient 默认客户端实例
var DefaultClient = NewClient()

// GetStatsSummary 获取服务摘要信息（使用默认客户端）
func GetStatsSummary(ctx context.Context) (SummaryResponse, error) {
	return DefaultClient.GetStatsSummary(ctx)
}

// GetStatsTrends 获取模块趋势统计（使用默认客户端）
func GetStatsTrends(ctx context.Context, trend TrendType) ([]TrendItem, error) {
	return DefaultClient.GetStatsTrends(ctx, trend)
}

// GetStatsModule 获取指定模块(版本)的统计信息（使用默认客户端）
func GetStatsModule(ctx context.Context, modulePath string, moduleVersion string) (ModuleStatsResponse, error) {
	return DefaultClient.GetStatsModule(ctx, modulePath, moduleVersion)
}
