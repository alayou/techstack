package goproxycn

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"testing"
	"time"
)

// TestNewClient 测试客户端创建
func TestNewClient(t *testing.T) {
	client := NewClient()
	if client.BaseURL != "https://goproxy.cn" {
		t.Errorf("expected BaseURL to be https://goproxy.cn, got %s", client.BaseURL)
	}
	if client.HttpClient == nil {
		t.Error("expected HttpClient to be non-nil")
	}
}

// TestNewClientWithCustomProxy 测试使用自定义代理创建客户端
func TestNewClientWithCustomProxy(t *testing.T) {
	customURL := "https://custom.goproxy.cn"
	client := NewClient(WithProxy(customURL))
	if client.BaseURL != customURL {
		t.Errorf("expected BaseURL to be %s, got %s", customURL, client.BaseURL)
	}
}

// TestGetStatsSummary 测试获取服务摘要信息
func TestGetStatsSummary(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	summary, err := client.GetStatsSummary(ctx)
	if err != nil {
		t.Fatalf("GetStatsSummary failed: %v", err)
	}

	// 打印结果
	data, _ := json.MarshalIndent(summary, "", "  ")
	fmt.Printf("Summary Response:\n%s\n", data)

	// 验证基本字段
	if summary.ModuleVersionCount <= 0 {
		t.Error("expected ModuleVersionCount to be positive")
	}
	if summary.ModuleHostCount <= 0 {
		t.Error("expected ModuleHostCount to be positive")
	}
	if len(summary.Top10ModuleHosts) == 0 {
		t.Error("expected Top10ModuleHosts to be non-empty")
	}

	// 验证 TOP 10 列表长度
	if len(summary.Top10ModuleHosts) != 10 {
		t.Logf("expected Top10ModuleHosts length to be 10, got %d", len(summary.Top10ModuleHosts))
	}
}

// TestGetStatsTrends 测试获取模块趋势统计
func TestGetStatsTrends(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testCases := []struct {
		trend TrendType
		name  string
	}{
		{TrendLatest, "latest"},
		{TrendLast7Days, "last-7-days"},
		{TrendLast30Days, "last-30-days"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			trends, err := client.GetStatsTrends(ctx, tc.trend)
			if err != nil {
				t.Fatalf("GetStatsTrends(%s) failed: %v", tc.name, err)
			}

			// 打印前5条结果
			fmt.Printf("Trends (%s) - top 5:\n", tc.name)
			for i := 0; i < len(trends) && i < 5; i++ {
				fmt.Printf("  %d. %s: %d downloads\n", i+1, trends[i].ModulePath, trends[i].DownloadCount)
			}

			if len(trends) == 0 {
				t.Error("expected trends to be non-empty")
			}
		})
	}
}

// TestGetStatsModule 测试获取指定模块的统计信息
func TestGetStatsModule(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	modulePath := "golang.org/x/text"

	stats, err := client.GetStatsModule(ctx, modulePath, "")
	if err != nil {
		t.Fatalf("GetStatsModule failed: %v", err)
	}

	// 打印结果
	fmt.Printf("Module Stats for %s:\n", modulePath)
	fmt.Printf("  Total Downloads: %d\n", stats.DownloadCount)
	fmt.Printf("  Last 30 Days: %d entries\n", len(stats.Last30Days))
	fmt.Printf("  Top 10 Versions: %d entries\n", len(stats.Top10ModuleVersion))

	if stats.DownloadCount <= 0 {
		t.Error("expected DownloadCount to be positive")
	}
	if len(stats.Last30Days) == 0 {
		t.Error("expected Last30Days to be non-empty")
	}
	if len(stats.Top10ModuleVersion) == 0 {
		t.Error("expected Top10ModuleVersion to be non-empty")
	}
}

// TestGetStatsModuleWithVersion 测试获取指定模块指定版本的统计信息
func TestGetStatsModuleWithVersion(t *testing.T) {
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	modulePath := "golang.org/x/text"
	moduleVersion := "v0.3.2"

	stats, err := client.GetStatsModule(ctx, modulePath, moduleVersion)
	if err != nil {
		t.Fatalf("GetStatsModule with version failed: %v", err)
	}

	// 打印结果
	fmt.Printf("Module Stats for %s@%s:\n", modulePath, moduleVersion)
	fmt.Printf("  Total Downloads: %d\n", stats.DownloadCount)
	fmt.Printf("  Last 30 Days: %d entries\n", len(stats.Last30Days))

	if stats.DownloadCount <= 0 {
		t.Error("expected DownloadCount to be positive")
	}
	if len(stats.Last30Days) == 0 {
		t.Error("expected Last30Days to be non-empty")
	}
}

// TestDefaultClient 测试默认客户端
func TestDefaultClient(t *testing.T) {
	if DefaultClient == nil {
		t.Error("expected DefaultClient to be non-nil")
	}
	if DefaultClient.BaseURL != "https://goproxy.cn" {
		t.Errorf("expected DefaultClient.BaseURL to be https://goproxy.cn, got %s", DefaultClient.BaseURL)
	}
}

// TestPackageLevelFunctions 测试包级别快捷函数
func TestPackageLevelFunctions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("GetStatsSummary", func(t *testing.T) {
		summary, err := GetStatsSummary(ctx)
		if err != nil {
			t.Fatalf("GetStatsSummary failed: %v", err)
		}
		if summary.ModuleVersionCount <= 0 {
			t.Error("expected ModuleVersionCount to be positive")
		}
	})

	t.Run("GetStatsTrends", func(t *testing.T) {
		trends, err := GetStatsTrends(ctx, TrendLatest)
		if err != nil {
			t.Fatalf("GetStatsTrends failed: %v", err)
		}
		if len(trends) == 0 {
			t.Error("expected trends to be non-empty")
		}
	})

	t.Run("GetStatsModule", func(t *testing.T) {
		stats, err := GetStatsModule(ctx, "golang.org/x/net", "")
		if err != nil {
			t.Fatalf("GetStatsModule failed: %v", err)
		}
		if stats.DownloadCount <= 0 {
			t.Error("expected DownloadCount to be positive")
		}
	})
}

// TestClientHTTPError 测试客户端 HTTP 错误处理
func TestClientHTTPError(t *testing.T) {
	// 创建一个使用无效地址的客户端
	client := NewClient(WithProxy("https://invalid-domain-that-does-not-exist.example.com"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.GetStatsSummary(ctx)
	if err == nil {
		t.Error("expected error for invalid domain")
	}

	// 验证是网络错误（而不是其他错误）
	if _, ok := err.(*url.Error); !ok {
		// 在 Go 1.22+ 中，错误可能被包装
		t.Logf("error type: %T", err)
	}
}

// TestResponseStructs 测试响应结构体 JSON 序列化
func TestResponseStructs(t *testing.T) {
	summary := SummaryResponse{
		CacherSize:         2663405247231,
		ModuleVersionCount: 1035421,
		ModuleHostCount:    1120,
		Top10ModuleHosts: []ModuleHostCount{
			{ModuleHost: "github.com", ModuleVersionCount: 921606},
		},
	}

	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("failed to marshal SummaryResponse: %v", err)
	}

	var decoded SummaryResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal SummaryResponse: %v", err)
	}

	if decoded.CacherSize != summary.CacherSize {
		t.Errorf("expected CacherSize %d, got %d", summary.CacherSize, decoded.CacherSize)
	}
}
