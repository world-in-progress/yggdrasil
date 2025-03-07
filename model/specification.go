package model

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
)
