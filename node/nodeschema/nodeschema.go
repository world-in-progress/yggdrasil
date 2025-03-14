package nodeschema

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"sync"

	"github.com/google/uuid"
	nodeinterface "github.com/world-in-progress/yggdrasil/node/interface"
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

	SchemaDefinition struct {
		ID      string
		Name    string
		Extends string
		Fields  map[string]*FieldDefinition
	}

	SchemaManager struct {
		repo  nodeinterface.IRepository
		cache map[string]*SchemaDefinition
		mu    sync.RWMutex
	}
)

func NewSchemaManager(repo nodeinterface.IRepository) *SchemaManager {
	return &SchemaManager{
		repo:  repo,
		cache: make(map[string]*SchemaDefinition),
	}
}

func (sm *SchemaManager) RegisterSchema(schemaInfo map[string]any) (string, error) {
	ctx := context.Background()

	// Check if schema info is empty.
	if len(schemaInfo) == 0 {
		return "", fmt.Errorf("schemaInfo cannot be empty")
	}

	// Extract and verify schema name.
	name, ok := schemaInfo["name"].(string)
	if !ok || name == "" {
		return "", fmt.Errorf("schema name must be a non-empry string")
	}

	// Check if schema already exists.
	sm.mu.RLock()
	_, existsInCache := sm.cache[name]
	sm.mu.RUnlock()
	if existsInCache {
		return "", fmt.Errorf("schema name %s already exists", name)
	}

	count, err := sm.repo.Count(ctx, "nodeschema", map[string]any{"name": name})
	if err != nil {
		return "", fmt.Errorf("check for model name duplication failed: %v", err)
	}
	if count > 0 {
		return "", fmt.Errorf("schema name %s already exists", name)
	}

	// Extract and verify field of extends if existing.
	var extends string
	if ext, ok := schemaInfo["extends"]; ok {
		extends, ok = ext.(string)
		if !ok {
			return "", fmt.Errorf("extends of schema info must be a string")
		}
		if extends != "" && !sm.HasSchema(extends) {
			return "", fmt.Errorf("base schema %s does not exist", extends)
		}
	}

	// Extract and verify field of fields.
	fieldsRaw, ok := schemaInfo["fields"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("fields of schema info must be type of map[string]any")
	}

	// Try to parse fieldsRaw.
	fieldsJson, err := json.Marshal(fieldsRaw)
	if err != nil {
		return "", fmt.Errorf("failed to serialize fields of schema %s", name)
	}
	rawFields := make(map[string]json.RawMessage)
	if err := json.Unmarshal(fieldsJson, &rawFields); err != nil {
		return "", fmt.Errorf("failed to deserialize fields of schema %s: %v", name, err)
	}

	// First paring: build base model.
	fields := make(map[string]*FieldDefinition)
	if err := ParseFields(rawFields, fields, nil); err != nil {
		return "", err
	}

	// Second parsing: process inheritance and complex types.
	if extends != "" {
		baseModel, err := sm.LoadSchema(ctx, extends)
		if err != nil {
			return "", fmt.Errorf("failed to load base schema %s: %v", extends, err)
		}
		for k, v := range baseModel.Fields {
			if _, exists := fields[k]; !exists { // do not overwrite existing fields
				fields[k] = v
			}
		}
	}

	// Write schema to cache.
	sm.mu.Lock()
	sm.cache[name] = &SchemaDefinition{
		Name:   name,
		Fields: fields,
	}
	sm.mu.Unlock()

	// Write schema to repository.
	schemaID := uuid.New().String()
	record := map[string]any{
		"_id":     schemaID,
		"name":    name,
		"extends": extends,
		"fields":  fieldsRaw,
	}
	_, err = sm.repo.Create(ctx, "nodeschema", record)
	if err != nil {
		return "", fmt.Errorf("failed to store node schema to repository: %v", err)
	}

	return schemaID, nil
}

