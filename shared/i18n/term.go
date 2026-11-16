package i18n

type Term string

func (t Term) Localize(locale Locale) string {
	return Translate(locale, string(t))
}

func (t Term) String() string {
	return string(t)
}
