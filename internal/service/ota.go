package service

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"net/http"
	"strings"
	"time"

	dal "project/internal/dal"
	model "project/internal/model"
	query "project/internal/query"
	"project/mqtt/publish"
	"project/pkg/common"
	"project/pkg/errcode"
	global "project/pkg/global"
	utils "project/pkg/utils"

	"github.com/go-basic/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type OTA struct{}

func resolveOTADeviceKind(kind *int16) int16 {
	if kind == nil || *kind == 0 {
		return model.OTADeviceKindBMS
	}
	if *kind == model.OTADeviceKindMeter {
		return model.OTADeviceKindMeter
	}
	if *kind == model.OTADeviceKind4GModule {
		return model.OTADeviceKind4GModule
	}
	return model.OTADeviceKindBMS
}

func validateOTAUpgradePackageReq(kind int16, name, version, deviceConfigID string, packageURL *string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("name is required")
	}
	if packageURL == nil || strings.TrimSpace(*packageURL) == "" {
		return fmt.Errorf("package_url is required")
	}
	if kind == model.OTADeviceKindMeter {
		return nil
	}
	if strings.TrimSpace(version) == "" {
		return fmt.Errorf("version is required")
	}
	if kind == model.OTADeviceKind4GModule {
		return nil
	}
	return nil
}

func cleanOptionalStringPtr(v *string) *string {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(*v)
	if s == "" {
		return nil
	}
	return &s
}

func unsetOtherLatest4GPackages(tx *gorm.DB, tenantID, packageID string) error {
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(packageID) == "" {
		return nil
	}
	return tx.Model(&model.OtaUpgradePackage{}).
		Where("tenant_id = ? AND device_kind = ? AND id <> ?", tenantID, model.OTADeviceKind4GModule, packageID).
		Update("is_latest", false).Error
}

func resolveSignatureHash(signType string) (hash.Hash, error) {
	switch strings.ToUpper(strings.TrimSpace(signType)) {
	case "", "SHA256":
		return sha256.New(), nil
	case "MD5":
		return md5.New(), nil
	default:
		return nil, fmt.Errorf("unsupported signature type: %s", signType)
	}
}

func signPackageSource(packageURL, signType string) (string, error) {
	raw := strings.TrimSpace(packageURL)
	if raw == "" {
		return "", fmt.Errorf("package_url is empty")
	}

	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		hasher, err := resolveSignatureHash(signType)
		if err != nil {
			return "", err
		}

		resp, err := http.Get(raw)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", fmt.Errorf("download package failed: HTTP %d", resp.StatusCode)
		}

		if _, err = io.Copy(hasher, resp.Body); err != nil {
			return "", err
		}
		return hex.EncodeToString(hasher.Sum(nil)), nil
	}

	filePath := strings.Replace(raw, "/api/v1/ota/download", "", 1)
	return utils.FileSign(filePath, signType)
}

