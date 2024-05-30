package user

import (
	"fmt"
	"github.com/kubackup/kubackup/internal/entity/v1/sysuser"
	"github.com/kubackup/kubackup/internal/service/v1/common"
	"github.com/kubackup/kubackup/pkg/utils"
	"time"
)

type Service interface {
	common.DBService
	Create(user *sysuser.SysUser, options common.DBOptions) error
	List(options common.DBOptions) ([]sysuser.SysUser, error)
	Get(id int, options common.DBOptions) (*sysuser.SysUser, error)
	GetByUserName(name string, options common.DBOptions) (*sysuser.SysUser, error)
	Delete(id int, options common.DBOptions) error
	DeleteStruct(user *sysuser.SysUser, options common.DBOptions) error
	Update(user *sysuser.SysUser, options common.DBOptions) error
	ClearOtp(username string, options common.DBOptions) error
	ClearPwd(username string, options common.DBOptions) error
	InitAdmin() error
}

func GetService() Service {
	return &Sysuser{
		DefaultDBService: common.DefaultDBService{},
	}
}

type Sysuser struct {
	common.DefaultDBService
}

func (s Sysuser) ClearPwd(username string, options common.DBOptions) error {
	user, err := s.GetByUserName(username, options)
	if err != nil {
		return err
	}
	pwd := utils.RandomString(8)
	fmt.Println("新密码:", pwd)
	encodePWD, err := utils.EncodePWD(pwd)
	if err != nil {
		return err
	}
	user.Password = encodePWD
	err = s.Update(user, options)
	if err != nil {
		return err
	}
	return nil
}

// ClearOtp 清理otp信息
func (s Sysuser) ClearOtp(username string, options common.DBOptions) error {
	user, err := s.GetByUserName(username, options)
	if err != nil {
		return err
	}
	user.OtpSecret = "1" //默认值
	user.OtpInterval = 0
	err = s.Update(user, options)
	if err != nil {
		return err
	}
	return nil
}

func (s Sysuser) DeleteStruct(user *sysuser.SysUser, options common.DBOptions) error {
	db := s.GetDB(options)
	return db.DeleteStruct(user)
}

func (s Sysuser) InitAdmin() error {
	user, err := s.GetByUserName("admin", common.DBOptions{})
	if err != nil && err.Error() != "not found" {
		return err
	}
	if user == nil {
		pwd := utils.RandomString(8)
		fmt.Println("初始用户: admin")
		fmt.Println("初始密码:", pwd)
		encodePWD, err := utils.EncodePWD(pwd)
		if err != nil {
			return err
		}
		admin := &sysuser.SysUser{
			Username: "admin",
			NickName: "Admin",
			Password: encodePWD,
		}
		err = s.Create(admin, common.DBOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s Sysuser) GetByUserName(name string, options common.DBOptions) (*sysuser.SysUser, error) {
	db := s.GetDB(options)
	var user sysuser.SysUser
	err := db.One("Username", name, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s Sysuser) Create(user *sysuser.SysUser, options common.DBOptions) error {
	db := s.GetDB(options)
	user.CreatedAt = time.Now()
	return db.Save(user)
}

func (s Sysuser) List(options common.DBOptions) ([]sysuser.SysUser, error) {
	db := s.GetDB(options)
	var users []sysuser.SysUser
	err := db.All(&users)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (s Sysuser) Get(id int, options common.DBOptions) (*sysuser.SysUser, error) {
	db := s.GetDB(options)
	var user sysuser.SysUser
	err := db.One("Id", id, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s Sysuser) Delete(id int, options common.DBOptions) error {
	db := s.GetDB(options)
	user, err := s.Get(id, options)
	if err != nil {
		return err
	}
	return db.DeleteStruct(user)
}

func (s Sysuser) Update(user *sysuser.SysUser, options common.DBOptions) error {
	db := s.GetDB(options)
	user.UpdatedAt = time.Now()
	return db.Update(user)
}
