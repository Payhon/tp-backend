package service

import (
	"fmt"
	"strings"
	"time"

	dal "project/internal/dal"
	model "project/internal/model"
	"project/pkg/errcode"
	utils "project/pkg/utils"

	"github.com/go-basic/uuid"
	"github.com/sirupsen/logrus"
)

type Dict struct{}

func (*Dict) CreateDictColumn(createDictReq *model.CreateDictReq, claims *utils.UserClaims) error {
	if claims.Authority != dal.SYS_ADMIN && claims.Authority != dal.TENANT_ADMIN {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": "wrong user authority",
		})
	}

	tenantID := "0"
	if claims.Authority == dal.TENANT_ADMIN {
		tenantID = claims.TenantID
	} else if createDictReq.TenantID != nil && *createDictReq.TenantID != "" {
		tenantID = *createDictReq.TenantID
	}

	var dict = model.SysDictRecord{
		ID:        uuid.New(),
		DictCode:  createDictReq.DictCode,
		DictValue: createDictReq.DictValue,
		TenantID:  tenantID,
		Category:  createDictReq.Category,
		CreatedAt: time.Now().UTC(),
		Remark:    createDictReq.Remark,
	}

	err := dal.CreateDict(&dict, nil)
	if err != nil {
		logrus.Error(err)
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": err.Error(),
		})
	}
	return err
}

func (*Dict) CreateDictLanguage(createDictLanguage *model.CreateDictLanguageReq, claims *utils.UserClaims) error {
	// 验证sys_dict的id是否存在
	d, err := dal.GetDictById(createDictLanguage.DictId)
	if err != nil {
		logrus.Error(err)
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": err.Error(),
		})
	}
	if d.TenantID == "0" && claims.Authority != dal.SYS_ADMIN {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": "only SYS_ADMIN can modify global dict",
		})
	}
	if d.TenantID != "0" && claims.Authority != dal.SYS_ADMIN && (claims.Authority != dal.TENANT_ADMIN || claims.TenantID != d.TenantID) {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": "wrong user authority",
		})
	}
	// 创建 sys_dict_language
	var dictLanguage = model.SysDictLanguage{}

	dictLanguage.ID = uuid.New()
	dictLanguage.DictID = createDictLanguage.DictId
	dictLanguage.LanguageCode = createDictLanguage.LanguageCode
	dictLanguage.Translation = createDictLanguage.Translation

	err = dal.CreateDictLanguage(&dictLanguage, nil)
	if err != nil {
		logrus.Error(err)
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": err.Error(),
		})
	}
	return err
}

func (*Dict) DeleteDict(id string, claims *utils.UserClaims) error {
	d, err := dal.GetDictById(id)
	if err != nil {
		logrus.Error(err)
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": err.Error(),
		})
	}
	if d.TenantID == "0" && claims.Authority != dal.SYS_ADMIN {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": "only SYS_ADMIN can delete global dict",
		})
	}
	if d.TenantID != "0" && claims.Authority != dal.SYS_ADMIN && (claims.Authority != dal.TENANT_ADMIN || claims.TenantID != d.TenantID) {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": "wrong user authority",
		})
	}

	err = dal.DeleteDictById(id, nil)
	if err != nil {
		logrus.Error(err)
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": err.Error(),
		})
	}
	return err
}

func (*Dict) DeleteDictLanguage(id string, claims *utils.UserClaims) error {
	lang, err := dal.GetDictLanguageById(id)
	if err != nil {
		logrus.Error(err)
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": err.Error(),
		})
	}
	d, err := dal.GetDictById(lang.DictID)
	if err != nil {
		logrus.Error(err)
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": err.Error(),
		})
	}

	if d.TenantID == "0" && claims.Authority != dal.SYS_ADMIN {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": "only SYS_ADMIN can modify global dict",
		})
	}
	if d.TenantID != "0" && claims.Authority != dal.SYS_ADMIN && (claims.Authority != dal.TENANT_ADMIN || claims.TenantID != d.TenantID) {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": "wrong user authority",
		})
	}

	err = dal.DeleteDictLanguageById(id)
	if err != nil {
		logrus.Error(err)
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": err.Error(),
		})
	}

	return nil
}

