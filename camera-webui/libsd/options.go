package libsd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

type SDOptions struct {
	SDModelCheckpoint string `json:"sd_model_checkpoint"`
}

func (opt SDOptions) ApplySDOptions() error {
	url := fmt.Sprintf("%s/sdapi/v1/options", webuiHost)

	byteMsg, err := json.Marshal(opt)
	if err != nil {
		return err
	}
	request, _ := http.NewRequest("POST", url, bytes.NewReader(byteMsg))
	request.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println("设置模型：", string(result))
	return err
}
