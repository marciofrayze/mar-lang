package expr

import "testing"

func TestParseEvalElmStyleFunctions(t *testing.T) {
	ctx := map[string]any{
		"title":     "Todo",
		"email":     "dev@company.com",
		"user_role": "admin",
	}

	tests := []struct {
		expression string
		want       any
	}{
		{expression: `length title >= 3`, want: true},
		{expression: `contains "@" email`, want: true},
		{expression: `starts_with "dev@" email`, want: true},
		{expression: `ends_with "@company.com" email`, want: true},
		{expression: `matches "^[^@]+@company\\.com$" email`, want: true},
		{expression: `user_role == "admin"`, want: true},
	}

	opts := ParserOptions{AllowedVariables: map[string]struct{}{
		"title":     {},
		"email":     {},
		"user_role": {},
	}}

	for _, tc := range tests {
		node, err := Parse(tc.expression, opts)
		if err != nil {
			t.Fatalf("Parse(%q) returned error: %v", tc.expression, err)
		}
		got, err := node.Eval(ctx)
		if err != nil {
			t.Fatalf("Eval(%q) returned error: %v", tc.expression, err)
		}
		if got != tc.want {
			t.Fatalf("Eval(%q) = %#v, want %#v", tc.expression, got, tc.want)
		}
	}
}

func TestParseRejectsLegacyFunctionSyntax(t *testing.T) {
	opts := ParserOptions{AllowedVariables: map[string]struct{}{"title": {}, "email": {}}}
	tests := []string{
		`len(title) >= 3`,
		`contains(email, "@")`,
		`startsWith(email, "dev@")`,
		`endsWith(email, "@company.com")`,
		`matches(email, "^[^@]+$")`,
	}

	for _, expression := range tests {
		if _, err := Parse(expression, opts); err == nil {
			t.Fatalf("expected Parse(%q) to fail", expression)
		}
	}
}
