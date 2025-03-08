package model

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"reflect"
	"sync"

	"github.com/world-in-progress/yggdrasil/config"
)

var basicTypes = map[string]bool{
	"string":  true,
	"int":     true,
	"float64": true,
	"bool":    true,
	"array":   true,
	"map":     true,
}

type (
	FieldDefinition struct {
		Type     string
		Required bool
		Fields   map[string]*FieldDefinition
		Item     *FieldDefinition
		Ref      string
	}

	ModelDefinition struct {
		Name    string
		Extends string
		Fields  map[string]*FieldDefinition
	}

	ModelManager struct {
		models map[string]*ModelDefinition
		mu     sync.RWMutex
	}
)

func NewModelManager() (*ModelManager, error) {

	modelConfig := config.LoadModelConfig()

	data, err := os.ReadFile(modelConfig.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config struct {
		Models []struct {
			Name    string                     `json:"name"`
			Extends string                     `json:"extends,omitempty"`
			Fields  map[string]json.RawMessage `json:"fields"`
		} `json:"models"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarsal JSON: %v", err)
	}

	// First parsing: build base model.
	tempModels := make(map[string]*ModelDefinition)
	for _, m := range config.Models {
		fields := make(map[string]*FieldDefinition)
		if err := parseFields(m.Fields, fields, nil); err != nil {
			return nil, err
		}
		tempModels[m.Name] = &ModelDefinition{
			Name:    m.Name,
			Extends: m.Extends,
			Fields:  fields,
		}
	}

	// Second parsing: process inheritance and complex types.
	models := make(map[string]*ModelDefinition)
	for _, m := range config.Models {
		fields := make(map[string]*FieldDefinition)
		if err := parseFields(m.Fields, fields, tempModels); err != nil {
			return nil, err
		}

		// process inheritance
		if m.Extends != "" {
			if base, ok := tempModels[m.Extends]; ok {
				for k, v := range base.Fields {
					if _, exists := fields[k]; !exists { // do not overwrite existed fields
						fields[k] = v
					}
				}
			} else {
				return nil, fmt.Errorf("base model %s not found for %s", m.Extends, m.Name)
			}
		}

		models[m.Name] = &ModelDefinition{
			Name:   m.Name,
			Fields: fields,
		}
	}
	return &ModelManager{
		models: models,
	}, nil
}

func (mm *ModelManager) HasModel(modelName string) bool {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	_, ok := mm.models[modelName]
	return ok
}

func (mm *ModelManager) Validate(modelName string, data map[string]any) error {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	model, ok := mm.models[modelName]
	if !ok {
		return fmt.Errorf("model %s not found", modelName)
	}
	return mm.validateFields(model.Fields, data)
}

func (mm *ModelManager) ValidateField(modelName string, filedName string, data any) error {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	var ok bool
	var def *FieldDefinition
	var model *ModelDefinition

	model, ok = mm.models[modelName]
	if !ok {
		return fmt.Errorf("model %s not found", modelName)
	}
	def, ok = model.Fields[filedName]
	if !ok {
		return fmt.Errorf("model %s does not have field %s", modelName, filedName)
	}
	return mm.validateField(filedName, data, def)
}

func (mm *ModelManager) validateFields(fields map[string]*FieldDefinition, data map[string]any) error {
	for name, def := range fields {
		value, exists := data[name]
		if !exists {
			if def.Required {
				return fmt.Errorf("field %s is required", name)
			}
			continue
		}
		if err := mm.validateField(name, value, def); err != nil {
			return err
		}
	}
	return nil
}

func parseFields(rawFields map[string]json.RawMessage, fields map[string]*FieldDefinition, models map[string]*ModelDefinition) error {

	for name, raw := range rawFields {
		var def struct {
			Type     string                     `json:"type"`
			Required bool                       `json:"required"`
			Fields   map[string]json.RawMessage `json:"fields,omitempty"`
			Item     json.RawMessage            `json:"item,omitempty"`
			Ref      string                     `json:"ref,omitempty"`
		}
		if err := json.Unmarshal(raw, &def); err != nil {
			return fmt.Errorf("failed to unmarshal field %s: %v", name, err)
		}
		if def.Type == "" {
			return fmt.Errorf("field %s missing type", name)
		}

		fieldDef := &FieldDefinition{
			Type:     def.Type,
			Required: def.Required,
			Ref:      def.Ref,
		}

		// process nested fields (type == "object")
		if def.Fields != nil {
			if def.Type != "object" {
				return fmt.Errorf("field %s: fields only allowed with type 'object'", name)
			}
			fieldDef.Fields = make(map[string]*FieldDefinition)
			if err := parseFields(def.Fields, fieldDef.Fields, models); err != nil {
				return err
			}
		}

		// process array or map item
		if def.Item != nil {
			if def.Type != "array" && def.Type != "map" {
				return fmt.Errorf("field %s: item only allowed with type 'array' or 'map'", name)
			}
			var itemDef struct {
				Type   string                     `json:"type"`
				Fields map[string]json.RawMessage `json:"fields,omitempty"`
				Ref    string                     `json:"ref,omitempty"`
			}
			if err := json.Unmarshal(def.Item, &itemDef); err != nil {
				return fmt.Errorf("failed to unmarshal item for field %s: %v", name, err)
			}
			fieldDef.Item = &FieldDefinition{
				Type: itemDef.Type,
				Ref:  itemDef.Ref,
			}
			if itemDef.Fields != nil {
				if itemDef.Type != "object" {
					return fmt.Errorf("field %s: item fields only allowed wity type 'object'", name)
				}
				fieldDef.Item.Fields = make(map[string]*FieldDefinition)
				if err := parseFields(itemDef.Fields, fieldDef.Fields, models); err != nil {
					return err
				}
			}
		}

		// process complex type (model reference)
		if models != nil {
			// for object referencing a model
			if def.Type != "object" && def.Type != "array" && def.Type != "map" {
				if _, isBasic := basicTypes[def.Type]; !isBasic || def.Ref != "" {
					refType := def.Type
					if def.Ref != "" {
						refType = def.Ref // backwards compatibility
					}
					if refModel, ok := models[refType]; ok {
						fieldDef.Type = "object"
						fieldDef.Fields = make(map[string]*FieldDefinition)
						maps.Copy(fieldDef.Fields, refModel.Fields)
					} else if !isBasic {
						return fmt.Errorf("field %s: type %s is not a basic type or defined model", name, def.Type)
					}
				}
			}
			// for array or map item referencing a model
			if (def.Type == "array" || def.Type == "map") && fieldDef.Item != nil {
				if _, isBasic := basicTypes[fieldDef.Item.Type]; !isBasic || fieldDef.Item.Ref != "" {
					refType := fieldDef.Item.Type
					if fieldDef.Item.Ref != "" {
						refType = fieldDef.Item.Ref
					}
					if refModel, ok := models[refType]; ok {
						fieldDef.Item.Type = "object"
						fieldDef.Item.Fields = make(map[string]*FieldDefinition)
						maps.Copy(fieldDef.Item.Fields, refModel.Fields)
					} else if !isBasic {
						return fmt.Errorf("field %s: item type %s is not a basic type or defined model", name, fieldDef.Item.Type)
					}
				}
			}
		}
		fields[name] = fieldDef
	}
	return nil
}

type validatorFunc func(name string, value any) error

var typeValidators = map[string]validatorFunc{
	"string": func(name string, value any) error {
		if _, ok := value.(string); !ok {
			return fmt.Errorf("%s must be a string", name)
		}
		return nil
	},
	"int": func(name string, value any) error {
		if _, ok := value.(int); !ok {
			if f, ok := value.(float64); ok && float64(int(f)) == f {
				reflect.ValueOf(&value).Elem().Set(reflect.ValueOf(int(f)))
				return nil
			}
			return fmt.Errorf("%s must be an integer", name)
		}
		return nil
	},
	"float64": func(name string, value any) error {
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("%s must be a float64", name)
		}
		return nil
	},
	"bool": func(name string, value any) error {
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%s must be a bool", name)
		}
		return nil
	},
}

func (mm *ModelManager) validateField(name string, value any, def *FieldDefinition) error {
	if modelDef, ok := mm.models[def.Type]; ok {
		return mm.validateFields(modelDef.Fields, value.(map[string]any))
	}

	switch def.Type {
	case "string", "int", "float64", "bool":
		validator, ok := typeValidators[def.Type]
		if !ok {
			return fmt.Errorf("unsupported type %s for %s", def.Type, name)
		}
		return validator(name, value)

	case "object":
		nested, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("%s must be an object", name)
		}
		return mm.validateFields(def.Fields, nested)

	case "array":
		arr, ok := value.([]any)
		if !ok {
			return fmt.Errorf("%s must be an array", name)
		}
		if def.Item == nil {
			return nil
		}
		for i, item := range arr {
			itemName := fmt.Sprintf("item %d in %s", i, name)
			if err := mm.validateField(itemName, item, def.Item); err != nil {
				return err
			}
		}
		return nil

	case "map":
		m, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("%s must be a map", name)
		}
		if def.Item == nil {
			return nil
		}
		for key, val := range m {
			keyName := fmt.Sprintf("%s[%s]", name, key)
			if err := mm.validateField(keyName, val, def.Item); err != nil {
				return err
			}
		}
		return nil

	default:
		return fmt.Errorf("unsupported type %s for %s", def.Type, name)
	}
}
