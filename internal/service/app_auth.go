package service

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	dal "project/internal/dal"
	"project/internal/model"
	"project/pkg/common"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/go-basic/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type AppAuth struct{}

const (
	AuthCodeTTL = 5 * time.Minute
)

func normalizePhone(prefix, number string) string {
	prefix = strings.TrimSpace(prefix)
	number = strings.TrimSpace(number)
	if prefix == "" {
		return number
	}
	return strings.TrimSpace(prefix + " " + number)
}

func placeholderEmail(userID string) string {
	// 必须符合 email 格式，且尽量避免与真实邮箱冲突。
	return fmt.Sprintf("u_%s@app.local", strings.ReplaceAll(userID, "-", ""))
}

func authCodeKey(tenantID, channel, scene, identifier string) string {
	h := sha1.Sum([]byte(strings.TrimSpace(identifier)))
	return fmt.Sprintf("auth_code:%s:%s:%s:%s", tenantID, channel, scene, hex.EncodeToString(h[:]))
}

func (a *AppAuth) sendCode(ctx context.Context, tenantID, channel, scene, identifier string) (string, error) {
	if tenantID == "" {
		return "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{"error": "tenant_id is empty"})
	}
	if channel == "" || scene == "" || identifier == "" {
		return "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{"error": "invalid code params"})
	}

	code, err := common.GenerateNumericCode(6)
	if err != nil {
		return "", errcode.WithData(errcode.CodeSystemError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	key := authCodeKey(tenantID, channel, scene, identifier)
	if err := global.REDIS.Set(ctx, key, code, AuthCodeTTL).Err(); err != nil {
		return "", errcode.WithData(errcode.CodeCacheError, map[string]interface{}{
			"operation": "save_auth_code",
			"error":     err.Error(),
		})
	}
	return code, nil
}

func (a *AppAuth) verifyCode(ctx context.Context, tenantID, channel, scene, identifier, code string) error {
	key := authCodeKey(tenantID, channel, scene, identifier)
	val, err := global.REDIS.Get(ctx, key).Result()
	if err != nil {
		return errcode.New(200011) // 验证码已过期（复用现有错误码）
	}
	if strings.TrimSpace(val) != strings.TrimSpace(code) {
		return errcode.New(200012) // 验证码错误（复用现有错误码）
	}
	_ = global.REDIS.Del(ctx, key).Err()
	return nil
}

func (a *AppAuth) SendEmailCode(ctx context.Context, tenantID, email, scene string) error {
	email = strings.TrimSpace(email)
	if !utils.ValidateEmail(email) {
		return errcode.New(200014) // 邮箱格式错误（复用已有错误码）
	}
	scene = strings.ToUpper(strings.TrimSpace(scene))

	tpl, err := dal.GetAuthMessageTemplate(ctx, tenantID, dal.TemplateChannelEmail, scene)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"error": err.Error()})
	}
	if tpl == nil || tpl.Status != dal.TemplateStatusOpen {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"error":   "email template not configured",
			"channel": dal.TemplateChannelEmail,
			"scene":   scene,
		})
	}

	code, err := a.sendCode(ctx, tenantID, dal.TemplateChannelEmail, scene, email)
	if err != nil {
		return err
	}

	subject := "验证码"
	if tpl.Subject != nil && strings.TrimSpace(*tpl.Subject) != "" {
		subject = *tpl.Subject
	}

	body := "Your verification code is " + code
	if tpl.Content != nil && strings.TrimSpace(*tpl.Content) != "" {
		body = *tpl.Content
		body = strings.ReplaceAll(body, "{{code}}", code)
		body = strings.ReplaceAll(body, "${code}", code)
	}

	if err := sendEmailMessage(body, subject, tenantID, email); err != nil {
		return errcode.WithData(200010, map[string]interface{}{
			"email": email,
			"error": err.Error(),
		})
	}
	return nil
}

