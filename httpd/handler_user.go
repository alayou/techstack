package httpd

import (
	"encoding/json"
	"net/http"

	"github.com/alayou/techstack/httpd/bind"
	"github.com/alayou/techstack/httpd/buserr"
	"github.com/alayou/techstack/httpd/dao"
	. "github.com/alayou/techstack/httpd/httputil"
	"github.com/alayou/techstack/model"
)

// Profile 获取当前用户信息
func (s *Server) Profile(w http.ResponseWriter, req *http.Request) {
	uid := UidGet(req)
	user := dao.User.Find(uid)
	if user == nil {
		BadError(w, 404, "用户不存在")
		return
	}
	Ok(w, user)
}

// UpdateProfile 更新当前用户信息
func (s *Server) UpdateProfile(w http.ResponseWriter, req *http.Request) {
	uid := UidGet(req)

	var body bind.BodyUserUpdateProfile
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		BadError(w, 400, "无效的请求参数")
		return
	}

	user := dao.User.Find(uid)
	if user == nil {
		BadError(w, 404, "用户不存在")
		return
	}

	// 更新用户信息
	if err := dao.User.UpdateByID(uid, bind.BodyUserUpdate{
		Email:    body.Email,
		Phone:    body.Phone,
		Nickname: body.Nickname,
	}); err != nil {
		BadError(w, 500, "更新失败")
		return
	}

	// 重新获取更新后的用户信息
	user = dao.User.Find(uid)
	Ok(w, user)
}

// UpdateUserPassword 修改当前用户密码
func (s *Server) UpdateUserPassword(w http.ResponseWriter, req *http.Request) {
	uid := UidGet(req)

	var body bind.BodyUserChangePassword
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		BadError(w, 400, "无效的请求参数")
		return
	}

	if body.Password == "" {
		BadError(w, 400, "新密码不能为空")
		return
	}

	user := dao.User.Find(uid)
	if user == nil {
		BadError(w, 404, "用户不存在")
		return
	}

	matched, err := user.PasswordEqual(body.OldPassword)
	if !matched || err != nil {
		err = buserr.ErrFuncInvalidParams("PasswordNotEqual")
		Bad(w, err)
		return
	}

	// 更新密码
	if err := dao.User.UpdateByID(uid, bind.BodyUserUpdate{Password: body.Password}); err != nil {
		BadError(w, 500, "密码更新失败")
		return
	}
	s.Logout(w, req)
}

// GetSystemSetting 获取系统配置（仅管理员）
func (s *Server) GetSystemSetting(w http.ResponseWriter, req *http.Request) {
	// 获取所有配置
	settings, err := dao.Setting.FindSettings()
	if err != nil {
		BadError(w, 500, "获取配置失败")
		return
	}

	// 构建响应数据
	result := make(map[string]interface{})
	for _, setting := range settings {
		result[setting.Name] = setting.Opts
	}

	Ok(w, result)
}

// UpdateSystemSetting 更新系统配置（仅管理员）
func (s *Server) UpdateSystemSetting(w http.ResponseWriter, req *http.Request) {
	var body map[string]interface{}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		BadError(w, 400, "无效的请求参数")
		return
	}

	// 更新每个配置项
	for name, optsRaw := range body {
		opts, ok := optsRaw.(map[string]interface{})
		if !ok {
			continue
		}

		// 转换为 model.Opts
		modelOpts := make(model.Opts)
		for k, v := range opts {
			modelOpts[k] = v
		}

		if err := dao.Setting.Set(name, modelOpts); err != nil {
			BadError(w, 500, "更新配置失败: "+name)
			return
		}
	}

	Ok(w, "配置更新成功")
}

// GetBasicSetting 获取基本配置
func (s *Server) GetBasicSetting(w http.ResponseWriter, req *http.Request) {
	opts, err := dao.Setting.Get(string(model.SettingBasic))
	if err != nil {
		BadError(w, 500, "获取配置失败")
		return
	}
	Ok(w, opts)
}

// UpdateBasicSetting 更新基本配置
func (s *Server) UpdateBasicSetting(w http.ResponseWriter, req *http.Request) {
	var body model.Opts
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		BadError(w, 400, "无效的请求参数")
		return
	}

	if err := dao.Setting.Set(string(model.SettingBasic), body); err != nil {
		BadError(w, 500, "更新配置失败")
		return
	}

	Ok(w, "配置更新成功")
}