func (*Dict) GetDict(params *model.DictListReq, lang string, claims *utils.UserClaims) (list []model.DictListRsp, err error) {
	if claims == nil {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": "claims is nil",
		})
	}

	lanCode := normalizeDictLanguageCode(utils.FormatLangCode(lang))
	if params.LanguageCode != nil && *params.LanguageCode != "" {
		lanCode = normalizeDictLanguageCode(*params.LanguageCode)
	}

	tenantIDs := []string{"0"}
	preferTenant := "0"
	if claims.TenantID != "" && claims.TenantID != "0" {
		tenantIDs = append(tenantIDs, claims.TenantID)
		preferTenant = claims.TenantID
	}

	list, err = dal.GetDictEnum(params.DictCode, lanCode, tenantIDs, preferTenant)
	if err != nil {
		logrus.Error(err)
		return list, errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": err.Error(),
		})
	}
	return list, nil
}

// 获取协议接入下拉菜单
func (*Dict) GetProtocolMenu(protocolMenuReq *model.ProtocolMenuReq) (reqData []map[string]interface{}, err error) {
	if protocolMenuReq.LanguageCode == nil {
		protocolMenuReq.LanguageCode = StringPtr("zh")
	}
	var reqDataList []map[string]interface{}
	tenantIDs := []string{"0"}
	preferTenant := "0"
	if protocolMenuReq.LanguageCode == nil {
		protocolMenuReq.LanguageCode = StringPtr("zh")
	}

	dict1, err := dal.GetDictEnum("DRIECT_ATTACHED_PROTOCOL", *protocolMenuReq.LanguageCode, tenantIDs, preferTenant)
	if err != nil {
		logrus.Error(err)
		return reqDataList, err
	}
	dict2, err := dal.GetDictEnum("GATEWAY_PROTOCOL", *protocolMenuReq.LanguageCode, tenantIDs, preferTenant)
	if err != nil {
		logrus.Error(err)
		return reqDataList, err
	}
	for _, v := range dict1 {
		reqDataList = append(reqDataList, map[string]interface{}{
			"dict_value":  v.DictValue,
			"translation": v.Translation,
			"device_type": "1",
		})
	}
	for _, v := range dict2 {
		reqDataList = append(reqDataList, map[string]interface{}{
			"dict_value":  v.DictValue,
			"translation": v.Translation,
			"device_type": "2",
		})
	}
	for _, v := range dict2 {
		reqDataList = append(reqDataList, map[string]interface{}{
			"dict_value":  v.DictValue,
			"translation": v.Translation,
			"device_type": "3",
		})
	}
	return reqDataList, nil
}

func (*Dict) GetDictListByPage(params *model.GetDictLisyByPageReq, claims *utils.UserClaims) (map[string]interface{}, error) {
	tenantIDs, _, err := resolveTenantScope(params.Scope, params.TenantID, claims)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": err.Error(),
		})
	}

	total, list, err := dal.GetDictListByPage(params, claims, tenantIDs)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": err.Error(),
		})
	}
	dictListRspMap := make(map[string]interface{})
	dictListRspMap["total"] = total
	dictListRspMap["list"] = list
	return dictListRspMap, nil

}

func (*Dict) GetDictLanguageListById(id string) ([]*model.SysDictLanguage, error) {
	data, err := dal.GetDictLanguageListByDictId(id)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": err.Error(),
		})
	}
	return data, err
}

func (*Dict) UpdateDictColumn(id string, req *model.UpdateDictReq, claims *utils.UserClaims) error {
	d, err := dal.GetDictById(id)
	if err != nil {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": err.Error(),
		})
	}
	if d.TenantID == "0" && claims.Authority != dal.SYS_ADMIN {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": "only SYS_ADMIN can modify global dict",
		})
	}
	if d.TenantID != "0" && claims.Authority != dal.SYS_ADMIN && (claims.Authority != dal.TENANT_ADMIN || claims.TenantID != d.TenantID) {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": "wrong user authority",
		})
	}

	updates := map[string]interface{}{}
	if req.DictCode != nil {
		updates["dict_code"] = *req.DictCode
	}
	if req.DictValue != nil {
		updates["dict_value"] = *req.DictValue
	}
	if req.Category != nil {
		updates["category"] = *req.Category
	}
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}
	if len(updates) == 0 {
		return nil
	}
	if err := dal.UpdateDictById(id, updates); err != nil {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": err.Error(),
		})
	}
	return nil
}

