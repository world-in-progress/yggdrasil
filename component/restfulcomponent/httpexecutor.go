package restfulcomponent

import "net/http"

type HTTPExecutor struct {
	Client  *http.Client
	Headers map[string]string
}

func (e *HTTPExecutor) Execute(req *http.Request) (*http.Response, error) {
	if e.Client == nil {
		e.Client = &http.Client{}
	}
	for key, value := range e.Headers {
		req.Header.Set(key, value)
	}
	return e.Client.Do(req)
}
