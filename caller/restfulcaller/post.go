package restfulcaller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/world-in-progress/yggdrasil/caller"
)

type (
	RestPostTask struct {
		caller.BaseTask
		API       string
		JsonBody  []byte
		ReqSchema string
		ResSchema string
	}
)

func NewRestPostTask(c *RestfulCalling, jsonData []byte) *RestPostTask {

	return &RestPostTask{
		BaseTask:  caller.BaseTask{ID: uuid.New().String()},
		JsonBody:  jsonData,
		API:       c.API,
		ReqSchema: c.ReqSchema,
		ResSchema: c.ResSchema,
	}
}

func (rpt *RestPostTask) Process() error {
	req, err := http.NewRequest("POST", rpt.API, bytes.NewBuffer(rpt.JsonBody))
	if err != nil {
		return fmt.Errorf("error create request: %v", err)
	}

	req.Header.Set("Content-Type", rpt.ReqSchema)
	// TODO: authorization
	// req.Header.Set("Authorization", "Bearer your-token-here")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed: %s, body: %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("error unmarshaling response: %v", err)
	}
	fmt.Println("Response JSON:", result)
	return nil
}