func (*OTA) CreateOTAUpgradePackage(req *model.CreateOTAUpgradePackageReq, tenantID string) error {
	deviceKind := resolveOTADeviceKind(req.DeviceKind)
	if err := validateOTAUpgradePackageReq(deviceKind, req.Name, req.Version, req.DeviceConfigID, req.PackageUrl); err != nil {
		return err
	}

	var ota = model.OtaUpgradePackage{}
	ota.ID = uuid.New()
	ota.Name = req.Name
	ota.DeviceKind = deviceKind
	ota.Version = strings.TrimSpace(req.Version)
	ota.TargetVersion = req.TargetVersion
	ota.DeviceConfigID = strings.TrimSpace(req.DeviceConfigID)
	ota.BatteryModelID = cleanOptionalStringPtr(req.BatteryModelID)
	ota.BatchNumber = cleanOptionalStringPtr(req.BatchNumber)
	ota.ItemUUID = cleanOptionalStringPtr(req.ItemUUID)
	ota.Module = req.Module
	ota.SignatureType = req.SignatureType
	ota.PackageType = 2
	if req.PackageType != nil {
		ota.PackageType = *req.PackageType
	}
	if req.IsLatest != nil {
		ota.IsLatest = *req.IsLatest
	}
	if deviceKind == model.OTADeviceKindBMS {
		ota.DeviceConfigID = ""
	}
	if deviceKind == model.OTADeviceKindMeter {
		ota.Version = ""
		ota.TargetVersion = nil
		ota.DeviceConfigID = ""
		ota.BatteryModelID = nil
		ota.BatchNumber = nil
		ota.ItemUUID = nil
		ota.Module = nil
		ota.PackageType = 2
		ota.IsLatest = false
	}
	if deviceKind == model.OTADeviceKind4GModule {
		ota.TargetVersion = nil
		ota.DeviceConfigID = ""
		ota.BatteryModelID = nil
		ota.BatchNumber = nil
		ota.ItemUUID = nil
		ota.Module = nil
		ota.PackageType = 2
	}

	// 生成文件签名
	fileurl := *req.PackageUrl
	signatureType := "SHA256"
	if req.SignatureType != nil && strings.TrimSpace(*req.SignatureType) != "" {
		signatureType = *req.SignatureType
	}
	ota.SignatureType = &signatureType

	signature, err := signPackageSource(fileurl, signatureType)
	if err != nil {
		return err
	}
	ota.Signature = &signature

	ota.AdditionalInfo = req.AdditionalInfo
	defaultAdditionalInfo := "{}"
	if req.AdditionalInfo == nil || *req.AdditionalInfo == "" {
		ota.AdditionalInfo = &defaultAdditionalInfo
	}
	ota.Description = req.Description
	ota.PackageURL = req.PackageUrl
	ota.TenantID = &tenantID

	t := time.Now().UTC()
	ota.CreatedAt = t
	ota.UpdatedAt = &t
	ota.Remark = req.Remark
	return global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&ota).Error; err != nil {
			return err
		}
		if ota.DeviceKind == model.OTADeviceKind4GModule && ota.IsLatest {
			return unsetOtherLatest4GPackages(tx, tenantID, ota.ID)
		}
		return nil
	})
}

