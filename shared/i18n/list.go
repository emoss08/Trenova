package i18n

import "context"

// List joins items the way a sentence in the request's language lists them:
// "code, name and address". Each language's joins are translations of the
// two patterns, so adding a language needs no code.
func List(ctx context.Context, items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	head := items[0]
	for _, item := range items[1 : len(items)-1] {
		head = T(ctx, "{0}, {1}", head, item)
	}

	return T(ctx, "{0} and {1}", head, items[len(items)-1])
}
