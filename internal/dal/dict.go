package dal

import (
	"fmt"

	model "project/internal/model"
	query "project/internal/query"
	utils "project/pkg/utils"

	"gorm.io/gorm"
)

func dictDB(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return query.SysDict.UnderlyingDB()
}

func CreateDict(dict *model.SysDictRecord, tx *gorm.DB) error {
	return dictDB(tx).Create(dict).Error
}

func GetDictById(dictId string) (*model.SysDictRecord, error) {
	var dict model.SysDictRecord
	err := dictDB(nil).Where("id = ?", dictId).First(&dict).Error
	if err != nil {
		return nil, err
	}
	return &dict, nil
}

func DeleteDictById(dictId string, tx *gorm.DB) error {
	return dictDB(tx).Where("id = ?", dictId).Delete(&model.SysDictRecord{}).Error
}

func GetDictListByCode(dictCode string, tenantIDs []string, preferTenant string) ([]*model.SysDictRecord, error) {
	var dict []*model.SysDictRecord
	db := dictDB(nil).
		Where("dict_code = ?", dictCode).
		Where("tenant_id IN ?", tenantIDs).
		Order(gorm.Expr("CASE WHEN tenant_id = ? THEN 0 ELSE 1 END, created_at DESC", preferTenant))
	if err := db.Find(&dict).Error; err != nil {
		return nil, err
	}
	return dict, nil
}

func GetDictListByPage(dictListReq *model.GetDictLisyByPageReq, claims *utils.UserClaims, tenantIDs []string) (count int64, dictList []*model.SysDictRecord, err error) {
	if claims.Authority != SYS_ADMIN && claims.Authority != TENANT_ADMIN {
		return 0, nil, fmt.Errorf("authority exception")
	}

	db := dictDB(nil).Model(&model.SysDictRecord{}).Where("tenant_id IN ?", tenantIDs)

	if dictListReq.Category != nil {
		db = db.Where("category = ?", *dictListReq.Category)
	}
	if dictListReq.DictCode != nil {
		db = db.Where("dict_code = ?", *dictListReq.DictCode)
	}
	if dictListReq.DictValue != nil {
		db = db.Where("dict_value ILIKE ?", "%"+*dictListReq.DictValue+"%")
	}

	if err = db.Count(&count).Error; err != nil {
		return 0, nil, err
	}

	err = db.
		Order("created_at DESC").
		Offset((dictListReq.Page - 1) * dictListReq.PageSize).
		Limit(dictListReq.PageSize).
		Find(&dictList).Error
	return count, dictList, err
}

func GetDictCategories(claims *utils.UserClaims, tenantIDs []string) ([]model.DictCategoryItem, error) {
	if claims.Authority != SYS_ADMIN && claims.Authority != TENANT_ADMIN {
		return nil, fmt.Errorf("authority exception")
	}
	var rows []model.DictCategoryItem
	err := dictDB(nil).
		Table("sys_dict").
		Select("category, COUNT(1) as count").
		Where("tenant_id IN ?", tenantIDs).
		Group("category").
		Order("category").
		Scan(&rows).Error
	return rows, err
}

func GetDictEnum(dictCode, languageCode string, tenantIDs []string, preferTenant string) ([]model.DictListRsp, error) {
	type row struct {
		DictValue   string  `gorm:"column:dict_value"`
		Translation *string `gorm:"column:translation"`
		TenantID    string  `gorm:"column:tenant_id"`
	}
	var rows []row
	err := dictDB(nil).
		Table("sys_dict sd").
		Select("sd.dict_value, sd.tenant_id, sdl.translation").
		Joins("LEFT JOIN sys_dict_language sdl ON sdl.dict_id = sd.id AND sdl.language_code = ?", languageCode).
		Where("sd.dict_code = ?", dictCode).
		Where("sd.tenant_id IN ?", tenantIDs).
		Order(gorm.Expr("CASE WHEN sd.tenant_id = ? THEN 0 ELSE 1 END, sd.created_at DESC", preferTenant)).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	// 按 dict_value 去重：优先租户字典，其次系统全局
	seen := make(map[string]bool, len(rows))
	var result []model.DictListRsp
	for _, r := range rows {
		if seen[r.DictValue] {
			continue
		}
		seen[r.DictValue] = true
		trans := ""
		if r.Translation != nil {
			trans = *r.Translation
		}
		result = append(result, model.DictListRsp{
			DictValue:   r.DictValue,
			Translation: trans,
		})
	}
	return result, nil
}

func GetDictTranslation(dictCode, dictValue, languageCode string, tenantIDs []string, preferTenant string) (string, error) {
	type row struct {
		Translation *string `gorm:"column:translation"`
	}
	var rows []row
	err := dictDB(nil).
		Table("sys_dict sd").
		Select("sdl.translation").
		Joins("LEFT JOIN sys_dict_language sdl ON sdl.dict_id = sd.id AND sdl.language_code = ?", languageCode).
		Where("sd.dict_code = ?", dictCode).
		Where("sd.dict_value = ?", dictValue).
		Where("sd.tenant_id IN ?", tenantIDs).
		Order(gorm.Expr("CASE WHEN sd.tenant_id = ? THEN 0 ELSE 1 END, sd.created_at DESC", preferTenant)).
		Limit(1).
		Scan(&rows).Error
	if err != nil {
		return "", err
	}
	if len(rows) == 0 || rows[0].Translation == nil {
		return "", nil
	}
	return *rows[0].Translation, nil
}

func UpdateDictById(dictId string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	return dictDB(nil).Model(&model.SysDictRecord{}).Where("id = ?", dictId).Updates(updates).Error
}

// Deprecated: 请使用 GetDictEnum（支持租户/全局优先级）.
func GetDictLanguageByDictCodeAndLanguageCode(dictCode, languageCode string) ([]map[string]interface{}, error) {
	var data []map[string]interface{}
	sd := query.SysDict
	sdl := query.SysDictLanguage
	err := sd.Select(sd.DictValue, sdl.Translation).
		LeftJoin(sdl, sdl.DictID.EqCol(sd.ID)).
		Where(sd.DictCode.Eq(dictCode), sdl.LanguageCode.Eq(languageCode)).
		Scan(&data)
	if err != nil {
		return nil, err
	}
	return data, err
}
