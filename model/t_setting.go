package model

type SettingKeyType string

const (
	SettingBasic     SettingKeyType = "sys.basic" // 基本设置
	IntegrationEmail SettingKeyType = "ig.email"  // 集成邮件

)
const DefaultAdmin = "admin"
const DefaultAdminPwd = "techstack"

var SettingsList = []SettingKeyType{SettingBasic, IntegrationEmail}
var SettingsListStr []string

func init() {
	SettingsListStr = make([]string, len(SettingsList))
	for i, v := range SettingsList {
		SettingsListStr[i] = string(v)
	}
}

const (
	SettingBasicRegisterApprovalSwitch = "register_approval_switch"  // 注册审批开关
	SettingBasicLoginCaptchaSwitch     = "login_captcha_switch"      // 登录验证码开关
	SettingBasicLoginCaptchaTTL        = "login_captcha_ttl"         // 登录验证码有效期
	SettingBasicRegisterSwitch         = "register_switch"           // 注册开关
	SettingBasicPasswordRuleMinLength  = "password_rule_min_length"  // 密码规则，最小长度字符
	SettingBasicPasswordRuleIncludeStr = "password_rule_include_str" // 密码规则，包含字符
	SettingBasicAccountMaxOnlineTime   = "account_max_online_time"   // 账号最大在线时间
)

const (
	SettingStorageDownloadApproveSwitch = "file_download_approval_switch"
)

var (
	DefaultBasicOpts = Opts{
		"name":                             "techstack",
		"intro":                            "techstack",
		"locale":                           "zh-CN", // 语言中文
		"logo":                             "",      // logo
		SettingBasicRegisterApprovalSwitch: true,    // 注册审批
		SettingBasicLoginCaptchaSwitch:     false,   // 登录验证码开关
		SettingBasicLoginCaptchaTTL:        3600,    // 登录验证码超时时间 // 1 小时
		SettingBasicRegisterSwitch:         false,   // 开启注册开关
		SettingBasicPasswordRuleMinLength:  6,       // 密码规则，最小长度字符
		SettingBasicPasswordRuleIncludeStr: 0,       // 密码规划，1大写，2小写，4数字，8符号  符号包括： !@#$%^&
		SettingBasicAccountMaxOnlineTime:   1440,    // 最大在线时长，分钟
	}

	DefaultEmailOpts = Opts{
		"address":  "smtpdm.aliyun.com:25", // 邮件地址
		"username": "no-reply@saltbo.fun",  // 用户名
		"password": "yourpassword",         // 密码
		"sender":   "techstack",            // 发送者
	}
)

type Setting struct {
	ID        ID     `json:"id,string" gorm:"primaryKey"`
	Name      string `json:"name" gorm:"size:64;not null;uniqueIndex:setting_unique_idx"`
	Opts      Opts   `json:"opts" gorm:"type:text;not null"`
	CreatedAt int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (Setting) TableName() string {
	return SettingTableName
}
