package service

import (
	"context"
	"strings"

	"project/internal/model"
	"project/pkg/errcode"

	"gorm.io/gorm"
)

// deleteUserCascadeTx clears known user-linked records before deleting the user row.
func deleteUserCascadeTx(ctx context.Context, tx *gorm.DB, user *model.User) error {
	if tx == nil {
		return errcode.WithData(errcode.CodeSystemError, map[string]interface{}{"error": "delete user tx is nil"})
	}
	if user == nil {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"error": "user is nil"})
	}

	userID := strings.TrimSpace(user.ID)
	if userID == "" {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"error": "user id is empty"})
	}

	tenantID := ""
	if user.TenantID != nil {
		tenantID = strings.TrimSpace(*user.TenantID)
	}

	deleteByTable := func(table string, where string, args ...interface{}) error {
		if err := tx.WithContext(ctx).Exec("DELETE FROM "+table+" WHERE "+where, args...).Error; err != nil {
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"table": table,
				"error": err.Error(),
			})
		}
		return nil
	}

	if tenantID != "" {
		if err := deleteByTable("user_identities", "tenant_id = ? AND user_id = ?", tenantID, userID); err != nil {
			return err
		}
		if err := deleteByTable("app_device_added_records", "tenant_id = ? AND user_id = ?", tenantID, userID); err != nil {
			return err
		}
	} else {
		if err := deleteByTable("user_identities", "user_id = ?", userID); err != nil {
			return err
		}
		if err := deleteByTable("app_device_added_records", "user_id = ?", userID); err != nil {
			return err
		}
	}

	if err := deleteByTable("device_user_bindings", "user_id = ?", userID); err != nil {
		return err
	}
	if err := deleteByTable("message_push_manage", "user_id = ?", userID); err != nil {
		return err
	}
	if err := deleteByTable("message_push_log", "user_id = ?", userID); err != nil {
		return err
	}
	if err := deleteByTable("user_roles", "user_id = ?", userID); err != nil {
		return err
	}
	if err := deleteByTable("user_address", "user_id = ?", userID); err != nil {
		return err
	}

	if err := tx.WithContext(ctx).Table("users").Where("id = ?", userID).Delete(&model.User{}).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"table": "users",
			"error": err.Error(),
		})
	}

	return nil
}
