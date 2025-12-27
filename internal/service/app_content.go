package service

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"time"

	"project/internal/model"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/google/uuid"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AppContent APP内容管理（单页/FAQ/用户反馈）
type AppContent struct{}

const (
	contentKeyUserPolicy    = "user_policy"
	contentKeyPrivacyPolicy = "privacy_policy"

	langZhCN = "zh-CN"
	langEnUS = "en-US"
)

var md = goldmark.New(goldmark.WithExtensions(extension.GFM))

func normalizeLang(lang string) string {
	l := strings.TrimSpace(lang)
	if strings.EqualFold(l, "zh-cn") {
		return langZhCN
	}
	return langEnUS
}

func markdownToHTML(markdownText string) string {
	if strings.TrimSpace(markdownText) == "" {
		return ""
	}
	var buf bytes.Buffer
	_ = md.Convert([]byte(markdownText), &buf)
	return buf.String()
}

func validateContentKey(k string) error {
	switch strings.TrimSpace(k) {
	case contentKeyUserPolicy, contentKeyPrivacyPolicy:
		return nil
	default:
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"field": "content_key",
			"allow": []string{contentKeyUserPolicy, contentKeyPrivacyPolicy},
		})
	}
}

func resolveTenantID(ctx context.Context, tenantHeader string) (string, error) {
	tid := strings.TrimSpace(tenantHeader)
	if tid != "" {
		return tid, nil
	}

	// 默认第一个租户：取任意 tenant_admin 的 tenant_id（按创建时间/ID稳定排序）
	var defaultTenantID string
	err := global.DB.WithContext(ctx).
		Table("users").
		Select("tenant_id").
		Where("authority = ? AND tenant_id IS NOT NULL AND tenant_id <> ''", "TENANT_ADMIN").
		Order("created_at ASC NULLS LAST, id ASC").
		Limit(1).
		Scan(&defaultTenantID).Error
	if err != nil {
		return "", errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if strings.TrimSpace(defaultTenantID) == "" {
		return "", errcode.New(errcode.CodeNotFound)
	}
	return defaultTenantID, nil
}

type appRef struct {
	ID    string `gorm:"column:id"`
	AppID string `gorm:"column:appid"`
}

func getAppByAppID(ctx context.Context, tenantID, appid string) (*appRef, error) {
	var a appRef
	if err := global.DB.WithContext(ctx).Table("apps").
		Select("id, appid").
		Where("tenant_id = ? AND appid = ?", tenantID, strings.TrimSpace(appid)).
		Limit(1).
		Scan(&a).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if a.ID == "" {
		return nil, errcode.New(errcode.CodeNotFound)
	}
	return &a, nil
}

func ensureAppOwned(ctx context.Context, tenantID, appID string) error {
	var id string
	if err := global.DB.WithContext(ctx).Table("apps").
		Select("id").
		Where("tenant_id = ? AND id = ?", tenantID, strings.TrimSpace(appID)).
		Limit(1).
		Scan(&id).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if id == "" {
		return errcode.New(errcode.CodeNotFound)
	}
	return nil
}

// ---------------------------------------------------------------------------
// APP端（无需登录）：单页内容
// ---------------------------------------------------------------------------

func (*AppContent) GetPageForApp(ctx context.Context, tenantHeader, appid, contentKey, lang string) (*model.AppContentPageResp, error) {
	if err := validateContentKey(contentKey); err != nil {
		return nil, err
	}

	tenantID, err := resolveTenantID(ctx, tenantHeader)
	if err != nil {
		return nil, err
	}
	lang = normalizeLang(lang)

	app, err := getAppByAppID(ctx, tenantID, appid)
	if err != nil {
		return nil, err
	}

	type pageRow struct {
		ID        string     `gorm:"column:id"`
		Published bool       `gorm:"column:published"`
		UpdatedAt *time.Time `gorm:"column:updated_at"`
	}
	var p pageRow
	if err := global.DB.WithContext(ctx).
		Table("app_content_pages").
		Select("id, published, updated_at").
		Where("tenant_id = ? AND app_id = ? AND content_key = ?", tenantID, app.ID, contentKey).
		Limit(1).
		Scan(&p).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if p.ID == "" || !p.Published {
		return nil, errcode.New(errcode.CodeNotFound)
	}

	type i18nRow struct {
		ID      string     `gorm:"column:id"`
		Title   *string    `gorm:"column:title"`
		MD      *string    `gorm:"column:content_markdown"`
		HTML    *string    `gorm:"column:content_html"`
		Updated *time.Time `gorm:"column:updated_at"`
	}

	var tr i18nRow
	db := global.DB.WithContext(ctx).Table("app_content_page_i18n").Where("page_id = ? AND lang = ?", p.ID, lang)
	if err := db.Select("id, title, content_markdown, content_html, updated_at").Limit(1).Scan(&tr).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if tr.ID == "" && lang != langEnUS {
		if err := global.DB.WithContext(ctx).Table("app_content_page_i18n").
			Select("id, title, content_markdown, content_html, updated_at").
			Where("page_id = ? AND lang = ?", p.ID, langEnUS).
			Limit(1).
			Scan(&tr).Error; err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
		lang = langEnUS
	}
	if tr.ID == "" {
		return nil, errcode.New(errcode.CodeNotFound)
	}

	title := ""
	if tr.Title != nil {
		title = *tr.Title
	}
	contentMD := ""
	if tr.MD != nil {
		contentMD = *tr.MD
	}
	contentHTML := ""
	if tr.HTML != nil {
		contentHTML = *tr.HTML
	}

	updatedAt := formatLocalTime(p.UpdatedAt)
	if tr.Updated != nil {
		updatedAt = formatLocalTime(tr.Updated)
	}

	return &model.AppContentPageResp{
		ContentKey:      contentKey,
		Lang:            lang,
		Title:           title,
		ContentMarkdown: contentMD,
		ContentHTML:     contentHTML,
		UpdatedAt:       updatedAt,
	}, nil
}

// ---------------------------------------------------------------------------
// APP端（无需登录）：FAQ
// ---------------------------------------------------------------------------

func (*AppContent) ListFaqsForApp(ctx context.Context, tenantHeader, appid, lang string, page, pageSize int) (*model.AppFaqListResp, error) {
	tenantID, err := resolveTenantID(ctx, tenantHeader)
	if err != nil {
		return nil, err
	}
	lang = normalizeLang(lang)

	app, err := getAppByAppID(ctx, tenantID, appid)
	if err != nil {
		return nil, err
	}

	db := global.DB.WithContext(ctx).Table("app_faq f").
		Where("f.tenant_id = ? AND f.app_id = ? AND f.published = true", tenantID, app.ID)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	offset := (page - 1) * pageSize
	type row struct {
		ID         string     `gorm:"column:id"`
		IsPinned   bool       `gorm:"column:is_pinned"`
		Sort       int        `gorm:"column:sort"`
		UpdatedAt  *time.Time `gorm:"column:updated_at"`
		Question   *string    `gorm:"column:question"`
		AnswerMD   *string    `gorm:"column:answer_markdown"`
		AnswerHTML *string    `gorm:"column:answer_html"`
	}
	rows := make([]row, 0, pageSize)

	if err := db.Select(`
			f.id, f.is_pinned, f.sort, f.updated_at,
			COALESCE(i_lang.question, i_en.question) AS question,
			COALESCE(i_lang.answer_markdown, i_en.answer_markdown) AS answer_markdown,
			COALESCE(i_lang.answer_html, i_en.answer_html) AS answer_html
		`).
		Joins("LEFT JOIN app_faq_i18n i_lang ON i_lang.faq_id = f.id AND i_lang.lang = ?", lang).
		Joins("LEFT JOIN app_faq_i18n i_en ON i_en.faq_id = f.id AND i_en.lang = ?", langEnUS).
		Order("f.is_pinned DESC, f.sort DESC, f.updated_at DESC").
		Offset(offset).
		Limit(pageSize).
		Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	list := make([]model.AppFaqItemResp, 0, len(rows))
	for _, r := range rows {
		q := ""
		if r.Question != nil {
			q = *r.Question
		}
		mdText := ""
		if r.AnswerMD != nil {
			mdText = *r.AnswerMD
		}
		htmlText := ""
		if r.AnswerHTML != nil {
			htmlText = *r.AnswerHTML
		}
		list = append(list, model.AppFaqItemResp{
			ID:             r.ID,
			Question:       q,
			AnswerMarkdown: mdText,
			AnswerHTML:     htmlText,
			IsPinned:       r.IsPinned,
			Sort:           r.Sort,
			UpdatedAt:      formatLocalTime(r.UpdatedAt),
		})
	}

	return &model.AppFaqListResp{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// ---------------------------------------------------------------------------
// 管理端：单页内容
// ---------------------------------------------------------------------------

func (*AppContent) AdminGetPage(ctx context.Context, claims *utils.UserClaims, appID, contentKey, lang string) (*model.AdminContentPageResp, error) {
	if err := validateContentKey(contentKey); err != nil {
		return nil, err
	}
	if err := ensureAppOwned(ctx, claims.TenantID, appID); err != nil {
		return nil, err
	}
	lang = normalizeLang(lang)

	type pageRow struct {
		ID          string     `gorm:"column:id"`
		Published   bool       `gorm:"column:published"`
		PublishedAt *time.Time `gorm:"column:published_at"`
		UpdatedAt   *time.Time `gorm:"column:updated_at"`
	}
	var p pageRow
	if err := global.DB.WithContext(ctx).
		Table("app_content_pages").
		Select("id, published, published_at, updated_at").
		Where("tenant_id = ? AND app_id = ? AND content_key = ?", claims.TenantID, appID, contentKey).
		Limit(1).
		Scan(&p).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	if p.ID == "" {
		// 读取时自动创建草稿记录，方便前端直接编辑
		now := time.Now().UTC()
		id := uuid.NewString()
		if err := global.DB.WithContext(ctx).Table("app_content_pages").Create(map[string]interface{}{
			"id":          id,
			"tenant_id":   claims.TenantID,
			"app_id":      appID,
			"content_key": contentKey,
			"published":   false,
			"created_by":  claims.ID,
			"updated_by":  claims.ID,
			"created_at":  now,
			"updated_at":  now,
		}).Error; err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
		p.ID = id
		p.Published = false
		p.UpdatedAt = &now
	}

	type i18nRow struct {
		ID      string     `gorm:"column:id"`
		Title   *string    `gorm:"column:title"`
		MD      *string    `gorm:"column:content_markdown"`
		HTML    *string    `gorm:"column:content_html"`
		Updated *time.Time `gorm:"column:updated_at"`
	}
	var tr i18nRow
	if err := global.DB.WithContext(ctx).Table("app_content_page_i18n").
		Select("id, title, content_markdown, content_html, updated_at").
		Where("page_id = ? AND lang = ?", p.ID, lang).
		Limit(1).
		Scan(&tr).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if tr.ID == "" && lang != langEnUS {
		if err := global.DB.WithContext(ctx).Table("app_content_page_i18n").
			Select("id, title, content_markdown, content_html, updated_at").
			Where("page_id = ? AND lang = ?", p.ID, langEnUS).
			Limit(1).
			Scan(&tr).Error; err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
	}

	title := ""
	if tr.Title != nil {
		title = *tr.Title
	}
	contentMD := ""
	if tr.MD != nil {
		contentMD = *tr.MD
	}
	contentHTML := ""
	if tr.HTML != nil {
		contentHTML = *tr.HTML
	}

	publishedAt := (*string)(nil)
	if p.PublishedAt != nil {
		s := formatLocalTime(p.PublishedAt)
		publishedAt = &s
	}
	updatedAt := formatLocalTime(p.UpdatedAt)
	if tr.Updated != nil {
		updatedAt = formatLocalTime(tr.Updated)
	}

	return &model.AdminContentPageResp{
		AppID:           appID,
		ContentKey:      contentKey,
		Published:       p.Published,
		PublishedAt:     publishedAt,
		Lang:            lang,
		Title:           title,
		ContentMarkdown: contentMD,
		ContentHTML:     contentHTML,
		UpdatedAt:       updatedAt,
	}, nil
}

func (*AppContent) AdminUpsertPage(ctx context.Context, claims *utils.UserClaims, contentKey string, req model.AdminContentPageUpsertReq) error {
	if err := validateContentKey(contentKey); err != nil {
		return err
	}
	if err := ensureAppOwned(ctx, claims.TenantID, req.AppID); err != nil {
		return err
	}
	lang := normalizeLang(req.Lang)

	now := time.Now().UTC()
	tx := global.DB.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	type pageRow struct {
		ID string `gorm:"column:id"`
	}
	var p pageRow
	if err := tx.Table("app_content_pages").
		Select("id").
		Where("tenant_id = ? AND app_id = ? AND content_key = ?", claims.TenantID, req.AppID, contentKey).
		Limit(1).
		Scan(&p).Error; err != nil {
		tx.Rollback()
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if p.ID == "" {
		p.ID = uuid.NewString()
		if err := tx.Table("app_content_pages").Create(map[string]interface{}{
			"id":          p.ID,
			"tenant_id":   claims.TenantID,
			"app_id":      req.AppID,
			"content_key": contentKey,
			"published":   false,
			"created_by":  claims.ID,
			"updated_by":  claims.ID,
			"created_at":  now,
			"updated_at":  now,
		}).Error; err != nil {
			tx.Rollback()
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
	}

	html := markdownToHTML(req.ContentMarkdown)

	// upsert i18n
	type i18nRow struct {
		ID string `gorm:"column:id"`
	}
	var tr i18nRow
	if err := tx.Table("app_content_page_i18n").
		Select("id").
		Where("page_id = ? AND lang = ?", p.ID, lang).
		Limit(1).
		Scan(&tr).Error; err != nil {
		tx.Rollback()
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if tr.ID == "" {
		if err := tx.Table("app_content_page_i18n").Create(map[string]interface{}{
			"id":               uuid.NewString(),
			"page_id":          p.ID,
			"lang":             lang,
			"title":            strings.TrimSpace(req.Title),
			"content_markdown": req.ContentMarkdown,
			"content_html":     html,
			"updated_at":       now,
		}).Error; err != nil {
			tx.Rollback()
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
	} else {
		if err := tx.Table("app_content_page_i18n").
			Where("id = ?", tr.ID).
			Updates(map[string]interface{}{
				"title":            strings.TrimSpace(req.Title),
				"content_markdown": req.ContentMarkdown,
				"content_html":     html,
				"updated_at":       now,
			}).Error; err != nil {
			tx.Rollback()
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
	}

	if err := tx.Table("app_content_pages").
		Where("id = ?", p.ID).
		Updates(map[string]interface{}{
			"updated_by": claims.ID,
			"updated_at": now,
		}).Error; err != nil {
		tx.Rollback()
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	return tx.Commit().Error
}

func (*AppContent) AdminSetPagePublish(ctx context.Context, claims *utils.UserClaims, contentKey, appID string, published bool) error {
	if err := validateContentKey(contentKey); err != nil {
		return err
	}
	if err := ensureAppOwned(ctx, claims.TenantID, appID); err != nil {
		return err
	}
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"published":  published,
		"updated_by": claims.ID,
		"updated_at": now,
	}
	if published {
		updates["published_at"] = now
	} else {
		updates["published_at"] = nil
	}

	if err := global.DB.WithContext(ctx).Table("app_content_pages").
		Where("tenant_id = ? AND app_id = ? AND content_key = ?", claims.TenantID, appID, contentKey).
		Updates(updates).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return nil
}

// ---------------------------------------------------------------------------
// 管理端：FAQ
// ---------------------------------------------------------------------------

func (*AppContent) AdminListFaqs(ctx context.Context, claims *utils.UserClaims, req model.AdminFaqListReq) (*model.AdminFaqListResp, error) {
	if err := ensureAppOwned(ctx, claims.TenantID, req.AppID); err != nil {
		return nil, err
	}
	lang := langEnUS
	if req.Lang != nil {
		lang = normalizeLang(*req.Lang)
	}

	db := global.DB.WithContext(ctx).Table("app_faq f").
		Where("f.tenant_id = ? AND f.app_id = ?", claims.TenantID, req.AppID)

	if req.Published != nil {
		db = db.Where("f.published = ?", *req.Published)
	}
	if req.Keyword != nil && strings.TrimSpace(*req.Keyword) != "" {
		kw := "%" + strings.TrimSpace(*req.Keyword) + "%"
		db = db.Where(`EXISTS (
			SELECT 1 FROM app_faq_i18n i
			WHERE i.faq_id = f.id AND (i.question ILIKE ? OR i.answer_markdown ILIKE ?)
		)`, kw, kw)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	offset := (req.Page - 1) * req.PageSize
	type row struct {
		ID        string     `gorm:"column:id"`
		IsPinned  bool       `gorm:"column:is_pinned"`
		Sort      int        `gorm:"column:sort"`
		Published bool       `gorm:"column:published"`
		UpdatedAt *time.Time `gorm:"column:updated_at"`
		Question  *string    `gorm:"column:question"`
	}
	rows := make([]row, 0, req.PageSize)
	if err := db.Select(`
			f.id, f.is_pinned, f.sort, f.published, f.updated_at,
			COALESCE(i_lang.question, i_en.question) AS question
		`).
		Joins("LEFT JOIN app_faq_i18n i_lang ON i_lang.faq_id = f.id AND i_lang.lang = ?", lang).
		Joins("LEFT JOIN app_faq_i18n i_en ON i_en.faq_id = f.id AND i_en.lang = ?", langEnUS).
		Order("f.is_pinned DESC, f.sort DESC, f.updated_at DESC").
		Offset(offset).
		Limit(req.PageSize).
		Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	list := make([]model.AdminFaqListItemResp, 0, len(rows))
	for _, r := range rows {
		q := ""
		if r.Question != nil {
			q = *r.Question
		}
		list = append(list, model.AdminFaqListItemResp{
			ID:        r.ID,
			Question:  q,
			IsPinned:  r.IsPinned,
			Sort:      r.Sort,
			Published: r.Published,
			UpdatedAt: formatLocalTime(r.UpdatedAt),
		})
	}

	return &model.AdminFaqListResp{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (*AppContent) AdminGetFaq(ctx context.Context, claims *utils.UserClaims, id string) (*model.AdminFaqDetailResp, error) {
	type baseRow struct {
		ID        string     `gorm:"column:id"`
		AppID     string     `gorm:"column:app_id"`
		IsPinned  bool       `gorm:"column:is_pinned"`
		Sort      int        `gorm:"column:sort"`
		Published bool       `gorm:"column:published"`
		UpdatedAt *time.Time `gorm:"column:updated_at"`
	}
	var b baseRow
	if err := global.DB.WithContext(ctx).Table("app_faq").
		Select("id, app_id, is_pinned, sort, published, updated_at").
		Where("tenant_id = ? AND id = ?", claims.TenantID, strings.TrimSpace(id)).
		Limit(1).
		Scan(&b).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if b.ID == "" {
		return nil, errcode.New(errcode.CodeNotFound)
	}

	if err := ensureAppOwned(ctx, claims.TenantID, b.AppID); err != nil {
		return nil, err
	}

	type trRow struct {
		Lang string  `gorm:"column:lang"`
		Q    *string `gorm:"column:question"`
		MD   *string `gorm:"column:answer_markdown"`
	}
	trs := make([]trRow, 0, 4)
	if err := global.DB.WithContext(ctx).Table("app_faq_i18n").
		Select("lang, question, answer_markdown").
		Where("faq_id = ? AND lang IN ?", b.ID, []string{langZhCN, langEnUS}).
		Find(&trs).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	i18n := map[string]model.AdminFaqI18nPayload{
		langZhCN: {},
		langEnUS: {},
	}
	for _, tr := range trs {
		payload := model.AdminFaqI18nPayload{}
		if tr.Q != nil {
			payload.Question = *tr.Q
		}
		if tr.MD != nil {
			payload.AnswerMarkdown = *tr.MD
		}
		i18n[normalizeLang(tr.Lang)] = payload
	}

	return &model.AdminFaqDetailResp{
		ID:        b.ID,
		AppID:     b.AppID,
		IsPinned:  b.IsPinned,
		Sort:      b.Sort,
		Published: b.Published,
		I18n:      i18n,
		UpdatedAt: formatLocalTime(b.UpdatedAt),
	}, nil
}

func (*AppContent) AdminCreateFaq(ctx context.Context, claims *utils.UserClaims, req model.AdminFaqCreateReq) (string, error) {
	if err := ensureAppOwned(ctx, claims.TenantID, req.AppID); err != nil {
		return "", err
	}

	now := time.Now().UTC()
	id := uuid.NewString()
	tx := global.DB.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Table("app_faq").Create(map[string]interface{}{
		"id":        id,
		"tenant_id": claims.TenantID,
		"app_id":    req.AppID,
		"is_pinned": req.IsPinned,
		"sort":      req.Sort,
		"published": req.Published,
		"published_at": func() interface{} {
			if req.Published {
				return now
			}
			return nil
		}(),
		"created_by": claims.ID,
		"updated_by": claims.ID,
		"created_at": now,
		"updated_at": now,
	}).Error; err != nil {
		tx.Rollback()
		return "", errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	if err := upsertFaqI18n(tx, id, req.I18n, now); err != nil {
		tx.Rollback()
		return "", err
	}

	if err := tx.Commit().Error; err != nil {
		return "", errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return id, nil
}

func (*AppContent) AdminUpdateFaq(ctx context.Context, claims *utils.UserClaims, id string, req model.AdminFaqUpdateReq) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errcode.New(errcode.CodeParamError)
	}

	type baseRow struct {
		ID    string `gorm:"column:id"`
		AppID string `gorm:"column:app_id"`
	}
	var b baseRow
	if err := global.DB.WithContext(ctx).Table("app_faq").
		Select("id, app_id").
		Where("tenant_id = ? AND id = ?", claims.TenantID, id).
		Limit(1).
		Scan(&b).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if b.ID == "" {
		return errcode.New(errcode.CodeNotFound)
	}
	if err := ensureAppOwned(ctx, claims.TenantID, b.AppID); err != nil {
		return err
	}

	now := time.Now().UTC()
	tx := global.DB.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	updates := map[string]interface{}{
		"updated_by": claims.ID,
		"updated_at": now,
	}
	if req.IsPinned != nil {
		updates["is_pinned"] = *req.IsPinned
	}
	if req.Sort != nil {
		updates["sort"] = *req.Sort
	}
	if req.Published != nil {
		updates["published"] = *req.Published
		if *req.Published {
			updates["published_at"] = now
		} else {
			updates["published_at"] = nil
		}
	}

	if err := tx.Table("app_faq").Where("id = ?", id).Updates(updates).Error; err != nil {
		tx.Rollback()
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	if req.I18n != nil {
		if err := upsertFaqI18n(tx, id, req.I18n, now); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func upsertFaqI18n(tx *gorm.DB, faqID string, i18n map[string]model.AdminFaqI18nPayload, now time.Time) error {
	if i18n == nil {
		return nil
	}

	for langKey, payload := range i18n {
		lang := normalizeLang(langKey)
		if strings.TrimSpace(payload.Question) == "" && strings.TrimSpace(payload.AnswerMarkdown) == "" {
			continue
		}

		answerHTML := markdownToHTML(payload.AnswerMarkdown)

		type row struct {
			ID string `gorm:"column:id"`
		}
		var r row
		if err := tx.Table("app_faq_i18n").
			Select("id").
			Where("faq_id = ? AND lang = ?", faqID, lang).
			Limit(1).
			Scan(&r).Error; err != nil {
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
		if r.ID == "" {
			if err := tx.Table("app_faq_i18n").Create(map[string]interface{}{
				"id":              uuid.NewString(),
				"faq_id":          faqID,
				"lang":            lang,
				"question":        strings.TrimSpace(payload.Question),
				"answer_markdown": payload.AnswerMarkdown,
				"answer_html":     answerHTML,
				"updated_at":      now,
			}).Error; err != nil {
				return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
			}
		} else {
			if err := tx.Table("app_faq_i18n").Where("id = ?", r.ID).Updates(map[string]interface{}{
				"question":        strings.TrimSpace(payload.Question),
				"answer_markdown": payload.AnswerMarkdown,
				"answer_html":     answerHTML,
				"updated_at":      now,
			}).Error; err != nil {
				return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
			}
		}
	}
	return nil
}

func (*AppContent) AdminDeleteFaq(ctx context.Context, claims *utils.UserClaims, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errcode.New(errcode.CodeParamError)
	}
	if err := global.DB.WithContext(ctx).Table("app_faq").Where("tenant_id = ? AND id = ?", claims.TenantID, id).Delete(map[string]interface{}{}).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return nil
}

func (*AppContent) AdminBatchDeleteFaqs(ctx context.Context, claims *utils.UserClaims, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	for i := range ids {
		ids[i] = strings.TrimSpace(ids[i])
	}
	if err := global.DB.WithContext(ctx).Table("app_faq").Where("tenant_id = ? AND id IN ?", claims.TenantID, ids).Delete(map[string]interface{}{}).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return nil
}

// ---------------------------------------------------------------------------
// 管理端：用户反馈
// ---------------------------------------------------------------------------

func (*AppContent) AdminListFeedback(ctx context.Context, claims *utils.UserClaims, req model.AdminFeedbackListReq) (*model.AdminFeedbackListResp, error) {
	if err := ensureAppOwned(ctx, claims.TenantID, req.AppID); err != nil {
		return nil, err
	}

	db := global.DB.WithContext(ctx).Table("app_feedback f").
		Where("f.tenant_id = ? AND f.app_id = ?", claims.TenantID, req.AppID)

	if req.Status != nil && strings.TrimSpace(*req.Status) != "" {
		db = db.Where("f.status = ?", strings.TrimSpace(*req.Status))
	}
	if req.Keyword != nil && strings.TrimSpace(*req.Keyword) != "" {
		kw := "%" + strings.TrimSpace(*req.Keyword) + "%"
		db = db.Where("(f.content ILIKE ?)", kw)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	offset := (req.Page - 1) * req.PageSize
	type row struct {
		ID        string         `gorm:"column:id"`
		UserID    *string        `gorm:"column:user_id"`
		Phone     *string        `gorm:"column:phone_number"`
		Email     *string        `gorm:"column:email"`
		Content   string         `gorm:"column:content"`
		Images    datatypes.JSON `gorm:"column:images"`
		Status    string         `gorm:"column:status"`
		Reply     *string        `gorm:"column:reply"`
		CreatedAt *time.Time     `gorm:"column:created_at"`
		UpdatedAt *time.Time     `gorm:"column:updated_at"`
	}
	rows := make([]row, 0, req.PageSize)
	if err := db.Select("f.id, f.user_id, u.phone_number, u.email, f.content, f.images, f.status, f.reply, f.created_at, f.updated_at").
		Joins("LEFT JOIN users u ON u.id = f.user_id").
		Order("f.created_at DESC").
		Offset(offset).
		Limit(req.PageSize).
		Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	list := make([]model.AdminFeedbackListItemResp, 0, len(rows))
	for _, r := range rows {
		var imgs []string
		if len(r.Images) != 0 {
			_ = json.Unmarshal(r.Images, &imgs)
		}
		list = append(list, model.AdminFeedbackListItemResp{
			ID:        r.ID,
			UserID:    r.UserID,
			Phone:     r.Phone,
			Email:     r.Email,
			Content:   r.Content,
			ImageCnt:  len(imgs),
			Status:    r.Status,
			Reply:     r.Reply,
			CreatedAt: formatLocalTime(r.CreatedAt),
			UpdatedAt: formatLocalTime(r.UpdatedAt),
		})
	}

	return &model.AdminFeedbackListResp{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (*AppContent) AdminGetFeedback(ctx context.Context, claims *utils.UserClaims, id string) (*model.AdminFeedbackDetailResp, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errcode.New(errcode.CodeParamError)
	}

	type row struct {
		ID          string         `gorm:"column:id"`
		AppID       string         `gorm:"column:app_id"`
		AppIDText   string         `gorm:"column:appid"`
		UserID      *string        `gorm:"column:user_id"`
		Phone       *string        `gorm:"column:phone_number"`
		Email       *string        `gorm:"column:email"`
		Content     string         `gorm:"column:content"`
		Images      datatypes.JSON `gorm:"column:images"`
		Platform    *string        `gorm:"column:platform"`
		AppVersion  *string        `gorm:"column:app_version"`
		DeviceModel *string        `gorm:"column:device_model"`
		OSVersion   *string        `gorm:"column:os_version"`
		Status      string         `gorm:"column:status"`
		Reply       *string        `gorm:"column:reply"`
		RepliedAt   *time.Time     `gorm:"column:replied_at"`
		HandleNote  *string        `gorm:"column:handle_note"`
		CreatedAt   *time.Time     `gorm:"column:created_at"`
		UpdatedAt   *time.Time     `gorm:"column:updated_at"`
	}
	var r row
	if err := global.DB.WithContext(ctx).Table("app_feedback f").
		Select(`f.id, f.app_id, f.appid, f.user_id, u.phone_number, u.email, f.content, f.images,
			f.platform, f.app_version, f.device_model, f.os_version, f.status, f.reply, f.replied_at, f.handle_note,
			f.created_at, f.updated_at`).
		Joins("LEFT JOIN users u ON u.id = f.user_id").
		Where("f.tenant_id = ? AND f.id = ?", claims.TenantID, id).
		Limit(1).
		Scan(&r).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if r.ID == "" {
		return nil, errcode.New(errcode.CodeNotFound)
	}
	if err := ensureAppOwned(ctx, claims.TenantID, r.AppID); err != nil {
		return nil, err
	}

	var imgs []string
	if len(r.Images) != 0 {
		_ = json.Unmarshal(r.Images, &imgs)
	}

	var repliedAt *string
	if r.RepliedAt != nil {
		s := formatLocalTime(r.RepliedAt)
		repliedAt = &s
	}

	return &model.AdminFeedbackDetailResp{
		ID:          r.ID,
		AppID:       r.AppID,
		AppIDText:   r.AppIDText,
		UserID:      r.UserID,
		Phone:       r.Phone,
		Email:       r.Email,
		Content:     r.Content,
		Images:      imgs,
		Platform:    r.Platform,
		AppVersion:  r.AppVersion,
		DeviceModel: r.DeviceModel,
		OSVersion:   r.OSVersion,
		Status:      r.Status,
		Reply:       r.Reply,
		RepliedAt:   repliedAt,
		HandleNote:  r.HandleNote,
		CreatedAt:   formatLocalTime(r.CreatedAt),
		UpdatedAt:   formatLocalTime(r.UpdatedAt),
	}, nil
}

func (*AppContent) AdminUpdateFeedback(ctx context.Context, claims *utils.UserClaims, id string, req model.AdminFeedbackUpdateReq) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errcode.New(errcode.CodeParamError)
	}
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"updated_at":  now,
		"handler_uid": claims.ID,
	}
	if req.Status != nil {
		updates["status"] = strings.TrimSpace(*req.Status)
	}
	if req.Reply != nil {
		updates["reply"] = req.Reply
		updates["replied_at"] = now
	}
	if req.HandleNote != nil {
		updates["handle_note"] = req.HandleNote
	}

	if err := global.DB.WithContext(ctx).Table("app_feedback").
		Where("tenant_id = ? AND id = ?", claims.TenantID, id).
		Updates(updates).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return nil
}

// ---------------------------------------------------------------------------
// APP端（需登录）：用户反馈
// ---------------------------------------------------------------------------

func (*AppContent) CreateFeedbackForApp(ctx context.Context, claims *utils.UserClaims, tenantHeader string, req model.AppFeedbackCreateReq) (string, error) {
	// header 必须存在且与 token tenant 匹配；此处直接使用 claims.TenantID（router middleware 已校验一致性）
	_ = tenantHeader

	app, err := getAppByAppID(ctx, claims.TenantID, req.AppID)
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	id := uuid.NewString()
	b, _ := json.Marshal(req.Images)
	if err := global.DB.WithContext(ctx).Table("app_feedback").Create(map[string]interface{}{
		"id":           id,
		"tenant_id":    claims.TenantID,
		"app_id":       app.ID,
		"appid":        app.AppID,
		"user_id":      claims.ID,
		"content":      strings.TrimSpace(req.Content),
		"images":       datatypes.JSON(b),
		"platform":     req.Platform,
		"app_version":  req.AppVersion,
		"device_model": req.DeviceModel,
		"os_version":   req.OSVersion,
		"status":       "NEW",
		"created_at":   now,
		"updated_at":   now,
	}).Error; err != nil {
		return "", errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return id, nil
}

func (*AppContent) ListMyFeedbackForApp(ctx context.Context, claims *utils.UserClaims, req model.AppFeedbackListReq) (*model.AppFeedbackListResp, error) {
	db := global.DB.WithContext(ctx).Table("app_feedback").
		Where("tenant_id = ? AND user_id = ?", claims.TenantID, claims.ID)

	if req.AppID != nil && strings.TrimSpace(*req.AppID) != "" {
		app, err := getAppByAppID(ctx, claims.TenantID, *req.AppID)
		if err != nil {
			return nil, err
		}
		db = db.Where("app_id = ?", app.ID)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	offset := (req.Page - 1) * req.PageSize
	type row struct {
		ID        string         `gorm:"column:id"`
		AppID     string         `gorm:"column:appid"`
		Content   string         `gorm:"column:content"`
		Images    datatypes.JSON `gorm:"column:images"`
		Status    string         `gorm:"column:status"`
		Reply     *string        `gorm:"column:reply"`
		RepliedAt *time.Time     `gorm:"column:replied_at"`
		CreatedAt *time.Time     `gorm:"column:created_at"`
		UpdatedAt *time.Time     `gorm:"column:updated_at"`
	}
	rows := make([]row, 0, req.PageSize)
	if err := db.Select("id, appid, content, images, status, reply, replied_at, created_at, updated_at").
		Order("created_at DESC").
		Offset(offset).
		Limit(req.PageSize).
		Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	list := make([]model.AppFeedbackItemResp, 0, len(rows))
	for _, r := range rows {
		var imgs []string
		if len(r.Images) != 0 {
			_ = json.Unmarshal(r.Images, &imgs)
		}
		var repliedAt *string
		if r.RepliedAt != nil {
			s := formatLocalTime(r.RepliedAt)
			repliedAt = &s
		}
		list = append(list, model.AppFeedbackItemResp{
			ID:        r.ID,
			AppID:     r.AppID,
			Content:   r.Content,
			Images:    imgs,
			Status:    r.Status,
			Reply:     r.Reply,
			RepliedAt: repliedAt,
			CreatedAt: formatLocalTime(r.CreatedAt),
			UpdatedAt: formatLocalTime(r.UpdatedAt),
		})
	}

	return &model.AppFeedbackListResp{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (*AppContent) GetMyFeedbackForApp(ctx context.Context, claims *utils.UserClaims, id string) (*model.AppFeedbackItemResp, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errcode.New(errcode.CodeParamError)
	}
	type row struct {
		ID        string         `gorm:"column:id"`
		AppID     string         `gorm:"column:appid"`
		Content   string         `gorm:"column:content"`
		Images    datatypes.JSON `gorm:"column:images"`
		Status    string         `gorm:"column:status"`
		Reply     *string        `gorm:"column:reply"`
		RepliedAt *time.Time     `gorm:"column:replied_at"`
		CreatedAt *time.Time     `gorm:"column:created_at"`
		UpdatedAt *time.Time     `gorm:"column:updated_at"`
	}
	var r row
	if err := global.DB.WithContext(ctx).Table("app_feedback").
		Select("id, appid, content, images, status, reply, replied_at, created_at, updated_at").
		Where("tenant_id = ? AND user_id = ? AND id = ?", claims.TenantID, claims.ID, id).
		Limit(1).
		Scan(&r).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if r.ID == "" {
		return nil, errcode.New(errcode.CodeNotFound)
	}

	var imgs []string
	if len(r.Images) != 0 {
		_ = json.Unmarshal(r.Images, &imgs)
	}
	var repliedAt *string
	if r.RepliedAt != nil {
		s := formatLocalTime(r.RepliedAt)
		repliedAt = &s
	}

	return &model.AppFeedbackItemResp{
		ID:        r.ID,
		AppID:     r.AppID,
		Content:   r.Content,
		Images:    imgs,
		Status:    r.Status,
		Reply:     r.Reply,
		RepliedAt: repliedAt,
		CreatedAt: formatLocalTime(r.CreatedAt),
		UpdatedAt: formatLocalTime(r.UpdatedAt),
	}, nil
}
