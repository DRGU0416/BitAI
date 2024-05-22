package monitor

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"camera/logger"
	"camera/models"

	jsoniter "github.com/json-iterator/go"
)

const (
	BARK_URL = "https://api.day.app/%s/%s/%s"
)

type BarkType int

const (
	CHATGPT BarkType = iota + 1
	MJ_TASK
	MJ_PUSH
	MJ_IMAGE
	BD_TRANSLATE
	BD_CENSOR
	MYSQL
	REDIS
	QINIU
	TASK_TIMEOUT
	WEBUI
	ORDER_PAY
	CARD_LORA
)

var (
	client = &http.Client{Timeout: 10 * time.Second}

	logBark = logger.New("logs/bark.log")

	json = jsoniter.ConfigCompatibleWithStandardLibrary

	// 阀门
	valver   = make(map[BarkType]time.Time, 0)
	duration = 10 * time.Minute
)

func canBark(barkType BarkType) bool {
	v, ok := valver[barkType]
	if !ok {
		valver[barkType] = time.Now()
		return true
	}

	d := time.Now().Sub(v)
	if d > duration {
		valver[barkType] = time.Now()
		return true
	}
	return false
}

type Bark struct {
	Title   string
	Message string
}

type BarkResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (bark *Bark) SendMessage(barkType BarkType) {
	if bark.Title == "" || bark.Message == "" {
		return
	}
	if !canBark(barkType) {
		return
	}

	tokens, err := models.GetBarkTokens()
	if err != nil {
		logBark.Errorf("get token error: %s", err.Error())
		return
	}

	for _, token := range tokens {
		url := fmt.Sprintf(BARK_URL, token, bark.Title, bark.Message)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			logBark.Errorf("bad request: %s", err)
			return
		}

		resp, err := client.Do(req)
		if err != nil {
			logBark.Errorf("bad request: %s", err)
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logBark.Errorf("bad body: %s", err)
			return
		}
		barkRes := BarkResponse{}
		if err = json.Unmarshal(body, &barkRes); err != nil {
			logBark.Errorf("bad json: %s, error: %s", string(body), err)
			return
		}
		if barkRes.Code != 200 || barkRes.Message != "success" {
			logBark.Errorf("bad response, %+v", barkRes)
		}
	}
}