func (*OTA) UpdateOTAUpgradePackage(req *model.UpdateOTAUpgradePackageReq) error {
	oldota, err := dal.GetOtaUpgradePackageByID(req.Id)
	if err != nil {
		return err
	}

	deviceKind := oldota.DeviceKind
	if req.DeviceKind != nil && *req.DeviceKind != 0 {
		deviceKind = resolveOTADeviceKind(req.DeviceKind)
	}
	nextName := oldota.Name
	if strings.TrimSpace(req.Name) != "" {
		nextName = req.Name
	}
	nextVersion := oldota.Version
	if strings.TrimSpace(req.Version) != "" {
		nextVersion = req.Version
	}
	nextDeviceConfigID := oldota.DeviceConfigID
	if strings.TrimSpace(req.DeviceConfigID) != "" {
		nextDeviceConfigID = req.DeviceConfigID
	}
	nextPackageURL := oldota.PackageURL
	if req.PackageUrl != nil {
		nextPackageURL = req.PackageUrl
	}
	if err := validateOTAUpgradePackageReq(deviceKind, nextName, nextVersion, nextDeviceConfigID, nextPackageURL); err != nil {
		return err
	}

	updates := map[string]interface{}{
		"device_kind": deviceKind,
	}
	if strings.TrimSpace(req.Name) != "" {
		updates["name"] = req.Name
	}
	if strings.TrimSpace(req.Version) != "" {
		updates["version"] = strings.TrimSpace(req.Version)
	}
	if req.TargetVersion != nil {
		updates["target_version"] = req.TargetVersion
	}
	if strings.TrimSpace(req.DeviceConfigID) != "" {
		updates["device_config_id"] = strings.TrimSpace(req.DeviceConfigID)
	}
	if req.BatteryModelID != nil {
		updates["battery_model_id"] = cleanOptionalStringPtr(req.BatteryModelID)
	}
	if req.BatchNumber != nil {
		updates["batch_number"] = cleanOptionalStringPtr(req.BatchNumber)
	}
	if req.ItemUUID != nil {
		updates["item_uuid"] = cleanOptionalStringPtr(req.ItemUUID)
	}
	if req.Module != nil {
		updates["module"] = req.Module
	}
	if req.PackageType != nil {
		updates["package_type"] = *req.PackageType
	}
	if req.SignatureType != nil {
		updates["signature_type"] = req.SignatureType
	}
	if req.AdditionalInfo != nil {
		updates["additional_info"] = req.AdditionalInfo
	}
	if req.Description != nil {
		updates["description"] = req.Description
	}
	if req.PackageUrl != nil {
		updates["package_url"] = req.PackageUrl
	}
	if req.Remark != nil {
		updates["remark"] = req.Remark
	}
	if req.IsLatest != nil {
		updates["is_latest"] = *req.IsLatest
	}
	if deviceKind == model.OTADeviceKindBMS {
		updates["device_config_id"] = ""
	}
	if deviceKind == model.OTADeviceKindMeter {
		updates["version"] = ""
		updates["target_version"] = nil
		updates["device_config_id"] = ""
		updates["battery_model_id"] = nil
		updates["batch_number"] = nil
		updates["item_uuid"] = nil
		updates["module"] = nil
		updates["package_type"] = int16(2)
		updates["is_latest"] = false
	}
	if deviceKind == model.OTADeviceKind4GModule {
		updates["target_version"] = nil
		updates["device_config_id"] = ""
		updates["battery_model_id"] = nil
		updates["batch_number"] = nil
		updates["item_uuid"] = nil
		updates["module"] = nil
		updates["package_type"] = int16(2)
		if req.IsLatest == nil {
			updates["is_latest"] = oldota.IsLatest
		}
	}
	packageChanged := req.PackageUrl != nil && (oldota.PackageURL == nil || strings.TrimSpace(*req.PackageUrl) != strings.TrimSpace(*oldota.PackageURL))
	if packageChanged {
		// 生成文件签名
		fileurl := *req.PackageUrl
		signatureType := "SHA256"
		if req.SignatureType != nil && strings.TrimSpace(*req.SignatureType) != "" {
			signatureType = *req.SignatureType
		} else if oldota.SignatureType != nil && strings.TrimSpace(*oldota.SignatureType) != "" {
			signatureType = *oldota.SignatureType
		}
		signature, err := signPackageSource(fileurl, signatureType)
		if err != nil {
			return err
		}
		updates["signature"] = signature
	}

	t := time.Now().UTC()
	updates["updated_at"] = &t
	tenantID := ""
	if oldota.TenantID != nil {
		tenantID = strings.TrimSpace(*oldota.TenantID)
	}
	return global.DB.Transaction(func(tx *gorm.DB) error {
		info := tx.Model(&model.OtaUpgradePackage{}).Where("id = ?", req.Id).Updates(updates)
		if info.Error != nil {
			return info.Error
		}
		if info.RowsAffected == 0 {
			return fmt.Errorf("no data updated")
		}
		if deviceKind == model.OTADeviceKind4GModule {
			if latest, ok := updates["is_latest"].(bool); ok && latest {
				return unsetOtherLatest4GPackages(tx, tenantID, req.Id)
			}
		}
		return nil
	})
}

func (*OTA) DeleteOTAUpgradePackage(packageId string) error {
	err := dal.DeleteOtaUpgradePackage(packageId)
	return err
}

func (*OTA) GetOTAUpgradePackageListByPage(req *model.GetOTAUpgradePackageLisyByPageReq, userClaims *utils.UserClaims) (map[string]interface{}, error) {
	total, list, err := dal.GetOtaUpgradePackageListByPage(req, userClaims.TenantID)
	if err != nil {
		return nil, err
	}
	packageListRspMap := make(map[string]interface{})
	packageListRspMap["total"] = total
	packageListRspMap["list"] = list
	return packageListRspMap, nil

}

