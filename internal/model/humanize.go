package model

import "unicode"

// HumanizeIdentifier turns identifiers like SharedTodo or shared_todo into
// human-friendly labels such as "Shared Todo" and "Shared Todo".
func HumanizeIdentifier(value string) string {
	if value == "" {
		return ""
	}

	runes := []rune(value)
	out := make([]rune, 0, len(runes)+4)

	for i, r := range runes {
		if r == '_' || r == '-' {
			if len(out) > 0 && out[len(out)-1] != ' ' {
				out = append(out, ' ')
			}
			continue
		}

		if i > 0 {
			prev := runes[i-1]
			var next rune
			hasNext := i+1 < len(runes)
			if hasNext {
				next = runes[i+1]
			}

			shouldSeparate :=
				(unicode.IsLower(prev) && unicode.IsUpper(r)) ||
					(unicode.IsLetter(prev) && unicode.IsDigit(r)) ||
					(unicode.IsDigit(prev) && unicode.IsLetter(r)) ||
					(unicode.IsUpper(prev) && unicode.IsUpper(r) && hasNext && unicode.IsLower(next))

			if shouldSeparate && len(out) > 0 && out[len(out)-1] != ' ' {
				out = append(out, ' ')
			}
		}

		out = append(out, r)
	}

	return string(out)
}