func (a *AppAuth) SendPhoneCode(ctx context.Context, tenantID, phonePrefix, phoneNumber, scene string) error {
	phone := normalizePhone(phonePrefix, phoneNumber)
	if strings.TrimSpace(phone) == "" {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"error": "phone is empty"})
	}
	scene = strings.ToUpper(strings.TrimSpace(scene))

	tpl, err := dal.GetAuthMessageTemplate(ctx, tenantID, dal.TemplateChannelSMS, scene)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"error": err.Error()})
	}
	if tpl == nil || tpl.Status != dal.TemplateStatusOpen {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"error":   "sms template not configured",
			"channel": dal.TemplateChannelSMS,
			"scene":   scene,
		})
	}

	templateCode := ""
	if tpl.ProviderTemplateCode != nil {
		templateCode = strings.TrimSpace(*tpl.ProviderTemplateCode)
	}

	code, err := a.sendCode(ctx, tenantID, dal.TemplateChannelSMS, scene, phone)
	if err != nil {
		return err
	}

	params := map[string]string{"code": code}
	if err := GroupApp.NotificationServicesConfig.SendSMSByTemplate(ctx, tenantID, phone, templateCode, params); err != nil {
		return err
	}

	return nil
}

func (a *AppAuth) PhoneLoginByCode(ctx context.Context, tenantID, phonePrefix, phoneNumber, verifyCode string) (*model.LoginRsp, error) {
	phone := normalizePhone(phonePrefix, phoneNumber)
	if err := a.verifyCode(ctx, tenantID, dal.TemplateChannelSMS, dal.TemplateSceneLogin, phone, verifyCode); err != nil {
		return nil, err
	}

	identity, err := dal.GetUserIdentity(ctx, tenantID, dal.IdentityTypePhone, phone)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.New(errcode.CodeInvalidAuth)
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"error": err.Error()})
	}

	user, err := dal.GetUsersById(identity.UserID)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"error": err.Error()})
	}
	if user.Status != nil && *user.Status != "N" {
		return nil, errcode.New(errcode.CodeUserDisabled)
	}

	loginRsp, err := GroupApp.User.UserLoginAfter(user)
	if err != nil {
		return nil, err
	}
	_ = dal.UserQuery{}.UpdateLastVisitTime(ctx, user.ID)
	return loginRsp, nil
}

func (a *AppAuth) EmailLoginByCode(ctx context.Context, tenantID, email, verifyCode string) (*model.LoginRsp, error) {
	email = strings.TrimSpace(email)
	if err := a.verifyCode(ctx, tenantID, dal.TemplateChannelEmail, dal.TemplateSceneLogin, email, verifyCode); err != nil {
		return nil, err
	}

	identity, err := dal.GetUserIdentity(ctx, tenantID, dal.IdentityTypeEmail, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.New(errcode.CodeInvalidAuth)
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"error": err.Error()})
	}

	user, err := dal.GetUsersById(identity.UserID)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"error": err.Error()})
	}
	if user.Status != nil && *user.Status != "N" {
		return nil, errcode.New(errcode.CodeUserDisabled)
	}

	loginRsp, err := GroupApp.User.UserLoginAfter(user)
	if err != nil {
		return nil, err
	}
	_ = dal.UserQuery{}.UpdateLastVisitTime(ctx, user.ID)
	return loginRsp, nil
}

func randomPassword() string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		// 兜底：即使随机源不可用，也返回一个不为空的字符串
		return fmt.Sprintf("p_%d", time.Now().UnixNano())
	}
	var b strings.Builder
	for i := 0; i < len(buf); i++ {
		b.WriteByte(alphabet[int(buf[i])%len(alphabet)])
	}
	return b.String()
}

