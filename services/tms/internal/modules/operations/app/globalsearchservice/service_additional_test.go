package globalsearchservice

import "testing"

func TestStringValueDecodesByteEncodedStrings(t *testing.T) {
	t.Parallel()

	document := map[string]any{
		"id":         []byte(`"shp_123"`),
		"pro_number": []byte(`"S26030001580609"`),
	}

	if got := stringValue(document, "id"); got != "shp_123" {
		t.Fatalf("expected decoded id, got %q", got)
	}

	if got := stringValue(document, "pro_number"); got != "S26030001580609" {
		t.Fatalf("expected decoded pro number, got %q", got)
	}
}
