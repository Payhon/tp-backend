package service

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	dal "project/internal/dal"
	model "project/internal/model"
	"project/pkg/common"
	"project/pkg/errcode"
	"project/pkg/utils"

	"github.com/go-basic/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	qbox "github.com/qiniu/go-sdk/v7/auth/qbox"
	qstorage "github.com/qiniu/go-sdk/v7/storage"
)

type File struct{}

const (
	fileStorageConfigID = "file_storage_config_1"
	secretMask          = "********"
)

func boolOrDefault(v *bool, def bool) bool {
	if v == nil {
		return def
	}
	return *v
}

func normalizeDomain(domain string, useHTTPS bool) (string, error) {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return "", fmt.Errorf("domain is empty")
	}
	if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") {
		_, err := url.Parse(domain)
		return strings.TrimRight(domain, "/"), err
	}
	scheme := "https"
	if !useHTTPS {
		scheme = "http"
	}
	u := scheme + "://" + domain
	_, err := url.Parse(u)
	return strings.TrimRight(u, "/"), err
}

func requestOrigin(scheme, host string) string {
	scheme = strings.TrimSpace(scheme)
	host = strings.TrimSpace(host)
	if scheme == "" {
		scheme = "http"
	}
	return scheme + "://" + host
}

func generateObjectKey(bizType, filename string) (string, string, error) {
	if bizType == "" {
		return "", "", fmt.Errorf("bizType is empty")
	}
	if err := utils.CheckPath(bizType); err != nil {
		return "", "", fmt.Errorf("invalid bizType: %w", err)
	}
	ext := strings.ToLower(filepath.Ext(filename))
	dateDir := time.Now().Format("2006-01-02")

	randomStr, err := common.GenerateRandomString(16)
	if err != nil {
		return "", "", err
	}
	timeStr := time.Now().Format("20060102150405")
	hashStr := fmt.Sprintf("%x", md5.Sum([]byte(timeStr+randomStr)))

	objectKey := filepath.ToSlash(filepath.Join(bizType, dateDir, hashStr+ext))
	return objectKey, ext, nil
}

func defaultStorageConfig() model.UpsertFileStorageConfigReq {
	return model.UpsertFileStorageConfigReq{
		StorageType: "local",
		Provider:    "",
		Local: model.LocalStorageConfig{
			BaseDir:         "./files",
			PublicPathPrefix: "/files",
		},
		Aliyun: model.AliyunOSSConfig{
			DirPrefix: "uploads/",
			UseHTTPS:  func() *bool { v := true; return &v }(),
		},
		Qiniu: model.QiniuKodoConfig{
			DirPrefix: "uploads/",
			Region:    "huadong",
			UseHTTPS:  func() *bool { v := true; return &v }(),
		},
	}
}

func (s *File) getEffectiveStorageConfig(ctx context.Context) (*model.UpsertFileStorageConfigReq, *model.FileStorageConfig, error) {
	cfgRow, err := dal.GetFileStorageConfig(ctx, fileStorageConfigID)
	if err != nil {
		return nil, nil, err
	}
	cfg := defaultStorageConfig()
	if cfgRow == nil {
		return &cfg, nil, nil
	}
	if len(cfgRow.Config) > 0 {
		_ = json.Unmarshal(cfgRow.Config, &cfg)
	}
	cfg.StorageType = cfgRow.StorageType
	if cfgRow.Provider != nil {
		cfg.Provider = *cfgRow.Provider
	}
	cfg.Remark = cfgRow.Remark
	return &cfg, cfgRow, nil
}

func maskSecrets(cfg *model.UpsertFileStorageConfigReq) {
	if strings.TrimSpace(cfg.Aliyun.AccessKeySecret) != "" {
		cfg.Aliyun.AccessKeySecret = secretMask
		cfg.Aliyun.AccessKeySecretSet = true
	}
	if strings.TrimSpace(cfg.Qiniu.SecretKey) != "" {
		cfg.Qiniu.SecretKey = secretMask
		cfg.Qiniu.SecretKeySet = true
	}
}