func (a *AppAuth) PhoneRegister(ctx context.Context, tenantID, phonePrefix, phoneNumber, verifyCode string, password *string) (*model.LoginRsp, error) {
	phone := normalizePhone(phonePrefix, phoneNumber)
	if err := a.verifyCode(ctx, tenantID, dal.TemplateChannelSMS, dal.TemplateSceneRegister, phone, verifyCode); err != nil {
		return nil, err
	}

	if _, err := dal.GetUserIdentity(ctx, tenantID, dal.IdentityTypePhone, phone); err == nil {
		return nil, errcode.New(errcode.CodePhoneDuplicated)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"error": err.Error()})
	}

	now := time.Now().UTC()
	userID := uuid.New()
	email := placeholderEmail(userID)

	pass := randomPassword()
	if password != nil && strings.TrimSpace(*password) != "" {
		if err := utils.ValidatePassword(*password); err != nil {
			return nil, err
		}
		pass = *password
	}
	passHash := utils.BcryptHash(pass)

	u := &model.User{
		ID:                  userID,
		Name:                nil,
		PhoneNumber:         phone,
		Email:               email,
		Status:              StringPtr("N"),
		Authority:           StringPtr(dal.TENANT_USER),
		Password:            passHash,
		TenantID:            StringPtr(tenantID),
		UserKind:            StringPtr(model.UserKindEndUser),
		CreatedAt:           &now,
		UpdatedAt:           &now,
		PasswordLastUpdated: &now,
	}

	identity := &dal.UserIdentity{
		ID:             uuid.New(),
		UserID:         userID,
		TenantID:       tenantID,
		IdentityType:   dal.IdentityTypePhone,
		Identifier:     phone,
		CredentialType: dal.CredentialTypePassword,
		PasswordHash:   &passHash,
		VerifiedAt:     &now,
		IsPrimary:      true,
		Status:         "ACTIVE",
		Extra:          StringPtr("{}"),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("users").Create(u).Error; err != nil {
			return err
		}
		return dal.CreateUserIdentity(ctx, tx, identity)
	})
	if err != nil {
		logrus.Error(err)
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "phone_register",
			"error":     err.Error(),
		})
	}

	return GroupApp.User.UserLoginAfter(u)
}

func (a *AppAuth) EmailRegister(ctx context.Context, tenantID, email, verifyCode string, password *string) (*model.LoginRsp, error) {
	email = strings.TrimSpace(email)
	if !utils.ValidateEmail(email) {
		return nil, errcode.New(200014)
	}
	if err := a.verifyCode(ctx, tenantID, dal.TemplateChannelEmail, dal.TemplateSceneRegister, email, verifyCode); err != nil {
		return nil, err
	}

	if _, err := dal.GetUserIdentity(ctx, tenantID, dal.IdentityTypeEmail, email); err == nil {
		return nil, errcode.New(200008) // 邮箱已注册
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"error": err.Error()})
	}

	now := time.Now().UTC()
	userID := uuid.New()
	phone := ""

	pass := randomPassword()
	if password != nil && strings.TrimSpace(*password) != "" {
		if err := utils.ValidatePassword(*password); err != nil {
			return nil, err
		}
		pass = *password
	}
	passHash := utils.BcryptHash(pass)

	u := &model.User{
		ID:                  userID,
		Name:                nil,
		PhoneNumber:         phone,
		Email:               email,
		Status:              StringPtr("N"),
		Authority:           StringPtr(dal.TENANT_USER),
		Password:            passHash,
		TenantID:            StringPtr(tenantID),
		UserKind:            StringPtr(model.UserKindEndUser),
		CreatedAt:           &now,
		UpdatedAt:           &now,
		PasswordLastUpdated: &now,
	}

	identity := &dal.UserIdentity{
		ID:             uuid.New(),
		UserID:         userID,
		TenantID:       tenantID,
		IdentityType:   dal.IdentityTypeEmail,
		Identifier:     email,
		CredentialType: dal.CredentialTypePassword,
		PasswordHash:   &passHash,
		VerifiedAt:     &now,
		IsPrimary:      true,
		Status:         "ACTIVE",
		Extra:          StringPtr("{}"),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("users").Create(u).Error; err != nil {
			return err
		}
		return dal.CreateUserIdentity(ctx, tx, identity)
	})
	if err != nil {
		logrus.Error(err)
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "email_register",
			"error":     err.Error(),
		})
	}

	return GroupApp.User.UserLoginAfter(u)
}

