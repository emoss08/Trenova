package emaildesign

import (
	"html/template"
	"sort"

	"github.com/emoss08/trenova/shared/i18n"
)

func Funcs(locale i18n.Locale) template.FuncMap {
	return template.FuncMap{
		"preheader":  preheader,
		"kicker":     kicker,
		"heading":    headingFor(locale),
		"para":       para,
		"section":    section,
		"facts":      facts,
		"fact":       fact,
		"steps":      steps,
		"tableOpen":  tableOpen,
		"tableClose": tableClose,
		"row":        row,
		"totalRow":   totalRow,
		"col":        col,
		"numCol":     numCol,
		"cell":       cell,
		"num":        num,
		"label":      label,
		"step":       step,
		"wide":       wide,
		"item":       item,
		"quote":      quote,
		"code":       code,
		"notice":     notice,
		"link":       link,
		"action":     action,
		"signoff":    signoff,
		"footer":     footer,
	}
}

func TextFuncs(locale i18n.Locale) map[string]any {
	return map[string]any{"heading": plainHeadingFor(locale)}
}

func Names() []string {
	fm := Funcs(i18n.Default)
	out := make([]string, 0, len(fm))
	for name := range fm {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
