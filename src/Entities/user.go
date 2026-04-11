package Entities

import (
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Id                int        `json:"id"`
	Username          string     `json:"username"`
	Avatar            *string    `json:"avatar"`
	Password          string     `json:"password,omitempty"` // Don't return password in JSON
	InterfaceLanguage *string    `json:"interface_language"`
	InterfaceTimezone *string    `json:"interface_timezone"`
	InterfaceFontSize float64    `json:"interface_font_size"`
	UserStatus        UserStatus `json:"user_status"`
	TotalPosts        int        `json:"total_posts"`
	TotalGeneralPosts int        `json:"total_general_posts"`
	DisableSound      bool       `json:"disable_sound"`
	InterfaceDesign   *string    `json:"interface_design"`
	Roles             []Role     `json:"roles"`
}

type ShortUser struct {
	Id       int    `json:"id"`
	Username string `json:"username"`
}

type UserProfile struct {
	UserId            int    `json:"user_id"`
	UserName          string `json:"user_name"`
	Avatar            string `json:"avatar"`
	TotalPosts        int    `json:"total_posts"`
	TotalGeneralPosts int    `json:"total_general_posts"`
	CurrencyAmount    *int   `json:"currency_amount,omitempty"`
}

type UserStatus int

const (
	ActiveUser  UserStatus = 0
	BlockedUser UserStatus = 1
)

type PrivateKey struct {
	UserId        int    `json:"user_id"`
	PrivateKey    string `json:"private_key"`
	Salt          string `json:"salt"`
	IV            string `json:"iv"`
	RecoveryKeyId *int   `json:"recover_key_id"`
}

type PublicKey struct {
	UserId    int    `json:"user_id"`
	PublicKey string `json:"public_key"`
}

func (u *User) HashPassword(password string) error {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return err
	}
	u.Password = string(bytes)
	return nil
}

func (u *User) CheckPassword(providedPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(providedPassword))
}