func applySecretPreserve(in *model.UpsertFileStorageConfigReq, existing *model.UpsertFileStorageConfigReq) {
	if strings.TrimSpace(in.Aliyun.AccessKeySecret) == "" || in.Aliyun.AccessKeySecret == secretMask {
		in.Aliyun.AccessKeySecret = existing.Aliyun.AccessKeySecret
	}
	if strings.TrimSpace(in.Qiniu.SecretKey) == "" || in.Qiniu.SecretKey == secretMask {
		in.Qiniu.SecretKey = existing.Qiniu.SecretKey
	}
}

func (s *File) GetFileStorageConfig(ctx context.Context) (*model.GetFileStorageConfigRsp, error) {
	cfg, cfgRow, err := s.getEffectiveStorageConfig(ctx)
	if err != nil {
		return nil, err
	}
	maskSecrets(cfg)

	rsp := &model.GetFileStorageConfigRsp{
		ID:          fileStorageConfigID,
		StorageType: cfg.StorageType,
		Provider:    cfg.Provider,
		Local:       cfg.Local,
		Aliyun:      cfg.Aliyun,
		Qiniu:       cfg.Qiniu,
		Remark:      cfg.Remark,
	}
	if cfgRow != nil {
		rsp.UpdatedAt = cfgRow.UpdatedAt.Unix()
	} else {
		rsp.UpdatedAt = 0
	}
	return rsp, nil
}

func (s *File) UpsertFileStorageConfig(ctx context.Context, req *model.UpsertFileStorageConfigReq) error {
	existing, existingRow, err := s.getEffectiveStorageConfig(ctx)
	if err != nil {
		return err
	}
	applySecretPreserve(req, existing)

	// storage_type=cloud 时校验 provider 与对应配置字段
	if req.StorageType == "cloud" {
		if req.Provider != "aliyun" && req.Provider != "qiniu" {
			return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"provider": "provider required when storage_type=cloud"})
		}
		switch req.Provider {
		case "aliyun":
			if strings.TrimSpace(req.Aliyun.AccessKeyID) == "" || strings.TrimSpace(req.Aliyun.AccessKeySecret) == "" ||
				strings.TrimSpace(req.Aliyun.Endpoint) == "" || strings.TrimSpace(req.Aliyun.Bucket) == "" || strings.TrimSpace(req.Aliyun.Domain) == "" {
				return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"aliyun": "aliyun config incomplete"})
			}
		case "qiniu":
			if strings.TrimSpace(req.Qiniu.AccessKey) == "" || strings.TrimSpace(req.Qiniu.SecretKey) == "" ||
				strings.TrimSpace(req.Qiniu.Bucket) == "" || strings.TrimSpace(req.Qiniu.Domain) == "" {
				return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"qiniu": "qiniu config incomplete"})
			}
		}
	}

	// 固化本地默认路径，避免与 /files 静态路由不一致
	if strings.TrimSpace(req.Local.BaseDir) == "" {
		req.Local.BaseDir = "./files"
	}
	if strings.TrimSpace(req.Local.PublicPathPrefix) == "" {
		req.Local.PublicPathPrefix = "/files"
	}

	cfgBytes, err := json.Marshal(req)
	if err != nil {
		return err
	}

	row := &model.FileStorageConfig{
		ID:          fileStorageConfigID,
		StorageType: req.StorageType,
		Config:      datatypes.JSON(cfgBytes),
		UpdatedAt:   time.Now(),
		Remark:      req.Remark,
	}
	if req.Provider != "" {
		row.Provider = StringPtr(req.Provider)
	} else {
		row.Provider = nil
	}
	if existingRow != nil {
		row.ID = existingRow.ID
	}
	return dal.UpsertFileStorageConfig(ctx, row)
}

