package dal

import (
	"context"

	model "project/internal/model"
	global "project/pkg/global"

	"gorm.io/gorm"
)

func CreateFile(ctx context.Context, f *model.File) error {
	return global.DB.WithContext(ctx).Create(f).Error
}

func GetFileByID(ctx context.Context, id string) (*model.File, error) {
	var f model.File
	err := global.DB.WithContext(ctx).First(&f, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

type ListFilesResult struct {
	Total int64
	List  []model.File
}

func ListFilesByTenant(ctx context.Context, tenantID string, req *model.GetFileListByPageReq) (*ListFilesResult, error) {
	db := global.DB.WithContext(ctx).Model(&model.File{}).Where("tenant_id = ?", tenantID)

	if req.UploadedBy != nil && *req.UploadedBy != "" {
		db = db.Where("uploaded_by = ?", *req.UploadedBy)
	}
	if req.Keyword != nil && *req.Keyword != "" {
		kw := "%" + *req.Keyword + "%"
		db = db.Where("(file_name ILIKE ? OR original_file_name ILIKE ? OR file_path ILIKE ?)", kw, kw, kw)
	}
	if req.BizType != nil && *req.BizType != "" {
		db = db.Where("biz_type = ?", *req.BizType)
	}
	if req.StorageLocation != nil && *req.StorageLocation != "" {
		db = db.Where("storage_location = ?", *req.StorageLocation)
	}
	if req.StartTime != nil {
		db = db.Where("uploaded_at >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		db = db.Where("uploaded_at <= ?", *req.EndTime)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	var list []model.File
	if err := db.Order("uploaded_at DESC").Limit(req.PageSize).Offset((req.Page - 1) * req.PageSize).Find(&list).Error; err != nil {
		return nil, err
	}

	return &ListFilesResult{Total: total, List: list}, nil
}
