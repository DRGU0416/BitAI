package lib

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rwcarlsen/goexif/exif"
	"golang.org/x/image/webp"
)

// 下载文件
func DownloadFile(url, fileName string) error {
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
}

// 下载图片返回Image
func DownloadImage(url string) (image.Image, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	img, err := png.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return img, nil
}

// 切割文件
func SplitImage(imagePath string) ([]image.Image, error) {
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

	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	top := img.Bounds().Min.Y
	left := img.Bounds().Min.X
	right := img.Bounds().Max.X
	bottom := img.Bounds().Max.Y
	halfWidth := width / 2
	halfHeight := height / 2

	var topLeft, topRight, bottomLeft, bottomRight image.Image
	if format == "webp" {
		topLeft = img.(*image.YCbCr).SubImage(image.Rect(left, top, left+halfWidth, top+halfHeight))
		topRight = img.(*image.YCbCr).SubImage(image.Rect(left+halfWidth, top, right, top+halfHeight))
		bottomLeft = img.(*image.YCbCr).SubImage(image.Rect(left, top+halfHeight, left+halfWidth, bottom))
		bottomRight = img.(*image.YCbCr).SubImage(image.Rect(left+halfWidth, top+halfHeight, right, bottom))
	} else {
		topLeft = img.(*image.RGBA).SubImage(image.Rect(left, top, left+halfWidth, top+halfHeight))
		topRight = img.(*image.RGBA).SubImage(image.Rect(left+halfWidth, top, right, top+halfHeight))
		bottomLeft = img.(*image.RGBA).SubImage(image.Rect(left, top+halfHeight, left+halfWidth, bottom))
		bottomRight = img.(*image.RGBA).SubImage(image.Rect(left+halfWidth, top+halfHeight, right, bottom))
	}
	return []image.Image{topLeft, topRight, bottomLeft, bottomRight}, nil
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

func IsImage(file io.Reader) bool {
	buffer := make([]byte, 512)
	_, err := file.Read(buffer)
	if err != nil {
		return false
	}
	mimeType := http.DetectContentType(buffer)
	return strings.HasPrefix(mimeType, "image/")
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

// 处理标准Mask图
func MaskStandard(imagePath string) error {
	m, err := GetImage(imagePath)
	if err != nil {
		return err
	}
	bounds := m.Bounds()
	dx := bounds.Dx()
	dy := bounds.Dy()
	newRgba := image.NewRGBA(bounds)
	for x := 0; x < dx; x++ {
		for y := 0; y < dy; y++ {
			r, g, b, a := m.At(x, y).RGBA()
			if r == 0 && g == 0 && b == 0 && a == 0 {
				newRgba.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
			} else {
				newRgba.SetRGBA(x, y, color.RGBA{255, 255, 255, 255})
			}
		}
	}
	osf, err := os.Create(imagePath)
	if err != nil {
		return err
	}
	return png.Encode(osf, newRgba)
}

func GetFileMd5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	md5hash := md5.New()
	if _, err := io.Copy(md5hash, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", md5hash.Sum(nil)), nil
}

// 加水印
func AddWatermark(imageData image.Image, watermarkPath string) (image.Image, error) {
	// Open the watermark image file
	watermarkFile, err := os.Open(watermarkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open watermark file: %v", err)
	}
	defer watermarkFile.Close()

	watermarkData, err := png.Decode(watermarkFile)
	if err != nil {
		return nil, fmt.Errorf("failed to decode watermark image: %v", err)
	}

	outputImage := image.NewRGBA(imageData.Bounds())
	draw.Draw(outputImage, outputImage.Bounds(), imageData, image.Point{}, draw.Src)
	numXRepeats := (outputImage.Bounds().Dx() + watermarkData.Bounds().Dx() - 1) / (watermarkData.Bounds().Dx() + 100)
	numYRepeats := (outputImage.Bounds().Dy() + watermarkData.Bounds().Dy() - 1) / (watermarkData.Bounds().Dy() + 100)
	for y := 0; y < numYRepeats; y++ {
		for x := 0; x < numXRepeats; x++ {
			draw.Draw(outputImage, watermarkData.Bounds().Add(image.Pt(x*(watermarkData.Bounds().Dx()+100), y*(watermarkData.Bounds().Dy()+100))), watermarkData, image.Point{}, draw.Over)
		}
	}

	return outputImage.SubImage(image.Rect(0, 0, outputImage.Bounds().Dx(), outputImage.Bounds().Dy())), nil
}

// png图片没有Orientation信息
func ReadOrientation(imgReader io.Reader) (int, error) {
	// file, err := os.Open(filename)
	// if err != nil {
	// 	fmt.Println("failed to open file, err: ", err)
	// 	return 0
	// }
	// defer file.Close()

	x, err := exif.Decode(imgReader)
	if err != nil {
		return 0, fmt.Errorf("failed to decode file, err: %s", err)
	}

	orientation, err := x.Get(exif.Orientation)
	if err != nil {
		return 0, fmt.Errorf("failed to get orientation, err: %s", err)
	}
	orientVal, err := orientation.Int(0)
	if err != nil {
		return 0, fmt.Errorf("failed to get orientation, err: %s", err)
	}

	return orientVal, nil
}
