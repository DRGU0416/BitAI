package libsd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

type SDControlNetUnit struct {
	RgbbgrMode   bool `json:"rgbbgr_mode"`
	Enabled      bool `json:"enabled"`
	ScribbleMode bool `json:"scribble_mode"`

	InputImage    string  `json:"input_image"`
	Model         string  `json:"model"`
	Module        string  `json:"module"`
	Weight        float64 `json:"weight"`
	ResizeMode    string  `json:"resize_mode"`
	Lowvram       bool    `json:"lowvram"`
	PixelPerfect  bool    `json:"pixel_perfect"`
	ProcessorRes  int     `json:"processor_res"`
	ThresholdA    int     `json:"threshold_a"`
	ThresholdB    int     `json:"threshold_b"`
	GuidanceStart float64 `json:"guidance_start"`
	GuidanceEnd   float64 `json:"guidance_end"`
	ControlMode   string  `json:"control_mode"`
}

type RoopUnit struct {
	ImgBase64              string  `json:"imgBase64"`
	Enabled                bool    `json:"enabled"`
	FacesIndex             string  `json:"faces_index"`
	FaceRestorerName       string  `json:"face_restorer_name"`
	Model                  string  `json:"model"`
	FaceRestorerVisibility float64 `json:"face_restorer_visibility"`
	UpscalerName           string  `json:"upscaler_name"`
	UpscalerScale          float64 `json:"upscaler_scale"`
	UpscalerVisibility     float64 `json:"upscaler_visibility"`
	SwapInSource           bool    `json:"swap_in_source"`
	SwapInGenerated        bool    `json:"swap_in_generated"`
}

type ADetailerUnit struct {
	AdModel                    string  `json:"ad_model"`
	AdPrompt                   string  `json:"ad_prompt"`
	AdNegativePrompt           string  `json:"ad_negative_prompt"`
	AdConfidence               float64 `json:"ad_confidence"`
	AdMaskMinRatio             float64 `json:"ad_mask_min_ratio"`
	AdMaskMaxRatio             float64 `json:"ad_mask_max_ratio"`
	AdXOffset                  int     `json:"ad_x_offset"`
	AdYOffset                  int     `json:"ad_y_offset"`
	AdDilateErode              int     `json:"ad_dilate_erode"`
	AdMaskMergeInvert          string  `json:"ad_mask_merge_invert"`
	AdMaskBlur                 int     `json:"ad_mask_blur"`
	AdDenoisingStrength        float64 `json:"ad_denoising_strength"`
	AdInpaintOnlyMasked        bool    `json:"ad_inpaint_only_masked"`
	AdInpaintOnlyMaskedPadding int     `json:"ad_inpaint_only_masked_padding"`
	AdUseInpaintWidthHeight    bool    `json:"ad_use_inpaint_width_height"`
	AdInpaintWidth             int     `json:"ad_inpaint_width"`
	AdInpaintHeight            int     `json:"ad_inpaint_height"`
	AdUseSteps                 bool    `json:"ad_use_steps"`
	AdSteps                    int     `json:"ad_steps"`
	AdUseCfgScale              bool    `json:"ad_use_cfg_scale"`
	AdCfgScale                 float64 `json:"ad_cfg_scale"`
	AdRestoreFace              bool    `json:"ad_restore_face"`
	AdControlnetModel          string  `json:"ad_controlnet_model"`
	AdControlnetWeight         float64 `json:"ad_controlnet_weight"`
	AdControlnetGuidanceStart  float64 `json:"ad_controlnet_guidance_start"`
	AdControlnetGuidanceEnd    float64 `json:"ad_controlnet_guidance_end"`
	AdUseClipSkip              bool    `json:"ad_use_clip_skip"`
	AdClipSkip                 float64 `json:"ad_clip_skip"`
}

func (t RoopUnit) toArray() []any {
	return []any{t.ImgBase64, t.Enabled, t.FacesIndex, t.Model, t.FaceRestorerName, t.FaceRestorerVisibility, t.UpscalerName, t.UpscalerScale, t.UpscalerVisibility, t.SwapInSource, t.SwapInGenerated}
}

type SDTextToImageGenerator struct {
	DenoisingStrength float64 `json:"denoising_strength"`
	Prompt            string  `json:"prompt"`
	NegativePrompt    string  `json:"negative_prompt"`
	SamplerName       string  `json:"sampler_name"`
	Steps             int     `json:"steps"`
	RestoreFaces      bool    `json:"restore_faces"`
	Tiling            bool    `json:"tiling"`
	EnableHr          bool    `json:"enable_hr"`
	HrScale           float64 `json:"hr_scale"`
	HrUpscaler        string  `json:"hr_upscaler"`
	HrSecondPassSteps int     `json:"hr_second_pass_steps"`
	HrResizeX         int     `json:"hr_resize_x"`
	HrResizeY         int     `json:"hr_resize_y"`

	Width  int `json:"width"`
	Height int `json:"height"`

	FirstphaseWidth  int `json:"firstphase_width"`
	FirstphaseHeight int `json:"firstphase_height"`

	CfgScale   float64 `json:"cfg_scale"`
	NIter      int     `json:"n_iter"`
	BatchSize  int     `json:"batch_size"`
	BatchCount int     `json:"batch_count"`

	Seed             int64          `json:"seed"`
	SubSeed          int64          `json:"subseed"`
	SubseedStrength  float64        `json:"subseed_strength"`
	SeedResizeFromH  int            `json:"seed_resize_from_h"`
	SeedResizeFromW  int            `json:"seed_resize_from_w"`
	DoNotSaveSamples bool           `json:"do_not_save_samples"`
	DoNotSaveGrid    bool           `json:"do_not_save_grid"`
	Eta              float64        `json:"eta"`
	SChurn           float64        `json:"s_churn"`
	STmax            float64        `json:"s_tmax"`
	STmin            float64        `json:"s_tmin"`
	SNoise           float64        `json:"s_noise"`
	OverrideSettings map[string]any `json:"override_settings"`

	OverrideSettingsRestoreAfterwards bool `json:"override_settings_restore_afterwards"`

	ScriptArgs   []string `json:"script_args"`
	SamplerIndex string   `json:"sampler_index"`
	ScriptName   string   `json:"script_name"`
	SendImages   bool     `json:"send_images"`
	SaveImages   bool     `json:"save_images"`

	AlwaysonScripts map[string]any     `json:"alwayson_scripts"`
	Styles          []string           `json:"styles"`
	ControlNetUnits []SDControlNetUnit `json:"-"`
	RoopUnit        RoopUnit           `json:"-"`
	ADetailerUnits  []ADetailerUnit    `json:"-"`
}