func (*Dict) UpsertDictLanguage(req *model.UpsertDictLanguageReq, claims *utils.UserClaims) error {
	d, err := dal.GetDictById(req.DictId)
	if err != nil {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": err.Error(),
		})
	}
	if d.TenantID == "0" && claims.Authority != dal.SYS_ADMIN {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": "only SYS_ADMIN can modify global dict",
		})
	}
	if d.TenantID != "0" && claims.Authority != dal.SYS_ADMIN && (claims.Authority != dal.TENANT_ADMIN || claims.TenantID != d.TenantID) {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": "wrong user authority",
		})
	}

	var dictLanguage = model.SysDictLanguage{
		ID:           uuid.New(),
		DictID:       req.DictId,
		LanguageCode: req.LanguageCode,
		Translation:  req.Translation,
	}
	if err := dal.UpsertDictLanguage(&dictLanguage); err != nil {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": err.Error(),
		})
	}
	return nil
}

func (*Dict) GetDictCategories(req *model.DictCategoriesReq, claims *utils.UserClaims) ([]model.DictCategoryItem, error) {
	tenantIDs, _, err := resolveTenantScope(req.Scope, req.TenantID, claims)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": err.Error(),
		})
	}
	data, err := dal.GetDictCategories(claims, tenantIDs)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": err.Error(),
		})
	}
	return data, nil
}

func (*Dict) GetDictValue(req *model.DictValueReq, lang string, claims *utils.UserClaims) (string, error) {
	lanCode := normalizeDictLanguageCode(utils.FormatLangCode(lang))
	if req.LanguageCode != nil && *req.LanguageCode != "" {
		lanCode = normalizeDictLanguageCode(*req.LanguageCode)
	}
	tenantIDs := []string{"0"}
	preferTenant := "0"
	if claims != nil && claims.TenantID != "" && claims.TenantID != "0" {
		tenantIDs = append(tenantIDs, claims.TenantID)
		preferTenant = claims.TenantID
	}
	val, err := dal.GetDictTranslation(req.DictCode, req.DictValue, lanCode, tenantIDs, preferTenant)
	if err != nil {
		return "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"err": err.Error(),
		})
	}
	return val, nil
}

func normalizeDictLanguageCode(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return code
	}
	// 兼容 Accept-Language 解析后的 zh_CN/en_US；sys_dict_language 通常使用 zh/en
	if strings.Contains(code, "_") {
		parts := strings.Split(code, "_")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}
	return code
}

func resolveTenantScope(scope *string, tenantIDParam *string, claims *utils.UserClaims) (tenantIDs []string, preferTenant string, err error) {
	if claims == nil {
		return nil, "", fmt.Errorf("claims is nil")
	}
	s := "all"
	if scope != nil && *scope != "" {
		s = *scope
	}

	if claims.Authority == dal.SYS_ADMIN {
		targetTenant := "0"
		if tenantIDParam != nil && *tenantIDParam != "" {
			targetTenant = *tenantIDParam
		}
		switch s {
		case "global":
			return []string{"0"}, "0", nil
		case "tenant":
			if targetTenant == "" {
				return nil, "", fmt.Errorf("tenant_id is required for scope=tenant")
			}
			return []string{targetTenant}, targetTenant, nil
		case "all":
			if targetTenant != "" && targetTenant != "0" {
				return []string{"0", targetTenant}, targetTenant, nil
			}
			return []string{"0"}, "0", nil
		default:
			return nil, "", fmt.Errorf("invalid scope")
		}
	}

	if claims.Authority == dal.TENANT_ADMIN {
		targetTenant := claims.TenantID
		switch s {
		case "global":
			return []string{"0"}, targetTenant, nil
		case "tenant":
			return []string{targetTenant}, targetTenant, nil
		case "all":
			return []string{"0", targetTenant}, targetTenant, nil
		default:
			return nil, "", fmt.Errorf("invalid scope")
		}
	}
	return nil, "", fmt.Errorf("authority exception")
}
