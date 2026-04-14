package httpd

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/alayou/techstack/global"
	"github.com/alayou/techstack/httpd/dao"
	"github.com/alayou/techstack/model"
)

type MiddlewareFunc func(http.Handler) http.HandlerFunc

func Middleware(handlerFunc http.HandlerFunc, handlers ...MiddlewareFunc) http.HandlerFunc {
	if len(handlers) == 0 {
		return handlerFunc
	}
	var m MiddlewareFunc
	for {
		if len(handlers) == 0 {
			break
		}
		m = handlers[0]
		handlerFunc = m(handlerFunc)
		handlers = handlers[1:]
	}
	return handlerFunc
}

// checkCors 跨域配置
func checkCors(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		method := req.Method
		origin := req.Header.Get("Origin")
		if global.Config.Debug {
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token,X-AUTH-Token, Authorization, Token,X-Token,X-User-SourceID, X-REQUESTED-WITH")
			w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, GET, POST, DELETE, PUT")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type, X-REQUESTED-WITH")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			// 放行所有OPTIONS方法
			if method == "OPTIONS" {
				w.WriteHeader(204)
				return
			}
			//logger.Info(logSender,"checkCors start")
			handler.ServeHTTP(w, req)
			//logger.Info(logSender,"checkCors end")

			return
		}
		w.Header().Set("Access-Control-Allow-Origin", global.Config.Cors.AllowOrigin)
		w.Header().Set("Access-Control-Allow-Headers", global.Config.Cors.AllowHeaders)
		w.Header().Set("Access-Control-Allow-Methods", global.Config.Cors.AllowMethods)
		w.Header().Set("Access-Control-Expose-Headers", global.Config.Cors.AllowHeaders)
		w.Header().Set("Access-Control-Allow-Credentials", global.Config.Cors.AllowCredentials)
		if method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		log.Info().Str("logSender", logSender).Str("method", req.Method).
			Str("path", req.URL.Path).
			Str("origin", origin).Send()

		handler.ServeHTTP(w, req)
		//logger.Info(logSender,"checkCors end")
	}
}

const BearerPrefix = "Bearer "

func (s *Server) checkUserToken(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var err error
		uid := UidGetStr(req)
		if uid != "" {
			handler.ServeHTTP(w, req)
			return
		}
		uid, err = s.sessionMgr.AuthorizationHandler()(w, req)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		req = UidSetStr(req, uid)
		handler.ServeHTTP(w, req)
	}
}

// checkAKSKAuth AK/SK 认证中间件
func (s *Server) checkAKSKAuth(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// 检查是否使用新的签名认证方式 (version=1)
		akskVersion := req.Header.Get("x-aksk-version")
		if akskVersion == "1" {
			// 使用签名认证方式
			uid := s.verifyAKSKSignature(w, req)
			if uid == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			req = UidSetStr(req, uid)
			handler.ServeHTTP(w, req)
			return
		}
		handler.ServeHTTP(w, req)
	}
}

// checkAdminToken 管理员权限验证中间件
func (s *Server) checkAdminToken(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		uidStr := UidGetStr(req)
		if uidStr == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		uid, err := strconv.ParseInt(uidStr, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		user := dao.User.Find(uid)
		if user == nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// 检查用户角色是否为管理员
		if user.Role != model.RoleAdmin {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		handler.ServeHTTP(w, req)
	}
}

// verifyAKSKSignature 验证 AK/SK 签名认证
func (s *Server) verifyAKSKSignature(w http.ResponseWriter, req *http.Request) string {
	// 获取签名参数：可以从 Header 或 Query 参数中获取
	accessKey := req.Header.Get("x-access-key")
	if accessKey == "" {
		accessKey = req.URL.Query().Get("access_key")
	}

	timestamp := req.Header.Get("x-timestamp")
	if timestamp == "" {
		timestamp = req.URL.Query().Get("timestamp")
	}

	signature := req.Header.Get("x-signature")
	if signature == "" {
		signature = req.URL.Query().Get("signature")
	}

	// 验证必要参数
	if accessKey == "" || timestamp == "" || signature == "" {
		return ""
	}

	// 根据 access_key 查询用户
	user := dao.User.FindByAccountKey(accessKey)
	if user == nil || user.AccountSecret == "" {
		return ""
	}

	// 验证时间戳（防止重放攻击）
	if !s.validateTimestamp(timestamp) {
		return ""
	}

	// 收集所有请求参数（除了 signature 本身）
	params := make(map[string]string)
	for k, v := range req.URL.Query() {
		if k != "signature" && len(v) > 0 {
			params[k] = v[0]
		}
	}

	// 验证签名
	if !verifySignature(user.AccountSecret, signature, req.Method, req.URL.Path, params, timestamp) {
		return ""
	}

	return user.ID.String()
}

// validateTimestamp 验证时间戳是否在有效期内（5分钟）
func (s *Server) validateTimestamp(timestamp string) bool {
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}

	now := time.Now().Unix()
	// 允许 ±5 分钟的误差
	drift := int64(5 * 60)
	if ts < now-drift || ts > now+drift {
		return false
	}

	return true
}

// buildSignatureString 构造签名字符串
// 格式: {method}&{path}&{sorted_params}&{timestamp}
func buildSignatureString(method, path string, params map[string]string, timestamp string) string {
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
func computeHmacSHA256(secret, message string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return string(h.Sum(nil))
}

// verifySignature 验证签名
func verifySignature(sk, signature, method, path string, params map[string]string, timestamp string) bool {
	sig, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	// 构造签名字符串
	signatureString := buildSignatureString(method, path, params, timestamp)
	// 计算签名
	computedSignature := computeHmacSHA256(sk, signatureString)
	// 使用 ConstantTimeCompare 防止时序攻击
	return subtle.ConstantTimeCompare([]byte(computedSignature), []byte(sig)) == 1
}
