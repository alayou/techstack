package dao

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/alayou/techstack/global"
	"github.com/alayou/techstack/httpd/bind"
	"github.com/alayou/techstack/httpd/buserr"
	"github.com/alayou/techstack/model"
	"github.com/alayou/techstack/utils"
	argon2id "github.com/alexedwards/argon2id"
	"github.com/gorpher/gone/osutil"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type dUser struct {
}

var User = NewUser()

func NewUser() *dUser {
	return &dUser{}
}

func (u *dUser) Find(uid int64) *model.User {
	return u.FindByID(uid)
}

func (u *dUser) FindByID(id int64) *model.User {
	user := new(model.User)
	err := gdb.First(user, id).Error
	if err != nil {
		return nil
	}
	//gdb.Model(&model.UserGroup{}).Select("name").First(&user.GroupName, "id=?", user.GroupID) //nolint
	//gdb.Model(&model.LogUser{}).Select("ip").
	//	Where("uid =? and opt_type=?", user.ID, model.UserActionLogin).
	//	Order("created_at desc").Limit(1).First(&user.LastLoginIP) // nolint
	return user
}
func (u *dUser) FindUsernameByID(id int64) (username string) {
	gdb.Model(&model.User{}).Select("username").First(&username, "id=?", id) //nolint
	return username
}

func (u *dUser) FindByUsername(username string) *model.User {
	user := new(model.User)
	err := gdb.First(user, "username=?", username).Error
	if err != nil {
		return nil
	}
	return user
}

func (u *dUser) FindByAccountKey(accountKey string) *model.User {
	user := new(model.User)
	err := gdb.First(user, "account_key=?", accountKey).Error
	if err != nil {
		return nil
	}
	return user
}

func (u *dUser) FindExistUser(username ...string) (list []string, err error) {
	list = make([]string, 0)
	err = gdb.Model(&model.User{}).Select("username").Find(&list, "username in (?)", username).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return list, err
	}
	return list, nil
}

func (u *dUser) Create(user *model.User) error {
	if us := u.FindByUsername(user.Username); us != nil {
		return buserr.ErrFuncNotExist("user")
	}
	if user.Nickname == "" {
		user.Nickname = user.Username

	}
	hash := utils.Sha256Hash([]byte(user.Username))
	user.AccountKey = osutil.ID.UUID4().String()
	user.AccountSecret = hash[:6] + "-" + osutil.ID.XString()
	err := CreateUserPasswordHash(user)
	if err != nil {
		return err
	}
	return gdb.Create(user).Error
}

type BodyBatchUserStatus struct {
	ID     int64 `json:"id,string" binding:"required"`
	Status uint8 `json:"status" binding:"required,gte=1,lte=3"`
}

func (u *dUser) UpdateBatchStatus(users ...BodyBatchUserStatus) error {
	tx := gdb.Begin()
	for _, user := range users {
		updates := map[string]interface{}{"status": user.Status}
		if user.Status == model.UserStatusDisabled {
			updates["disabled_at"] = time.Now().Unix()
		}
		err := tx.Model(model.User{}).Where("id=?", user.ID).Updates(updates).Error
		if err != nil {
			return err
		}
	}
	return tx.Commit().Error
}

// Delete 删除单个用户
func (u *dUser) Delete(user *model.User) (err error) {
	return gdb.Transaction(func(tx *gorm.DB) error {
		err = tx.Delete(user).Error
		if err != nil {
			return err
		}
		return nil
	})
}

