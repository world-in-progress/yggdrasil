package db

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
)

type FieldDefinition struct {
	Type     string
	Required bool
	Fields   map[string]*FieldDefinition
	Item     *FieldDefinition
	Ref      string
}

type ModelDefinition struct {
	Name    string
	Extends string
	Fields  map[string]*FieldDefinition
}

type ModelManager struct {
	models map[string]*ModelDefinition
	mu     sync.RWMutex
}

var basicTypes = map[string]bool{
	"string":  true,
	"int":     true,
	"float64": true,
	"bool":    true,
	"array":   true,
}

func NewModelManager(configPath string) (*ModelManager, error) {
	data, err := os.ReadFile(configPath)
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

func (m *ModelManager) ValidateData(modelName string, data map[string]any) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	model, ok := m.models[modelName]
	if !ok {
		return fmt.Errorf("model %s not found", modelName)
	}
	return validateFields(model.Fields, data)
}

func (m *ModelManager) ToBSON(modelName string, data map[string]any) (bson.M, error) {
	if err := m.ValidateData(modelName, data); err != nil {
		return nil, err
	}
	return bson.M(data), nil
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

		// process nested fields
		if def.Fields != nil {
			if def.Type != "object" {
				return fmt.Errorf("field %s: fields only allowed with type 'object'", name)
			}
			fieldDef.Fields = make(map[string]*FieldDefinition)
			if err := parseFields(def.Fields, fieldDef.Fields, models); err != nil {
				return err
			}
		}

		// process array item
		if def.Item != nil {
			if def.Type != "array" {
				return fmt.Errorf("field %s: item only allowed with type 'array'", name)
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
			if def.Type != "object" && def.Type != "array" {
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
			// for array item referencing a model
			if def.Type == "array" && fieldDef.Item != nil {
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

func validateFields(fields map[string]*FieldDefinition, data map[string]any) error {
	for name, def := range fields {
		value, exists := data[name]
		if !exists && def.Required {
			return fmt.Errorf("field %s is required", name)
		}
		if exists {
			switch def.Type {
			case "string":
				if _, ok := value.(string); !ok {
					return fmt.Errorf("field %s must be a string", name)
				}
			case "int":
				if _, ok := value.(int); !ok {
					if f, ok := value.(float64); ok && float64((int(f))) == f {
						data[name] = int(f)
					} else {
						return fmt.Errorf("field %s must be an integer", name)
					}
				}
			case "float64":
				if _, ok := value.(float64); !ok {
					return fmt.Errorf("field %s must be a float64", name)
				}
			case "bool":
				if _, ok := value.(bool); !ok {
					return fmt.Errorf("field %s must be a bool", name)
				}
			case "object":
				if nested, ok := value.(map[string]any); ok {
					if err := validateFields(def.Fields, nested); err != nil {
						return fmt.Errorf("field %s: %v", name, err)
					}
				} else {
					return fmt.Errorf("field %s must be an object", name)
				}
			case "array":
				if arr, ok := value.([]any); ok {
					if def.Item != nil {
						for i, item := range arr {
							switch def.Item.Type {
							case "string":
								if _, ok := item.(string); !ok {
									return fmt.Errorf("item %d in %s must be a string", i, name)
								}
							case "int":
								if _, ok := item.(int); !ok {
									if f, ok := item.(float64); ok && float64(int(f)) == f {
										arr[i] = int(f)
									} else {
										return fmt.Errorf("item %d in %s must be an integer", i, name)
									}
								}
							case "float64":
								if _, ok := item.(float64); !ok {
									return fmt.Errorf("item %d in %s must be a float64", i, name)
								}
							case "bool":
								if _, ok := item.(bool); !ok {
									return fmt.Errorf("item %d in %s must be a bool", i, name)
								}
							case "object":
								if nested, ok := item.(map[string]any); ok {
									if err := validateFields(def.Item.Fields, nested); err != nil {
										return fmt.Errorf("item %d in %s: %v", i, name, err)
									}
								} else {
									return fmt.Errorf("item %d in %s must be an object", i, name)
								}
							}
						}
					}
				} else {
					return fmt.Errorf("field %s must be an array", name)
				}
			}
		}
	}
	return nil
}
