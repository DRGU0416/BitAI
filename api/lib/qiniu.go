package lib

import (
	"bytes"
	"context"
	"image"
	"image/png"

	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"
)

// 上传七牛云
func UploadQNCDN(img image.Image, key string) (int64, error) {
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

	buffer := new(bytes.Buffer)
	if err := png.Encode(buffer, img); err != nil {
		return 0, err
	}

	data := buffer.Bytes()
	dataLen := int64(len(data))
	if err := formUploader.Put(context.Background(), &ret, upToken, key, bytes.NewReader(data), dataLen, nil); err != nil {
		return 0, err
	}
	return dataLen, nil
}

// 删除七牛云
func DeleteQNCDN(key string) error {
	mac := qbox.NewMac(QiniuAccessKey, QiniuSecretKey)
	cfg := storage.Config{
		Region:        &storage.ZoneHuanan,
		UseHTTPS:      true,
		UseCdnDomains: false,
	}
	bucketManager := storage.NewBucketManager(mac, &cfg)
	return bucketManager.Delete(QiniuBucket, key)
}
