package lib

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	mrand "math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
)

var (
	ctx = context.Background()

	client = &http.Client{Timeout: time.Second * 10}

	json = jsoniter.ConfigCompatibleWithStandardLibrary

	rr = mrand.New(mrand.NewSource(time.Now().UnixNano()))
)

func GenGUID() string {
	b := make([]byte, 48)

	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}

	h := md5.New()
	h.Write([]byte(base64.URLEncoding.EncodeToString(b)))
	return hex.EncodeToString(h.Sum(nil))
}

func VerifyMobileFormat(mobileNum string) bool {
	regular := "^1\\d{10}$"

	reg := regexp.MustCompile(regular)
	return reg.MatchString(mobileNum)
}

func DayStart(curtime time.Time) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", curtime.Format("2006-01-02"), time.Local)
}

func DayEnd(curtime time.Time) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", curtime.Add(time.Hour*24).Format("2006-01-02"), time.Local)
}

func GetRand(min, max int) int {
	return rr.Intn(max-min) + min
}

func Shuffle[T any](t []T) {
	rr.Shuffle(len(t), func(i, j int) {
		t[i], t[j] = t[j], t[i]
	})
}

func GetMd5(data []byte) string {
	return fmt.Sprintf("%x", md5.Sum(data))
}

func InArray(array []string, key string) (exists bool) {
	for _, v := range array {
		if strings.EqualFold(v, key) {
			exists = true
			break
		}
	}
	return
}

func HmacSha256(data, secret string) string {
	key := []byte(secret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
