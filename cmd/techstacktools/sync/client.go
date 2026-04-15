package sync

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// BatchSize 批量导入每批最大数量
const BatchSize = 56

// MaxRetries 最大重试次数
const MaxRetries = 3

// Client techstack API 客户端
type Client struct {
	serverURL        string
	accessKey        string
	secretKey        string
	token            string
	httpClient       *http.Client
	signatureVersion string // "1" for new signature auth
}

// NewClientWithAKSK 创建新的API客户端（AK/SK认证）
func NewClientWithAKSK(serverURL, accessKey, secretKey string) *Client {
	return &Client{
		serverURL:        serverURL,
		accessKey:        accessKey,
		secretKey:        secretKey,
		signatureVersion: "1",

		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientWithSignature 创建新的API客户端（签名认证）
func NewClientWithSignature(serverURL, accessKey, secretKey string) *Client {
	return &Client{
		serverURL:        serverURL,
		accessKey:        accessKey,
		secretKey:        secretKey,
		signatureVersion: "1",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AddPackage 添加库到 techstack 服务
func (c *Client) AddPackage(name string, pkgType string) error {
	addURL := fmt.Sprintf("%s/api/v1/c/libraries", c.serverURL)

	reqBody, err := json.Marshal(map[string]string{
		"name":      name,
		"purl_type": pkgType,
	})
	if err != nil {
		return fmt.Errorf("marshal add package request: %w", err)
	}

	req, err := http.NewRequest("POST", addURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("create add package request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 根据认证类型添加相应的 Header
	if err := c.SignRequest(req, nil); err != nil {
		return fmt.Errorf("sign request failed: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("add package request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read add package response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("add package failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	return nil
}

// AddLibraries 批量添加库
func (c *Client) AddLibraries(packages []map[string]string) error {
	addURL := fmt.Sprintf("%s/api/v1/c/libraries", c.serverURL)

	// 批量添加
	for _, pkg := range packages {
		reqBody, err := json.Marshal(map[string]string{
			"name":      pkg["name"],
			"purl_type": pkg["purl_type"],
		})
		if err != nil {
			return fmt.Errorf("marshal add package request: %w", err)
		}

		req, err := http.NewRequest("POST", addURL, bytes.NewBuffer(reqBody))
		if err != nil {
			return fmt.Errorf("create add package request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		// 使用签名认证
		if err := c.SignRequest(req, nil); err != nil {
			return fmt.Errorf("sign request failed: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("add package request failed: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read add package response: %w", err)
		}

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			if resp.StatusCode == 400 {
				fmt.Println(strings.Repeat("==", 10), pkg["name"], string(body), strings.Repeat("==", 10))
				continue
			}
			return fmt.Errorf("add package failed: status=%d, body=%s", resp.StatusCode, string(body))
		}
	}

	return nil
}

// SignRequest 为请求添加签名认证 Header
// queryParams 是 URL 查询参数（可选，可以是 nil）
func (c *Client) SignRequest(req *http.Request, queryParams map[string]string) error {
	// 添加版本 Header
	req.Header.Set("x-aksk-version", c.signatureVersion)
	req.Header.Set("x-access-key", c.accessKey)

	// 添加时间戳
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	req.Header.Set("x-timestamp", timestamp)

	// 收集所有请求参数（除了 signature 本身）
	params := make(map[string]string)
	for k, v := range req.URL.Query() {
		if k != "signature" && len(v) > 0 {
			params[k] = v[0]
		}
	}
	// 添加额外的查询参数
	for k, v := range queryParams {
		if k != "signature" {
			params[k] = v
		}
	}

	// 构造签名字符串
	signatureString := c.buildSignatureString(req.Method, req.URL.Path, params, timestamp)

	// 使用 HMAC-SHA256 计算签名
	signature := c.computeHmacSHA256(signatureString)
	req.Header.Set("x-signature", signature)

	return nil
}

// buildSignatureString 构造签名字符串
// 格式: {method}&{path}&{sorted_params}&{timestamp}
func (c *Client) buildSignatureString(method, path string, params map[string]string, timestamp string) string {
	var sb strings.Builder
	sb.WriteString(method)
	sb.WriteString("&")
	sb.WriteString(path)
	sb.WriteString("&")

	// 按参数名排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 构造排序后的参数字符串
	for i, k := range keys {
		if i > 0 {
			sb.WriteString("&")
		}
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(params[k])
	}

	// 添加时间戳
	sb.WriteString("&")
	sb.WriteString(timestamp)

	return sb.String()
}

// computeHmacSHA256 使用 HMAC-SHA256 计算签名
func (c *Client) computeHmacSHA256(message string) string {
	h := hmac.New(sha256.New, []byte(c.secretKey))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// BatchImportLibraries 批量导入库（支持分批处理）
// 使用 /api/v1/c/libraries/batch 接口，单次最多5000条
func (c *Client) BatchImportLibraries(packages []map[string]string) error {
	total := len(packages)
	if total == 0 {
		fmt.Println("没有需要导入的库")
		return nil
	}

	fmt.Printf("开始批量导入库，共 %d 条记录\n", total)
	fmt.Printf("每批最大 %d 条，共分 %d 批\n", BatchSize, (total+BatchSize-1)/BatchSize)

	// 计算分批数量
	batchCount := (total + BatchSize - 1) / BatchSize
	totalBatchCount := batchCount

	// 用于统计
	var successCount int
	var failCount int

	// 分批处理
	for batchIdx := 0; batchIdx < batchCount; batchIdx++ {
		// 计算当前批的起始和结束索引
		start := batchIdx * BatchSize
		end := start + BatchSize
		if end > total {
			end = total
		}

		batchNum := batchIdx + 1
		currentBatchSize := end - start

		fmt.Printf("\n[%d/%d] 正在导入第 %d-%d 条（共 %d 条）...\n",
			batchNum, totalBatchCount, start+1, end, currentBatchSize)

		// 准备当前批的 purl 列表
		var batchReq []map[string]string
		for i := start; i < end; i++ {
			pkg := packages[i]
			// 构造 PURL 格式: pkgtype:name
			batchReq = append(batchReq, pkg)
		}

		// 重试机制
		var lastErr error
		for retry := 0; retry < MaxRetries; retry++ {
			if retry > 0 {
				fmt.Printf("[重试 %d/%d] 第 %d 批...\n", retry, MaxRetries, batchNum)
				// 指数退避等待
				waitTime := time.Duration(retry*retry) * time.Second
				fmt.Printf("等待 %v 后重试...\n", waitTime)
				time.Sleep(waitTime)
			}

			// 调用批量导入接口
			rows, err := c.doBatchImport(batchReq)
			if err == nil {
				// 成功
				successCount += int(rows)
				fmt.Printf("[%d/%d] 导入成功: %d 条\n", batchNum, totalBatchCount, rows)
				break
			}

			lastErr = err
			fmt.Printf("[%d/%d] 导入失败: %v\n", batchNum, totalBatchCount, err)
		}

		if lastErr != nil {
			// 所有重试都失败
			failCount += currentBatchSize
			fmt.Printf("[%d/%d] 批次导入失败，已跳过 %d 条记录\n", batchNum, totalBatchCount, currentBatchSize)
			// 继续处理下一批
		}

		// 每批之间短暂延迟，避免请求过快
		if batchIdx < batchCount-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// 打印统计结果
	fmt.Printf("\n========== 批量导入完成 ==========\n")
	fmt.Printf("总计: %d 条\n", total)
	fmt.Printf("成功: %d 条\n", successCount)
	fmt.Printf("失败: %d 条\n", failCount)
	fmt.Printf("==================================\n")

	if failCount > 0 {
		return fmt.Errorf("批量导入完成，但有 %d 条记录失败", failCount)
	}

	return nil
}

// doBatchImport 执行单次批量导入请求
func (c *Client) doBatchImport(batchReq []map[string]string) (int64, error) {
	batchURL := fmt.Sprintf("%s/api/v1/c/libraries/batch", c.serverURL)

	// 构造请求体
	reqBody, err := json.Marshal(map[string]interface{}{
		"list": batchReq,
	})
	if err != nil {
		return 0, fmt.Errorf("marshal batch request: %w", err)
	}

	req, err := http.NewRequest("POST", batchURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return 0, fmt.Errorf("create batch request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 签名认证
	if err := c.SignRequest(req, nil); err != nil {
		return 0, fmt.Errorf("sign request failed: %w", err)
	}

	// 设置超时
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("batch import request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("read batch response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("batch import failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	// 解析响应，检查是否有错误
	var respBody map[string]interface{}
	if err := json.Unmarshal(body, &respBody); err != nil {
		// 响应可能不是 JSON 格式，但状态码是 200/201，认为成功
		v, _ := respBody["rows"].(string)
		rows, _ := strconv.ParseInt(v, 10, 64)
		return rows, nil
	}

	// 检查响应中的错误码
	if code, ok := respBody["code"]; ok {
		if codeFloat, ok := code.(float64); ok && codeFloat != 0 {
			if msg, ok := respBody["msg"].(string); ok {
				return 0, fmt.Errorf("batch import error: %s", msg)
			}
			return 0, fmt.Errorf("batch import error code: %.0f", codeFloat)
		}
	}

	return 0, nil
}