func (a *AppAuth) ResetPasswordByPhone(ctx context.Context, tenantID, phonePrefix, phoneNumber, verifyCode, newPassword string) error {
	phone := normalizePhone(phonePrefix, phoneNumber)
	if err := utils.ValidatePassword(newPassword); err != nil {
		return err
	}
	if err := a.verifyCode(ctx, tenantID, dal.TemplateChannelSMS, dal.TemplateSceneResetPassword, phone, verifyCode); err != nil {
		return err
	}
	identity, err := dal.GetUserIdentity(ctx, tenantID, dal.IdentityTypePhone, phone)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.New(errcode.CodeInvalidAuth)
		}
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"error": err.Error()})
	}

	passHash := utils.BcryptHash(newPassword)
	now := time.Now().UTC()

	return global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("users").
			Where("id = ?", identity.UserID).
			Updates(map[string]interface{}{
				"password":              passHash,
				"password_last_updated": now,
				"updated_at":            now,
			}).Error; err != nil {
			return err
		}

		return dal.UpdateUserIdentity(ctx, tx, tenantID, identity.ID, map[string]interface{}{
			"credential_type": dal.CredentialTypePassword,
			"password_hash":   passHash,
			"updated_at":      now,
		})
	})
}

func (a *AppAuth) ResetPasswordByEmail(ctx context.Context, tenantID, email, verifyCode, newPassword string) error {
	email = strings.TrimSpace(email)
	if !utils.ValidateEmail(email) {
		return errcode.New(200014)
	}
	if err := utils.ValidatePassword(newPassword); err != nil {
		return err
	}
	if err := a.verifyCode(ctx, tenantID, dal.TemplateChannelEmail, dal.TemplateSceneResetPassword, email, verifyCode); err != nil {
		return err
	}
	identity, err := dal.GetUserIdentity(ctx, tenantID, dal.IdentityTypeEmail, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.New(errcode.CodeInvalidAuth)
		}
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"error": err.Error()})
	}

	passHash := utils.BcryptHash(newPassword)
	now := time.Now().UTC()

	return global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("users").
			Where("id = ?", identity.UserID).
			Updates(map[string]interface{}{
				"password":              passHash,
				"password_last_updated": now,
				"updated_at":            now,
			}).Error; err != nil {
			return err
		}

		return dal.UpdateUserIdentity(ctx, tx, tenantID, identity.ID, map[string]interface{}{
			"credential_type": dal.CredentialTypePassword,
			"password_hash":   passHash,
			"updated_at":      now,
		})
	})
}

func (a *AppAuth) GetBindings(ctx context.Context, tenantID, userID string) (*model.AppAuthBindingsResp, error) {
	list, err := dal.ListUserIdentitiesByUser(ctx, tenantID, userID)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"error": err.Error()})
	}
	resp := &model.AppAuthBindingsResp{UserID: userID}
	for _, it := range list {
		var verifiedAt *string
		if it.VerifiedAt != nil {
			s := it.VerifiedAt.UTC().Format(time.RFC3339)
			verifiedAt = &s
		}
		resp.List = append(resp.List, model.AppAuthBindingItem{
			ID:           it.ID,
			IdentityType: it.IdentityType,
			Identifier:   it.Identifier,
			IsPrimary:    it.IsPrimary,
			VerifiedAt:   verifiedAt,
			Status:       it.Status,
		})
	}
	return resp, nil
}

func (a *AppAuth) BindPhone(ctx context.Context, tenantID, userID, phonePrefix, phoneNumber, verifyCode string) error {
	phone := normalizePhone(phonePrefix, phoneNumber)
	if err := a.verifyCode(ctx, tenantID, dal.TemplateChannelSMS, dal.TemplateSceneBind, phone, verifyCode); err != nil {
		return err
	}

	// 防止手机号被其它用户占用
	if exists, err := dal.CheckPhoneNumberExists(phone, userID); err != nil {
		return err
	} else if exists {
		return errcode.New(errcode.CodePhoneDuplicated)
	}

	now := time.Now().UTC()
	return global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 若已存在该身份，则视为已绑定
		if _, err := dal.GetUserIdentity(ctx, tenantID, dal.IdentityTypePhone, phone); err == nil {
			return errcode.New(errcode.CodePhoneDuplicated)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// 若用户暂无任何身份，则该手机号作为主身份
		list, err := dal.ListUserIdentitiesByUser(ctx, tenantID, userID)
		if err != nil {
			return err
		}
		isPrimary := len(list) == 0

		passHash := ""
		user, err := dal.GetUsersById(userID)
		if err == nil && user != nil {
			passHash = user.Password
		}
		var passHashPtr *string
		if passHash != "" {
			passHashPtr = &passHash
		}

		identity := &dal.UserIdentity{
			ID:             uuid.New(),
			UserID:         userID,
			TenantID:       tenantID,
			IdentityType:   dal.IdentityTypePhone,
			Identifier:     phone,
			CredentialType: dal.CredentialTypePassword,
			PasswordHash:   passHashPtr,
			VerifiedAt:     &now,
			IsPrimary:      isPrimary,
			Status:         "ACTIVE",
			Extra:          StringPtr("{}"),
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		if err := dal.CreateUserIdentity(ctx, tx, identity); err != nil {
			return err
		}
		return tx.Table("users").Where("id = ?", userID).Updates(map[string]interface{}{
			"phone_number": phone,
			"updated_at":   now,
		}).Error
	})
}