// DeleteBatchUser  批量删除用户
func (u *dUser) DeleteBatchUser(ids ...int64) (err error) {
	return gdb.Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			err = tx.Where("id=?", id).Delete(&model.User{}).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (u *dUser) GetAllUserQuotaTotalExceptUid(uid int64) (total uint64, err error) {
	err = gdb.Model(&model.User{}).Where("id !=?", uid).Pluck("sum(quota)", &total).Error
	return
}

func (u *dUser) GetAllUserQuotaTotal() (total uint64, err error) {
	err = gdb.Model(&model.User{}).Where("roles!=?", model.RoleAdmin).Pluck("sum(quota)", &total).Error
	return
}

// FindAllDeletedUsers 得到所有已经删除的用户
func (u *dUser) FindAllDeletedUsers() (deletedUsers []model.User, err error) {
	err = gdb.Model(&model.User{}).Unscoped().Find(&deletedUsers, "deleted_at>0").Error
	return
}

// FindUsernameMapByIds  得到用户id对应用户名信息
func (u *dUser) FindUsernameMapByIds(ids []int64) (maps map[int64]string, err error) {
	var users []model.User
	maps = make(map[int64]string, 0)
	err = gdb.Model(&model.User{}).Find(&users, ids).Error
	if err != nil {
		return
	}
	for _, user := range users {
		maps[int64(user.ID)] = user.Username
	}
	return
}

// Count 用户数量
func (u *dUser) Count() (count int64, err error) {
	err = gdb.Model(&model.User{}).Count(&count).Error
	return
}

// CountUsersWithWhere 用户数量
func (u *dUser) CountUsersWithWhere(wh string) (count int64, err error) {
	err = gdb.Model(&model.User{}).Where(wh).Count(&count).Error
	return
}

// FindUsersMapByIds  得到用户id对应用户名信息
func (u *dUser) FindUsersMapByIds(ids []int64) (maps map[int64]model.User, err error) {
	var users []model.User
	maps = make(map[int64]model.User, 0)
	err = gdb.Model(&model.User{}).Find(&users, ids).Error
	if err != nil {
		return
	}
	for _, user := range users {
		maps[int64(user.ID)] = user
	}
	return
}

func (u *dUser) UpdateByID(id int64, in bind.BodyUserUpdate) error {
	return gdb.Transaction(func(tx *gorm.DB) error {
		var values = make(map[string]any, 4)

		if in.Email != "" {
			values["email"] = in.Email
		}
		if in.Phone != "" {
			values["phone"] = in.Phone
		}
		if in.Nickname != "" {
			values["nickname"] = in.Nickname
		}
		if in.Password != "" {
			password, err := hashPlainPassword(in.Password)
			if err != nil {
				return err
			}
			values["password"] = password
		}
		return tx.Model(&model.User{}).Where("id=?", id).Updates(values).Error
	})
}

func (u *dUser) DeleteByID(id int64) error {
	return gdb.Transaction(func(tx *gorm.DB) error {
		return tx.Where("id=?", id).Delete(&model.User{}).Error
	})
}

type BodyUserPatch struct {
	Password    string   `json:"password"`
	Status      uint8    `json:"status"`
	LastLoginAt int64    `json:"last_login_at"`
	Nickname    string   `json:"nickname"`
	Email       string   `json:"email"`
	Phone       string   `json:"phone"`
	Fields      []string `json:"fields"`
}

func (p *BodyUserPatch) Size() int {
	// 可更新字段数
	return 6
}

func (u *dUser) UpdatePatch(id int64, in BodyUserPatch) error {
	return gdb.Transaction(func(tx *gorm.DB) error {
		var values = make(map[string]any, in.Size())
		if utils.In("password", in.Fields) {
			if in.Password != "" {
				password, err := hashPlainPassword(in.Password)
				if err != nil {
					return err
				}
				values["password"] = password
				values["change_pwd"] = model.NeedNoChangePwd
			}
		}
		if utils.In("status", in.Fields) {
			values["status"] = in.Status
			if in.Status == model.UserStatusDisabled {
				values["disabled_at"] = time.Now().Unix()
			}
		}
		if utils.In("last_login_at", in.Fields) {
			values["last_login_at"] = in.LastLoginAt
		}
		if utils.In("nickname", in.Fields) {
			values["nickname"] = in.Nickname
		}
		if utils.In("email", in.Fields) {
			values["email"] = in.Email
		}
		if utils.In("phone", in.Fields) {
			values["phone"] = in.Phone
		}
		if len(values) == 0 {
			return nil
		}
		return tx.Model(&model.User{}).Where("id=?", id).Updates(values).Error
	})
}

func (u *dUser) GetAllUserIds() (li []int64) {
	gdb.Model(&model.User{}).Select("id").Find(&li, "deleted_at=0") //nolint
	return
}

func (u *dUser) UpdateMoney(id int64, money float64) error {
	return gdb.Raw("update user set money=money+? where id=?", money, id).Error
	//return gdb.Model(&model.User{}).Where("id=?", id).Update("money", money).Error
}

func CreateUserPasswordHash(user *model.User) error {
	if user.Password != "" && !user.IsPasswordHashed() {
		hashedPwd, err := hashPlainPassword(user.Password)
		if err != nil {
			return err
		}
		user.Password = hashedPwd
	}
	return nil
}

var (
	argon2Params         *argon2id.Params
	initArgon2ParamsOnce sync.Once
)

func initializeHashingAlgo() {
	initArgon2ParamsOnce.Do(func() {
		parallelism := global.Config.PasswordHashing.Argon2Options.Parallelism
		if parallelism == 0 {
			parallelism = uint8(runtime.NumCPU())
		}
		argon2Params = &argon2id.Params{
			Memory:      global.Config.PasswordHashing.Argon2Options.Memory,
			Iterations:  global.Config.PasswordHashing.Argon2Options.Iterations,
			Parallelism: parallelism,
			SaltLength:  16,
			KeyLength:   32,
		}

	})
}

func hashPlainPassword(plainPwd string) (string, error) {
	initializeHashingAlgo()
	if global.Config.PasswordHashing.Algo == global.HashingAlgoBcrypt {
		pwd, err := bcrypt.GenerateFromPassword([]byte(plainPwd), global.Config.PasswordHashing.BcryptOptions.Cost)
		if err != nil {
			return "", fmt.Errorf("bcrypt hashing error: %w", err)
		}
		return string(pwd), nil
	}
	pwd, err := argon2id.CreateHash(plainPwd, argon2Params)
	if err != nil {
		return "", fmt.Errorf("argon2ID hashing error: %w", err)
	}
	return pwd, nil
}
