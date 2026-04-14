package bind

type BodyCaptcha struct {
	Action uint8 `json:"action"` // 1 登录 2 注册
}
type BodyLogin struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
	Captcha  string `json:"captcha"`
}

type BodyUserUpdate struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Nickname string `json:"nickname"`
	Password string `json:"password"`
}
type BodyUserCreation struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Password string `json:"password"`
}

type BodyUserReset struct {
	Activated bool   `json:"activated"`
	Token     string `json:"token"`
	Password  string `json:"password"`
}

type BodyUserChangePassword struct {
	OldPassword string `json:"old_password"`
	Password    string `json:"password"`
}

type BodyUserUpdateProfile struct {
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
}
