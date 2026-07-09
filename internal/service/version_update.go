package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"project/internal/model"
	"project/pkg/errcode"
	global "project/pkg/global"

	"github.com/go-basic/uuid"
	"gorm.io/gorm"
)

type VersionUpdate struct{}

func normalizeVersionUpdateProject(project string) (string, error) {
	value := strings.TrimSpace(strings.ToUpper(project))
	switch value {
	case model.VersionUpdateProjectMobile, model.VersionUpdateProjectCloudFrontend, model.VersionUpdateProjectCloudBackend:
		return value, nil
	default:
		return "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "项目类型不正确"})
	}
}

func normalizeVersionUpdateText(value, fieldName string, maxRunes int) (string, error) {
	text := strings.TrimSpace(value)
	if text == "" {
		return "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": fieldName + "不能为空"})
	}
	if utf8.RuneCountInString(text) > maxRunes {
		return "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": fieldName + "长度超出限制"})
	}
	return text, nil
}

func parseVersionUpdateDate(value string) (time.Time, error) {
	text := strings.TrimSpace(value)
	if text == "" {
		return time.Time{}, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "日期不能为空"})
	}
	parsed, err := time.ParseInLocation("2006-01-02", text, time.Local)
	if err != nil {
		return time.Time{}, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "日期格式必须为 YYYY-MM-DD"})
	}
	return parsed, nil
}

func ensureVersionUpdateUnique(ctx context.Context, project, versionNo string, releaseDate time.Time, excludeID string) error {
	q := global.DB.WithContext(ctx).
		Model(&model.VersionUpdateRecord{}).
		Where("project = ? AND version_no = ? AND DATE(release_date) = ?", project, versionNo, releaseDate.Format("2006-01-02"))
	if excludeID != "" {
		q = q.Where("id <> ?", excludeID)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if total > 0 {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "相同项目、版本号和日期的更新记录已存在"})
	}
	return nil
}

func versionUpdateResp(row model.VersionUpdateRecord) model.VersionUpdateResp {
	return model.VersionUpdateResp{
		ID:            row.ID,
		Project:       row.Project,
		VersionNo:     row.VersionNo,
		ReleaseDate:   row.ReleaseDate.In(time.Local).Format("2006-01-02"),
		UpdateContent: row.UpdateContent,
		Source:        row.Source,
		SourceRef:     row.SourceRef,
		CreatedAt:     row.CreatedAt.In(time.Local).Format("2006-01-02 15:04:05"),
		UpdatedAt:     row.UpdatedAt.In(time.Local).Format("2006-01-02 15:04:05"),
	}
}

