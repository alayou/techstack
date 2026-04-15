package httpd

import (
	"context"
	"embed"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorpher/gone/cache"
	"github.com/rs/zerolog/log"

	"github.com/alayou/techstack/global"
	"github.com/alayou/techstack/httpd/dao"
	"github.com/alayou/techstack/httpd/sessmgr"
	"github.com/alayou/techstack/model"
	"github.com/alayou/techstack/utils"
	"github.com/gorpher/gone/authed"
)

const logSender = "httpd"

type Server struct {
	ctx           context.Context
	cache         cache.Cache
	router        *AdapterRouter
	sMailService  *MailService
	sessionMgr    *sessmgr.SessionMgr
	webFS         embed.FS
	webFsMapFiles map[string][]byte
}

func NewServer(ctx context.Context, cache cache.Cache, webFS embed.FS) *Server {
	s := &Server{
		ctx:           ctx,
		router:        NewRouter(checkCors),
		webFS:         webFS,
		webFsMapFiles: utils.EmbedFS2Files(webFS),
	}
	s.cache = cache
	s.sessionMgr = sessmgr.NewSessionMgr(
		authed.WithCryptoKey([]byte(global.Config.SessionHashKey)),
		authed.WithCache(s.cache),
		authed.WithMultiSession(),
		authed.WithCookieCode([]byte(global.Config.SessionHashKey), []byte(global.Config.SessionCookieKey)),
	)
	err := s.init(global.Config.Database.Driver, global.Config.Database.DSN) //nolint
	if err != nil {
		log.Fatal().Err(err).Msg("初始化系统失败")
	}
	return s
}

func (s *Server) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/" {
		req.URL.Path = "/index.html"
	}
	path := strings.TrimPrefix(req.URL.Path, "/")
	for p, body := range s.webFsMapFiles {
		if p == "web/"+path {
			if strings.HasSuffix(p, ".html") {
				rw.Header().Set("Content-Type", "text/html; charset=utf-8")
			}
			if strings.HasSuffix(p, ".css") {
				rw.Header().Set("Content-Type", "text/css; charset=utf-8")
			}
			if strings.HasSuffix(p, ".js") {
				rw.Header().Set("Content-Type", "text/javascript; charset=utf-8")
			}
			if strings.HasSuffix(p, ".ico") {
				rw.Header().Set("Content-Type", "image/x-icon")
			}
			if strings.HasSuffix(p, ".png") {
				rw.Header().Set("Content-Type", "image/png")
			}
			if strings.HasSuffix(p, ".svg") {
				rw.Header().Set("Content-Type", "image/svg+xml")
			}
			rw.Header().Set("Content-Length", strconv.Itoa(len(body)))
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write(body)
			return
		}
	}
	s.router.ServeHTTP(rw, req)
}

// Init 初始化数据库连接，如果是第一次安装，则初始化默认配置
func (s *Server) init(driver, dsn string) (err error) {
	if err = dao.Init(driver, dsn); err != nil {
		return err
	}
	s.initSettings()
	s.initAdmin()
	s.initRouter()
	return nil
}

// initSettings  初始化配置表
func (s *Server) initSettings() {
	dao.Setting.OptRegister(string(model.SettingBasic), model.DefaultBasicOpts, nil)
	dao.Setting.OptRegister(string(model.IntegrationEmail), model.DefaultEmailOpts, nil)
}

// initAdmin  初始化管理员
func (s *Server) initAdmin() {
	username := model.DefaultAdmin
	admin := dao.User.FindByUsername(username)
	if admin != nil && admin.ID > 0 {
		return
	}
	user := &model.User{
		Username:  username,
		Nickname:  "超级管理员",
		Password:  model.DefaultAdminPwd,
		Role:      model.RoleAdmin,
		Status:    model.UserStatusOk,
		ChangePwd: model.NeedNoChangePwd,
	}

	err := dao.User.Create(user)
	if err != nil {
		log.Fatal().Msg("初始化管理员失败")
	}
}