func (*OTA) Check4GModuleUpgrade(ctx context.Context, req *model.GetOTA4GModuleUpgradeCheckReq, tenantID string) (*model.OTA4GModuleUpgradeCheckResp, error) {
	version := strings.TrimSpace(req.Version)
	imei := strings.TrimSpace(req.Imei)
	resp := &model.OTA4GModuleUpgradeCheckResp{
		NeedUpgrade:    false,
		CurrentVersion: version,
		Imei:           imei,
	}

	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "X-Tenant-ID is required")
	}

	var deviceCount int64
	if err := global.DB.WithContext(ctx).
		Table(model.TableNameDeviceBattery+" db").
		Joins("JOIN "+model.TableNameDevice+" d ON d.id = db.device_id").
		Where("d.tenant_id = ? AND (db.comm_chip_id = ? OR db.imei = ?)", tenantID, imei, imei).
		Count(&deviceCount).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if deviceCount == 0 {
		return resp, nil
	}

	var packages []model.OtaUpgradePackage
	if err := global.DB.WithContext(ctx).
		Table(model.TableNameOtaUpgradePackage).
		Where("(tenant_id = ? OR tenant_id IS NULL)", tenantID).
		Where("device_kind = ?", model.OTADeviceKind4GModule).
		Order("created_at DESC").
		Find(&packages).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if len(packages) == 0 {
		return resp, nil
	}

	candidates := make([]model.OtaUpgradePackage, 0, len(packages))
	for _, pkg := range packages {
		if compareVersion(pkg.Version, version) > 0 {
			candidates = append(candidates, pkg)
		}
	}
	if len(candidates) == 0 {
		return resp, nil
	}

	selected := candidates[0]
	if len(candidates) > 1 {
		foundLatest := false
		for _, pkg := range candidates {
			if pkg.IsLatest {
				if !foundLatest || compareVersion(pkg.Version, selected.Version) > 0 {
					selected = pkg
				}
				foundLatest = true
			}
		}
		if !foundLatest {
			return resp, nil
		}
	}

	resp.NeedUpgrade = true
	resp.Version = &selected.Version
	resp.FirmwareURL = buildOtaDownloadURL(selected.PackageURL)
	resp.PackageID = &selected.ID
	resp.Name = &selected.Name
	resp.Description = selected.Description
	resp.IsLatest = selected.IsLatest
	return resp, nil
}

type ota4GBMSDeviceRow struct {
	DeviceID       string  `gorm:"column:device_id"`
	ItemUUID       string  `gorm:"column:item_uuid"`
	BatteryModelID *string `gorm:"column:battery_model_id"`
	BatchNumber    *string `gorm:"column:batch_number"`
}

func (*OTA) Check4GBMSUpgrade(ctx context.Context, req *model.GetOTA4GBMSUpgradeCheckReq, tenantID string) (*model.OTA4GBMSUpgradeCheckResp, error) {
	version := strings.TrimSpace(req.Version)
	itemUUID := strings.TrimSpace(req.ItemUUID)
	resp := &model.OTA4GBMSUpgradeCheckResp{
		NeedUpgrade:    false,
		ItemUUID:       itemUUID,
		CurrentVersion: version,
	}

	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "tenant_id or X-Tenant-ID is required")
	}

	var row ota4GBMSDeviceRow
	if err := global.DB.WithContext(ctx).
		Table(model.TableNameDeviceBattery+" db").
		Select(`
			db.device_id AS device_id,
			db.item_uuid AS item_uuid,
			db.battery_model_id AS battery_model_id,
			db.batch_number AS batch_number
		`).
		Joins("JOIN "+model.TableNameDevice+" d ON d.id = db.device_id").
		Where("d.tenant_id = ? AND db.item_uuid = ?", tenantID, itemUUID).
		Scan(&row).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if row.DeviceID == "" {
		return resp, nil
	}
	if strings.TrimSpace(row.ItemUUID) != "" {
		resp.ItemUUID = strings.TrimSpace(row.ItemUUID)
	}

	var packages []model.OtaUpgradePackage
	if err := global.DB.WithContext(ctx).
		Table(model.TableNameOtaUpgradePackage).
		Where("(tenant_id = ? OR tenant_id IS NULL)", tenantID).
		Where("device_kind = ?", model.OTADeviceKindBMS).
		Order("created_at DESC").
		Find(&packages).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if len(packages) == 0 {
		return resp, nil
	}

	selected := selectAppBatteryOtaPackage(packages, version, appBatteryOtaMatchCriteria{
		BatteryModelID: firstTrimmed(row.BatteryModelID),
		BatchNumber:    firstTrimmed(row.BatchNumber),
		ItemUUID:       strings.TrimSpace(row.ItemUUID),
	})
	if selected == nil {
		return resp, nil
	}

	resp.NeedUpgrade = true
	resp.Version = &selected.Version
	resp.TargetVersion = selected.TargetVersion
	resp.FirmwareURL = buildOtaDownloadURL(selected.PackageURL)
	resp.PackageID = &selected.ID
	packageType := selected.PackageType
	resp.PackageType = &packageType
	resp.SignatureType = selected.SignatureType
	resp.Signature = selected.Signature
	resp.Module = selected.Module
	resp.Description = selected.Description
	resp.AdditionalInfo = selected.AdditionalInfo
	resp.Remark = selected.Remark
	return resp, nil
}

