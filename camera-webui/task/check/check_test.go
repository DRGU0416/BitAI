package main

import (
	"camera-webui/lib"
	"camera-webui/models"
	"camera-webui/task/check/cron"
	"encoding/json"
	"testing"
	"time"
)

func TestCheckImg(t *testing.T) {

	task := lib.TaskCheck{
		ImagesMap: map[uint64]string{
			1: "https://c-ssl.dtstatic.com/uploads/item/201905/18/20190518091538_qnacy.thumb.1000_0.jpg",
			2: "https://plc.jj20.com/up/allimg/mx07/0I11Z13341/1ZI1013341-3.jpg",
			3: "https://c-ssl.dtstatic.com/uploads/item/201912/06/20191206014607_rtrxu.thumb.1000_0.jpg",
			4: "https://n.sinaimg.cn/sinacn16/129/w2048h2881/20181110/f365-hnstwwq1720620.jpg",
			5: "https://n.sinaimg.cn/sinacn07/283/w2048h3035/20180902/81f7-hinpmnr6633563.jpg",
			6: "https://www.chinadaily.com.cn/language_tips/images/attachement/jpg/site1/20170904/64006a47a40a1b16edb001.jpg",
		},
		Callback: "http://baidu.com",
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Errorf("json序列化失败, %s", err)
		return
	}

	chWork := &models.CheckWork{
		ID:        1,
		JsonData:  string(data),
		Status:    0,
		CreatedAt: time.Now().Unix(),
		Callback:  "http://baidu.com",
	}
	cron.RunCheck(chWork)
}
