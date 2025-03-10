package restfulcomponent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	componentinterface "github.com/world-in-progress/yggdrasil/component/interface"
)

type (
	HTTPMethod string

	ParamKind string

	ParamDescription struct {
		Name         string             `json:"name"`
		Description  string             `json:"description,omitempty"`
		Type         string             `json:"type"`
		Kind         ParamKind          `json:"kind,omitempty"`
		Required     bool               `json:"required,omitempty"`
		Default      any                `json:"default,omitempty"`
		NestedParams []ParamDescription `json:"nestedParams,omitempty"` // nested params (only valid for object and array)
		IsPathParam  bool               `json:"isPathParam,omitempty"`
	}

	ResponseStatus struct {
		Code        int                `json:"code"`
		Description string             `json:"description,omitempty"`
		Schema      string             `json:"schema,omitempty"`
		Params      []ParamDescription `json:"params,omitempty"`
	}

	RestfulComponent struct {
		ID          string             `json:"_id"`
		Name        string             `json:"name"`
		API         string             `json:"api"`
		Method      HTTPMethod         `json:"method"`
		Description string             `json:"description,omitempty"`
		ReqSchema   string             `json:"reqSchema,omitempty"`
		ReqParams   []ParamDescription `json:"reqParams,omitempty"`
		ResStatuses []ResponseStatus   `json:"resStatuses,omitempty"`
		Deprecated  bool               `json:"deprecated,omitempty"`

		callTime time.Time
	}
)

const (
	GET    HTTPMethod = "GET"
	POST   HTTPMethod = "POST"
	PUT    HTTPMethod = "PUT"
	DELETE HTTPMethod = "DELETE"
	PATCH  HTTPMethod = "PATCH"
)

const (
	KindSimple ParamKind = "simple" // string, int, float64, bool
	KindObject ParamKind = "object" // complex object type
	KindArray  ParamKind = "array"  // array
)

var (
	ValidHTTPMethods = map[HTTPMethod]bool{
		GET:    true,
		POST:   true,
		PUT:    true,
		DELETE: true,
		PATCH:  true,
	}

	ValidParamKinds = map[ParamKind]bool{
		KindSimple: true,
		KindObject: true,
		KindArray:  true,
	}

	ValidParamTypes = map[string]bool{
		"string":  true,
		"int":     true,
		"float64": true,
		"bool":    true,
		"object":  true,
		"array":   true,
	}
)

func NewRestfulComponentInstance(componentInfo map[string]any) (*RestfulComponent, error) {
	if c, err := convertToStruct[*RestfulComponent](componentInfo); err != nil {
		return nil, fmt.Errorf("faied to build restful component: %v", err)
	} else {
		c.callTime = time.Now()
		return c, nil
	}
}

func NewRestfulComponent(schema map[string]any) (map[string]any, error) {
	c, err := convertToStruct[*RestfulComponent](schema)
	if err != nil {
		return nil, fmt.Errorf("faied to build restful component: %v", err)
	}

	// calculate uuid for this new schema
	c.ID = "RESTFUL" + "-" + uuid.New().String()

	// verify required fields
	if c.Name == "" || c.API == "" || c.Method == "" {
		return nil, fmt.Errorf("missing required fields: ID, Name, API, or Method")
	}

	// verify http method
	if !ValidHTTPMethods[c.Method] {
		return nil, fmt.Errorf("invalid HTTP method '%s'", c.Method)
	}

	// verify kind of request params and set default values
	for i := range c.ReqParams {
		if err := validateAndSetParamDefaults(&c.ReqParams[i], c.ReqParams[i].Name); err != nil {
			return nil, err
		}
	}

	// verify kind of response params and set default values
	for i := range c.ResStatuses {
		for j := range c.ResStatuses[i].Params {
			if err := validateAndSetParamDefaults(&c.ResStatuses[i].Params[j], c.ResStatuses[i].Params[j].Name); err != nil {
				return nil, err
			}
		}
		if c.ResStatuses[i].Schema == "" {
			c.ResStatuses[i].Schema = "application/json"
		}
	}

	// convert component to map
	if schema, err := convertToMap(c); err != nil {
		return nil, fmt.Errorf("failed to build component schema in type of map: %v", err)
	} else {
		return schema, nil
	}
}

