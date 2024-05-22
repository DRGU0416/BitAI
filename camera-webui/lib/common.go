package lib

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	mrand "math/rand"
	"time"
)

var (
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

func GetRand(min, max int) int {
	return rr.Intn(max-min) + min
}

func GetMd5(data []byte) string {
	return fmt.Sprintf("%x", md5.Sum(data))
}
