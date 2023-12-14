package internal

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// 仅支持HTTP、POST、JSON
func PostJSON(url string, request, response any) error {
	buf, err := json.Marshal(request)
	if err != nil {
		return err
	}
	// log.Infof("report url %s body %s", url, buf)

	body, err := Post(url, "application/json", buf)
	return json.Unmarshal(body, response)
}

func Post(url, contentType string, buf []byte) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", contentType)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