func (a *AppAuth) BindEmail(ctx context.Context, tenantID, userID, email, verifyCode string) error {
	email = strings.TrimSpace(email)
	if !utils.ValidateEmail(email) {
		return errcode.New(200014)
	}
	if err := a.verifyCode(ctx, tenantID, dal.TemplateChannelEmail, dal.TemplateSceneBind, email, verifyCode); err != nil {
		return err
	}

	now := time.Now().UTC()
	return global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 若已存在该身份，则视为已绑定
		if _, err := dal.GetUserIdentity(ctx, tenantID, dal.IdentityTypeEmail, email); err == nil {
			return errcode.New(200008)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// users.email 是全局唯一约束，必须先确认无占用
		if u, err := dal.GetUsersByEmail(email); err == nil && u != nil && u.ID != userID {
			return errcode.New(200008)
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		list, err := dal.ListUserIdentitiesByUser(ctx, tenantID, userID)
		if err != nil {
			return err
		}
		isPrimary := len(list) == 0

		passHash := ""
		user, err := dal.GetUsersById(userID)
		if err == nil && user != nil {
			passHash = user.Password
		}
		var passHashPtr *string
		if passHash != "" {
			passHashPtr = &passHash
		}

		identity := &dal.UserIdentity{
			ID:             uuid.New(),
			UserID:         userID,
			TenantID:       tenantID,
			IdentityType:   dal.IdentityTypeEmail,
			Identifier:     email,
			CredentialType: dal.CredentialTypePassword,
			PasswordHash:   passHashPtr,
			VerifiedAt:     &now,
			IsPrimary:      isPrimary,
			Status:         "ACTIVE",
			Extra:          StringPtr("{}"),
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		if err := dal.CreateUserIdentity(ctx, tx, identity); err != nil {
			return err
		}
		return tx.Table("users").Where("id = ?", userID).Updates(map[string]interface{}{
			"email":      email,
			"updated_at": now,
		}).Error
	})
}

func (a *AppAuth) UnbindIdentity(ctx context.Context, tenantID, userID, identityType string) error {
	list, err := dal.ListUserIdentitiesByUser(ctx, tenantID, userID)
	if err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"error": err.Error()})
	}

	var target *dal.UserIdentity
	var remaining []dal.UserIdentity
	for _, it := range list {
		if it.IdentityType == identityType && target == nil {
			cp := it
			target = &cp
			continue
		}
		remaining = append(remaining, it)
	}
	if target == nil {
		return errcode.WithData(errcode.CodeNotFound, map[string]interface{}{"error": "identity not found"})
	}
	if len(remaining) == 0 {
		return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{
			"error": "cannot unbind last login identity",
		})
	}

	now := time.Now().UTC()
	return global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := dal.DeleteUserIdentity(ctx, tx, tenantID, target.ID); err != nil {
			return err
		}

		// 如果解绑的是主身份，则把最早创建的剩余身份设为主身份
		if target.IsPrimary {
			next := remaining[0]
			for _, it := range remaining {
				if it.CreatedAt.Before(next.CreatedAt) {
					next = it
				}
			}
			if err := dal.UpdateUserIdentity(ctx, tx, tenantID, next.ID, map[string]interface{}{
				"is_primary": true,
				"updated_at": now,
			}); err != nil {
				return err
			}
		}

		// 同步 users 表字段（解绑后置空；不影响现有WEB账号，因为WEB不走解绑接口）
		switch identityType {
		case dal.IdentityTypePhone:
			if err := tx.Table("users").Where("id = ?", userID).Updates(map[string]interface{}{
				"phone_number": "",
				"updated_at":   now,
			}).Error; err != nil {
				return err
			}
		case dal.IdentityTypeEmail:
			// email 置为占位符，满足 NOT NULL + UNIQUE
			if err := tx.Table("users").Where("id = ?", userID).Updates(map[string]interface{}{
				"email":      placeholderEmail(userID),
				"updated_at": now,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// WxmpLogin 微信小程序一键登录：通过 code2session 获取 openid，然后按租户查/建身份并登录
func (a *AppAuth) WxmpLogin(ctx context.Context, tenantID, code string) (*model.LoginRsp, error) {
	code = strings.TrimSpace(code)
	if tenantID == "" || code == "" {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"error": "tenant_id/code is empty"})
	}

	wxConf, err := dal.GetWxMpAppByTenant(ctx, tenantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"error": "wx miniapp not configured"})
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"error": err.Error()})
	}
	if strings.ToUpper(wxConf.Status) != "OPEN" {
		return nil, errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{"error": "wx miniapp disabled"})
	}

	openid, err := a.wxCode2Session(ctx, wxConf.AppID, wxConf.AppSecret, code)
	if err != nil {
		return nil, err
	}

	identity, err := dal.GetUserIdentity(ctx, tenantID, dal.IdentityTypeWxmpOpenID, openid)
	if err == nil {
		user, err := dal.GetUsersById(identity.UserID)
		if err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"error": err.Error()})
		}
		if user.Status != nil && *user.Status != "N" {
			return nil, errcode.New(errcode.CodeUserDisabled)
		}
		loginRsp, err := GroupApp.User.UserLoginAfter(user)
		if err != nil {
			return nil, err
		}
		_ = dal.UserQuery{}.UpdateLastVisitTime(ctx, user.ID)
		return loginRsp, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"error": err.Error()})
	}

	now := time.Now().UTC()
	userID := uuid.New()
	u := &model.User{
		ID:                  userID,
		Name:                nil,
		PhoneNumber:         "",
		Email:               placeholderEmail(userID),
		Status:              StringPtr("N"),
		Authority:           StringPtr(dal.TENANT_USER),
		Password:            utils.BcryptHash(randomPassword()),
		TenantID:            StringPtr(tenantID),
		UserKind:            StringPtr(model.UserKindEndUser),
		CreatedAt:           &now,
		UpdatedAt:           &now,
		PasswordLastUpdated: &now,
	}

	identity = &dal.UserIdentity{
		ID:             uuid.New(),
		UserID:         userID,
		TenantID:       tenantID,
		IdentityType:   dal.IdentityTypeWxmpOpenID,
		Identifier:     openid,
		CredentialType: dal.CredentialTypeCode,
		PasswordHash:   nil,
		VerifiedAt:     &now,
		IsPrimary:      true,
		Status:         "ACTIVE",
		Extra:          StringPtr("{}"),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("users").Create(u).Error; err != nil {
			return err
		}
		return dal.CreateUserIdentity(ctx, tx, identity)
	}); err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "wxmp_create_user",
			"error":     err.Error(),
		})
	}

	return GroupApp.User.UserLoginAfter(u)
}

