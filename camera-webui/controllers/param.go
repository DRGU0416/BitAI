package controllers

type RespCode int

const (
	FAILURE       RespCode = 0 // 失败
	SUCCESS       RespCode = 1 // 成功
	INVALID_PARAM RespCode = 2 // 参数错误
)

type Response struct {
	Code RespCode    `json:"code"`
	Data interface{} `json:"data,omitempty"`
}