func (s *File) UploadFile(ctx context.Context, claims *utils.UserClaims, scheme, host string, file *multipart.FileHeader, bizType string) (*model.UploadFileRsp, error) {
	if claims == nil || claims.TenantID == "" {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"tenant_id": "missing"})
	}
	if file == nil {
		return nil, errcode.New(errcode.CodeFileEmpty)
	}

	// 文件类型检查（沿用原逻辑）
	if err := utils.CheckPath(bizType); err != nil {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"type": "invalid"})
	}
	if !utils.ValidateFileType(file.Filename, bizType) {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"type": "file type not allowed"})
	}

	cfg, _, err := s.getEffectiveStorageConfig(ctx)
	if err != nil {
		return nil, err
	}

	// OTA升级包保持本地存储（与现有下载链路兼容）
	if bizType == "upgradePackage" {
		cfg.StorageType = "local"
	}

	objectKey, ext, err := generateObjectKey(bizType, file.Filename)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeFilePathGenError, map[string]interface{}{"error": err.Error()})
	}

	src, err := file.Open()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeFileSaveError, map[string]interface{}{"error": err.Error()})
	}
	defer src.Close()

	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	now := time.Now()
	fileID := uuid.New()

	var storageLocation string
	var dbFilePath string
	var fullURL string
	var responsePath string

	switch cfg.StorageType {
	case "cloud":
		switch cfg.Provider {
		case "aliyun":
			storageLocation = "aliyun"
			var key string
			key, fullURL, err = s.putAliyunOSS(ctx, cfg, objectKey, src, mimeType)
			if err != nil {
				return nil, err
			}
			dbFilePath = key
			responsePath = "./files-cloud/" + fileID
		case "qiniu":
			storageLocation = "qiniu"
			var key string
			key, fullURL, err = s.putQiniu(ctx, cfg, objectKey, src, file.Size, mimeType)
			if err != nil {
				return nil, err
			}
			dbFilePath = key
			responsePath = "./files-cloud/" + fileID
		default:
			return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"provider": "unknown provider"})
		}
	default:
		storageLocation = "local"
		baseDir := strings.TrimSpace(cfg.Local.BaseDir)
		// 强制在 ./files 下（与 /files 静态路由一致）
		absBase, _ := filepath.Abs("./files")
		absDir, _ := filepath.Abs(baseDir)
		if !strings.HasPrefix(absDir, absBase) {
			baseDir = "./files"
		}

		diskPath := filepath.Join(baseDir, filepath.FromSlash(objectKey))
		if err := os.MkdirAll(filepath.Dir(diskPath), 0755); err != nil {
			return nil, errcode.WithData(errcode.CodeFileSaveError, map[string]interface{}{"error": err.Error()})
		}
		dst, err := os.Create(diskPath)
		if err != nil {
			return nil, errcode.WithData(errcode.CodeFileSaveError, map[string]interface{}{"error": err.Error()})
		}
		if _, err := io.Copy(dst, src); err != nil {
			_ = dst.Close()
			return nil, errcode.WithData(errcode.CodeFileSaveError, map[string]interface{}{"error": err.Error()})
		}
		_ = dst.Close()

		dbFilePath = "./files/" + objectKey
		origin := requestOrigin(scheme, host)
		if bizType == "upgradePackage" {
			accessPath := "/api/v1/ota/download/files/upgradePackage/" + strings.Split(objectKey, "/")[1] + "/" + filepath.Base(objectKey)
			fullURL = origin + accessPath
			responsePath = "." + accessPath
		} else {
			publicPrefix := strings.TrimSpace(cfg.Local.PublicPathPrefix)
			if publicPrefix == "" {
				publicPrefix = "/files"
			}
			fullURL = origin + strings.TrimRight(publicPrefix, "/") + "/" + objectKey
			responsePath = dbFilePath
		}
	}

	originalFileName := filepath.Base(file.Filename)
	fileName := originalFileName
	uploadedBy := claims.ID
	meta := datatypes.JSON([]byte(`{}`))
	fileExt := ext

	row := &model.File{
		ID:               fileID,
		TenantID:         claims.TenantID,
		FileName:         fileName,
		OriginalFileName: &originalFileName,
		FileSize:         file.Size,
		StorageLocation:  storageLocation,
		BizType:          bizType,
		MimeType:         &mimeType,
		FileExt:          &fileExt,
		FilePath:         dbFilePath,
		FullURL:          fullURL,
		UploadedAt:       now,
		UploadedBy:       &uploadedBy,
		Meta:             meta,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := dal.CreateFile(ctx, row); err != nil {
		logrus.WithError(err).Error("create file record failed")
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"err": err.Error()})
	}

	return &model.UploadFileRsp{
		ID:              fileID,
		StorageLocation: storageLocation,
		Path:            responsePath,
		URL:             fullURL,
	}, nil
}

