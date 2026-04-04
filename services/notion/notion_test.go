package notion

import (
	"encoding/json"
	"testing"
)

func TestMarshalBlogPostsQuery_includesActiveCheckbox(t *testing.T) {
	raw, err := marshalBlogPostsQuery("travel", true)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	f, ok := decoded["filter"].(map[string]any)
	if !ok {
		t.Fatalf("filter: %v", decoded["filter"])
	}
	andRaw, ok := f["and"].([]any)
	if !ok || len(andRaw) != 2 {
		t.Fatalf("expected and with 2 clauses, got %v", f["and"])
	}
	box, ok := andRaw[1].(map[string]any)
	if !ok {
		t.Fatal("second clause not object")
	}
	if box["property"] != notionPublishedCheckboxProperty {
		t.Fatalf("property %q", box["property"])
	}
	ch, ok := box["checkbox"].(map[string]any)
	if !ok || ch["equals"] != true {
		t.Fatalf("checkbox equals true: %v", box["checkbox"])
	}
}
