package service

import (
	"fmt"
	"time"

	dal "project/internal/dal"
	model "project/internal/model"
	utils "project/pkg/utils"

	"github.com/go-basic/uuid"
	"github.com/sirupsen/logrus"
)

type Role struct{}

func (*Role) CreateRole(createRoleReq *model.CreateRoleReq, userClaims *utils.UserClaims) error {

	var role = model.Role{}

	role.ID = uuid.New()
	role.Name = createRoleReq.Name
	role.Description = createRoleReq.Description
	role.Authority = createRoleReq.Authority
	role.UserKind = createRoleReq.UserKind
	role.OrgType = createRoleReq.OrgType

	if role.Authority == nil || *role.Authority == "" {
		role.Authority = StringPtr(userClaims.Authority)
	}
	if role.UserKind == nil || *role.UserKind == "" {
		role.UserKind = StringPtr(model.UserKindOrgUser)
	}

	t := time.Now().UTC()
	role.CreatedAt = &t
	role.UpdatedAt = &t
	role.TenantID = &userClaims.TenantID

	err := dal.CreateRole(&role)

	if err != nil {
		logrus.Error(err)
	}

	return err
}

func (*Role) UpdateRole(updateRoleReq *model.UpdateRoleReq, userClaims *utils.UserClaims) (model.Role, error) {
	var role = model.Role{}
	existing, err := dal.GetRoleByID(updateRoleReq.Id)
	if err != nil {
		return role, err
	}
	if existing.TenantID != nil && userClaims.TenantID != "" && *existing.TenantID != userClaims.TenantID {
		return role, fmt.Errorf("no permission to update role")
	}
	role.ID = updateRoleReq.Id
	if updateRoleReq.Description != nil {
		role.Description = updateRoleReq.Description
	}
	if updateRoleReq.Name != "" {
		role.Name = updateRoleReq.Name
	}
	if updateRoleReq.Authority != nil {
		role.Authority = updateRoleReq.Authority
	}
	if updateRoleReq.UserKind != nil {
		role.UserKind = updateRoleReq.UserKind
	}
	if updateRoleReq.OrgType != nil {
		role.OrgType = updateRoleReq.OrgType
	}
	info, err := dal.UpdateRole(&role)
	if err != nil {
		logrus.Error(err)
	}

	if info.RowsAffected == 0 {
		return role, fmt.Errorf("no data updated")
	}

	role, err = dal.GetRoleByID(role.ID)
	if err != nil {
		logrus.Error(err)
	}

	return role, err
}

func (*Role) DeleteRole(id string, userClaims *utils.UserClaims) error {
	existing, err := dal.GetRoleByID(id)
	if err != nil {
		return err
	}
	if existing.TenantID != nil && userClaims.TenantID != "" && *existing.TenantID != userClaims.TenantID {
		return fmt.Errorf("no permission to delete role")
	}
	GroupApp.Casbin.RemoveRoleAndFunction(id)
	err = dal.DeleteRole(id)
	return err
}

func (*Role) GetRoleListByPage(params *model.GetRoleListByPageReq, userClaims *utils.UserClaims) (map[string]interface{}, error) {

	total, list, err := dal.GetRoleListByPage(params, userClaims.TenantID)
	if err != nil {
		return nil, err
	}
	roleListRsp := make(map[string]interface{})
	roleListRsp["total"] = total
	roleListRsp["list"] = list

	return roleListRsp, err
}