func (s *File) GetFileListByPage(ctx context.Context, claims *utils.UserClaims, req *model.GetFileListByPageReq) (*model.GetFileListByPageRsp, error) {
	if claims == nil || claims.TenantID == "" {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"tenant_id": "missing"})
	}
	if req.Mine != nil && *req.Mine {
		req.UploadedBy = &claims.ID
	}
	res, err := dal.ListFilesByTenant(ctx, claims.TenantID, req)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"err": err.Error()})
	}

	list := make([]model.FileListItem, 0, len(res.List))
	for i := range res.List {
		f := res.List[i]
		path := f.FilePath
		if f.StorageLocation != "local" {
			path = "./files-cloud/" + f.ID
		}
		// OTA 固件下载需要走专用接口（支持 Range/CRC16），这里为 upgradePackage 统一返回 ota download 路径
		if f.StorageLocation == "local" && f.BizType == "upgradePackage" {
			objectKey := strings.TrimPrefix(f.FilePath, "./files/")
			parts := strings.Split(objectKey, "/")
			if len(parts) >= 3 {
				dateDir := parts[1]
				filename := parts[len(parts)-1]
				path = "./api/v1/ota/download/files/upgradePackage/" + dateDir + "/" + filename
			}
		}
		list = append(list, model.FileListItem{
			ID:              f.ID,
			FileName:        f.FileName,
			FileSize:        f.FileSize,
			StorageLocation: f.StorageLocation,
			BizType:         f.BizType,
			MimeType:        f.MimeType,
			FileExt:         f.FileExt,
			UploadedAt:      f.UploadedAt,
			UploadedBy:      f.UploadedBy,
			Path:            path,
			URL:             f.FullURL,
		})
	}

	return &model.GetFileListByPageRsp{Total: res.Total, List: list}, nil
}

func qiniuZone(region string) (*qstorage.Zone, error) {
	switch strings.ToLower(strings.TrimSpace(region)) {
	case "", "huadong":
		return &qstorage.ZoneHuadong, nil
	case "huabei":
		return &qstorage.ZoneHuabei, nil
	case "huanan":
		return &qstorage.ZoneHuanan, nil
	case "beimei":
		return &qstorage.ZoneBeimei, nil
	case "xinjiapo":
		return &qstorage.ZoneXinjiapo, nil
	default:
		return nil, fmt.Errorf("unknown qiniu region: %s", region)
	}
}

func (s *File) putQiniu(ctx context.Context, cfg *model.UpsertFileStorageConfigReq, objectKey string, src multipart.File, size int64, mimeType string) (string, string, error) {
	zone, err := qiniuZone(cfg.Qiniu.Region)
	if err != nil {
		return "", "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{"qiniu.region": err.Error()})
	}
	useHTTPS := boolOrDefault(cfg.Qiniu.UseHTTPS, true)

	mac := qbox.NewMac(cfg.Qiniu.AccessKey, cfg.Qiniu.SecretKey)
	key := objectKey
	prefix := strings.TrimSpace(cfg.Qiniu.DirPrefix)
	if prefix != "" {
		key = filepath.ToSlash(filepath.Join(prefix, objectKey))
	}
	putPolicy := qstorage.PutPolicy{Scope: cfg.Qiniu.Bucket + ":" + key, Expires: 3600}
	upToken := putPolicy.UploadToken(mac)

	uploader := qstorage.NewFormUploader(&qstorage.Config{Zone: zone, UseHTTPS: useHTTPS})
	ret := qstorage.PutRet{}
	putExtra := qstorage.PutExtra{MimeType: mimeType}

	if err := uploader.Put(ctx, &ret, upToken, key, src, size, &putExtra); err != nil {
		logrus.WithError(err).Error("qiniu upload failed")
		return "", "", errcode.WithData(errcode.CodeSystemError, map[string]interface{}{"qiniu": err.Error()})
	}

	domain, err := normalizeDomain(cfg.Qiniu.Domain, useHTTPS)
	if err != nil {
		return "", "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{"qiniu.domain": err.Error()})
	}
	return key, domain + "/" + key, nil
}

