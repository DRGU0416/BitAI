package lib

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var (
	MessageCode        = "【喵喵相机】您的验证码为%s，验证码5分钟有效。为保证您的账号安全，验证码请勿转发他人。如您不是操作者本人请忽略此短信"
	MessageCardSuccess = "【喵喵相机】您的数字分身已经制作完成，快来查看您的专属形象，一键生成写真"
)

var SmsChannel = 2 // 1 微网通联 2 助通科技

var SmsWwtlName = "dlbojie0"
var SmsWwtlPassword = "zxm1190abc"
var SmsWwtlHyProductId = "1012888"
var SmsWwtlYxProductId = "1012812"

var SmsZtkjHyName = "qdkyhy"
var SmsZtkjHyPassword = "2OhERaqd"
var SmsZtkjYxName = "qdkyyx"
var SmsZtkjYxPassword = "3cevN0eb"

// SendPhoneMessage 发送短信
// 1:注册验证码 2:触发报警
func SendPhoneMessage(phone, msg string) error {
	if SmsChannel == 1 {
		return sendPhoneMessageWwtl(phone, msg, 1)
	}

	if SmsChannel == 2 {
		return sendPhoneMessageZtkj(phone, msg, 1)
	}

	return fmt.Errorf("sms channel not exist")
}

// sendPhoneMessageWwtl 微网通联 发送短信
// 手机号码，号码中间用英文逗号分隔，每个包最大支持10万条
func sendPhoneMessageWwtl(phone, msg string, smsType int) error {
	SmsWwtlProductId := SmsWwtlHyProductId
	if smsType == 2 {
		SmsWwtlProductId = SmsWwtlYxProductId
	}

	apiUrl := "http://cf.51welink.com/submitdata/Service.asmx/g_Submit"
	postBody := fmt.Sprintf(`sname=%s&spwd=%s&scorpid=&sprdid=%s&sdst=%s&smsg=%s`, SmsWwtlName, SmsWwtlPassword, SmsWwtlProductId, phone, url.QueryEscape(msg))

	byteMsg := []byte(postBody)
	request, _ := http.NewRequest("POST", apiUrl, bytes.NewReader(byteMsg))
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Add("Content-Length", fmt.Sprintf("%d", len(byteMsg)))
	request.Header.Add("Connection", "close")

	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("[wwtl] %v %s %s", phone, msg, string(result))
	}

	msgResp := struct {
		State    int
		MsgID    string
		MsgState string
		Reserve  int
	}{}
	if err = xml.Unmarshal(result, &msgResp); err != nil {
		return err
	}
	if msgResp.State != 0 {
		return fmt.Errorf("send phone message failed, body: %v", msgResp)
	}

	return nil
}

// sendPhoneMessageZtkj 助通科技 发送短信
// https://doc.zthysms.com/web
// 手机号码，号码中间用英文逗号分隔，每个包最大支持2000个号码
func sendPhoneMessageZtkj(phone, msg string, smsType int) error {
	apiUrl := "https://api.mix2.zthysms.com/v2/sendSms"

	username := SmsZtkjHyName
	password := SmsZtkjHyPassword
	if smsType == 2 {
		username = SmsZtkjYxName
		password = SmsZtkjYxPassword
	}

	tKey := strconv.Itoa(int(int32(time.Now().Unix())))
	pwd := GetMd5([]byte(GetMd5([]byte(password)) + tKey))

	post := "{" +
		"\"username\":\"" + username +
		"\",\"password\":\"" + pwd +
		"\",\"mobile\":\"" + phone +
		"\",\"content\":\"" + msg +
		"\",\"tKey\":\"" + tKey +
		"\"}"

	var jsonStr = []byte(post)

	req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if body, err := io.ReadAll(resp.Body); err != nil {
		return fmt.Errorf("[ztkj] %s %s %s", phone, msg, string(body))
	}
	return nil
}
