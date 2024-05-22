package lib

import (
	"context"
	"os"

	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"
)

func UploadQNCDN(fileName string, key string) (int64, error) {
	putPolicy := storage.PutPolicy{
		Scope: QiniuBucket,
	}
	mac := qbox.NewMac(QiniuAccessKey, QiniuSecretKey)
	upToken := putPolicy.UploadToken(mac)
	cfg := storage.Config{
		Region:        &storage.ZoneHuanan,
		UseHTTPS:      true,
		UseCdnDomains: false,
	}
	formUploader := storage.NewFormUploader(&cfg)
	ret := storage.PutRet{}

	reader, err := os.Open(fileName)
	if err != nil {
		return 0, err
	}
	defer reader.Close()

	fileInfo, _ := reader.Stat()
	var dataLen int64 = fileInfo.Size()

	if err := formUploader.Put(context.Background(), &ret, upToken, key, reader, dataLen, nil); err != nil {
		return 0, err
	}
	return dataLen, nil
}
