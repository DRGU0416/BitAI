package libsd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

type SDImageTagsGenerator struct {
	Image     string  `json:"image"`
	Model     string  `json:"model"`
	Threshold float64 `json:"threshold"`
}

type SDTagResponse struct {
	Caption map[string]float64 `json:"caption"`
}

type SDTagError struct {
	Error  string `json:"error"`
	Errors string `json:"errors"`
}

func (tag SDImageTagsGenerator) GenerateImageTags() (map[string]float64, error) {
	tag.Model = "wd14-swinv2-v1" //"wd14-swinv2-v2-git" //"wd14-vit-v2"
	tag.Threshold = 0.45

	url := fmt.Sprintf("%s/tagger/v1/interrogate", webuiHost)

	byteMsg, err := json.Marshal(tag)
	if err != nil {
		return nil, err
	}
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

	data := SDTagResponse{}
	if err := json.Unmarshal(result, &data); err != nil || data.Caption == nil {
		errResp := SDTagError{}
		if err := json.Unmarshal(result, &errResp); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%s: %s", errResp.Error, errResp.Errors)
	}
	return data.Caption, nil
}
