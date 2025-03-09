package restfulcomponent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type RequestBuilder struct{}

func (b *RequestBuilder) BuildRequest(c *RestfulComponent, params map[string]any) (*http.Request, error) {
	// build URL
	apiURL := c.API
	for _, param := range c.ReqParams {
		if param.IsPathParam {
			value, exists := params[param.Name]
			if !exists && param.Required && param.Default == nil {
				return nil, fmt.Errorf("missing required path parameter '%s'", param.Name)
			}
			var paramValue string
			if value != nil {
				switch v := value.(type) {
				case string:
					paramValue = v
				case int:
					paramValue = fmt.Sprintf("%d", v)
				case float64:
					paramValue = fmt.Sprintf("%f", v)
				default:
					return nil, fmt.Errorf("path parameter '%s' must be a string, int, or float64, got %T", param.Name, value)
				}
			} else if param.Default != nil {
				paramValue = fmt.Sprintf("%v", param.Default)
			}
			apiURL = strings.Replace(apiURL, "{"+param.Name+"}", url.PathEscape(paramValue), 1)
		}
	}

	// deal with query params and request body
	queryParams := url.Values{}
	var bodyParams map[string]any
	if c.Method == GET || c.Method == DELETE {
		for _, param := range c.ReqParams {
			if !param.IsPathParam {
				if value, exists := params[param.Name]; exists {
					switch v := value.(type) {
					case string:
						queryParams.Add(param.Name, v)
					case int:
						queryParams.Add(param.Name, fmt.Sprintf("%d", v))
					case float64:
						queryParams.Add(param.Name, fmt.Sprintf("%f", v))
					case bool:
						queryParams.Add(param.Name, fmt.Sprintf("%t", v))
					case []any:
						for _, item := range v {
							queryParams.Add(param.Name, fmt.Sprintf("%v", item))
						}
					default:
						return nil, fmt.Errorf("unsupported query parameter type for '%s': %T", param.Name, value)
					}
				}
			}
		}
	} else {
		bodyParams = make(map[string]any)
		for paramName, value := range params {
			isPathParam := false
			for _, reqParam := range c.ReqParams {
				if reqParam.Name == paramName && reqParam.IsPathParam {
					isPathParam = true
					break
				}
			}
			if !isPathParam {
				bodyParams[paramName] = value
			}
		}
	}

	// build request body
	var reqBody *bytes.Buffer
	if c.Method == POST || c.Method == PUT || c.Method == PATCH {
		if len(bodyParams) > 0 {
			body, err := json.Marshal(bodyParams)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %v", err)
			}
			reqBody = bytes.NewBuffer(body)
		}
	}

	// create request
	req, err := http.NewRequest(string(c.Method), apiURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// add query params
	if len(queryParams) > 0 {
		req.URL.RawQuery = queryParams.Encode()
	}

	return req, nil
}
