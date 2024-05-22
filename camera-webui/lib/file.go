package lib

import (
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/chai2010/webp"
)

// 拷贝文件
func CopyFile(src string, dst string, preferLink bool) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	err = nil
	if preferLink {
		os.MkdirAll(filepath.Dir(dst), 0755)
		err = os.Link(src, dst)
		if err != nil {
			fmt.Printf("link error, %s", err)
		}
	}

	if !preferLink || err != nil {
		source, err := os.Open(src)
		if err != nil {
			return err
		}

		defer source.Close()
		os.MkdirAll(filepath.Dir(dst), 0755)
		destination, err := os.Create(dst)
		if err != nil {
			return err
		}

		defer destination.Close()
		_, err = io.Copy(destination, source)

		if err != nil {
			return err
		}
	}

	return nil
}

// 读取文件大小
func FileSize(path string) int64 {
	sourceFileStat, err := os.Stat(path)
	if err != nil {
		fmt.Printf("FileSize: Stat error %s\n", err)
		return 0
	}
	if !sourceFileStat.Mode().IsRegular() {
		fmt.Printf("FileSize: %s is not a regular file", path)
		return 0
	}

	return sourceFileStat.Size()
}

// 下载文件
func DownloadFile(url, fileName string) error {
	if strings.HasPrefix(url, "http") {
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		os.MkdirAll(filepath.Dir(fileName), 0755)
		destination, err := os.Create(fileName)
		if err != nil {
			return err
		}

		defer destination.Close()
		_, err = io.Copy(destination, resp.Body)
		if err != nil {
			return err
		}
		return nil
	} else {
		return CopyFile(url, fileName, true)
	}
}

// 图片类型
func ImageFormat(fi *os.File) (string, error) {
	header := make([]byte, 12)
	_, err := fi.Read(header)
	if err != nil {
		return "", err
	}
	fi.Seek(0, 0)

	isPng := header[0] == 0x89 && header[1] == 0x50 && header[2] == 0x4e && header[3] == 0x47 &&
		header[4] == 0x0d && header[5] == 0x0a && header[6] == 0x1a && header[7] == 0x0a

	isJpeg := header[0] == 0xff && header[1] == 0xd8
	// Print the file format
	if isPng {
		return "png", nil
	} else if isJpeg {
		return "jpeg", nil
	}
	return "webp", nil
}

// 获取Image
func GetImage(imagePath string) (image.Image, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	format, err := ImageFormat(file)
	if err != nil {
		return nil, err
	}

	var img image.Image
	switch format {
	case "jpeg":
		img, err = jpeg.Decode(file)
	case "png":
		img, err = png.Decode(file)
	case "webp":
		img, err = webp.Decode(file)
	}
	if err != nil {
		return nil, err
	}
	return img, nil
}

// 是否图片文件
func IsImageFile(path string) bool {
	extension := strings.ToLower(filepath.Ext(path))
	imageExtensions := []string{".jpg", ".jpeg", ".png"} // 可以根据需要添加其他图片扩展名

	for _, ext := range imageExtensions {
		if ext == extension {
			return true
		}
	}

	return false
}

// 文件是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// 图片Base64转换
func ImageFileToBase64(path string) (string, error) {
	imageData, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(imageData), nil
}

// Base64转PNG图片
func Base64ToPNG(base64String string, outputPath string) error {
	// 解码 Base64 字符串
	data, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		return err
	}

	// 创建图像对象
	img, _, err := image.Decode(strings.NewReader(string(data)))
	if err != nil {
		return err
	}

	// 创建输出文件
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 将图像保存为 PNG 格式
	err = png.Encode(file, img)
	if err != nil {
		return err
	}

	return nil
}
