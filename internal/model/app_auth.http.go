package model

// APPAuth: send code
type AppSendEmailCodeReq struct {
	Email string `json:"email" validate:"required,email"`
	Scene string `json:"scene" validate:"required,max=32"` // LOGIN/REGISTER/RESET_PASSWORD/BIND
}

type AppSendPhoneCodeReq struct {
	PhonePrefix string `json:"phone_prefix" validate:"omitempty,max=10"`      // +86
	PhoneNumber string `json:"phone_number" validate:"required,min=5,max=50"` // 13100000000
	Scene       string `json:"scene" validate:"required,max=32"`              // LOGIN/REGISTER/RESET_PASSWORD/BIND
}

// APPAuth: login/register/reset
type AppPhoneLoginByCodeReq struct {
	PhonePrefix string `json:"phone_prefix" validate:"omitempty,max=10"`
	PhoneNumber string `json:"phone_number" validate:"required,min=5,max=50"`
	VerifyCode  string `json:"verify_code" validate:"required,min=4,max=10"`
}

type AppEmailLoginByCodeReq struct {
	Email      string `json:"email" validate:"required,email"`
	VerifyCode string `json:"verify_code" validate:"required,min=4,max=10"`
}

type AppPhoneRegisterReq struct {
	PhonePrefix string  `json:"phone_prefix" validate:"omitempty,max=10"`
	PhoneNumber string  `json:"phone_number" validate:"required,min=5,max=50"`
	VerifyCode  string  `json:"verify_code" validate:"required,min=4,max=10"`
	Password    *string `json:"password" validate:"omitempty,min=6,max=255"`
}

type AppEmailRegisterReq struct {
	Email      string  `json:"email" validate:"required,email"`
	VerifyCode string  `json:"verify_code" validate:"required,min=4,max=10"`
	Password   *string `json:"password" validate:"omitempty,min=6,max=255"`
}

type AppPhoneResetPasswordReq struct {
	PhonePrefix string `json:"phone_prefix" validate:"omitempty,max=10"`
	PhoneNumber string `json:"phone_number" validate:"required,min=5,max=50"`
	VerifyCode  string `json:"verify_code" validate:"required,min=4,max=10"`
	Password    string `json:"password" validate:"required,min=6,max=255"`
}

type AppEmailResetPasswordReq struct {
	Email      string `json:"email" validate:"required,email"`
	VerifyCode string `json:"verify_code" validate:"required,min=4,max=10"`
	Password   string `json:"password" validate:"required,min=6,max=255"`
}

type AppWxmpLoginReq struct {
	Code string `json:"code" validate:"required"`
}

type AppWxmpBindPhoneReq struct {
	PhoneCode string `json:"phone_code" validate:"required"` // wx.getPhoneNumber 返回的 code
}

type AppWxmpProfileReq struct {
	NickName  *string `json:"nick_name" validate:"omitempty,max=50"`
	AvatarURL *string `json:"avatar_url" validate:"omitempty,max=500"`
	Gender    *int    `json:"gender" validate:"omitempty,oneof=0 1 2"` // 0未知 1男 2女
	Country   *string `json:"country" validate:"omitempty,max=50"`
	Province  *string `json:"province" validate:"omitempty,max=50"`
	City      *string `json:"city" validate:"omitempty,max=50"`
	Language  *string `json:"language" validate:"omitempty,max=20"`
}

// APPAuth: binding
type AppBindPhoneReq struct {
	PhonePrefix string `json:"phone_prefix" validate:"omitempty,max=10"`
	PhoneNumber string `json:"phone_number" validate:"required,min=5,max=50"`
	VerifyCode  string `json:"verify_code" validate:"required,min=4,max=10"`
}

type AppBindEmailReq struct {
	Email      string `json:"email" validate:"required,email"`
	VerifyCode string `json:"verify_code" validate:"required,min=4,max=10"`
}

type AppUnbindReq struct {
	IdentityType string `json:"identity_type" validate:"required,max=32,oneof=PHONE EMAIL"`
}

type AppAuthBindingsResp struct {
	UserID string               `json:"user_id"`
	List   []AppAuthBindingItem `json:"list"`
}

type AppAuthBindingItem struct {
	ID           string  `json:"id"`
	IdentityType string  `json:"identity_type"`
	Identifier   string  `json:"identifier"`
	IsPrimary    bool    `json:"is_primary"`
	VerifiedAt   *string `json:"verified_at"` // RFC3339 string
	Status       string  `json:"status"`
}

// WEB配置：模板、微信小程序配置（按租户）
type UpsertAuthMessageTemplateReq struct {
	Channel              string  `json:"channel" validate:"required,max=16,oneof=EMAIL SMS"`
	Scene                string  `json:"scene" validate:"required,max=32,oneof=LOGIN REGISTER RESET_PASSWORD BIND"`
	Subject              *string `json:"subject" validate:"omitempty,max=200"`
	Content              *string `json:"content" validate:"omitempty,max=10000"`
	Provider             *string `json:"provider" validate:"omitempty,max=36"`
	ProviderTemplateCode *string `json:"provider_template_code" validate:"omitempty,max=64"`
	Status               string  `json:"status" validate:"required,max=16,oneof=OPEN CLOSE"`
	Remark               *string `json:"remark" validate:"omitempty,max=255"`
}

type UpsertWxMpAppReq struct {
	AppID     string  `json:"appid" validate:"required,max=64"`
	AppSecret string  `json:"app_secret" validate:"required,max=128"`
	Status    string  `json:"status" validate:"required,max=16,oneof=OPEN CLOSE"`
	Remark    *string `json:"remark" validate:"omitempty,max=255"`
}