func (o *OTA) CreateOTAUpgradeTask(req *model.CreateOTAUpgradeTaskReq) error {
	tasks, err := dal.CreateOTAUpgradeTaskWithDetail(req)
	if err == nil {
		go func() {
			for _, t := range tasks {
				o.PushOTAUpgradePackage(t)
			}
		}()
	}
	return err
}

func (*OTA) DeleteOTAUpgradeTask(id string) error {
	err := dal.DeleteOTAUpgradeTask(id)
	return err
}

func (*OTA) GetOTAUpgradeTaskListByPage(req *model.GetOTAUpgradeTaskListByPageReq) (map[string]interface{}, error) {
	total, list, err := dal.GetOtaUpgradeTaskListByPage(req)
	if err != nil {
		return nil, err
	}
	dataMap := make(map[string]interface{})
	dataMap["total"] = total
	dataMap["list"] = list
	return dataMap, nil
}

func (*OTA) GetOTAUpgradeTaskDetailListByPage(req *model.GetOTAUpgradeTaskDetailReq) (map[string]interface{}, error) {
	total, list, statistics, err := dal.GetOtaUpgradeTaskDetailListByPage(req)
	if err != nil {
		return nil, err
	}
	dataMap := make(map[string]interface{})
	dataMap["total"] = total
	dataMap["statistics"] = statistics
	dataMap["list"] = list
	return dataMap, nil
}

