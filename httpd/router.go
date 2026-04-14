package httpd

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gorpher/gone/logger"

	httpMux "github.com/gorilla/mux"

	"github.com/alayou/techstack/global"
	. "github.com/gorpher/gone/httputil"
)

func NewRouter(middles ...MiddlewareFunc) *AdapterRouter {
	return &AdapterRouter{
		//gorilla: httpMux.NewRouter(),
		httpMux: http.NewServeMux(),
		middles: middles,
	}
}

type Router interface {
	ServeHTTP(rw http.ResponseWriter, req *http.Request)
	HandleFunc(path string, f func(http.ResponseWriter, *http.Request))
}

type AdapterRouter struct {
	gorilla *httpMux.Router
	httpMux *http.ServeMux
	middles []MiddlewareFunc
}

func (a *AdapterRouter) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	logger.Debug(logSender, "method:%s path:%s", req.Method, req.RequestURI)
	rw.Header().Set("Accept-Language", req.Header.Get("Accept-Language"))

	middles := a.middles
	if len(middles) == 0 {
		a.httpMux.ServeHTTP(rw, req)
		return
	}
	var handler http.Handler = a.httpMux
	var middle MiddlewareFunc
	for {
		if len(middles) == 0 {
			break
		}
		middle = middles[0]
		handler = middle(handler)
		middles = middles[1:]
	}
	handler.ServeHTTP(rw, req)
}

func (a *AdapterRouter) HandleFunc(fullPath string, f func(http.ResponseWriter, *http.Request)) {
	if a.httpMux != nil {
		a.httpMux.HandleFunc(fullPath, f)
		return
	}
	n := strings.SplitN(fullPath, " ", 2)
	var path string
	var method string
	if len(n) >= 2 {
		path = n[1]
		method = n[0]
	} else {
		path = n[0]
	}
	isPrefix := strings.HasSuffix(path, "/")
	if path == "/" {
		isPrefix = false
	}
	var route *httpMux.Route
	if isPrefix {
		route = a.gorilla.PathPrefix(path)
	} else {
		route = a.gorilla.Path(path)
	}
	route = route.HandlerFunc(f)
	if method != "" {
		route.Methods(method)
	}
}
func (a *AdapterRouter) PathValue(req *http.Request, key string) string {
	vars := httpMux.Vars(req)
	return vars[key]
}

func (a *AdapterRouter) ParamInt(r *http.Request, key string) int {
	valueStr := a.PathValue(r, key)
	if valueStr == "" {
		valueStr = r.FormValue(key)
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0
	}
	return value
}