// GetSchemaID get the ID of a specific shcema by its name
func (sm *SchemaManager) GetSchemaID(schemaName string) (string, error) {
	// Find in cache.
	sm.mu.RLock()
	if cached, ok := sm.cache[schemaName]; ok {
		sm.mu.RUnlock()
		return cached.ID, nil
	}
	sm.mu.RUnlock()

	// Find in repository.
	ctx := context.Background()
	record, err := sm.repo.ReadOne(ctx, "nodeschema", map[string]any{"name": schemaName})
	if err != nil {
		return "", fmt.Errorf("failed to find schema having name '%s': %v", schemaName, err)
	}

	return record["_id"].(string), nil
}

// GetSchema gets a specific schema by its ID.
func (sm *SchemaManager) GetSchema(schemaID string) (*SchemaDefinition, error) {
	// Query schema from repository.
	ctx := context.Background()
	record, err := sm.repo.ReadOne(ctx, "nodeschema", map[string]any{"_id": schemaID})
	if err != nil {
		return nil, fmt.Errorf("cannot find schema by ID %s: %v", schemaID, err)
	}
	if len(record) == 0 {
		return nil, fmt.Errorf("schema having ID %s does not exist", schemaID)
	}

	name, _ := record["name"].(string)
	extends, _ := record["extends"].(string)
	fieldsRaw, _ := record["field"].(map[string]any)

	sm.mu.RLock()
	if cached, ok := sm.cache[name]; ok {
		sm.mu.RUnlock()
		return cached, nil
	}
	sm.mu.RUnlock()

	fieldsJson, err := json.Marshal(fieldsRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize fields of schema %s", name)
	}
	rawFields := make(map[string]json.RawMessage)
	if err := json.Unmarshal(fieldsJson, &rawFields); err != nil {
		return nil, fmt.Errorf("failed to deserialize fields of schema %s: %v", name, err)
	}

	// First paring: build base model.
	fields := make(map[string]*FieldDefinition)
	if err := ParseFields(rawFields, fields, nil); err != nil {
		return nil, err
	}

	// Second parsing: process inheritance and complex types.
	if extends != "" {
		baseModel, err := sm.LoadSchema(ctx, extends)
		if err != nil {
			return nil, fmt.Errorf("failed to load base schema %s: %v", extends, err)
		}
		for k, v := range baseModel.Fields {
			if _, exists := fields[k]; !exists { // do not overwrite existing fields
				fields[k] = v
			}
		}
	}

	// Write schema to cache.
	schema := &SchemaDefinition{
		Name:   name,
		Fields: fields,
	}
	sm.mu.Lock()
	sm.cache[name] = schema
	sm.mu.Unlock()

	return schema, nil
}

// LoadSchema loads a specific schema by its name.
func (sm *SchemaManager) LoadSchema(ctx context.Context, schemaName string) (*SchemaDefinition, error) {
	sm.mu.RLock()
	if cached, ok := sm.cache[schemaName]; ok {
		sm.mu.RUnlock()
		return cached, nil
	}
	sm.mu.RUnlock()

	// Query schema from repository.
	record, err := sm.repo.ReadOne(ctx, "nodeschema", map[string]any{"name": schemaName})
	if err != nil {
		return nil, fmt.Errorf("cannot find schema %s: %v", schemaName, err)
	}
	if len(record) == 0 {
		return nil, fmt.Errorf("schema %s does not exist", schemaName)
	}

	name, _ := record["name"].(string)
	extends, _ := record["extends"].(string)
	fieldsRaw, _ := record["field"].(map[string]any)

	fieldsJson, err := json.Marshal(fieldsRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize fields of schema %s", name)
	}
	rawFields := make(map[string]json.RawMessage)
	if err := json.Unmarshal(fieldsJson, &rawFields); err != nil {
		return nil, fmt.Errorf("failed to deserialize fields of schema %s: %v", name, err)
	}

	// First paring: build base model.
	fields := make(map[string]*FieldDefinition)
	if err := ParseFields(rawFields, fields, nil); err != nil {
		return nil, err
	}

	// Second parsing: process inheritance and complex types.
	if extends != "" {
		baseModel, err := sm.LoadSchema(ctx, extends)
		if err != nil {
			return nil, fmt.Errorf("failed to load base schema %s: %v", extends, err)
		}
		for k, v := range baseModel.Fields {
			if _, exists := fields[k]; !exists { // do not overwrite existing fields
				fields[k] = v
			}
		}
	}

	// Write schema to cache.
	schema := &SchemaDefinition{
		Name:   name,
		Fields: fields,
	}
	sm.mu.Lock()
	sm.cache[schemaName] = schema
	sm.mu.Unlock()

	return schema, nil
}

