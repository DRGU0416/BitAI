package lib

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Baidu-AIP/golang-sdk/aip/censor"
)

var (
	baiduTransUrl = "https://fanyi-api.baidu.com/api/trans/vip/translate"
)

type baiduTransResponse struct {
	From         string `json:"from,omitempty"`
	To           string `json:"to,omitempty"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_msg,omitempty"`
	TransResult  []struct {
		Dst string `json:"dst"`
	} `json:"trans_result,omitempty"`
}

// 百度翻译
// API文档: http://api.fanyi.baidu.com/product/113
// 标准版QPS（每秒请求量）=1，高级版（适用于个人，QPS=10）或尊享版（适用于企业，QPS=100）
func BaiduTranslate(baiduAppID, baiduAppKey, msg string) (string, error) {
	salt := strconv.Itoa(GetRand(100000, 999999))
	content := fmt.Sprintf("%s%s%s%s", baiduAppID, msg, salt, baiduAppKey)
	sign := GetMd5([]byte(content))
	url := fmt.Sprintf(`%s?q=%s&from=auto&to=en&appid=%s&salt=%s&sign=%s`, baiduTransUrl, url.QueryEscape(msg), baiduAppID, salt, sign)

	request, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("request error: %v", err)
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read error: %v", err)
	}

	respData := baiduTransResponse{}
	if err = json.Unmarshal(result, &respData); err != nil {
		return "", fmt.Errorf("json unmarshal error: %v", err)
	}

	if respData.ErrorCode != "52000" && respData.ErrorCode != "" {
		return "", fmt.Errorf("baidu error code: %s %s", respData.ErrorCode, respData.ErrorMessage)
	}
	if len(respData.TransResult) == 0 {
		return "", fmt.Errorf("no result")
	}
	return respData.TransResult[0].Dst, nil
}

// 判断是否全英文
func IsEnglish(text string) bool {
	for _, char := range text {
		if char < 32 || char > 126 {
			return false
		}
	}
	return true
}

type baiduCensorResponse struct {
	LogId          uint64 `json:"log_id"`
	ErrorCode      uint64 `json:"error_code"`
	ErrorMsg       string `json:"error_msg"`
	ConclusionType uint64 `json:"conclusionType"`
}

// 百度图片审核
// 文档：https://ai.baidu.com/ai-doc/ANTIPORN/jk42xep4e
// 个人认证QPS（每秒请求量）=2，企业认证QPS（每秒请求量）=2
func BaiduImageCensor(baiduAppID, baiduAppKey, imgPath string) (uint64, error) {
	client := censor.NewClient(baiduAppID, baiduAppKey)
	// base64d := util.ReadFileToBase64(imgPath)
	// result := client.ImgCensor(base64d, param)
	result := client.ImgCensorUrl(imgPath, nil)

	respData := baiduCensorResponse{}
	if err := json.Unmarshal([]byte(result), &respData); err != nil {
		return 0, fmt.Errorf("json unmarshal error: %v", err)
	}
	if respData.ErrorCode != 0 {
		return 0, fmt.Errorf("baidu censor error: %d %s", respData.ErrorCode, respData.ErrorMsg)
	}
	return respData.ConclusionType, nil
}

// 百度文本审核
// 文档：https://ai.baidu.com/ai-doc/ANTIPORN/jk42xep4e
// 个人认证QPS（每秒请求量）=2，企业认证QPS（每秒请求量）=5
func BaiduTextCensor(baiduAppID, baiduAppKey, msg string) (uint64, error) {
	client := censor.NewClient(baiduAppID, baiduAppKey)
	result := client.TextCensor(msg)

	respData := baiduCensorResponse{}
	if err := json.Unmarshal([]byte(result), &respData); err != nil {
		return 0, fmt.Errorf("json unmarshal error: %v", err)
	}
	if respData.ErrorCode != 0 {
		return 0, fmt.Errorf("baidu censor error: %d %s", respData.ErrorCode, respData.ErrorMsg)
	}
	return respData.ConclusionType, nil
}
