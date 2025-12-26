package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	dal "project/internal/dal"
	model "project/internal/model"
	"project/pkg/errcode"
	utils "project/pkg/utils"
	"project/third_party/others/http_client"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v4/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/go-basic/uuid"
	"github.com/sirupsen/logrus"
	"gopkg.in/gomail.v2"
)

type NotificationServicesConfig struct{}

// 统一的通知历史记录保存方法
func (n *NotificationServicesConfig) saveNotificationHistory(notificationType, tenantID, target, content, status string, remark *string) error {
	history := &model.NotificationHistory{
		ID:               uuid.New(),
		SendTime:         time.Now().UTC(),
		SendContent:      &content,
		SendTarget:       target,
		SendResult:       &status,
		NotificationType: notificationType,
		TenantID:         tenantID,
		Remark:           remark,
	}

	err := GroupApp.NotificationHisory.SaveNotificationHistory(history)
	if err != nil {
		logrus.Error("保存通知历史记录失败:", err)
		return err
	}
	return nil
}

// 发送webhook通知的方法
func (n *NotificationServicesConfig) sendWebhookMessage(payloadURL, secret, alertJson, tenantID string) error {
	// 验证JSON格式并确保不转义
	var alertData map[string]interface{}
	err := json.Unmarshal([]byte(alertJson), &alertData)
	if err != nil {
		logrus.Error("告警JSON格式错误:", err)
		return err
	}

	// 重新序列化JSON，禁用HTML转义
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false) // 关键：不转义HTML字符，避免 > 变成 \u003e
	err = encoder.Encode(alertData)
	if err != nil {
		logrus.Error("重新序列化JSON失败:", err)
		return err
	}
	cleanJson := strings.TrimSpace(buffer.String()) // 去掉encoder.Encode添加的换行符

	// 创建PENDING记录
	historyID := uuid.New()
	pendingStatus := "PENDING"
	history := &model.NotificationHistory{
		ID:               historyID,
		SendTime:         time.Now().UTC(),
		SendContent:      &cleanJson,
		SendTarget:       payloadURL,
		SendResult:       &pendingStatus,
		NotificationType: model.NoticeType_Webhook,
		TenantID:         tenantID,
		Remark:           nil,
	}

	err = GroupApp.NotificationHisory.SaveNotificationHistory(history)
	if err != nil {
		logrus.Error("创建webhook通知历史记录失败:", err)
		return err
	}

	// 发送webhook，带重试机制
	var lastErr error
	maxRetries := 2 // 总共尝试2次（第一次+重试1次）

	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			logrus.Info(fmt.Sprintf("Webhook发送重试，第%d次", i))
		}

		// 创建带超时的context
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = http_client.SendSignedRequestWithTimeout(ctx, payloadURL, cleanJson, secret)
		if err == nil {
			// 发送成功，更新记录
			successStatus := "SUCCESS"
			_, updateErr := dal.UpdateNotificationHistory(historyID, &successStatus, nil)
			if updateErr != nil {
				logrus.Error("更新webhook通知历史记录失败:", updateErr)
			}
			logrus.Info("Webhook发送成功:", payloadURL)
			return nil
		}
		lastErr = err
		logrus.Error(fmt.Sprintf("Webhook发送失败，第%d次尝试:", i+1), err)
	}

	// 所有重试都失败，更新记录，在JSON后追加错误信息
	failureStatus := "FAILURE"
	errorContent := cleanJson + "; Webhook发送失败: " + lastErr.Error()
	remarkText := lastErr.Error()

	// 更新记录的内容和状态
	_, updateErr := dal.UpdateNotificationHistoryWithContent(historyID, &failureStatus, &remarkText, &errorContent)
	if updateErr != nil {
		logrus.Error("更新webhook通知历史记录失败:", updateErr)
	}

	return lastErr
}

func (*NotificationServicesConfig) SaveNotificationServicesConfig(req *model.SaveNotificationServicesConfigReq) (*model.NotificationServicesConfig, error) {
	// 查找数据库中是否存在
	c, err := dal.GetNotificationServicesConfigByType(req.NoticeType)
	if err != nil {
		return nil, err
	}

	config := model.NotificationServicesConfig{}

	var strconf []byte
	switch req.NoticeType {
	case model.NoticeType_Email:
		strconf, err = json.Marshal(req.EMailConfig)
		if err != nil {
			return nil, err
		}
	case model.NoticeType_SME_CODE:
		strconf, err = json.Marshal(req.SMEConfig)
		if err != nil {
			return nil, err
		}
	}

	if c == nil {
		config.ID = uuid.New()
	} else {
		config.ID = c.ID
	}

	configStr := string(strconf)
	config.NoticeType = req.NoticeType
	config.Remark = req.Remark
	config.Status = req.Status
	config.Config = &configStr

	data, err := dal.SaveNotificationServicesConfig(&config)
	if err != nil {
		return nil, err
	}

	return data, err
}

