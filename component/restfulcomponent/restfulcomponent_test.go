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

func TestRestfulComponent(t *testing.T) {
	// test API having nested JSON
	userComponent, err := NewRestfulComponent(restfulCreateTagsComponent)
	if err != nil {
		fmt.Println("Error creating user component:", err)
		return
	}
	fmt.Printf("User Component: %+v\n", userComponent)

	// test API having array
	itemsComponent, err := NewRestfulComponent(restfulCreateItemsComponent)
	if err != nil {
		fmt.Println("Error creating items component:", err)
		return
	}
	fmt.Printf("Items Component: %+v\n", itemsComponent)

	// test invalid HttpMethod
	invalidMethodComponent, err := NewRestfulComponent(restfulInvalidMethodComponent)
	if err != nil {
		fmt.Println("Error creating invalid method component:", err)
	} else {
		fmt.Printf("Invalid Method Component: %+v\n", invalidMethodComponent)
	}

	// test invalid ParamKind
	invalidKindComponent, err := NewRestfulComponent(restfulInvalidKindComponent)
	if err != nil {
		fmt.Println("Error creating invalid kind component:", err)
	} else {
		fmt.Printf("Invalid Kind Component: %+v\n", invalidKindComponent)
	}

	// testing default value handling
	defaultComponent, err := NewRestfulComponent(restfulDefaultComponent)
	if err != nil {
		fmt.Println("Error creating default component:", err)
	} else {
		fmt.Printf("Default Component: %+v\n", defaultComponent)
	}
}
