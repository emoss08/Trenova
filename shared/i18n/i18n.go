package i18n

import "context"

func Translate(locale Locale, message string, args ...any) string {
	if message == "" {
		return ""
	}

	if !locale.IsValid() {
		locale = Default
	}

	return format(locale, lookup(locale, message), args)
}

func lookup(locale Locale, message string) string {
	if locale == Default {
		return message
	}
	if messages := catalog(locale); messages != nil {
		if found, ok := messages[message]; ok && found != "" {
			return found
		}
	}

	return message
}

func Format(message string, args ...any) string {
	return format(Default, message, args)
}

func T(ctx context.Context, message string, args ...any) string {
	return Translate(FromContext(ctx), message, args...)
}

func Has(locale Locale, message string) bool {
	messages := catalog(locale)
	if messages == nil {
		return false
	}
	_, ok := messages[message]
	return ok
}
