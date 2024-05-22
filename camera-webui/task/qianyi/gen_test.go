package main

import (
	"camera-webui/lib"
	"camera-webui/models"
	"camera-webui/task/qianyi/cron"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func TestGenImg(t *testing.T) {

	// err := lib.CopyFile("D:\\lora\\1\\camera_1_4-000036.safetensors", "E:\\sdwebui\\stable-diffusion-webui\\models\\Lora\\camera_1_4-000036.safetensors", true)
	// t.Logf("err: %s", err)
	// return
	wch := make(chan int, 2)
	wcs := new(sync.WaitGroup)

	task := &lib.Task{}
	task.UserId = 1
	task.TaskId = 10
	task.Callback = "http://baidu.com"
	task.Stype = lib.Stype{
		// EnableHr:          true,
		// HrScale:           2,
		// HiresUpscaler:     "4x-UltraSharp",
		// HrSecondPassSteps: 20,
		DenoisingStrength: 0.22,
		SamplerName:       "DPM++ 2M SDE Karras",
		Prompt:            "1girl, solo, realistic, brown hair, brown eyes, looking at viewer, hair ornament, dress, bare shoulders, lips, upper body, white dress, flower, hair flower, off shoulder, collarbone, parted lips, dark backgroud, simple background, high light on the face,<lora:gudianweimei:1>",
		NegativePrompt:    "(low quality, worst quality:1.4)",
		Width:             512,
		Height:            768,
		Seed:              -1,
		Steps:             20,
		CfgScale:          7,
		BatchSize:         1,
		BatchCount:        1,
		RestoreFace:       true,
		Tiling:            false,
		MainModelPath:     "E:\\sdwebui\\stable-diffusion-webui\\models\\Stable-diffusion\\realisticVisionV50_v50VAE.safetensors",
		ControlNets: []*lib.ControlNet{
			{
				ImagePath:     "https://aicimg.catcamai.com/card/card.png",
				Preprocessor:  "lineart_standard (from white bg & black line)",
				ModelName:     "control_v11p_sd15_lineart [43d4be0d]",
				Weight:        0.5,
				StartCtrlStep: 0,
				EndCtrlStep:   0.3,
				PreprocRes:    768,
				ControlMode:   "Balanced",
				ResizeMode:    "Scale to Fit (Inner Fit)",
				PixelPerfect:  true,
			},
		},
		Roop: &lib.Roop{
			ImagePath: "https://aicimg.catcamai.com/material/67/f8/c2a92affce5952bfcf4cd2.jpg",
		},
		ADetailer: []*lib.ADetailer{
			{
				ModelUrl:        "http://down.haitaotaopa.com/lora/1/camera_1_4-000036.safetensors",
				AdPrompt:        "Jknnje, <lora:camera_1_4-000036:0.8>",
				AdInpaintWidth:  512,
				AdInpaintHeight: 768,
			},
		},
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Errorf("json序列化失败, %s", err)
		return
	}

	sdword := &models.SDWork{
		ID:        1,
		JsonData:  string(data),
		Status:    0,
		CreatedAt: time.Now().Unix(),
		Callback:  "http://baidu.com",
	}
	cron.RunTask(wch, wcs, sdword)
	wcs.Wait()
}