func (*NotificationServicesConfig) GetNotificationServicesConfig(noticeType string) (*model.NotificationServicesConfig, error) {
	c, err := dal.GetNotificationServicesConfigByType(noticeType)
	return c, err
}

func (*NotificationServicesConfig) SendTestEmail(req *model.SendTestEmailReq) error {
	// 校验邮箱
	if !utils.ValidateEmail(req.Email) {
		return errcode.New(200014)
	}
	c, err := dal.GetNotificationServicesConfigByType(model.NoticeType_Email)
	if err != nil {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"notice_type": err.Error(),
		})
	}
	if c == nil {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"error": "邮件服务配置不存在",
		})
	}
	if c.Config == nil {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"error": "邮件服务配置内容为空",
		})
	}
	var emailConf model.EmailConfig
	err = json.Unmarshal([]byte(*c.Config), &emailConf)
	if err != nil {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	m := gomail.NewMessage()
	// 设置发件人
	m.SetHeader("From", emailConf.FromEmail)
	// 设置收件人，可以有多个
	m.SetHeader("To", req.Email)
	// 设置邮件主题
	m.SetHeader("Subject", "Iot平台-验证码通知")
	// 设置邮件正文。可以是纯文本或者HTML
	m.SetBody("text/html", req.Body)

	// cokyahsoudtdbahe
	// 设置SMTP服务器（以Gmail为例），并提供认证信息
	d := gomail.NewDialer(emailConf.Host, emailConf.Port, emailConf.FromEmail, emailConf.FromPassword)

	// 发送邮件
	if err := d.DialAndSend(m); err != nil {
		return errcode.WithData(errcode.CodeSystemError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return nil
}

// SendSMSByTemplate 发送短信（目前仅支持阿里云）
// - templateCode 为空时，回退使用 notification_services_config(SME_CODE) 中的默认 template_code
// - params 为短信模板参数（会被序列化为 JSON 字符串）
func (*NotificationServicesConfig) SendSMSByTemplate(ctx context.Context, tenantID, phone, templateCode string, params map[string]string) error {
	c, err := dal.GetNotificationServicesConfigByType(model.NoticeType_SME_CODE)
	if err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"notice_type": model.NoticeType_SME_CODE,
			"error":       err.Error(),
		})
	}
	if c == nil || c.Config == nil {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"error": "短信服务配置不存在",
		})
	}
	if c.Status != model.OPEN {
		return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{
			"error": "短信服务未开启",
		})
	}

	var smeConf model.SMEConfig
	if err := json.Unmarshal([]byte(*c.Config), &smeConf); err != nil {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"error": err.Error(),
		})
	}
	if strings.ToUpper(strings.TrimSpace(smeConf.Provider)) != "ALIYUN" {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"error":    "unsupported sms provider",
			"provider": smeConf.Provider,
		})
	}
	if smeConf.AliyunSMSConfig == nil {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"error": "aliyun sms config is empty",
		})
	}

	ali := smeConf.AliyunSMSConfig
	if strings.TrimSpace(templateCode) == "" {
		templateCode = ali.TemplateCode
	}
	if strings.TrimSpace(templateCode) == "" {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"error": "sms template_code is empty",
		})
	}

	paramJSON, _ := json.Marshal(params)

	openapiConf := &openapi.Config{
		AccessKeyId:     tea.String(ali.AccessKeyID),
		AccessKeySecret: tea.String(ali.AccessKeySecret),
		Endpoint:        tea.String(ali.Endpoint),
	}
	client, err := dysmsapi.NewClient(openapiConf)
	if err != nil {
		return errcode.WithData(errcode.CodeSystemError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	req := &dysmsapi.SendSmsRequest{
		PhoneNumbers:  tea.String(phone),
		SignName:      tea.String(ali.SignName),
		TemplateCode:  tea.String(templateCode),
		TemplateParam: tea.String(string(paramJSON)),
	}
	resp, err := client.SendSms(req)
	if err != nil {
		return errcode.WithData(errcode.CodeSystemError, map[string]interface{}{
			"error": err.Error(),
		})
	}
	if resp == nil || resp.Body == nil || resp.Body.Code == nil || *resp.Body.Code != "OK" {
		code := ""
		msg := ""
		if resp != nil && resp.Body != nil {
			if resp.Body.Code != nil {
				code = *resp.Body.Code
			}
			if resp.Body.Message != nil {
				msg = *resp.Body.Message
			}
		}
		return errcode.WithData(errcode.CodeSystemError, map[string]interface{}{
			"error":      "sms send failed",
			"provider":   "ALIYUN",
			"code":       code,
			"message":    msg,
			"tenant_id":  tenantID,
			"phone":      phone,
			"template":   templateCode,
			"request_id": "",
		})
	}
	return nil
}

// Send email message
func sendEmailMessage(message string, subject string, tenantId string, to ...string) (err error) {
	c, err := dal.GetNotificationServicesConfigByType(model.NoticeType_Email)
	if err != nil {
		return err
	}
	if c == nil {
		return fmt.Errorf("邮件服务配置不存在")
	}
	if c.Config == nil {
		return fmt.Errorf("邮件服务配置内容为空")
	}
	var emailConf model.EmailConfig
	err = json.Unmarshal([]byte(*c.Config), &emailConf)
	if err != nil {
		return err
	}

	d := gomail.NewDialer(emailConf.Host, emailConf.Port, emailConf.FromEmail, emailConf.FromPassword)

	m := gomail.NewMessage()
	m.SetHeader("From", emailConf.FromEmail)
	m.SetHeader("To", to...)
	m.SetBody("text/plain", message)
	m.SetHeader("Subject", subject)

	// 使用统一的通知历史记录方法
	nsc := &NotificationServicesConfig{}

	// 记录数据库
	if err := d.DialAndSend(m); err != nil {
		logrus.Error("邮件发送失败:", err)
		remark := err.Error()
		nsc.saveNotificationHistory(model.NoticeType_Email, tenantId, to[0], message, "FAILURE", &remark)
		return err
	} else {
		logrus.Info("邮件发送成功:", to[0])
		nsc.saveNotificationHistory(model.NoticeType_Email, tenantId, to[0], message, "SUCCESS", nil)
	}
	return nil
}

// Send notification
func (*NotificationServicesConfig) ExecuteNotification(notificationGroupId, alertJson string) {
	notificationGroup, err := dal.GetNotificationGroupById(notificationGroupId)
	if err != nil {
		logrus.Error("获取通知组失败:", err)
		return
	}

	if notificationGroup.Status != "OPEN" {
		logrus.Info("通知组未开启:", notificationGroupId)
		return
	}

	switch notificationGroup.NotificationType {
	case model.NoticeType_Member:
		// TODO: SEND TO MEMBER - 成员通知功能待实现
		logrus.Info("成员通知功能尚未实现:", notificationGroupId)

	case model.NoticeType_Email:
		nConfig := make(map[string]string)
		err := json.Unmarshal([]byte(*notificationGroup.NotificationConfig), &nConfig)
		if err != nil {
			logrus.Error("解析邮件配置失败:", err)
			return
		}

		// 解析标准通知JSON
		var alertData map[string]interface{}
		err = json.Unmarshal([]byte(alertJson), &alertData)
		if err != nil {
			logrus.Error("解析告警JSON失败:", err)
			return
		}

		subject, _ := alertData["subject"].(string)
		content, _ := alertData["content"].(string)

		// 邮件特定格式：添加邮件签名
		emailBody := content + "\n\n---\nThis email was sent by ThingsPanel"

		emailList := strings.Split(nConfig["EMAIL"], ",")
		for _, emailAddr := range emailList {
			emailAddr = strings.TrimSpace(emailAddr)
			if emailAddr != "" {
				err := sendEmailMessage(emailBody, subject, notificationGroup.TenantID, emailAddr)
				if err != nil {
					// 在JSON后追加错误信息
					errorContent := alertJson + "; 邮件发送失败: " + err.Error()
					nsc := &NotificationServicesConfig{}
					nsc.saveNotificationHistory(model.NoticeType_Email, notificationGroup.TenantID, emailAddr, errorContent, "FAILURE", nil)
					logrus.Error("发送邮件失败:", err)
				}
			}
		}

	case model.NoticeType_Webhook:
		type WebhookConfig struct {
			PayloadURL string
			Secret     string
		}
		var nConfig WebhookConfig
		err = json.Unmarshal([]byte(*notificationGroup.NotificationConfig), &nConfig)
		if err != nil {
			logrus.Error("解析Webhook配置失败:", err)
			return
		}

		// 使用新的webhook发送方法，传递原始JSON
		nsc := &NotificationServicesConfig{}
		err = nsc.sendWebhookMessage(nConfig.PayloadURL, nConfig.Secret, alertJson, notificationGroup.TenantID)
		if err != nil {
			logrus.Error("Webhook通知发送失败:", err)
		}

	default:
		logrus.Warn("未支持的通知类型:", notificationGroup.NotificationType)
		return
	}
}
