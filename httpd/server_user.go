package httpd

import (
	"fmt"
	"strconv"

	"github.com/alayou/techstack/httpd/buserr"
	"github.com/alayou/techstack/httpd/dao"

	"github.com/alayou/techstack/model"
)

// signIn 登录
func (s *Server) signIn(username, password string, ttl int, multiClientLogin bool) (user *model.User, err error) {
	// 默认登录
	user = dao.User.FindByUsername(username)
	if user == nil {
		err = buserr.ErrFuncInvalidParams("usernameOrPassword")
		return
	}
	matched, err := user.PasswordEqual(password)
	if !matched || err != nil {
		err = buserr.ErrFuncInvalidParams("usernameOrPassword")
		return
	}

	//审批状态
	if user.Status == model.UserStatusInactivated {
		err = buserr.ErrUserApproving
		return
	}

	//禁用状态
	if user.Status == model.UserStatusDisabled {
		err = buserr.ErrUserDisabled
		return
	}

	err = s.createUserLoginToken(user, ttl)
	return
}

func (s *Server) createUserLoginToken(user *model.User, ttl int) (err error) {
	var token string
	token, _, err = s.sessionMgr.CreateAndCacheToken(user.ID.Int64(),
		ttl, user.Username, user.Nickname, true,
		user.GetRole())
	if err != nil {
		return
	}
	user.Token = token
	return
}

// signup 注册
func (s *Server) signup(username, password, email, phone, origin string, role string, status uint8) (*model.User, error) {
	// 创建基本信息
	user := &model.User{
		Username:  username,
		Password:  password,
		Email:     email,
		Phone:     phone,
		Status:    status,
		ChangePwd: model.NeedNoChangePwd,
	}

	err := dao.User.Create(user)
	if err != nil {
		return nil, err
	}

	// 如果如果启用了发信邮箱则发送一份激活邮件给用户
	if s.sMailService.Enabled() {
		token, _, err := s.sessionMgr.CreateAndCacheToken(int64(user.ID),
			3600*24, user.Username, user.Nickname, false, user.GetRole())
		if err != nil {
			return nil, err
		}

		return user, s.sMailService.NotifyActive(origin, user.Email, token)
	}

	return user, nil
}

// active 激活
func (s *Server) active(token string) error {
	rc, err := s.sessionMgr.Verify(token)
	if err != nil {
		return err
	}

	uid, _ := strconv.ParseInt(rc.Subject, 10, 64)
	user := dao.User.Find(uid)
	if user == nil {
		return buserr.ErrFuncNotExist("user")
	}

	if user.Status >= model.UserStatusOk {
		return fmt.Errorf("account already activated")
	}

	return dao.User.UpdatePatch(uid, dao.BodyUserPatch{
		Status: model.UserStatusOk, Fields: []string{"status"},
	})
}

// passwordReset 密码重置
func (s *Server) passwordReset(token, password string) error {
	rc, err := s.sessionMgr.Verify(token)
	if err != nil {
		return err
	}
	return dao.User.UpdatePatch(rc.GetUIDInt64(), dao.BodyUserPatch{
		Password: password,
		Fields:   []string{"password"},
	})
}
