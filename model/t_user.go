package model

import (
	"github.com/alayou/techstack/utils"
	"gorm.io/plugin/soft_delete"
)

// 用户角色
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

const (
	NeedChangePwd   = iota + 1 // 需要修改密码
	NeedNoChangePwd            // 不需要修改密码
)

const (
	TTL                 = 7 * 24 * 3600 //登录时长,单位（秒）
	DefaultUserPassword = "123456"
)

// 用户状态
const (
	UserStatusUnapproved  uint8 = iota + 1 // 未审批
	UserStatusOk                           // 正常
	UserStatusDisabled                     // 已禁用
	UserStatusInactivated                  // 未激活
)

var Status = map[uint8]string{
	UserStatusUnapproved:  "未审批",
	UserStatusOk:          "正常",
	UserStatusDisabled:    "已禁用",
	UserStatusInactivated: "未激活",
}

// User 用户表
type User struct {
	Username string `json:"username" gorm:"size:64;uniqueIndex:idx_username;not null"` // 用户名
	Nickname string `json:"nickname"`                                                  // 昵称
	Password string `json:"-"  gorm:"size:255;not null" `                              // 密码
	Email    string `json:"email,omitempty"`                                           // 邮箱
	Phone    string `json:"phone"  `                                                   // 电话
	Avatar   string `json:"avatar,omitempty" gorm:"size:255;not null"`                 // 头像

	ID            ID                    `json:"id,string,omitempty" gorm:"primaryKey;autoIncrement:false"`
	CreatedAt     int64                 `json:"created_at" gorm:"autoCreateTime;not null"` // 更新时间
	UpdatedAt     int64                 `json:"updated_at" gorm:"autoUpdateTime;not null"` // 更新时间
	DeletedAt     soft_delete.DeletedAt `json:"-" gorm:"uniqueIndex:idx_username"`         // 删除时间
	DisabledAt    int64                 `json:"disabled_at" `                              // 禁用时间
	LastLoginAt   int64                 `json:"last_login_at" gorm:"default:0;not null"`   // 最后登录时间
	ChangePwd     uint8                 `json:"change_pwd" gorm:"default:2;not null"`      // 是否需要修改密码，1需要修改，2不需要修改密码
	Status        uint8                 `json:"status" gorm:"size:1;default:1;not null"`   // 状态 1 未审批 2 正常 3 禁用 4 未激活
	Role          string                `json:"role"`
	Token         string                `json:"-" gorm:"-"`
	AccountKey    string                `json:"account_key" gorm:"not null;uniqueIndex:idx_user_account_key"` // 账户密钥
	AccountSecret string                `json:"account_secret" gorm:"not null"`                               // 账户密钥
	LastLoginIP   string                `json:"last_login_ip" gorm:"->;-:migration"`
}

func (User) TableName() string {
	return UserTableName
}

func (u User) GetRole() string {
	return RoleUser
}

// IsPasswordHashed returns true if the password is hashed
func (u *User) IsPasswordHashed() bool {
	return utils.IsStringPrefixInSlice(u.Password, hashPwdPrefixes)
}