type wxCode2SessionResp struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

func (a *AppAuth) wxCode2Session(ctx context.Context, appid, secret, code string) (string, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		appid, secret, code)
	req, err := httpRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", errcode.WithData(errcode.CodeSystemError, map[string]interface{}{"error": err.Error()})
	}
	body, err := doHTTPRequest(req)
	if err != nil {
		return "", errcode.WithData(errcode.CodeSystemError, map[string]interface{}{"error": err.Error()})
	}

	var resp wxCode2SessionResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", errcode.WithData(errcode.CodeSystemError, map[string]interface{}{"error": err.Error()})
	}
	if resp.ErrCode != 0 {
		return "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"error":   "wx code2session failed",
			"errcode": resp.ErrCode,
			"errmsg":  resp.ErrMsg,
		})
	}
	if strings.TrimSpace(resp.OpenID) == "" {
		return "", errcode.WithData(errcode.CodeSystemError, map[string]interface{}{"error": "wx openid empty"})
	}
	return resp.OpenID, nil
}

type wxAccessTokenResp struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

type wxPhoneNumberResp struct {
	ErrCode   int    `json:"errcode"`
	ErrMsg    string `json:"errmsg"`
	PhoneInfo struct {
		PhoneNumber     string `json:"phoneNumber"`
		PurePhoneNumber string `json:"purePhoneNumber"`
		CountryCode     string `json:"countryCode"`
	} `json:"phone_info"`
}

