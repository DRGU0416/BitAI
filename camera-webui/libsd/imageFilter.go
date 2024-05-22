package libsd

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/viper"
)

var (
	client = &http.Client{Timeout: time.Minute * 60}
	json   = jsoniter.ConfigCompatibleWithStandardLibrary

	webuiHost   string
	kohyaHost   string
	imageHost   string
	imageHrHost string
)

func init() {
	webuiHost = viper.GetString("webui.webui_host")
	kohyaHost = viper.GetString("webui.kohya_host")
	imageHost = viper.GetString("webui.image_host")
	imageHrHost = viper.GetString("webui.image_hr_host")
}

type SDResponse struct {
	Result string `json:"result"`
	Reason string `json:"reason"`
	Data   any    `json:"data"`
}

// 裁剪头像
func ClipFaces(fromDir string, toDir string, recursive bool) (map[string][]string, error) {
	url := fmt.Sprintf("%s/clipFaces?from=%s&to=%s&recursive=%t",
		imageHost, url.QueryEscape(fromDir), url.QueryEscape(toDir), recursive)

	request, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	data := SDResponse{}
	if err = json.Unmarshal(result, &data); err != nil {
		return nil, err
	}

	if data.Result == "succeeded" {
		rtData := map[string][]string{}
		for k, v := range data.Data.(map[string]any) {
			//类型转换
			if value, ok := v.([]any); ok {
				for _, val := range value {
					if value2, ok := val.([]any); ok && len(value2) == 2 {
						rtData[k] = append(rtData[k], value2[0].(string))
						if value2[1].(float64) > 1 {
							rtData[k] = append(rtData[k], "")
						}
					}
				}
			}
		}
		return rtData, nil
	}
	// fmt.Printf("ClipFaces errpr = %v\n", data)
	return nil, fmt.Errorf(data.Reason)
}

/**
 * 将某个目录中存放的全部图片进行整体缩放以适合生成和训练标准
 * fromDir: 原始图片存放目录
 * maxSideLen: 缩放后的图片其长边的边长
 * toDir: 缩放后的图片保存到的目录
 * recursive: 是否递归处理子目录,默认值false
 * makesquare: 缩放后的图片是否补成正方形，如果true则进行补充，false则保持图片原有边长比例,默认值true
 * 返回值：如果处理成功，返回true.
 * 如果服务器端计算出错则抛出异常
 **/
func BatchResizeImages(fromDir string, maxSideLen int, toDir string, recursive bool, makesquare bool) error {
	url := fmt.Sprintf("%s/resizeImages?from=%s&to=%s&sidelen=%d&recursive=%t&makesquare=%t",
		imageHost, url.QueryEscape(fromDir), url.QueryEscape(toDir), maxSideLen, recursive, makesquare)

	request, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	data := SDResponse{}
	if err = json.Unmarshal(result, &data); err != nil {
		return err
	}

	if data.Result == "succeeded" {
		return nil
	}
	return fmt.Errorf(data.Reason)
}

func ImageIsInKnownFaces(path string) (bool, error) {
	url := fmt.Sprintf("%s/detectKnownFaces?imagePath=%s", imageHost, url.QueryEscape(path))

	request, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(request)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	data := SDResponse{}
	if err = json.Unmarshal(result, &data); err != nil {
		return false, err
	}

	if data.Result == "succeeded" {
		return false, nil
	}
	if data.Reason == "The image is forbidden." {
		return true, nil
	}
	return false, fmt.Errorf(data.Reason)
}

func ImageIsIllegal(path string) (bool, error) {
	url := fmt.Sprintf("%s/isPornImage?imagePath=%s", imageHost, url.QueryEscape(path))

	request, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(request)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	data := SDResponse{}
	if err = json.Unmarshal(result, &data); err != nil {
		return false, err
	}

	if data.Result == "succeeded" {
		return false, nil
	}
	if data.Reason == "Porn/hentai image detected." {
		return true, nil
	}
	return false, fmt.Errorf(data.Reason)
}

// 去除生成图片中的信息
func CleanImageText(src string, dst string) error {
	url := fmt.Sprintf("%s/cleanpngtext?from=%s&to=%s", imageHost, url.QueryEscape(src), url.QueryEscape(dst))

	request, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	data := SDResponse{}
	if err = json.Unmarshal(result, &data); err != nil {
		return err
	}

	if data.Result == "succeeded" {
		return nil
	}
	return fmt.Errorf(data.Reason)
}

// 去除图片背景
func BatchRemoveBackground(fromDir, toDir string, recursive bool) error {
	url := fmt.Sprintf("%s/batchrembg?from=%s&to=%s&recursive=%t", imageHost, url.QueryEscape(fromDir), url.QueryEscape(toDir), recursive)

	request, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	data := SDResponse{}
	if err = json.Unmarshal(result, &data); err != nil {
		return err
	}

	if data.Result == "succeeded" {
		return nil
	}
	return fmt.Errorf(data.Reason)
}

// 单张图片加水印
func AddWatermark(fromFile, toFile string) error {
	url := fmt.Sprintf("%s/addWatermark?from=%s&to=%s", imageHost, url.QueryEscape(fromFile), url.QueryEscape(toFile))

	request, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	data := SDResponse{}
	if err = json.Unmarshal(result, &data); err != nil {
		return err
	}

	if data.Result == "succeeded" {
		return nil
	}
	return fmt.Errorf(data.Reason)
}

// 检查2张图片是否一个人
func IsSameFace(path1, path2 string) (bool, error) {
	url := fmt.Sprintf("%s/isSameFace?path1=%s&path2=%s", imageHost, url.QueryEscape(path1), url.QueryEscape(path2))

	request, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(request)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	data := SDResponse{}
	if err = json.Unmarshal(result, &data); err != nil {
		return false, err
	}

	return data.Result == "succeeded", nil
}

// 图片高清
func ImageHires(fromFile, toFile string) error {
	url := fmt.Sprintf("%s/hires?from=%s&to=%s", imageHrHost, url.QueryEscape(fromFile), url.QueryEscape(toFile))

	request, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	data := SDResponse{}
	if err = json.Unmarshal(result, &data); err != nil {
		return err
	}

	if data.Result == "succeeded" {
		return nil
	}
	return fmt.Errorf(data.Reason)
}
