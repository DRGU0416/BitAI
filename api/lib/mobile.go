package lib

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var (
	umengHost = "https://verify5.market.alicloudapi.com"
	umengUrl  = "/api/v1/mobile/info?appkey=%s&verifyId=%s"

	aliAppKey    = "203846413"
	aliAppSecret = "x3YYJqLONszW2Je4CPeCu6H50ZWwbfqH"
	aliAppCode   = "72a4fd07e3d042c1bee44631303435bb"

	UmengAppKey = "64dc676f5488fe7b3af4a7ea"
)

type UmengTokenResp struct {
	Success bool `json:"success"`
	Data    struct {
		Mobile      string  `json:"mobile"`
		Score       float32 `json:"score"`
		ActiveScore float32 `json:"activeScore"`
	} `json:"data"`
	Code      int    `json:"code"`
	Message   string `json:"message"`
	RequestId string `json:"requestId"`
}

// UmengTokenValidate 友盟 一键登录
func UmengTokenValidate(token, verifyId string) (string, error) {
	payload := struct {
		Token string `json:"token"`
	}{token}

	postBody, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	mm := md5.Sum(postBody)
	contentMd5 := base64.StdEncoding.EncodeToString(mm[:])

	reqTime := fmt.Sprintf("%d", time.Now().UnixNano()/1e6)
	nonce := GenGUID()
	url := fmt.Sprintf(umengUrl, UmengAppKey, verifyId)
	content := fmt.Sprintf("POST\napplication/json\n%s\napplication/json;charset=UTF-8\n\nX-Ca-Key:%s\nX-Ca-Nonce:%s\nX-Ca-Stage:RELEASE\nX-Ca-Timestamp:%s\n%s",
		contentMd5, aliAppKey, nonce, reqTime, strings.TrimRight(url, "="))

	request, _ := http.NewRequest("POST", umengHost+url, bytes.NewReader(postBody))
	request.Header.Add("Content-Type", "application/json;charset=UTF-8")
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-MD5", contentMd5)
	request.Header.Add("X-Ca-Signature-Headers", "X-Ca-Stage,X-Ca-Key,X-Ca-Timestamp,X-Ca-Nonce")
	request.Header.Add("X-Ca-Stage", "RELEASE")
	request.Header.Add("X-Ca-Key", aliAppKey)
	request.Header.Add("X-Ca-Timestamp", reqTime)
	request.Header.Add("X-Ca-Nonce", nonce)
	request.Header.Add("X-Ca-Signature", HmacSha256(content, aliAppSecret))

	resp, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("Code: %d, RequestId: %s, Error: %s", resp.StatusCode, resp.Header.Get("X-Ca-Request-Id"), resp.Header.Get("X-Ca-Error-Message"))
	}

	respBody := UmengTokenResp{}
	if err = json.Unmarshal(result, &respBody); err != nil {
		return "", err
	}

	if !respBody.Success {
		return "", fmt.Errorf("RequestId: %s, Error: %s", respBody.RequestId, respBody.Message)
	}

	return respBody.Data.Mobile, nil
}