// 设备状态修改(请求参数1-取消升级 2-重新升级)
// 1-待推送 2-已推送 3-升级中 修改为已取消
// 5-升级失败 修改为待推送
// 4-升级成功 6-已取消 不修改
func (o *OTA) UpdateOTAUpgradeTaskStatus(req *model.UpdateOTAUpgradeTaskStatusReq) error {
	taskDetail, err := query.OtaUpgradeTaskDetail.Where(query.OtaUpgradeTaskDetail.ID.Eq(req.Id)).First()
	if err != nil {
		return err
	}
	// 4-升级成功 6-已取消 不修改
	if taskDetail.Status == 4 || taskDetail.Status == 6 {
		return fmt.Errorf("the task status cannot be modified")
	}
	// 升级成功的任务不能取消升级
	if req.Action == 6 && taskDetail.Status == 5 {
		return fmt.Errorf("the task status cannot be modified")
	}
	// 1-待推送 2-已推送 3-升级中 不能重新升级
	if req.Action == 1 && taskDetail.Status <= 3 {
		return fmt.Errorf("the task is upgrading")
	}
	t := time.Now().UTC()
	if req.Action == 6 {
		//取消升级
		taskDetail.Status = 6
		taskDetail.UpdatedAt = &t
		desc := "手动取消升级"
		taskDetail.StatusDescription = &desc
		_, err := query.OtaUpgradeTaskDetail.Updates(taskDetail)
		return err
	}
	if req.Action == 1 {
		desc := "手动开始重新升级"
		startStep := int16(0)
		//重新升级
		taskDetail.Status = 1
		taskDetail.UpdatedAt = &t
		taskDetail.StatusDescription = &desc
		taskDetail.Step = &startStep

		_, err := query.OtaUpgradeTaskDetail.Updates(taskDetail)
		if err != nil {
			return err
		}
		// 重新升级后推送升级包
		err = o.PushOTAUpgradePackage(taskDetail)
		return err
	}

	return err
}
func (*OTA) PushOTAUpgradePackage(taskDetail *model.OtaUpgradeTaskDetail) error {
	// 查看设备是否在线
	device := &model.Device{}
	device, err := query.Device.Where(query.Device.ID.Eq(taskDetail.DeviceID)).First()
	if err != nil {
		return err
	}
	if device.IsOnline != 1 {
		//修改设备升级任务信息
		taskDetail.Status = 5
		desc := "设备离线"
		taskDetail.StatusDescription = &desc
		t := time.Now().UTC()
		taskDetail.UpdatedAt = &t
		_, err := query.OtaUpgradeTaskDetail.Updates(taskDetail)
		if err != nil {
			return err
		}
		return fmt.Errorf("the device is offline")
	}
	// 查看设备是否有其他升级中的任务
	count, err := query.OtaUpgradeTaskDetail.Where(query.OtaUpgradeTaskDetail.DeviceID.Eq(taskDetail.DeviceID), query.OtaUpgradeTaskDetail.Status.Lt(4)).Count()
	if err != nil {
		return err
	}
	if count > 0 {
		//修改设备升级任务信息
		taskDetail.Status = 5
		desc := "上次升级未完成"
		taskDetail.StatusDescription = &desc
		t := time.Now().UTC()
		taskDetail.UpdatedAt = &t
		_, err := query.OtaUpgradeTaskDetail.Updates(taskDetail)
		if err != nil {
			return err
		}
		return fmt.Errorf("the device is upgrading")
	}
	// 推送升级包
	taskQuery, err := query.OtaUpgradeTask.
		Select(query.OtaUpgradeTask.OtaUpgradePackageID).
		Where(query.OtaUpgradeTask.ID.Eq(taskDetail.OtaUpgradeTaskID)).
		First()
	if err != nil {
		return err
	}
	packageID := taskQuery.OtaUpgradePackageID
	otapackage, err := query.OtaUpgradePackage.Where(query.OtaUpgradePackage.ID.Eq(packageID)).First()
	if err != nil {
		return err
	}
	var otamsg = make(map[string]interface{})
	// 获取随机九位数字并转换为字符串
	randNum, err := common.GetRandomNineDigits()
	if err != nil {
		return err
	}
	otamsg["id"] = randNum
	otamsg["code"] = "200"
	var otamsgparams = make(map[string]interface{})
	otamsgparams["version"] = otapackage.Version
	otamsgparams["size"] = "0"
	downloadURL := buildOtaDownloadURL(otapackage.PackageURL)
	if downloadURL == nil {
		return fmt.Errorf("ota package url is empty")
	}
	otamsgparams["url"] = *downloadURL
	otamsgparams["signMethod"] = otapackage.SignatureType
	otamsgparams["sign"] = ""
	otamsgparams["module"] = otapackage.Module
	//其他配置格式成map
	var m map[string]interface{}
	err = json.Unmarshal([]byte(*otapackage.AdditionalInfo), &m)
	if err != nil {
		logrus.Error(err)
	}
	otamsgparams["extData"] = m
	otamsg["params"] = otamsgparams
	palyload, json_err := json.Marshal(otamsg)
	if json_err != nil {
		logrus.Error(err)
	} else {
		// 修改设备升级任务信息
		taskDetail.Status = 2
		desc := "已通知设备"
		taskDetail.StatusDescription = &desc
		t := time.Now().UTC()
		taskDetail.UpdatedAt = &t
		_, err := query.OtaUpgradeTaskDetail.Updates(taskDetail)
		if err != nil {
			return err
		}
		go publish.PublishOtaAdress(device.DeviceNumber, palyload)
	}

	return nil
}
