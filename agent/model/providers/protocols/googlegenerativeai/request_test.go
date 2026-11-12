package googlegenerativeai

import (
	"encoding/json"
	"testing"

	"github.com/invopop/jsonschema"
	agentschema "github.com/vevila2020/writing_platform_ai/agent/schema"
)

func TestRequestToolsStripsUnsupportedSchemaAnnotations(t *testing.T) {
	var schema jsonschema.Schema
	raw := `{
		"type": "object",
		"properties": {
			"items": {
				"type": "array",
				"items": {
					"type": "object",
					"$comment": "agent:independent-batch-item",
					"properties": {"$comment": {"type": "string"}},
					"anyOf": [{"type": "string", "$comment": "nested"}]
				}
			}
		}
	}`
	if err := json.Unmarshal([]byte(raw), &schema); err != nil {
		t.Fatal(err)
	}
	tools, err := requestTools([]*agentschema.ToolInfo{{
		Name: "send", ParamsOneOf: agentschema.NewParamsOneOfByJSONSchema(&schema),
	}})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(tools)
	if err != nil {
		t.Fatal(err)
	}
	var wire []struct {
		FunctionDeclarations []struct {
			Parameters map[string]any `json:"parametersJsonSchema"`
		} `json:"functionDeclarations"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		t.Fatal(err)
	}
	if len(wire) != 1 || len(wire[0].FunctionDeclarations) != 1 {
		t.Fatalf("tools = %#v, want one declaration", wire)
	}
	properties := wire[0].FunctionDeclarations[0].Parameters["properties"].(map[string]any)
	array := properties["items"].(map[string]any)
	item := array["items"].(map[string]any)
	if _, exists := item["$comment"]; exists {
		t.Fatalf("item schema kept $comment: %#v", item)
	}
	if nested := item["anyOf"].([]any)[0].(map[string]any); nested["$comment"] != nil {
		t.Fatalf("nested anyOf schema kept $comment: %#v", nested)
	}
	itemProperties := item["properties"].(map[string]any)
	if _, exists := itemProperties["$comment"]; !exists {
		t.Fatalf("a property named $comment must not be removed: %#v", itemProperties)
	}
}
