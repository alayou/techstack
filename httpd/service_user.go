package httpd

import "github.com/alayou/techstack/httpd/sessmgr"

// GetUserInfoFunc 获取用户信息
func (s *Server) GetUserInfoFunc(uid int64) *sessmgr.UserSession {
	//var (
	//	user *model.User
	//errv error
	//)
	//user, errv = s.dUser.Find(uid)
	//if errv != nil {
	//	return nil
	//}
	return &sessmgr.UserSession{
		//Kid: user.Kid,
		//Username: user.Username,
		//NickName: user.Profile.Nickname,
		//Roles:    user.RolesSplit(),
	}
}
