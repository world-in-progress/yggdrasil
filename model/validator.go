package model

// func ValidateFields(fields map[string]*FieldDefinition, data map[string]any) error {
// 	for name, def := range fields {
// 		value, exists := data[name]
// 		if !exists {
// 			if def.Required {
// 				return fmt.Errorf("field %s is required", name)
// 			}
// 			continue
// 		}
// 		if err := validateField(name, value, def); err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// type validatorFunc func(name string, value any) error

// var typeValidators = map[string]validatorFunc{
// 	"string": func(name string, value any) error {
// 		if _, ok := value.(string); !ok {
// 			return fmt.Errorf("%s must be a string", name)
// 		}
// 		return nil
// 	},
// 	"int": func(name string, value any) error {
// 		if _, ok := value.(int); !ok {
// 			if f, ok := value.(float64); ok && float64(int(f)) == f {
// 				reflect.ValueOf(&value).Elem().Set(reflect.ValueOf(int(f)))
// 				return nil
// 			}
// 			return fmt.Errorf("%s must be an integer", name)
// 		}
// 		return nil
// 	},
// 	"float64": func(name string, value any) error {
// 		if _, ok := value.(float64); !ok {
// 			return fmt.Errorf("%s must be a float64", name)
// 		}
// 		return nil
// 	},
// 	"bool": func(name string, value any) error {
// 		if _, ok := value.(bool); !ok {
// 			return fmt.Errorf("%s must be a bool", name)
// 		}
// 		return nil
// 	},
// }

// func validateField(name string, value any, def *FieldDefinition) error {
// 	switch def.Type {
// 	case "string", "int", "float64", "bool":
// 		validator, ok := typeValidators[def.Type]
// 		if !ok {
// 			return fmt.Errorf("unsupported type %s for %s", def.Type, name)
// 		}
// 		return validator(name, value)

// 	case "object":
// 		nested, ok := value.(map[string]any)
// 		if !ok {
// 			return fmt.Errorf("%s must be an object", name)
// 		}
// 		return ValidateFields(def.Fields, nested)

// 	case "array":
// 		arr, ok := value.([]any)
// 		if !ok {
// 			return fmt.Errorf("%s must be an array", name)
// 		}
// 		if def.Item == nil {
// 			return nil
// 		}
// 		for i, item := range arr {
// 			itemName := fmt.Sprintf("item %d in %s", i, name)
// 			if err := validateField(itemName, item, def.Item); err != nil {
// 				return err
// 			}
// 		}
// 		return nil

// 	case "map":
// 		m, ok := value.(map[string]any)
// 		if !ok {
// 			return fmt.Errorf("%s must be a map", name)
// 		}
// 		if def.Item == nil {
// 			return nil
// 		}
// 		for key, val := range m {
// 			keyName := fmt.Sprintf("%s[%s]", name, key)
// 			if err := validateField(keyName, val, def.Item); err != nil {
// 				return err
// 			}
// 		}
// 		return nil

// 	default:
// 		return fmt.Errorf("unsupported type %s for %s", def.Type, name)
// 	}
// }