func (s *File) putAliyunOSS(ctx context.Context, cfg *model.UpsertFileStorageConfigReq, objectKey string, src multipart.File, mimeType string) (string, string, error) {
	useHTTPS := boolOrDefault(cfg.Aliyun.UseHTTPS, true)

	endpoint := strings.TrimSpace(cfg.Aliyun.Endpoint)
	if endpoint == "" {
		return "", "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{"aliyun.endpoint": "empty"})
	}
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		if useHTTPS {
			endpoint = "https://" + endpoint
		} else {
			endpoint = "http://" + endpoint
		}
	}

	client, err := oss.New(endpoint, cfg.Aliyun.AccessKeyID, cfg.Aliyun.AccessKeySecret)
	if err != nil {
		logrus.WithError(err).Error("aliyun oss client init failed")
		return "", "", errcode.WithData(errcode.CodeSystemError, map[string]interface{}{"aliyun": err.Error()})
	}
	bucket, err := client.Bucket(cfg.Aliyun.Bucket)
	if err != nil {
		logrus.WithError(err).Error("aliyun oss bucket init failed")
		return "", "", errcode.WithData(errcode.CodeSystemError, map[string]interface{}{"aliyun": err.Error()})
	}

	key := objectKey
	prefix := strings.TrimSpace(cfg.Aliyun.DirPrefix)
	if prefix != "" {
		key = filepath.ToSlash(filepath.Join(prefix, objectKey))
	}

	if err := bucket.PutObject(key, src, oss.ContentType(mimeType)); err != nil {
		logrus.WithError(err).Error("aliyun oss put object failed")
		return "", "", errcode.WithData(errcode.CodeSystemError, map[string]interface{}{"aliyun": err.Error()})
	}

	domain, err := normalizeDomain(cfg.Aliyun.Domain, useHTTPS)
	if err != nil {
		return "", "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{"aliyun.domain": err.Error()})
	}
	return key, domain + "/" + key, nil
}

func (s *File) CreateCloudUploadCredential(ctx context.Context, claims *utils.UserClaims, req *model.CreateCloudUploadCredentialReq) (*model.CreateCloudUploadCredentialRsp, error) {
	if claims == nil || claims.TenantID == "" {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"tenant_id": "missing"})
	}
	cfg, _, err := s.getEffectiveStorageConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg.StorageType != "cloud" {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"storage_type": "cloud storage not enabled"})
	}

	objectKey, _, err := generateObjectKey(req.BizType, req.FileName)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeFilePathGenError, map[string]interface{}{"error": err.Error()})
	}

	mimeType := "application/octet-stream"
	if req.MimeType != nil && *req.MimeType != "" {
		mimeType = *req.MimeType
	}

	switch cfg.Provider {
	case "aliyun":
		useHTTPS := boolOrDefault(cfg.Aliyun.UseHTTPS, true)
		endpoint := strings.TrimSpace(cfg.Aliyun.Endpoint)
		if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
			if useHTTPS {
				endpoint = "https://" + endpoint
			} else {
				endpoint = "http://" + endpoint
			}
		}
		client, err := oss.New(endpoint, cfg.Aliyun.AccessKeyID, cfg.Aliyun.AccessKeySecret)
		if err != nil {
			return nil, errcode.WithData(errcode.CodeSystemError, map[string]interface{}{"aliyun": err.Error()})
		}
		bucket, err := client.Bucket(cfg.Aliyun.Bucket)
		if err != nil {
			return nil, errcode.WithData(errcode.CodeSystemError, map[string]interface{}{"aliyun": err.Error()})
		}
		key := objectKey
		prefix := strings.TrimSpace(cfg.Aliyun.DirPrefix)
		if prefix != "" {
			key = filepath.ToSlash(filepath.Join(prefix, objectKey))
		}
		expire := 10 * time.Minute
		signedURL, err := bucket.SignURL(key, oss.HTTPPut, int64(expire.Seconds()), oss.ContentType(mimeType))
		if err != nil {
			return nil, errcode.WithData(errcode.CodeSystemError, map[string]interface{}{"aliyun": err.Error()})
		}
		domain, err := normalizeDomain(cfg.Aliyun.Domain, useHTTPS)
		if err != nil {
			return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"aliyun.domain": err.Error()})
		}
		return &model.CreateCloudUploadCredentialRsp{
			Provider:  "aliyun",
			ObjectKey: key,
			AccessURL: domain + "/" + key,
			Upload: map[string]interface{}{
				"method":     "PUT",
				"url":        signedURL,
				"expire_at":  time.Now().Add(expire).Unix(),
				"headers":    map[string]string{"Content-Type": mimeType},
				"mime_type":  mimeType,
			},
		}, nil
	case "qiniu":
		useHTTPS := boolOrDefault(cfg.Qiniu.UseHTTPS, true)
		key := objectKey
		prefix := strings.TrimSpace(cfg.Qiniu.DirPrefix)
		if prefix != "" {
			key = filepath.ToSlash(filepath.Join(prefix, objectKey))
		}

		mac := qbox.NewMac(cfg.Qiniu.AccessKey, cfg.Qiniu.SecretKey)
		putPolicy := qstorage.PutPolicy{Scope: cfg.Qiniu.Bucket + ":" + key, Expires: 600}
		upToken := putPolicy.UploadToken(mac)

		uploadURL := strings.TrimSpace(cfg.Qiniu.UploadBaseURL)
		if uploadURL == "" {
			uploadURL = "https://up.qiniup.com"
		}
		if strings.HasPrefix(uploadURL, "http://") && useHTTPS {
			uploadURL = strings.Replace(uploadURL, "http://", "https://", 1)
		}
		domain, err := normalizeDomain(cfg.Qiniu.Domain, useHTTPS)
		if err != nil {
			return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"qiniu.domain": err.Error()})
		}
		return &model.CreateCloudUploadCredentialRsp{
			Provider:  "qiniu",
			ObjectKey: key,
			AccessURL: domain + "/" + key,
			Upload: map[string]interface{}{
				"method":    "POST",
				"url":       uploadURL,
				"expire_at": time.Now().Add(10 * time.Minute).Unix(),
				"fields": map[string]string{
					"token": upToken,
					"key":   key,
				},
			},
		}, nil
	default:
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"provider": "unknown provider"})
	}
}

