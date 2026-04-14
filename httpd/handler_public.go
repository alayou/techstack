package httpd

import (
	"net/http"
	"time"

	"github.com/alayou/techstack/httpd/bind"
	"github.com/alayou/techstack/httpd/buserr"
	"github.com/alayou/techstack/httpd/dao"
	. "github.com/alayou/techstack/httpd/httputil"
	"github.com/alayou/techstack/model"
)

// Captcha 验证码
func (s *Server) Captcha(w http.ResponseWriter, req *http.Request) {
	var in bind.BodyCaptcha
	if err := ShouldJson(req, &in); err != nil {
		Bad(w)
		return
	}

	Ok(w)
}

// Login 登录
func (s *Server) Login(w http.ResponseWriter, req *http.Request) {
	var in bind.BodyLogin
	if err := ShouldJson(req, &in); err != nil {
		Bad(w, err)
		return
	}
	expireSec := s.GetUserMaxOnlineTime()
	user, err := s.signIn(in.Username, in.Password, expireSec, true)
	if err != nil {
		Bad(w, err)
		return
	}
	s.sessionMgr.SetCookieToken(w, user.Token, expireSec)

	//更新最后登录时间
	user.LastLoginAt = time.Now().Unix()
	err = dao.User.UpdatePatch(user.ID.Int64(), dao.BodyUserPatch{
		LastLoginAt: user.LastLoginAt,
		Fields:      []string{"last_login_at"},
	})
	if err != nil {
		Bad(w, err)
		return
	}

	Ok(w, JsonRawBody{
		"token":      user.Token,
		"expired":    expireSec,
		"change_pwd": user.ChangePwd,
		"user":       user,
	})
}

// Logout 登出
func (s *Server) Logout(w http.ResponseWriter, req *http.Request) {
	session := s.sessionMgr.GetAuthedSession(req)
	if session == nil {
		Ok(w)
		return
	}
	if session.ID != "" {
		s.sessionMgr.DeleteCacheToken(session.ID)
	}

	s.sessionMgr.SetCookieToken(w, "", 1)
	s.sessionMgr.DeleteSessionValue(w, "x-token")

	user := dao.User.Find(session.GetUIDInt64())
	if user == nil {
		Bad(w, buserr.ErrFuncNotExist("user"))
		return
	}

	Ok(w, map[string]any{
		"status": "ok",
	})
}

// Signup 注册
func (s *Server) Signup(w http.ResponseWriter, req *http.Request) {
	opts, err := dao.Setting.Get(string(model.SettingBasic))
	if err != nil {
		Bad(w, err)
		return

	}
	if !opts.GetBool(model.SettingBasicRegisterSwitch) {
		Bad(w, buserr.ErrSystemRegistryClose)
		return
	}

	var in bind.BodyUserCreation
	if err = ShouldJson(req, &in); err != nil {
		Bad(w, err)
		return
	}

	if !IsUserName(in.Username) {
		Bad(w, buserr.ErrFuncInvalidParams("username"))
		return
	}

	if !IsNickname(in.Nickname) {
		Bad(w, buserr.ErrFuncInvalidParams("nickname"))
		return
	}
	password := in.Password

	if !IsPassword(password, s.isOkPassword) {
		Bad(w, buserr.ErrFuncInvalidParams("password"))
		return
	}

	status := model.UserStatusOk
	//开启注册审批后用户状态为未激活
	if opts.GetBool(model.SettingBasicRegisterApprovalSwitch) {
		status = model.UserStatusInactivated
	}

	// 注册成功后直接登录
	var user *model.User
	user, err = s.signup(in.Username, password, in.Email, in.Phone,
		GetOrigin(req),
		model.RoleUser, uint8(status))
	if err != nil {
		Bad(w, err)
		return
	}

	expireSec := s.GetUserMaxOnlineTime()
	user, err = s.signIn(user.Username, in.Password, expireSec, true)
	if err != nil {
		Bad(w, err)
		return
	}
	s.sessionMgr.SetCookieToken(w, user.Token, expireSec)

	//更新最后登录时间
	user.LastLoginAt = time.Now().Unix()
	err = dao.User.UpdatePatch(user.ID.Int64(), dao.BodyUserPatch{
		LastLoginAt: user.LastLoginAt,
		Fields:      []string{"last_login_at"},
	})
	if err != nil {
		Bad(w, err)
		return
	}

	Ok(w, JsonRawBody{
		"token":      user.Token,
		"expired":    expireSec,
		"change_pwd": user.ChangePwd,
	})
}

// PublicSys 系统信息
func (s *Server) PublicSys(w http.ResponseWriter, req *http.Request) {
	var (
		err  error
		opts model.Opts
	)
	if opts, err = dao.Setting.Get(string(model.SettingBasic)); err != nil {
		Bad(w, err)
		return
	}
	Ok(w, opts)
}

// ResetUser 激活用户或者重制密码
func (s *Server) ResetUser(w http.ResponseWriter, req *http.Request) {
	var in bind.BodyUserReset
	if err := ShouldJson(req, &in); err != nil {
		Bad(w, err)
		return
	}

	// account activate
	if in.Activated {
		if err := s.active(in.Token); err != nil {
			Bad(w, err)
			return
		}
		Ok(w)
		return
	}

	// password reset
	if in.Password != "" {
		if err := s.passwordReset(in.Token, in.Password); err != nil {
			Bad(w, err)
			return
		}
	}

	Ok(w)
}