func (sm *SchemaManager) HasSchema(schemaName string) bool {
	ctx := context.Background()
	_, err := sm.LoadSchema(ctx, schemaName)
	return err == nil
}

func (sm *SchemaManager) HasSchemaByID(schemaID string) bool {
	ctx := context.Background()
	_, err := sm.repo.ReadOne(ctx, "nodeschema", map[string]any{"_id": schemaID})
	return err == nil
}

func (sm *SchemaManager) Validate(schemaName string, data map[string]any) error {
	ctx := context.Background()
	schema, err := sm.LoadSchema(ctx, schemaName)
	if err != nil {
		return err
	}
	return sm.validateFields(ctx, schema.Fields, data)
}

func (sm *SchemaManager) ValidateField(schemaName string, fieldName string, data any) error {
	ctx := context.Background()
	schema, err := sm.LoadSchema(ctx, schemaName)
	if err != nil {
		return err
	}
	def, ok := schema.Fields[fieldName]
	if !ok {
		return fmt.Errorf("schema %s dose not have a field named %s", schemaName, fieldName)
	}
	return sm.validateField(ctx, fieldName, data, def)
}

func (sm *SchemaManager) validateFields(ctx context.Context, fields map[string]*FieldDefinition, data map[string]any) error {
	for name, def := range fields {
		value, exists := data[name]
		if !exists {
			if def.Required {
				return fmt.Errorf("field %s is required", name)
			}
			continue
		}
		if err := sm.validateField(ctx, name, value, def); err != nil {
			return err
		}
	}
	return nil
}

func ParseFields(rawFields map[string]json.RawMessage, fields map[string]*FieldDefinition, schemas map[string]*SchemaDefinition) error {

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

		// Process nested fields (type == "object").
		if def.Fields != nil {
			if def.Type != "object" {
				return fmt.Errorf("field %s: fields only allowed with type 'object'", name)
			}
			fieldDef.Fields = make(map[string]*FieldDefinition)
			if err := ParseFields(def.Fields, fieldDef.Fields, schemas); err != nil {
				return err
			}
		}

		// Process array or map item.
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
				if err := ParseFields(itemDef.Fields, fieldDef.Fields, schemas); err != nil {
					return err
				}
			}
		}

		// Process complex type (model reference).
		if schemas != nil {
			// for object referencing a model
			if def.Type != "object" && def.Type != "array" && def.Type != "map" {
				if _, isBasic := basicTypes[def.Type]; !isBasic || def.Ref != "" {
					refType := def.Type
					if def.Ref != "" {
						refType = def.Ref // backwards compatibility
					}
					if refModel, ok := schemas[refType]; ok {
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
					if refModel, ok := schemas[refType]; ok {
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

func (sm *SchemaManager) validateField(ctx context.Context, name string, value any, def *FieldDefinition) error {
	if ctx == nil {
		ctx = context.Background()
	}

	// If type of a field is a referenced schema, make recursively loading and validation.
	if _, isSchema := sm.cache[def.Type]; isSchema || (!basicTypes[def.Type] && def.Type != "object" && def.Type != "array" && def.Type != "map") {
		schema, err := sm.LoadSchema(ctx, def.Type)
		if err != nil {
			return fmt.Errorf("failed to load referenced schema %s: %v", def.Type, err)
		}
		nested, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("value of %s must be objet", name)
		}
		return sm.validateFields(ctx, schema.Fields, nested)
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
		return sm.validateFields(ctx, def.Fields, nested)

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
			if err := sm.validateField(ctx, itemName, item, def.Item); err != nil {
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
			if err := sm.validateField(ctx, keyName, val, def.Item); err != nil {
				return err
			}
		}
		return nil

	default:
		return fmt.Errorf("unsupported type %s for %s", def.Type, name)
	}
}
