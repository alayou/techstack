package httpd

import (
	"unicode"

	"github.com/alayou/techstack/httpd/dao"

	"github.com/alayou/techstack/httpd/httputil"

	"github.com/alayou/techstack/model"
)

// GetUserMaxOnlineTime 获取配置中账号最大在线时长，返回单位秒
func (s *Server) GetUserMaxOnlineTime() int {
	opts, err := dao.Setting.Get(string(model.SettingBasic))
	if err != nil {
		return model.TTL
	}
	return opts.GetInt(model.SettingBasicAccountMaxOnlineTime) * 60
}

// 1. 设置密码规则有，最小长度默认最小长度6个字符，字符不能小于6个，默认最大密码长度（20）
// 2. 设置包含：大写字母、小写字母、数字和字符（默认都不开启）
// 3. 开启限制后，用户注册必须按照规范注册用户
// 4. 已经注册过的用户，登录时不按照密码规则校验
// 5. 密码特殊字符包括： !@#$%^&.
// 6. 默认密码不开启验证时，密码必须属于写字母、小写字母、数字和字符中一种或多种组合
func (s *Server) isOkPassword(password string) bool {
	if len(password) < 6 || len(password) > 20 {
		return false
	}
	opt, err := dao.Setting.Get(string(model.SettingBasic))
	if err != nil {
		return false
	}
	// 密码特殊字符 !@#$%^&
	minLength := opt.GetInt(model.SettingBasicPasswordRuleMinLength)
	if len(password) < minLength {
		return false
	}
	flag := opt.GetInt(model.SettingBasicPasswordRuleIncludeStr)
	if flag <= 0 {
		return true
	}

	var digit, upper, lower, special bool
	for _, r := range password {
		if unicode.IsDigit(r) {
			digit = true
		}
		if unicode.IsUpper(r) {
			upper = true
		}
		if unicode.IsLower(r) {
			lower = true
		}
		if httputil.IsPasswordSpecialLetter(r) {
			special = true
		}
	}

	if flag&httputil.PasswordUpper != 0 && !upper {
		return false
	}

	if flag&httputil.PasswordLower != 0 && !lower {
		return false
	}

	if flag&httputil.PasswordDigit != 0 && !digit {
		return false
	}
	if flag&httputil.PasswordSpecial != 0 && !special {
		return false
	}
	return true
}
