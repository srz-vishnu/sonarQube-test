package internal

import (
	"fmt"
	"sonartest_cart/app/dto"
	"time"

	"gorm.io/gorm"
)

type UserRepo interface {
	SaveUserDetails(args *dto.UserDetailSaveRequest) (int64, error)
	GetUserByUsername(username string) (*Userdetail, error)
	IsUserActive(userID int64) (bool, error)
}

type UserRepoImpl struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) UserRepo {
	return &UserRepoImpl{
		db: db,
	}
}

type Userdetail struct {
	ID          int64     `gorm:"primaryKey"`
	Username    string    `gorm:"column:username;unique;not null"`
	Password    string    `gorm:"column:password;not null"`
	Address     string    `gorm:"column:address;not null"`
	Pincode     int64     `gorm:"column:pincode;not null"`
	Phonenumber int64     `gorm:"column:phone_number; not null"`
	Mail        string    `gorm:"column:mail;not null"`
	Status      bool      `gorm:"column:status;default:true;not null"` // Boolean field, default true
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`
	IsAdmin     bool      `gorm:"column:isadmin;default:false;not null"` // default false for user, true is used when  admin logins
}

func (Userdetail) TableName() string {
	return "userdetails"
}

func (r *UserRepoImpl) SaveUserDetails(args *dto.UserDetailSaveRequest) (int64, error) {

	user := Userdetail{
		ID:          args.UserID,
		Address:     args.Address,
		Mail:        args.Mail,
		Username:    args.UserName,
		Password:    args.Password,
		Pincode:     args.Pincode,
		Phonenumber: args.Phone,
		IsAdmin:     args.IsAdmin,
	}
	//GORM's Create method to insert the new user
	if err := r.db.Table("userdetails").Create(&user).Error; err != nil {
		return 0, err
	}
	return user.ID, nil
}

func (r *UserRepoImpl) GetUserByUsername(username string) (*Userdetail, error) {
	var user Userdetail
	if err := r.db.Table("userdetails").Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepoImpl) IsUserActive(userID int64) (bool, error) {
	var user Userdetail

	// Fetch the user details by userID
	if err := r.db.Table("userdetails").Where("id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, fmt.Errorf("user not found")
		}
		return false, err
	}

	return user.Status, nil
}
