package httputil

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
)

type JsonRawBody map[string]any

// JsonListBody 分页响应体
type JsonListBody struct {
	List  any    `json:"list"`
	Total int64  `json:"total"`
	Code  int    `json:"code,omitempty"`
	Msg   string `json:"msg,omitempty"`
}

// JsonDataBody json响应体
type JsonDataBody struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}

// Ok 返回成功信息, params作为动态参数，默认没有参数则返回204
func Ok(w http.ResponseWriter, params ...any) {
	if len(params) == 0 || params[0] == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	data := params[0]
	str, ok := data.(string)
	if ok {
		w.Header().Set("Content-Length", strconv.FormatInt(int64(len(str)), 10))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(str)) // nolint
		return
	}
	by, ok := data.([]byte)
	if ok {
		w.Header().Set("Content-Length", strconv.FormatInt(int64(len(by)), 10))
		w.WriteHeader(http.StatusOK)
		w.Write(by) // nolint
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data) // nolint
}

// OkList 返回成功列表
func OkList(w http.ResponseWriter, list any, total int64) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(JsonRawBody{ // nolint
		"list":  list,
		"total": total,
	})
}

// Bad 错误的请求
func Bad(w http.ResponseWriter, params ...any) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if len(params) == 0 || params[0] == nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(JsonRawBody{ // nolint
			"error": "请求参数有误",
			"msg":   "请求参数有误",
			"code":  http.StatusBadRequest,
		})
		return
	}
	data := params[0]
	BadError(w, http.StatusBadRequest, data, params[1:]...)
}

// BadError 返回错误信息
func BadError(w http.ResponseWriter, status int, data any, params ...any) {
	if data == nil {
		w.WriteHeader(status)
		return
	}
	// lang := w.Header().Get("Accept-Language")
	// if lang == "" {
	// 	lang = "en"
	// }
	switch v := data.(type) {
	case string:
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(JsonRawBody{"msg": v, "error": v, "code": status}) // nolint
	case validator.ValidationErrors:
		w.WriteHeader(status)
		var msg string
		for _, fieldError := range v {
			msg = validatorMsg(fieldError)
			break
		}
		json.NewEncoder(w).Encode(JsonRawBody{"msg": msg, "error": msg, "code": status}) // nolint
		return
	case error:
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(JsonRawBody{"msg": v.Error(), "error": v.Error(), "code": status}) // nolint
	default:
		w.WriteHeader(status)
	}
}

// Forbidden Forbidden
func Forbidden(w http.ResponseWriter, err any) {
	BadError(w, http.StatusForbidden, 0, err)
}

// BadW 返回错误信息
func BadW(w http.ResponseWriter, msg string) error {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(fmt.Sprintf("{\"msg\":\"%s\"}", msg))) //nolint
	return errors.New(msg)
}

var fieldMap = map[string]string{
	"Username":       "用户名",
	"Nickname":       "昵称",
	"Password":       "密码",
	"OldPassword":    "旧密码",
	"NewPassword":    "新密码",
	"Role":           "角色",
	"Sort":           "排序",
	"Name":           "名称",
	"Email":          "邮箱",
	"Phone":          "手机号",
	"Count":          "参数Count",
	"GroupId":        "组Id",
	"StartNum":       "开始序号",
	"UserPrefix":     "用户名前缀",
	"CreatedAtStart": "开始时间",
	"CreatedAtEnd":   "结束时间",
	"StartTime":      "开始时间",
	"EndTime":        "结束时间",
	"Quota":          "存储配额",
}
var tagMap = map[string]string{
	"required":      "不能为空",
	"min":           "长度太短",
	"max":           "长度太长",
	"lt":            "值太大",
	"gt":            "值太小",
	"lte":           "值太大",
	"gte":           "值太小",
	"email":         "格式不正确",
	"isUserName":    "格式不正确",
	"isNickName":    "格式不正确",
	"isGroupName":   "格式不正确",
	"isMobile":      "格式不正确",
	"isLessThanNow": "不能大于当前时间",
	"ltefield":      "不能大于区间范围的最大值",
	"isLessEndTime": "不能大于结束时间",
}

// validatorMsg
func validatorMsg(e validator.FieldError) string {
	field := e.Field()
	fieldVal, findField := fieldMap[field]
	tagVal, findTag := tagMap[e.Tag()]
	if field == "NewPassword" {
		return "新密码过于简单不安全，请使用更复杂安全的新密码"
	}
	if findField && findTag {
		return fieldVal + tagVal
	}
	if !findField && findTag {
		return "参数" + field + tagVal
	}
	return "请求参数错误"
}
