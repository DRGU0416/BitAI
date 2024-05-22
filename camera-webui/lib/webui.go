package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

var (
	client = &http.Client{Timeout: time.Second * 10}
)

type ControlNet struct {
	ImagePath     string  `json:"image_path"`      // 底图
	Preprocessor  string  `json:"preprocessor"`    // 预处理器
	ModelName     string  `json:"model_name"`      // 模型名称
	Weight        float64 `json:"weight"`          // 权重
	StartCtrlStep float64 `json:"start_ctrl_step"` // START_CTRL_STEP
	EndCtrlStep   float64 `json:"end_ctrl_step"`   // END_CTRL_STEP
	PreprocRes    int     `json:"preproc_res"`     // 预处理器分辨率
	ControlMode   string  `json:"control_mode"`    // 控制模式
	ResizeMode    string  `json:"resize_mode"`     // 大小调整模式
	PixelPerfect  bool    `json:"pixel_perfect"`   // 完美像素
}

type Roop struct {
	ImagePath string `json:"image_path"`
	// FacesIndex             string  `json:"faces_index"`
	FaceRestorerName string `json:"face_restorer_name"`
	// Model                  string  `json:"model"`
	FaceRestorerVisibility float64 `json:"face_restorer_visibility"`
	// UpscalerName           string  `json:"upscaler_name"`
	// UpscalerScale          float64 `json:"upscaler_scale"`
	// UpscalerVisibility     float64 `json:"upscaler_visibility"`
	// SwapInSource           bool    `json:"swap_in_source"`
	// SwapInGenerated        bool    `json:"swap_in_generated"`
}

type ADetailer struct {
	ModelUrl         string  `json:"model_url"`
	AdModel          string  `json:"ad_model"`
	AdPrompt         string  `json:"ad_prompt"`
	AdNegativePrompt string  `json:"ad_negative_prompt"`
	AdConfidence     float64 `json:"ad_confidence"`
	// AdMaskMinRatio             float64 `json:"ad_mask_min_ratio"`
	// AdMaskMaxRatio             float64 `json:"ad_mask_max_ratio"`
	// AdXOffset                  int     `json:"ad_x_offset"`
	// AdYOffset                  int     `json:"ad_y_offset"`
	AdDilateErode int `json:"ad_dilate_erode"`
	// AdMaskMergeInvert          string  `json:"ad_mask_merge_invert"`
	// AdMaskBlur                 int     `json:"ad_mask_blur"`
	AdDenoisingStrength float64 `json:"ad_denoising_strength"`
	// AdInpaintOnlyMasked        bool    `json:"ad_inpaint_only_masked"`
	// AdInpaintOnlyMaskedPadding int     `json:"ad_inpaint_only_masked_padding"`
	// AdUseInpaintWidthHeight    bool    `json:"ad_use_inpaint_width_height"`
	AdInpaintWidth  int `json:"ad_inpaint_width"`
	AdInpaintHeight int `json:"ad_inpaint_height"`
	// AdUseSteps                 bool    `json:"ad_use_steps"`
	// AdSteps                    int     `json:"ad_steps"`
	// AdUseCfgScale              bool    `json:"ad_use_cfg_scale"`
	// AdCfgScale                 float64 `json:"ad_cfg_scale"`
	// AdRestoreFace              bool    `json:"ad_restore_face"`
	// AdControlnetModel          string  `json:"ad_controlnet_model"`
	// AdControlnetWeight         float64 `json:"ad_controlnet_weight"`
	// AdControlnetGuidanceStart  float64 `json:"ad_controlnet_guidance_start"`
	// AdControlnetGuidanceEnd    float64 `json:"ad_controlnet_guidance_end"`
}

// 风格结构
type Stype struct {
	EnableHr          bool    `json:"enable_hr"`            // 是否超清放大
	HrScale           float64 `json:"hr_scale"`             // 超清放大倍数
	HiresUpscaler     string  `json:"hires_upscaler"`       // 需要超清放大时使用的HIRES超清放大模型名称
	HrSecondPassSteps int     `json:"hr_second_pass_steps"` // 需要超清放大时HIRES步数
	DenoisingStrength float64 `json:"denoising_strength"`   // 去噪强度

	SamplerName    string `json:"sampler_name"`    // 采样方法
	Prompt         string `json:"prompt"`          // 提示词
	NegativePrompt string `json:"negative_prompt"` // 反向提示词

	Width  int   `json:"width"`
	Height int   `json:"height"`
	Seed   int64 `json:"seed"`  // 随机种子
	Steps  int   `json:"steps"` // 采样步长

	RestoreFace bool    `json:"restore_face"` // 是否重绘面部
	Tiling      bool    `json:"tiling"`       // 是否分片
	CfgScale    float64 `json:"cfg_scale"`    // 提示词引导系数
	BatchSize   int     `json:"batch_size"`   // 单批次生成张数
	BatchCount  int     `json:"batch_count"`  // 单批次生成张数

	MainModelPath string `json:"main_model_path"` // 主模型路径
	SubModelUrl   string `json:"sub_model_url"`   // 子模型url

	RandnSource string `json:"randn_source"` // override_settings中的配置RNG=CPU

	ControlNets []*ControlNet `json:"control_nets"` // 风格姿势配置
	Roop        *Roop         `json:"roop"`         // 换脸配置
	ADetailer   []*ADetailer  `json:"adetailer"`    // 细节配置
}

// 训练结构
type LoraTrain struct {
	BaseModel string   `json:"base_model"` // 主模型
	ImageUrl  []string `json:"image_url"`  // 素材
	UUID      string   `json:"uuid"`       // tag
}

