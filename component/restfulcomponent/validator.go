package restfulcomponent

import "fmt"

type ParameterValidator struct{}

func (v *ParameterValidator) Validate(c *RestfulComponent, params map[string]any) error {
	for _, reqParam := range c.ReqParams {
		paramName := reqParam.Name
		value, exists := params[paramName]
		if err := v.validateParamValue(reqParam, value, paramName); err != nil {
			return err
		}
		if !exists && reqParam.Required && reqParam.Default == nil {
			return fmt.Errorf("missing required parameter '%s'", paramName)
		}
	}
	for paramName := range params {
		found := false
		for _, reqParam := range c.ReqParams {
			if reqParam.Name == paramName {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unknown parameter '%s' provided", paramName)
		}
	}
	return nil
}

func (v *ParameterValidator) validateParamValue(param ParamDescription, value any, paramPath string) error {
	if param.Required && value == nil {
		return fmt.Errorf("missing required parameter at '%s'", paramPath)
	}

	if value == nil {
		return nil
	}

	switch param.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("parameter '%s' must be a string, got %T", paramPath, value)
		}
	case "int":
		switch v := value.(type) {
		case int:
		case float64:
			if float64(int(v)) != v {
				return fmt.Errorf("parameter '%s' must be an integer, got %v (non-integer float)", paramPath, v)
			}
		default:
			return fmt.Errorf("parameter '%s' must be an int, got %T", paramPath, value)
		}
	case "float64":
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("parameter '%s' must be a float64, got %T", paramPath, value)
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("parameter '%s' must be a bool, got %T", paramPath, value)
		}
	case "object":
		obj, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("parameter '%s' must be an object (map[string]any), got %T", paramPath, value)
		}
		// verify nested field
		for _, nestedParam := range param.NestedParams {
			if nestedValue, exists := obj[nestedParam.Name]; exists {
				nestedPath := fmt.Sprintf("%s.%s", paramPath, nestedParam.Name)
				if err := v.validateParamValue(nestedParam, nestedValue, nestedPath); err != nil {
					return err
				}
			}
		}
	case "array":
		arr, ok := value.([]any)
		if !ok {
			return fmt.Errorf("parameter '%s' must be an array ([]any), got %T", paramPath, value)
		}
		// verify array elements
		if len(param.NestedParams) != 1 {
			return fmt.Errorf("array parameter '%s' must have exactly one nested parameter definition, got %d", paramPath, len(param.NestedParams))
		}
		nestedParam := param.NestedParams[0]
		for i, item := range arr {
			nestedPath := fmt.Sprintf("%s[%d]", paramPath, i)
			if err := v.validateParamValue(nestedParam, item, nestedPath); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported parameter type '%s' for '%s'", param.Type, paramPath)
	}
	return nil
}
