package libsd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

type SDImageToImageGenerator struct {
	DenoisingStrength      float64  `json:"denoising_strength"`
	InitImages             []string `json:"init_images"`
	ResizeMode             int      `json:"resize_mode"`
	ImageCfgScale          float64  `json:"image_cfg_scale"`
	Mask                   string   `json:"mask,omitempty"`
	MaskBlur               int      `json:"mask_blur"`                // 数值较小的时候，边缘越锐利
	InpaintingFill         int      `json:"inpainting_fill"`          // 0-fill 1-original 2-
	InpaintFullRes         int      `json:"inpaint_full_res"`         // 0-whole picture 1-only masked
	InpaintFullResPadding  int      `json:"inpaint_full_res_padding"` // only masked padding
	InpaintingMaskInvert   int      `json:"inpainting_mask_invert"`   // 0-inpaint masked黑底 1-inpaint not masked白底
	InitialNoiseMultiplier float64  `json:"initial_noise_multiplier"` // inpaint时设置=1.0
	Prompt                 string   `json:"prompt"`
	NegativePrompt         string   `json:"negative_prompt"`
	SamplerName            string   `json:"sampler_name"`
	Steps                  int      `json:"steps"`
	RestoreFaces           bool     `json:"restore_faces"`
	Tiling                 bool     `json:"tiling"`
	Width                  int      `json:"width"`
	Height                 int      `json:"height"`

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
}

/*
{\"init_images\":null,\"resize_mode\":0,\"image_cfg_scale\":0.0,\"mask\":null,\"mask_blur\":10,\"inpainting_fill\":0,\"inpaint_full_res\":true,\"inpaint_full_res_padding\":0,\"inpainting_mask_invert\":0,\"initial_noise_multiplier\":0.0,\"prompt\":\"best quality, masterpiece,Black hair, blue eyes, looking up, upper body\",\"negative_prompt\":\"(low quality, worst quality:1.4)\",\"sampler_name\":\"DPM++ SDE Karras\",\"steps\":20,\"restore_faces\":false,\"tiling\":false,\"denoising_strength\":0.75,\"width\":512,\"height\":832,\"cfg_scale\":11.0,\"n_iter\":1,\"batch_size\":1,\"batch_count\":1,\"seed\":29378540987,\"subseed\":-1,\"subseed_strength\":0.0,\"seed_resize_from_h\":-1,\"seed_resize_from_w\":-1,\"do_not_save_samples\":false,\"do_not_save_grid\":false,\"eta\":0.0,\"s_churn\":0.0,\"s_tmax\":0.0,\"s_tmin\":0.0,\"s_noise\":1.0,\"override_settings\":{},\"override_settings_restore_afterwards\":true,\"script_args\":[],\"sampler_index\":\"Euler\",\"script_name\":\"\",\"send_images\":true,\"save_images\":false,\"alwayson_scripts\":{},\"styles\":[]}
*/
func (t SDImageToImageGenerator) GenerateImages() ([]string, error) {
	t.NIter = 1
	t.SubSeed = -1
	t.SeedResizeFromH = -1
	t.SeedResizeFromW = -1
	t.SNoise = 1.0
	// t.SamplerIndex = "Euler"
	t.SendImages = true
	t.OverrideSettingsRestoreAfterwards = true
	t.AlwaysonScripts = make(map[string]any)

	//
	t.MaskBlur = 4
	t.InpaintFullRes = 0
	t.InpaintingFill = 1
	t.InpaintFullResPadding = 7
	t.InpaintingMaskInvert = 0
	t.InitialNoiseMultiplier = 1

	if len(t.ControlNetUnits) > 0 {
		args := make(map[string][]SDControlNetUnit)
		args["args"] = t.ControlNetUnits
		t.AlwaysonScripts = make(map[string]any)
		t.AlwaysonScripts["controlnet"] = args
	}

	url := fmt.Sprintf("%s/sdapi/v1/img2img", webuiHost)
	byteMsg, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	fmt.Println(string(byteMsg))

	request, _ := http.NewRequest("POST", url, bytes.NewReader(byteMsg))
	request.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	data := SDImageInfo{}
	if err = json.Unmarshal(result, &data); err != nil {
		return nil, err
	}
	return data.Images, nil
}
