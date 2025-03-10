package restfulcomponent

import (
	"fmt"
	"testing"
)

var restfulCreateTagsComponent = map[string]any{
	"_id":         "1",
	"name":        "Create Tags API",
	"api":         "localhost:8000/api/v0/tags",
	"method":      "POST",
	"reqSchema":   "application/json",
	"description": "Creates multiple tags",
	"reqParams": []any{
		map[string]any{
			"name":        "tags",
			"description": "List of tag names",
			"type":        "array",
			"kind":        "array",
			"required":    true,
			"nestedParams": []any{
				map[string]any{
					"name":        "tag",
					"description": "A single tag name",
					"type":        "string",
					"kind":        "simple",
				},
			},
		},
	},
	"resStatuses": []any{
		map[string]any{
			"code":        201,
			"description": "Tags created successfully",
			"schema":      "application/json",
			"params": []any{
				map[string]any{
					"name":        "ids",
					"description": "The IDs of the created tags",
					"type":        "array",
					"kind":        "array",
					"nestedParams": []any{
						map[string]any{
							"name":        "id",
							"description": "The ID of a tag",
							"type":        "string",
							"kind":        "simple",
						},
					},
				},
			},
		},
		map[string]any{
			"code":        400,
			"description": "Invalid input",
			"schema":      "application/json",
			"params": []any{
				map[string]any{
					"name":        "error",
					"description": "Error message",
					"type":        "string",
					"kind":        "simple",
				},
			},
		},
	},
}

var restfulCreateItemsComponent = map[string]any{
	"_id":         "2",
	"name":        "Create Items API",
	"api":         "localhost:8000/api/v0/items",
	"method":      "POST",
	"reqSchema":   "application/json",
	"description": "Creates multiple items",
	"reqParams": []any{
		map[string]any{
			"name":        "items",
			"description": "List of items to create",
			"type":        "array",
			"kind":        "array",
			"required":    true,
			"nestedParams": []any{
				map[string]any{
					"name":        "name",
					"description": "The name of the item",
					"type":        "string",
					"kind":        "simple",
					"required":    true,
				},
				map[string]any{
					"name":        "price",
					"description": "The price of the item",
					"type":        "float64",
					"kind":        "simple",
					"required":    true,
				},
			},
		},
	},
	"resStatuses": []any{
		map[string]any{
			"code":        201,
			"description": "Items created successfully",
			"schema":      "application/json",
			"params": []any{
				map[string]any{
					"name":        "ids",
					"description": "The IDs of the created items",
					"type":        "array",
					"kind":        "array",
					"nestedParams": []any{
						map[string]any{
							"name":        "id",
							"description": "The ID of an item",
							"type":        "string",
							"kind":        "simple",
						},
					},
				},
			},
		},
	},
}

var restfulInvalidMethodComponent = map[string]any{
	"_id":         "3",
	"name":        "Invalid Method API",
	"api":         "localhost:8000/api/v0/test",
	"method":      "INVALID",
	"reqSchema":   "application/json",
	"description": "Test invalid method",
}

var restfulInvalidKindComponent = map[string]any{
	"_id":         "4",
	"name":        "Invalid Kind API",
	"api":         "localhost:8000/api/v0/test",
	"method":      "POST",
	"reqSchema":   "application/json",
	"description": "Test invalid kind",
	"reqParams": []any{
		map[string]any{
			"name":        "data",
			"description": "Invalid kind parameter",
			"type":        "INVALID",
			"kind":        "INVALID",
		},
	},
}

var restfulDefaultComponent = map[string]any{
	"_id":         "5",
	"name":        "Default API",
	"api":         "localhost:8000/api/v0/default",
	"method":      "POST",
	"reqSchema":   "application/json",
	"description": "Test default values",
	"reqParams": []any{
		map[string]any{
			"name":        "data",
			"description": "Parameter with no type or kind",
		},
	},
}

var restfulCreateNodeComponent = map[string]any{
	"_id":         "6",
	"method":      "POST",
	"name":        "Register Node API",
	"description": "Create nodes",
	"api":         "http://127.0.0.1:8000/api/v0/node",
	"reqSchema":   "application/json",
	"reqParams": []any{
		map[string]any{
			"name":        "name",
			"description": "Node name",
			"type":        "string",
			"kind":        "simple",
			"required":    true,
		},
	},
	"resStatuses": []any{
		map[string]any{
			"code":        200,
			"description": "Node created successfully",
			"schema":      "application/json",
			"params": []any{
				map[string]any{
					"name":        "_id",
					"description": "Node id",
					"type":        "string",
				},
			},
		},
	},
}

func TestRestfulSchema(t *testing.T) {
	// test API having nested JSON
	userComponentSchema, err := NewRestfulComponent(restfulCreateTagsComponent)
	if err != nil {
		t.Fatalf("Error creating user component: %v\n\n\n", err)
		return
	}
	fmt.Printf("User Component: %+v\n\n\n", userComponentSchema)

	// test API having array
	itemsComponentSchema, err := NewRestfulComponent(restfulCreateItemsComponent)
	if err != nil {
		t.Fatalf("Error creating items component: %v\n\n\n", err)
		return
	}
	fmt.Printf("Items Component: %+v\n\n\n", itemsComponentSchema)

	// test invalid HttpMethod
	invalidMethodComponentSchema, err := NewRestfulComponent(restfulInvalidMethodComponent)
	if err != nil {
		fmt.Printf("Error creating invalid method component: %v\n\n\n", err)
	} else {
		t.Fatalf("Invalid Method Component: %+v\n\n\n", invalidMethodComponentSchema)
	}

	// test invalid ParamKind
	invalidKindComponentSchema, err := NewRestfulComponent(restfulInvalidKindComponent)
	if err != nil {
		fmt.Printf("Error creating invalid kind component: %v\n\n\n", err)
	} else {
		t.Fatalf("Invalid Kind Component: %+v\n\n\n", invalidKindComponentSchema)
	}

	// testing default value handling
	defaultComponentSchema, err := NewRestfulComponent(restfulDefaultComponent)
	if err != nil {
		t.Fatalf("Error creating default component: %v\n\n\n", err)
	} else {
		fmt.Printf("Default Component: %+v\n\n\n", defaultComponentSchema)
	}
}

func TestRestfulComponent(t *testing.T) {
	// check if component schema is valid
	nodeComponentSchema, err := NewRestfulComponent(restfulCreateNodeComponent)
	if err != nil {
		t.Fatalf("Error creating invalid kind component: %v\n\n\n", err)
	} else {
		fmt.Printf("%v", nodeComponentSchema)
	}

	// test create node component
	nodeComponnet, err := NewRestfulComponentInstance(restfulCreateNodeComponent)
	if err != nil {
		t.Fatalf("Error creating node component: %v\n\n\n", err)
	} else {
		fmt.Printf("Node Component: %+v\n\n\n", nodeComponnet)
	}
	testParams := map[string]any{
		"name": "Root Node",
	}
	result, err := nodeComponnet.Execute(nil, testParams, nil, nil)
	if err != nil {
		t.Fatalf("Error executing node component: %v\n\n\n", err)
	} else {
		fmt.Printf("Get Response: %+v\n", result)
	}
}