func (s *File) RegisterCloudFile(ctx context.Context, claims *utils.UserClaims, req *model.RegisterCloudFileReq) (*model.UploadFileRsp, error) {
	if claims == nil || claims.TenantID == "" {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"tenant_id": "missing"})
	}
	cfg, _, err := s.getEffectiveStorageConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg.StorageType != "cloud" {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"storage_type": "cloud storage not enabled"})
	}

	now := time.Now()
	fileID := uuid.New()
	mimeType := ""
	if req.MimeType != nil {
		mimeType = *req.MimeType
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	uploadedBy := claims.ID

	var storageLocation string
	var fullURL string
	switch cfg.Provider {
	case "aliyun":
		storageLocation = "aliyun"
		domain, err := normalizeDomain(cfg.Aliyun.Domain, boolOrDefault(cfg.Aliyun.UseHTTPS, true))
		if err != nil {
			return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"aliyun.domain": err.Error()})
		}
		fullURL = domain + "/" + req.ObjectKey
	case "qiniu":
		storageLocation = "qiniu"
		domain, err := normalizeDomain(cfg.Qiniu.Domain, boolOrDefault(cfg.Qiniu.UseHTTPS, true))
		if err != nil {
			return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"qiniu.domain": err.Error()})
		}
		fullURL = domain + "/" + req.ObjectKey
	default:
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"provider": "unknown provider"})
	}

	fileExt := strings.ToLower(filepath.Ext(req.FileName))
	originalFileName := filepath.Base(req.FileName)
	meta := datatypes.JSON([]byte(`{}`))

	row := &model.File{
		ID:               fileID,
		TenantID:         claims.TenantID,
		FileName:         originalFileName,
		OriginalFileName: &originalFileName,
		FileSize:         req.FileSize,
		StorageLocation:  storageLocation,
		BizType:          req.BizType,
		MimeType:         &mimeType,
		FileExt:          &fileExt,
		FilePath:         req.ObjectKey,
		FullURL:          fullURL,
		UploadedAt:       now,
		UploadedBy:       &uploadedBy,
		Meta:             meta,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := dal.CreateFile(ctx, row); err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"err": err.Error()})
	}

	return &model.UploadFileRsp{
		ID:              fileID,
		StorageLocation: storageLocation,
		Path:            "./files-cloud/" + fileID,
		URL:             fullURL,
	}, nil
}
