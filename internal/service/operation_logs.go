package service

import (
	"strings"

	dal "project/internal/dal"
	model "project/internal/model"
	"project/pkg/errcode"
	utils "project/pkg/utils"

	"github.com/sirupsen/logrus"
)

type OperationLogs struct{}

func inferOpModule(path string) string {
	switch {
	case strings.Contains(path, "/battery"):
		return "电池管理"
	case strings.Contains(path, "/battery_model"):
		return "电池型号"
	case strings.Contains(path, "/device_transfer"):
		return "设备转移"
	case strings.Contains(path, "/warranty"):
		return "维保管理"
	case strings.Contains(path, "/dealer"):
		return "经销商管理"
	case strings.Contains(path, "/end_user"):
		return "终端用户"
	case strings.Contains(path, "/ota"):
		return "OTA"
	case strings.Contains(path, "/sys_user") || strings.Contains(path, "/user"):
		return "系统管理"
	default:
		return "其它"
	}
}

func inferOpType(method string, path string) string {
	// 结合 path 进行更细粒度的“下发/导入”等判定
	if method == "POST" {
		if strings.Contains(path, "/import") {
			return "导入"
		}
		if strings.Contains(path, "/export") {
			return "导出"
		}
		if strings.Contains(path, "/batch") {
			return "批量操作"
		}
		if strings.Contains(path, "/bind") {
			return "绑定"
		}
		if strings.Contains(path, "/unbind") {
			return "解绑"
		}
		return "新增/下发"
	}
	if method == "PUT" {
		return "编辑"
	}
	if method == "DELETE" {
		return "删除"
	}
	return method
}

func inferContent(path string, method string, req *string) string {
	// 优先给出可读内容；没有就用 path+method 兜底
	if req != nil && *req != "" {
		// 只保留前 200 字符，避免页面过长
		s := *req
		if len(s) > 200 {
			s = s[:200] + "...(truncated)"
		}
		return s
	}
	return method + " " + path
}

func (*OperationLogs) CreateOperationLogs(operationLog *model.OperationLog) error {
	err := dal.CreateOperationLogs(operationLog)

	if err != nil {
		logrus.Error(err)
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	return err
}

// 分页查询日志
func (*OperationLogs) GetListByPage(Params *model.GetOperationLogListByPageReq, userClaims *utils.UserClaims) (map[string]interface{}, error) {

	total, list, err := dal.GetListByPage(Params, userClaims)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}
	OperationLogsListRsp := make(map[string]interface{})
	OperationLogsListRsp["total"] = total
	// 派生 module/op_type/content 字段
	if rows, ok := list.([]model.GetOperationLogListByPageRsp); ok {
		out := make([]model.GetOperationLogListByPageRsp, 0, len(rows))
		for _, r := range rows {
			p := SafeDeref(r.Path)
			m := SafeDeref(r.Name)
			r.Module = inferOpModule(p)
			r.OpType = inferOpType(m, p)
			// 默认用 request_message；remark 在此项目里常用于存 User-Agent 或其它补充
			r.Content = inferContent(p, m, r.RequestMessage)
			out = append(out, r)
		}

		// OpType 过滤（在派生后做）
		if Params.OpType != nil && *Params.OpType != "" {
			filtered := make([]model.GetOperationLogListByPageRsp, 0, len(out))
			for _, r := range out {
				if r.OpType == *Params.OpType {
					filtered = append(filtered, r)
				}
			}
			out = filtered
		}

		OperationLogsListRsp["list"] = out
	} else {
		OperationLogsListRsp["list"] = list
	}

	return OperationLogsListRsp, err
}