func (a *AdapterRouter) ParamInt64(r *http.Request, key string) int64 {
	valueStr := a.PathValue(r, key)
	if valueStr == "" {
		valueStr = r.FormValue(key)
	}
	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func (a *AdapterRouter) ParamString(r *http.Request, key string) string {
	valueStr := a.PathValue(r, key)
	if valueStr != "" {
		return valueStr
	}
	return r.FormValue(key)
}

func (s *Server) initRouter() {
	pingingHandler := func(w http.ResponseWriter, req *http.Request) {
		Ok(w, "pong")
	}
	healthHandler := func(w http.ResponseWriter, req *http.Request) {
		Ok(w, JsonRawBody{
			"health":   true,
			"revision": global.Revision,
			"version":  global.Version,
			"build_at": global.RBuiltAt,
		})
	}

	// 公共接口
	s.router.HandleFunc("GET /api/v1/ping", Middleware(pingingHandler)) // 路由执行顺序，从后往前，从右到左
	s.router.HandleFunc("HEAD /api/v1/ping", Middleware(pingingHandler))
	s.router.HandleFunc("GET /api/v1/health", Middleware(healthHandler)) // health

	// 客户端接口
	// 客户端无权限接口
	s.router.HandleFunc("POST /api/v1/c/login", Middleware(s.Login))     // 登录
	s.router.HandleFunc("POST /api/v1/c/captcha", Middleware(s.Captcha)) // 验证码
	s.router.HandleFunc("POST /api/v1/c/signup", Middleware(s.Signup))   // 注册

	// s.router.HandleFunc("GET /api/v1/c/auth/github", Middleware(s.GitHubOAuth, nil)) // GitHub OAuth跳转
	// s.router.HandleFunc("GET /api/v1/c/auth/github/callback", Middleware(s.GitHubCallback, nil)) // OAuth回调
	// // 客户端有权限接口
	s.router.HandleFunc("POST /api/v1/c/logout", Middleware(s.Logout, s.checkUserToken))                   // 退出登录
	s.router.HandleFunc("GET /api/v1/c/profile", Middleware(s.Profile, s.checkUserToken))                  // 个人信息
	s.router.HandleFunc("PUT /api/v1/c/user/password", Middleware(s.UpdateUserPassword, s.checkUserToken)) // 修改已登录用户密码

	// ==============================
	//  全局依赖库管理
	// ==============================
	s.router.HandleFunc("GET /api/v1/c/libraries", Middleware(s.SearchLibraries, s.checkUserToken, s.checkAKSKAuth))                                 // 分页查询收录的库
	s.router.HandleFunc("POST /api/v1/c/libraries", Middleware(s.ManualAddPackage, s.checkUserToken, s.checkAKSKAuth))                               // 手动添加全局库
	s.router.HandleFunc("GET /api/v1/c/libraries/{package_id}", Middleware(s.GetPackageDetail, s.checkUserToken, s.checkAKSKAuth))                   // 同步库版本
	s.router.HandleFunc("GET /api/v1/c/libraries/{package_id}/versions", Middleware(s.GetPackageVersions, s.checkUserToken, s.checkAKSKAuth))        // 库版本列表
	s.router.HandleFunc("POST /api/v1/c/libraries/{package_id}/sync-versions", Middleware(s.SyncPackageVersions, s.checkUserToken, s.checkAKSKAuth)) // 同步库版本
	s.router.HandleFunc("GET /api/v1/c/libraries/search", Middleware(s.SearchLibraries, s.checkUserToken, s.checkAKSKAuth))                          // 搜索全局库
	s.router.HandleFunc("POST /api/v1/c/libraries/batch", Middleware(s.BatchImportPackage, s.checkUserToken, s.checkAKSKAuth))                       // 手动添加全局库
	s.router.HandleFunc("GET /api/v1/c/libraries/get/purl", Middleware(s.GetPackage, s.checkUserToken, s.checkAKSKAuth))

	// ==============================
	//  公共开源仓库
	// ==============================
	s.router.HandleFunc("GET /api/v1/c/public-repos", Middleware(s.ListPublicRepos, s.checkUserToken, s.checkAKSKAuth))                  // 公共仓库列表
	s.router.HandleFunc("POST /api/v1/c/public-repos/import", Middleware(s.ImportPublicRepo, s.checkUserToken, s.checkAKSKAuth))         // 用户也可以添加收录公共仓库
	s.router.HandleFunc("GET /api/v1/c/public-repos/{id}", Middleware(s.GetPublicRepo, s.checkUserToken, s.checkAKSKAuth))               // 公共仓库详情
	s.router.HandleFunc("GET /api/v1/c/public-repos/{id}/analysis", Middleware(s.AnalysisPublicRepo, s.checkUserToken, s.checkAKSKAuth)) // 公共仓库分析

	// ==============================
	//  仓库收藏
	// ==============================
	s.router.HandleFunc("POST /api/v1/c/public-repos/{id}/star", Middleware(s.StarRepo, s.checkUserToken, s.checkAKSKAuth))     // 收藏公共仓库
	s.router.HandleFunc("DELETE /api/v1/c/public-repos/{id}/star", Middleware(s.UnStarRepo, s.checkUserToken, s.checkAKSKAuth)) // 取消收藏
	s.router.HandleFunc("GET /api/v1/c/user/stars", Middleware(s.ListUserStarRepos, s.checkUserToken, s.checkAKSKAuth))         // 我的收藏

	// ==============================
	//  异步任务
	// ==============================
	s.router.HandleFunc("GET /api/v1/c/tasks/{id}", Middleware(s.GetBackgroundTask, s.checkUserToken, s.checkAKSKAuth))          // 任务详情
	s.router.HandleFunc("GET /api/v1/c/tasks", Middleware(s.ListBackgroundTasks, s.checkUserToken, s.checkAKSKAuth))             // 任务列表
	s.router.HandleFunc("POST /api/v1/c/tasks/{id}/retry", Middleware(s.RetryBackgroundTask, s.checkUserToken, s.checkAKSKAuth)) // 重试失败任务

	// ==============================
	//  系统配置
	// ==============================
	s.router.HandleFunc("GET /api/v1/c/setting", Middleware(s.GetSystemSetting, s.checkAdminToken, s.checkUserToken))         // 获取所有配置（仅管理员）
	s.router.HandleFunc("PUT /api/v1/c/setting", Middleware(s.UpdateSystemSetting, s.checkAdminToken, s.checkUserToken))      // 更新所有配置（仅管理员）
	s.router.HandleFunc("GET /api/v1/c/setting/basic", Middleware(s.GetBasicSetting, s.checkUserToken))                       // 获取基本配置
	s.router.HandleFunc("PUT /api/v1/c/setting/basic", Middleware(s.UpdateBasicSetting, s.checkAdminToken, s.checkUserToken)) // 更新基本配置（仅管理员）

	// ==============================
	//  用户个人信息（登录用户）
	// ==============================
	s.router.HandleFunc("PUT /api/v1/c/profile", Middleware(s.UpdateProfile, s.checkUserToken, s.checkAKSKAuth)) // 更新个人信息

}