func (a *AppAuth) wxGetAccessToken(ctx context.Context, tenantID, appid, secret string) (string, error) {
	cacheKey := fmt.Sprintf("wxmp_access_token:%s", tenantID)
	if token, err := global.REDIS.Get(ctx, cacheKey).Result(); err == nil && strings.TrimSpace(token) != "" {
		return token, nil
	}

	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s", appid, secret)
	req, err := httpRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	body, err := doHTTPRequest(req)
	if err != nil {
		return "", err
	}
	var resp wxAccessTokenResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	if resp.ErrCode != 0 || strings.TrimSpace(resp.AccessToken) == "" {
		return "", errcode.WithData(errcode.CodeSystemError, map[string]interface{}{
			"error":   "wx get access_token failed",
			"errcode": resp.ErrCode,
			"errmsg":  resp.ErrMsg,
		})
	}

	ttl := 7000 * time.Second
	if resp.ExpiresIn > 0 {
		ttl = time.Duration(resp.ExpiresIn-200) * time.Second
	}
	_ = global.REDIS.Set(ctx, cacheKey, resp.AccessToken, ttl).Err()
	return resp.AccessToken, nil
}

func (a *AppAuth) wxGetPhoneNumber(ctx context.Context, accessToken, phoneCode string) (string, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/wxa/business/getuserphonenumber?access_token=%s", accessToken)
	payload, _ := json.Marshal(map[string]string{"code": phoneCode})
	req, err := httpRequestWithContext(ctx, http.MethodPost, url, payload)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	body, err := doHTTPRequest(req)
	if err != nil {
		return "", err
	}
	var resp wxPhoneNumberResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	if resp.ErrCode != 0 {
		return "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"error":   "wx get phone number failed",
			"errcode": resp.ErrCode,
			"errmsg":  resp.ErrMsg,
		})
	}
	cc := strings.TrimSpace(resp.PhoneInfo.CountryCode)
	pure := strings.TrimSpace(resp.PhoneInfo.PurePhoneNumber)
	if cc == "" || pure == "" {
		return "", errcode.WithData(errcode.CodeSystemError, map[string]interface{}{
			"error": "wx phone number empty",
		})
	}
	return normalizePhone("+"+cc, pure), nil
}

func (a *AppAuth) bindPhoneNoVerify(ctx context.Context, tenantID, userID, phone string) error {
	// 防止手机号被其它用户占用
	if exists, err := dal.CheckPhoneNumberExists(phone, userID); err != nil {
		return err
	} else if exists {
		return errcode.New(errcode.CodePhoneDuplicated)
	}

	now := time.Now().UTC()
	return global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := dal.GetUserIdentity(ctx, tenantID, dal.IdentityTypePhone, phone); err == nil {
			return errcode.New(errcode.CodePhoneDuplicated)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		list, err := dal.ListUserIdentitiesByUser(ctx, tenantID, userID)
		if err != nil {
			return err
		}
		isPrimary := len(list) == 0

		passHash := ""
		user, err := dal.GetUsersById(userID)
		if err == nil && user != nil {
			passHash = user.Password
		}
		var passHashPtr *string
		if passHash != "" {
			passHashPtr = &passHash
		}

		identity := &dal.UserIdentity{
			ID:             uuid.New(),
			UserID:         userID,
			TenantID:       tenantID,
			IdentityType:   dal.IdentityTypePhone,
			Identifier:     phone,
			CredentialType: dal.CredentialTypePassword,
			PasswordHash:   passHashPtr,
			VerifiedAt:     &now,
			IsPrimary:      isPrimary,
			Status:         "ACTIVE",
			Extra:          StringPtr("{}"),
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := dal.CreateUserIdentity(ctx, tx, identity); err != nil {
			return err
		}
		return tx.Table("users").Where("id = ?", userID).Updates(map[string]interface{}{
			"phone_number": phone,
			"updated_at":   now,
		}).Error
	})
}

