package sessmgr

import (
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorpher/gone/authed"
	"github.com/gorpher/gone/codec"

	"github.com/alayou/techstack/model"
	"github.com/mileusna/useragent"

	"github.com/alayou/techstack/global"

	"github.com/rs/zerolog/log"
)

type UserSession authed.UserSession

const appName = global.AppName
const (
	cookieName = "techstack_session"
	haskKey    = "IhVvnF5N4bNHpaSS"
	blockKey   = "C09lWMVNF9YobaYn"
	cookieKey  = "cPWFm7zUMaa8bn03"
)
const (
	RoleUser = model.RoleUser // 根据实际情况调整，默认使用oauth2.0 认证的用户角色
)

type SessionMgr struct {
	authed      *authed.Authed
	cookieCodec codec.CryptoCodec
	cookieName  string
	cookieKey   []byte
}

var Store *SessionMgr

type OptFunc func(session *SessionMgr) *SessionMgr

// NewSessionMgr 新建会话管理，multiSession  是否多用户模式
func NewSessionMgr(opts ...authed.OptFunc) *SessionMgr {
	Store = &SessionMgr{
		authed:      authed.NewAuthed(opts...),
		cookieCodec: codec.NewCookieCodec([]byte(haskKey), []byte(blockKey)),
		cookieName:  cookieName,
		cookieKey:   []byte(cookieKey),
	}
	return Store
}

// GetAuthedSession 从HTTP请求中获取用户会话信息，如果没有则返回空
func (s *SessionMgr) GetAuthedSession(req *http.Request) *authed.UserSession {
	return s.authed.GetHTTPSession(req)
}

// CreateAndCacheToken 创建token并存储缓存
func (s *SessionMgr) CreateAndCacheToken(uid int64, ttl int,
	username, nickname string, refresh bool, roles ...string) (token string, refreshToken string, err error) {
	token, refreshToken, err = s.authed.CreateToken(&authed.UserSession{
		Uid:      strconv.FormatInt(uid, 10),
		Extends:  map[string]any{},
		Username: username,
		Nickname: nickname,
		Roles:    roles,
	})
	if err != nil {
		return
	}
	if !refresh {
		refreshToken = ""
	}
	return
}

// SetCookieToken 设置token信息到cookie上
func (s *SessionMgr) SetCookieToken(w http.ResponseWriter, value string, maxAge int) {
	s.authed.SetCookieToken(w, value, maxAge)
}

// SetCookie 设信息到cookie上
func (s *SessionMgr) SetCookie(w http.ResponseWriter, key, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     key,
		Value:    url.QueryEscape(value),
		Path:     "/",
		Domain:   "",
		Expires:  time.Now().Add(time.Duration(maxAge)).UTC(),
		MaxAge:   maxAge,
		Secure:   false,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// SetSessionValue 设置会话值到浏览器
func (s *SessionMgr) SetSessionValue(w http.ResponseWriter, k, valueBytes []byte, ttl time.Duration) (string, error) {
	value, err := s.cookieCodec.Encode(s.cookieKey, valueBytes)
	if err != nil {
		return "", err
	}
	s.SetCookie(w, string(k), string(value), int(ttl.Seconds()))
	return string(value), nil
}

// GetSessionValue 从浏览器获取会话值
func (s *SessionMgr) GetSessionValue(req *http.Request, k string) string {
	var value []byte
	value = []byte(req.Header.Get(k))
	if len(value) == 0 {
		c, err := req.Cookie(k)
		if err != nil {
			return ""
		}
		value = []byte(c.Value)
	}
	if len(value) == 0 {
		return ""
	}
	v, err := s.cookieCodec.Decode(s.cookieKey, value)
	if err != nil {
		return ""
	}
	return string(v)
}

// DeleteSessionValue 删除浏览器会话
func (s *SessionMgr) DeleteSessionValue(w http.ResponseWriter, k string) {
	s.SetCookie(w, k, "", 1)
}

// DeleteCacheToken 删除cache存储token
func (s *SessionMgr) DeleteCacheToken(id string) {
	err := s.authed.DeleteToken(id)
	if err != nil {
		log.Error().Err(err).Msg("删除token失败")
	}
}

func (s *SessionMgr) AuthorizationHandler() func(w http.ResponseWriter, r *http.Request) (userID string, err error) {
	return func(w http.ResponseWriter, r *http.Request) (userID string, err error) {
		userSession := s.GetAuthedSession(r)
		if userSession == nil {
			return "", errors.New("not Authorized")
		}
		return userSession.Uid, nil
	}
}

func (s *SessionMgr) Verify(token string) (authed.Payload, error) {
	return s.authed.VerifyToken(token)

}

func GetUserAgent(r *http.Request) useragent.UserAgent {
	uaStr := r.Header.Get("User-Agent")
	if uaStr != "" {
		ua := useragent.Parse(uaStr)
		return ua
	}
	return useragent.UserAgent{
		OS: useragent.Windows,
	}
}

type Platform string

const (
	aix     = "aix"
	android = "android"
	darwin  = "darwin"
	freebsd = "freebsd"
	haiku   = "haiku"
	linux   = "linux"
	openbsd = "openbsd"
	sunos   = "sunos"
	win32   = "win32"
	cygwin  = "cygwin"
	netbsd  = "netbsd"
)

var PlatformMap = map[string]string{ //nolint
	aix:     "aix",
	android: "android",
	darwin:  "darwin",
	freebsd: "freebsd",
	haiku:   "haiku",
	linux:   "linux",
	openbsd: "openbsd",
	sunos:   "sunos",
	win32:   "win32",
	cygwin:  "cygwin",
	netbsd:  "netbsd",
} //nolint

type Architecture string

const (
	arm    = "arm"
	arm64  = "arm64"
	ia32   = "ia32"
	mips   = "mips"
	mipsel = "mipsel"
	pc     = "ppc"
	pcc64  = "ppc64"
	s390   = "s390"
	s390x  = "s390x"
	x64    = "x64"
)

var ArchitectureMap = map[string]string{ //nolint
	arm:    "arm",
	arm64:  "arm64",
	ia32:   "ia32",
	mips:   "mips",
	mipsel: "mipsel",
	pc:     "ppc",
	pcc64:  "ppc64",
	s390:   "s390",
	s390x:  "s390x",
	x64:    "x64",
} //nolint

// GetOrigin 获取客户端origin
func GetOrigin(r *http.Request) string {
	scheme := "http"
	host := r.Host
	forwardedHost := r.Header.Get("X-Forwarded-Host")
	if forwardedHost != "" {
		host = forwardedHost
	}
	forwardedProto := r.Header.Get("X-Forwarded-Proto")
	if forwardedProto == "https" {
		scheme = forwardedProto
	}

	return fmt.Sprintf("%s://%s", scheme, host)
}
