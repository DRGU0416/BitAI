package main

import (
	"camera-webui/lib"
	"camera-webui/models"
	"camera-webui/task/train/cron"
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestCheckImg(t *testing.T) {

	task := lib.Task{
		UserId: 1,
		TaskId: 1,
		Stype: lib.Stype{
			EnableHr:          false,
			HrScale:           0,
			HiresUpscaler:     "",
			HrSecondPassSteps: 0,
			DenoisingStrength: 0,
			SamplerName:       "",
			Prompt:            "",
			NegativePrompt:    "",
			Width:             0,
			Height:            0,
			Seed:              0,
			Steps:             0,
			RestoreFace:       false,
			Tiling:            false,
			CfgScale:          0,
			BatchSize:         0,
			BatchCount:        0,
			MainModelPath:     "",
			SubModelUrl:       "",
			ControlNets:       nil,
			Roop:              nil,
			ADetailer:         nil,
		},
		LoraTrain: lib.LoraTrain{
			BaseModel: "E:\\sdwebui\\stable-diffusion-webui\\models\\Stable-diffusion\\realisticVisionV50_v50VAE.safetensors",
			ImageUrl: []string{
				"https://c-ssl.dtstatic.com/uploads/item/201905/18/20190518091538_qnacy.thumb.1000_0.jpg",
				"https://plc.jj20.com/up/allimg/mx07/0I11Z13341/1ZI1013341-3.jpg",
				"https://c-ssl.dtstatic.com/uploads/item/201912/06/20191206014607_rtrxu.thumb.1000_0.jpg",
				"https://n.sinaimg.cn/sinacn16/129/w2048h2881/20181110/f365-hnstwwq1720620.jpg",
				"https://5b0988e595225.cdn.sohucs.com/images/20190107/30da1396020f4bff9e8f66220c40c8ba.jpeg",
			},
			UUID: "Jknnje",
		},
		Callback: "http://baidu.com",
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Errorf("json序列化失败, %s", err)
		return
	}

	trainWork := &models.TrainWork{
		ID:        1,
		JsonData:  string(data),
		Status:    0,
		CreatedAt: time.Now().Unix(),
		Callback:  "http://baidu.com",
	}
	ctx, _ := context.WithCancel(context.Background())
	cron.RunTrain(ctx, trainWork)
}