// WxmpBindPhone 微信小程序一键绑定手机号（使用 wx.getPhoneNumber code）
func (a *AppAuth) WxmpBindPhone(ctx context.Context, tenantID, userID, phoneCode string) error {
	phoneCode = strings.TrimSpace(phoneCode)
	if tenantID == "" || userID == "" || phoneCode == "" {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"error": "tenant_id/user_id/phone_code is empty"})
	}
	wxConf, err := dal.GetWxMpAppByTenant(ctx, tenantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"error": "wx miniapp not configured"})
		}
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"error": err.Error()})
	}
	if strings.ToUpper(wxConf.Status) != "OPEN" {
		return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{"error": "wx miniapp disabled"})
	}
	token, err := a.wxGetAccessToken(ctx, tenantID, wxConf.AppID, wxConf.AppSecret)
	if err != nil {
		return err
	}
	phone, err := a.wxGetPhoneNumber(ctx, token, phoneCode)
	if err != nil {
		return err
	}
	return a.bindPhoneNoVerify(ctx, tenantID, userID, phone)
}

// WxmpUpdateProfile 微信小程序用户信息解析/保存（wx.getUserProfile 返回）
func (a *AppAuth) WxmpUpdateProfile(ctx context.Context, tenantID, userID string, req *model.AppWxmpProfileReq) error {
	if tenantID == "" || userID == "" || req == nil {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"error": "tenant_id/user_id/req is empty"})
	}

	user, err := dal.GetUsersById(userID)
	if err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "query_user",
			"error":     err.Error(),
		})
	}
	if user.TenantID == nil || *user.TenantID != tenantID {
		return errcode.WithData(errcode.CodeNoPermission, map[string]interface{}{
			"error": "tenant mismatch",
		})
	}
	// 仅允许终端用户更新（避免误修改 WEB/业务账号资料）
	if user.UserKind != nil && *user.UserKind != model.UserKindEndUser {
		return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{
			"error": "only END_USER can update wx profile",
		})
	}

	now := time.Now().UTC()

	updates := map[string]interface{}{
		"updated_at": now,
	}
	if req.NickName != nil && strings.TrimSpace(*req.NickName) != "" {
		n := strings.TrimSpace(*req.NickName)
		updates["name"] = n
	}
	if req.AvatarURL != nil && strings.TrimSpace(*req.AvatarURL) != "" {
		updates["avatar_url"] = strings.TrimSpace(*req.AvatarURL)
	}

	// 追加到 additional_info.wx_profile
	var additional map[string]interface{}
	if user.AdditionalInfo != nil && strings.TrimSpace(*user.AdditionalInfo) != "" {
		_ = json.Unmarshal([]byte(*user.AdditionalInfo), &additional)
	}
	if additional == nil {
		additional = map[string]interface{}{}
	}
	additional["wx_profile"] = map[string]interface{}{
		"nick_name":  req.NickName,
		"avatar_url": req.AvatarURL,
		"gender":     req.Gender,
		"country":    req.Country,
		"province":   req.Province,
		"city":       req.City,
		"language":   req.Language,
	}
	additional["wx_profile_updated_at"] = now.Format(time.RFC3339)
	b, _ := json.Marshal(additional)
	s := string(b)
	updates["additional_info"] = s

	return global.DB.WithContext(ctx).Table("users").Where("id = ?", userID).Updates(updates).Error
}