func (*VersionUpdate) Create(ctx context.Context, req model.VersionUpdateCreateReq) (*model.VersionUpdateResp, error) {
	project, err := normalizeVersionUpdateProject(req.Project)
	if err != nil {
		return nil, err
	}
	versionNo, err := normalizeVersionUpdateText(req.VersionNo, "版本号", 64)
	if err != nil {
		return nil, err
	}
	updateContent, err := normalizeVersionUpdateText(req.UpdateContent, "更新内容", 10000)
	if err != nil {
		return nil, err
	}
	releaseDate, err := parseVersionUpdateDate(req.ReleaseDate)
	if err != nil {
		return nil, err
	}
	if err := ensureVersionUpdateUnique(ctx, project, versionNo, releaseDate, ""); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	row := &model.VersionUpdateRecord{
		ID:            uuid.New(),
		Project:       project,
		VersionNo:     versionNo,
		ReleaseDate:   releaseDate,
		UpdateContent: updateContent,
		Source:        model.VersionUpdateSourceManual,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := global.DB.WithContext(ctx).Create(row).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	resp := versionUpdateResp(*row)
	return &resp, nil
}

func (*VersionUpdate) Get(ctx context.Context, id string) (*model.VersionUpdateResp, error) {
	var row model.VersionUpdateRecord
	if err := global.DB.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errcode.New(errcode.CodeNotFound)
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	resp := versionUpdateResp(row)
	return &resp, nil
}

func (*VersionUpdate) Update(ctx context.Context, id string, req model.VersionUpdateUpdateReq) (*model.VersionUpdateResp, error) {
	var row model.VersionUpdateRecord
	if err := global.DB.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errcode.New(errcode.CodeNotFound)
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	project := row.Project
	versionNo := row.VersionNo
	releaseDate := row.ReleaseDate
	updateContent := row.UpdateContent

	updates := map[string]interface{}{}
	if req.Project != nil {
		normalized, err := normalizeVersionUpdateProject(*req.Project)
		if err != nil {
			return nil, err
		}
		project = normalized
		updates["project"] = normalized
	}
	if req.VersionNo != nil {
		normalized, err := normalizeVersionUpdateText(*req.VersionNo, "版本号", 64)
		if err != nil {
			return nil, err
		}
		versionNo = normalized
		updates["version_no"] = normalized
	}
	if req.ReleaseDate != nil {
		parsed, err := parseVersionUpdateDate(*req.ReleaseDate)
		if err != nil {
			return nil, err
		}
		releaseDate = parsed
		updates["release_date"] = parsed
	}
	if req.UpdateContent != nil {
		normalized, err := normalizeVersionUpdateText(*req.UpdateContent, "更新内容", 10000)
		if err != nil {
			return nil, err
		}
		updateContent = normalized
		updates["update_content"] = normalized
	}

	if len(updates) == 0 {
		resp := versionUpdateResp(row)
		return &resp, nil
	}

	if err := ensureVersionUpdateUnique(ctx, project, versionNo, releaseDate, row.ID); err != nil {
		return nil, err
	}
	updates["updated_at"] = time.Now().UTC()

	if err := global.DB.WithContext(ctx).
		Model(&model.VersionUpdateRecord{}).
		Where("id = ?", row.ID).
		Updates(updates).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	row.Project = project
	row.VersionNo = versionNo
	row.ReleaseDate = releaseDate
	row.UpdateContent = updateContent
	if updatedAt, ok := updates["updated_at"].(time.Time); ok {
		row.UpdatedAt = updatedAt
	}
	resp := versionUpdateResp(row)
	return &resp, nil
}

func (*VersionUpdate) Delete(ctx context.Context, id string) error {
	result := global.DB.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).Delete(&model.VersionUpdateRecord{})
	if result.Error != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": result.Error.Error()})
	}
	if result.RowsAffected == 0 {
		return errcode.New(errcode.CodeNotFound)
	}
	return nil
}

func (*VersionUpdate) List(ctx context.Context, req model.VersionUpdateListReq) (*model.VersionUpdateListResp, error) {
	q := global.DB.WithContext(ctx).Model(&model.VersionUpdateRecord{})

	if req.Project != nil && strings.TrimSpace(*req.Project) != "" {
		project, err := normalizeVersionUpdateProject(*req.Project)
		if err != nil {
			return nil, err
		}
		q = q.Where("project = ?", project)
	}
	if req.VersionNo != nil && strings.TrimSpace(*req.VersionNo) != "" {
		q = q.Where("LOWER(version_no) LIKE ?", "%"+strings.ToLower(strings.TrimSpace(*req.VersionNo))+"%")
	}
	if req.Keyword != nil && strings.TrimSpace(*req.Keyword) != "" {
		q = q.Where("LOWER(update_content) LIKE ?", "%"+strings.ToLower(strings.TrimSpace(*req.Keyword))+"%")
	}
	if req.StartDate != nil && strings.TrimSpace(*req.StartDate) != "" {
		start, err := parseVersionUpdateDate(*req.StartDate)
		if err != nil {
			return nil, err
		}
		q = q.Where("DATE(release_date) >= ?", start.Format("2006-01-02"))
	}
	if req.EndDate != nil && strings.TrimSpace(*req.EndDate) != "" {
		end, err := parseVersionUpdateDate(*req.EndDate)
		if err != nil {
			return nil, err
		}
		q = q.Where("DATE(release_date) <= ?", end.Format("2006-01-02"))
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	rows := make([]model.VersionUpdateRecord, 0, req.PageSize)
	if err := q.Order("release_date DESC").Order("created_at DESC").Order("id ASC").
		Limit(req.PageSize).
		Offset((req.Page - 1) * req.PageSize).
		Find(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	list := make([]model.VersionUpdateResp, 0, len(rows))
	for _, row := range rows {
		list = append(list, versionUpdateResp(row))
	}

	return &model.VersionUpdateListResp{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}
