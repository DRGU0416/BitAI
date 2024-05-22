package controllers

const (
	// 绘图最大上传数量
	UPLOAD_IMAGE_MAX = 6
	// 拼图最大上传数量
	SELECT_MATERIAL_MAX = 10

	// 下载消耗钻石数
	DIAMOND_DOWNLOAD = 2
	// 高清处理消耗钻石数
	DIAMOND_HIGHER = 2
)

type RespCode int

const (
	FAILURE               RespCode = 0 // 失败
	SUCCESS               RespCode = 1 // 成功
	INVALID_PARAM         RespCode = 2 // 参数错误
	RELOGIN               RespCode = 3 // 重新登录
	NO_CARD_TIMES         RespCode = 4 // 没有分身次数
	CARD_FRONT_NOT_ENOUGH RespCode = 5 // 正面照数量不足
	CARD_SIDE_NOT_ENOUGH  RespCode = 6 // 侧面照数量不足
	DIAMOND_NOT_ENOUGH    RespCode = 7 // 钻石不足

	MESSAGE_IS_SEND   RespCode = 1001 // 短信已发送，请等待
	GEN_TASK_RUNNING  RespCode = 1002 // 任务正在执行
	GEN_TASK_CANCELED RespCode = 1003 // 任务被撤消
	GEN_TASK_FAILED   RespCode = 1004 // 任务执行失败
	IMAGE_SIZE_BIG    RespCode = 1005 // 头像文件最大100KB
	PAY_CONFIRM_RETRY RespCode = 1006 // 支付确认失败，请重试
)

type Response struct {
	Code RespCode    `json:"code"`
	Data interface{} `json:"data,omitempty"`
}

const (
	MJ_SUCCESS               = 0 // 成功
	MJ_BAD_REQUEST           = 1 // 参数错误
	MJ_ERR_GEN_COMMAND       = 2 // 生成图片失败,通常是授权失败,要注意替换授权凭证
	MJ_ERR_PARSE_DISCORD_MSG = 3 // 解析Discord Message与预期不一致
	MJ_ERR_DOWNLOAD_IMAGE    = 4 // 下载图片失败
	MJ_ERR_UPLOAD_IMAGE      = 5 // 上传图片到OSS失败
)