// 任务结构
type Task struct {
	TaskType         int       `json:"task_type"`         // 0-训练 1-分身 2-写真
	UserId           int       `json:"user_id"`           // 用户ID
	TaskId           uint      `json:"task_id"`           // 任务ID
	Stype            Stype     `json:"stype"`             // 生图方式
	Callback         string    `json:"callback"`          // 回调地址
	LoraTrain        LoraTrain `json:"lora_train"`        // 训练
	SecondGeneration bool      `json:"second_generation"` // 是否二次生成
}

type LoraModel struct {
	LoraPath         string  `json:"lora_path"`
	Weight           float64 `json:"weight"`
	PromptWeight     float64 `json:"prompt_weight"`
	SecondGeneration bool    `json:"second_generation"`
}

// 回调结构
type SDCallback struct {
	Code       int         `json:"code"`
	Message    string      `json:"msg"`
	TaskId     uint        `json:"task_id"`
	Images     []string    `json:"images"`
	WaterMarks []string    `json:"water_marks"`
	Loras      []LoraModel `json:"loras"`
	Callback   string      `json:"callback"`
	FCT        int         `json:"-"`
	Gender     int         `json:"gender"` // 1-女性青年 2-男性青年
	Seed       int64       `json:"seed"`
}

// 记录错误次数
func (c *SDCallback) IncrFCT() {
	c.FCT++
}

// 检查任务结构
type TaskCheck struct {
	BaseImage string            `json:"base_image"` // 正面图，用于验证图片是否一致
	ImagesMap map[uint64]string `json:"image_urls"` // 图片id->url
	Callback  string            `json:"callback"`   // 回调地址
}

// 检查回调结构
type CheckCallback struct {
	Code     int            `json:"code"`
	Message  string         `json:"msg"`
	TaskId   uint           `json:"task_id"`
	Status   map[uint64]int `json:"check_status"`
	Callback string         `json:"callback"`
	FCT      int            `json:"-"`
}

// 记录错误次数
func (c *CheckCallback) IncrFCT() {
	c.FCT++
}

type CheckResponse struct {
	Code int       `json:"code"`
	Data TaskCheck `json:"data,omitempty"`
}

// 获取识别任务
func GetCheckTask() (TaskCheck, error) {
	url, err := url.JoinPath(WebUIHost, "api/work/recognize")
	if err != nil {
		return TaskCheck{}, err
	}

	request, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(request)
	if err != nil {
		return TaskCheck{}, fmt.Errorf("request error: %v", err)
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return TaskCheck{}, fmt.Errorf("read error: %v", err)
	}

	respData := CheckResponse{}
	if err = json.Unmarshal(result, &respData); err != nil {
		return TaskCheck{}, fmt.Errorf("json unmarshal error: %v", err)
	}
	return respData.Data, nil
}

type TaskResponse struct {
	Code int  `json:"code"`
	Data Task `json:"data,omitempty"`
}

// 获取训练任务
func GetTrainTask() (Task, error) {
	url, err := url.JoinPath(WebUIHost, "api/work/lora")
	if err != nil {
		return Task{}, err
	}

	request, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(request)
	if err != nil {
		return Task{}, fmt.Errorf("request error: %v", err)
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return Task{}, fmt.Errorf("read error: %v", err)
	}

	respData := TaskResponse{}
	if err = json.Unmarshal(result, &respData); err != nil {
		return Task{}, fmt.Errorf("json unmarshal error: %v", err)
	}
	return respData.Data, nil
}

// 获取出图任务
func GetPhotoTask() (Task, error) {
	url, err := url.JoinPath(WebUIHost, "api/work/photo")
	if err != nil {
		return Task{}, err
	}

	request, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(request)
	if err != nil {
		return Task{}, fmt.Errorf("request error: %v", err)
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return Task{}, fmt.Errorf("read error: %v", err)
	}

	respData := TaskResponse{}
	if err = json.Unmarshal(result, &respData); err != nil {
		return Task{}, fmt.Errorf("json unmarshal error: %v", err)
	}
	return respData.Data, nil
}

// 高清
type TaskPhotoHr struct {
	TaskId   int    `json:"task_id"`
	ImageUrl string `json:"image_url"`
	Callback string `json:"callback"` // 回调地址
}

type PhotoHrResponse struct {
	Code int         `json:"code"`
	Data TaskPhotoHr `json:"data,omitempty"`
}

// 检查回调结构
type PhotoHrCallback struct {
	Code          int    `json:"code"`
	Message       string `json:"msg"`
	TaskId        uint   `json:"task_id"`
	ImageUrl      string `json:"image_url"`
	WaterImageUrl string `json:"water_image_url"`
	Callback      string `json:"callback"`
	FCT           int    `json:"-"`
}

// 记录错误次数
func (c *PhotoHrCallback) IncrFCT() {
	c.FCT++
}

// 获取高清任务
func GetPhotoHrTask() (TaskPhotoHr, error) {
	url, err := url.JoinPath(WebUIHost, "api/work/photohr")
	if err != nil {
		return TaskPhotoHr{}, err
	}

	request, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(request)
	if err != nil {
		return TaskPhotoHr{}, fmt.Errorf("request error: %v", err)
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return TaskPhotoHr{}, fmt.Errorf("read error: %v", err)
	}

	respData := PhotoHrResponse{}
	if err = json.Unmarshal(result, &respData); err != nil {
		return TaskPhotoHr{}, fmt.Errorf("json unmarshal error: %v", err)
	}
	return respData.Data, nil
}
