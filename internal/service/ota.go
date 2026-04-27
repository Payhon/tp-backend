package service

import (
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
	utils "project/pkg/utils"

	"github.com/go-basic/uuid"
	"github.com/sirupsen/logrus"
)

type OTA struct{}

func resolveOTADeviceKind(kind *int16) int16 {
	if kind == nil || *kind == 0 {
		return model.OTADeviceKindBMS
	}
	if *kind == model.OTADeviceKindMeter {
		return model.OTADeviceKindMeter
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
	if strings.TrimSpace(deviceConfigID) == "" {
		return fmt.Errorf("device_config_id is required")
	}
	return nil
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
	ota.Module = req.Module
	ota.SignatureType = req.SignatureType
	ota.PackageType = 2
	if req.PackageType != nil {
		ota.PackageType = *req.PackageType
	}
	if deviceKind == model.OTADeviceKindMeter {
		ota.Version = ""
		ota.TargetVersion = nil
		ota.DeviceConfigID = ""
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
	err = dal.CreateOtaUpgradePackage(&ota)
	return err
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

	var ota = model.OtaUpgradePackage{}
	ota.ID = req.Id
	ota.DeviceKind = deviceKind

	ota.Name = req.Name
	ota.Version = req.Version
	ota.TargetVersion = req.TargetVersion
	ota.DeviceConfigID = req.DeviceConfigID
	ota.Module = req.Module
	if req.PackageType != nil {
		ota.PackageType = *req.PackageType
	}
	if req.SignatureType != nil {
		ota.SignatureType = req.SignatureType
	}
	ota.AdditionalInfo = req.AdditionalInfo
	ota.Description = req.Description
	ota.PackageURL = req.PackageUrl
	if deviceKind == model.OTADeviceKindMeter {
		ota.Version = ""
		ota.TargetVersion = nil
		ota.DeviceConfigID = ""
		ota.Module = nil
		ota.PackageType = 2
	}
	if req.PackageUrl != oldota.PackageURL {
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
		ota.Signature = &signature
	}

	t := time.Now().UTC()
	ota.UpdatedAt = &t
	ota.Remark = req.Remark
	info, err := dal.UpdateOtaUpgradePackage(&ota)
	if err != nil {
		return err
	}
	if info.RowsAffected == 0 {
		return fmt.Errorf("no data updated")
	}
	return nil
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
