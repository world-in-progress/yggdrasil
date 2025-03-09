package restfulcomponent

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type ResponseHandler struct{}

func (h *ResponseHandler) Handle(resp *http.Response, c *RestfulComponent) (map[string]any, error) {
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	for _, status := range c.ResStatuses {
		if resp.StatusCode == status.Code {
			return result, nil
		}
	}
	return result, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}
