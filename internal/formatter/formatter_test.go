package formatter

import (
	"strings"
	"testing"
)

func TestFormatIsIdempotent(t *testing.T) {
	src := `
app   TodoApi
entity Todo{
title:String
done:Bool optional
}
`

	once, err := Format(src)
	if err != nil {
		t.Fatalf("first format failed: %v", err)
	}
	twice, err := Format(once)
	if err != nil {
		t.Fatalf("second format failed: %v", err)
	}

	if once != twice {
		t.Fatalf("formatter is not idempotent\n--- once ---\n%s\n--- twice ---\n%s", once, twice)
	}
}

func TestFormatCanonicalOutput(t *testing.T) {
	src := `
app TodoApi
entity Todo{
-- user-facing title
title:String
done:Bool optional
}
`

	formatted, err := Format(src)
	if err != nil {
		t.Fatalf("format failed: %v", err)
	}

	expected := "" +
		"app TodoApi\n" +
		"entity Todo {\n" +
		"  -- user-facing title\n" +
		"  title: String\n" +
		"  done: Bool optional\n" +
		"}\n"

	if formatted != expected {
		t.Fatalf("unexpected formatted output\n--- expected ---\n%s\n--- got ---\n%s", expected, formatted)
	}
}

func TestFormatInvalidSourceReturnsParserError(t *testing.T) {
	src := `
app Broken
entity Todo {
  title String
}
`

	_, err := Format(src)
	if err == nil {
		t.Fatal("expected format to fail for invalid source")
	}
	if !strings.Contains(err.Error(), "invalid entity statement") {
		t.Fatalf("unexpected error: %v", err)
	}
}