func (c *RestfulComponent) GetID() string {
	c.callTime = time.Now()
	return c.ID
}

func (c *RestfulComponent) GetName() string {
	c.callTime = time.Now()
	return c.Name
}

func (c *RestfulComponent) GetCallTime() time.Time {
	return c.callTime
}

func (c *RestfulComponent) Execute(node componentinterface.INode, params map[string]any, client *http.Client, headers map[string]string) (map[string]any, error) {
	// overwrite specific param by node attribute if not provided by params
	if node != nil {
		for _, reqParam := range c.ReqParams {
			paramName := reqParam.Name
			_, exists := params[paramName]
			attribute := node.GetParam(paramName)
			if !exists && attribute != nil {
				params[paramName] = attribute
			}
		}
	}

	validator := &ParameterValidator{}
	if err := validator.Validate(c, params); err != nil {
		return nil, err
	}

	builder := &RequestBuilder{}
	req, err := builder.BuildRequest(c, params)
	if err != nil {
		return nil, err
	}

	if c.ReqSchema != "" {
		req.Header.Set("Content-Type", c.ReqSchema)
	} else if req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	executor := &HTTPExecutor{Client: client, Headers: headers}
	resp, err := executor.Execute(req)
	if err != nil {
		return nil, err
	}

	handler := &ResponseHandler{}
	return handler.Handle(resp, c)
}

func validateAndSetParamDefaults(param *ParamDescription, paramName string) error {
	// validate type
	if param.Type == "" {
		// If the type is not specified, it is inferred from the kind.
		switch param.Kind {
		case KindObject:
			param.Type = "object"
		case KindArray:
			param.Type = "array"
		case KindSimple, "":
			param.Type = "string"
		}
	} else if !ValidParamTypes[param.Type] {
		return fmt.Errorf("invalid Type '%s' for parameter '%s'", param.Type, paramName)
	}

	// verify and set Kind
	if param.Kind == "" {
		// If the kind is not specified, it is inferred from the type.
		switch param.Type {
		case "object":
			param.Kind = KindObject
		case "array":
			param.Kind = KindArray
		default:
			param.Kind = KindSimple
		}
	} else if !ValidParamKinds[param.Kind] {
		return fmt.Errorf("invalid ParamKind '%s' for parameter '%s'", param.Kind, paramName)
	}

	// Ensure Type and Kind are consistent.
	if (param.Type == "object" && param.Kind != KindObject) ||
		(param.Type == "array" && param.Kind != KindArray) ||
		(param.Type != "object" && param.Type != "array" && param.Kind != KindSimple) {
		return fmt.Errorf("type '%s' and kind '%s' are inconsistent for parameter '%s'", param.Type, param.Kind, paramName)
	}

	// If it is an array or object type, validate the nested parameters.
	if param.Kind == KindArray || param.Kind == KindObject {
		if len(param.NestedParams) == 0 {
			return fmt.Errorf("parameter '%s' with kind '%s' must have at least one nested parameter", paramName, param.Kind)
		}
		for i := range param.NestedParams {
			if err := validateAndSetParamDefaults(&param.NestedParams[i], param.NestedParams[i].Name); err != nil {
				return err
			}
		}
	}

	// If it is a simple type, there should be no nested parameters.
	if param.Kind == KindSimple && len(param.NestedParams) > 0 {
		return fmt.Errorf("simple parameter '%s' should not have nested parameters", paramName)
	}

	return nil
}

func convertToStruct[T any](source any) (T, error) {
	var result T

	bytes, err := json.Marshal(source)
	if err != nil {
		return result, fmt.Errorf("marshal error when transfer source to component: %v", err)
	}

	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return result, fmt.Errorf("unmarshal error source to component: %v", err)
	}

	return result, nil
}

func convertToMap[T any](component T) (map[string]any, error) {
	var result map[string]any

	bytes, err := json.Marshal(component)
	if err != nil {
		return result, fmt.Errorf("marshal error when transfer component to map %v", err)
	}

	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return result, fmt.Errorf("unmarshal error when transfer component to map %v", err)
	}

	return result, nil
}