type SDImageInfo struct {
	Images []string `json:"images"`
	Info   string   `json:"info"`
}

/*
{"prompt":"best quality, masterpiece,Black hair, blue eyes, looking up, upper body","negative_prompt":"(low quality, worst quality:1.4)","sampler_name":"DPM++ SDE Karras","steps":8,"restore_faces":false,"tiling":false,"enable_hr":false,"hr_scale":2.0,"hr_upscaler":"","hr_second_pass_steps":0,"hr_resize_x":0,"hr_resize_y":0,"denoising_strength":0.0,"width":1024,"height":1384,"firstphase_width":0,"firstphase_height":0,"cfg_scale":11.0,"n_iter":1,"batch_size":1,"batch_count":1,"seed":-1,"subseed":-1,"subseed_strength":0.0,"seed_resize_from_h":-1,"seed_resize_from_w":-1,"do_not_save_samples":false,"do_not_save_grid":false,"eta":0.0,"s_churn":0.0,"s_tmax":0.0,"s_tmin":0.0,"s_noise":1.0,"override_settings":{},"override_settings_restore_afterwards":true,"script_args":[],"sampler_index":"Euler","script_name":"","send_images":true,"save_images":false,"alwayson_scripts":{"controlnet":{"args":[{"rgbbgr_mode":false,"enabled":true,"scribble_mode":false,"input_image":"","module":"depth_midas","model":"control_v11f1p_sd15_depth [cfd03158]","weight":1.0,"resize_mode":"Scale to Fit (Inner Fit)","lowvram":false,"processor_res":512,"threshold_a":0,"threshold_b":0,"guidance_start":0.0,"guidance_end":1.0,"control_mode":"Balanced"}]}},"styles":[]}
*/

func (t SDTextToImageGenerator) GenerateImages() ([]string, int64, error) {
	t.AlwaysonScripts = make(map[string]any)
	if len(t.ControlNetUnits) > 0 {
		args := make(map[string][]SDControlNetUnit)
		args["args"] = t.ControlNetUnits
		t.AlwaysonScripts["controlnet"] = args
	}

	if t.RoopUnit.Enabled {
		args := make(map[string][]any)
		args["args"] = t.RoopUnit.toArray()
		t.AlwaysonScripts["roop"] = args
	}

	if len(t.ADetailerUnits) > 0 {
		args := make(map[string][]ADetailerUnit)
		args["args"] = t.ADetailerUnits
		t.AlwaysonScripts["adetailer"] = args
	}

	url := fmt.Sprintf("%s/sdapi/v1/txt2img", webuiHost)
	byteMsg, err := json.Marshal(t)
	if err != nil {
		return nil, -1, err
	}

	{
		//裁剪去图片数据进行打印
		if t.AlwaysonScripts["controlnet"] != nil && len(t.AlwaysonScripts["controlnet"].(map[string][]SDControlNetUnit)) > 0 {
			t.AlwaysonScripts["controlnet"].(map[string][]SDControlNetUnit)["args"][0].InputImage = fmt.Sprintf("Image(len=%d)", len(t.ControlNetUnits[0].InputImage))
		}
		if t.AlwaysonScripts["roop"] != nil && len(t.AlwaysonScripts["roop"].(map[string][]any)) > 0 {
			t.AlwaysonScripts["roop"].(map[string][]any)["args"][0] = fmt.Sprintf("Image(len=%d)", len(t.RoopUnit.ImgBase64))
		}
		logMsg, _ := json.Marshal(t)
		logTask.Infof("text2img param: %s", string(logMsg))
	}

	request, _ := http.NewRequest("POST", url, bytes.NewReader(byteMsg))
	request.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(request)
	if err != nil {
		return nil, -1, err
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, -1, err
	}

	data := SDImageInfo{}
	if err = json.Unmarshal(result, &data); err != nil {
		return nil, -1, err
	}

	var seed int64 = -1
	var imginfo map[string]any
	if err = json.Unmarshal([]byte(data.Info), &imginfo); err == nil {
		if v, ok := imginfo["seed"]; ok {
			seed = int64(v.(float64))
		}
	}

	return data.Images, seed, nil
}
