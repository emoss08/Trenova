package i18n

import (
	"sort"
	"strconv"
	"strings"
)

type Locale string

const (
	EN   Locale = "en"
	ES   Locale = "es"
	ZhTW Locale = "zh-TW"
	ZhCN Locale = "zh-CN"
)

const Default = EN

var supported = []Locale{EN, ES, ZhTW, ZhCN}

var canonical = map[string]Locale{
	"en":      EN,
	"es":      ES,
	"zh-tw":   ZhTW,
	"zh-cn":   ZhCN,
	"zh-hant": ZhTW,
	"zh-hans": ZhCN,
	"zh-hk":   ZhTW,
	"zh-mo":   ZhTW,
	"zh-sg":   ZhCN,
}

func Supported() []Locale {
	out := make([]Locale, len(supported))
	copy(out, supported)
	return out
}

func (l Locale) String() string { return string(l) }

func (l Locale) IsValid() bool {
	for _, s := range supported {
		if s == l {
			return true
		}
	}
	return false
}

func Parse(tag string) (Locale, bool) {
	trimmed := strings.TrimSpace(tag)
	if trimmed == "" {
		return Default, false
	}

	normalized := strings.ToLower(strings.ReplaceAll(trimmed, "_", "-"))
	if locale, ok := canonical[normalized]; ok {
		return locale, true
	}

	base, _, _ := strings.Cut(normalized, "-")
	switch base {
	case "en":
		return EN, true
	case "es":
		return ES, true
	case "zh":
		return ZhCN, true
	}

	return Default, false
}

type languageRange struct {
	tag     string
	quality float64
	order   int
}

func ParseAcceptLanguage(header string) Locale {
	if strings.TrimSpace(header) == "" {
		return Default
	}

	ranges := make([]languageRange, 0, 8)
	for i, part := range strings.Split(header, ",") {
		tag, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}

		quality := 1.0
		if params != "" {
			if key, value, found := strings.Cut(params, "="); found &&
				strings.EqualFold(strings.TrimSpace(key), "q") {
				parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
				if err == nil {
					quality = parsed
				}
			}
		}
		if quality <= 0 {
			continue
		}

		ranges = append(ranges, languageRange{tag: tag, quality: quality, order: i})
	}

	sort.SliceStable(ranges, func(i, j int) bool {
		if ranges[i].quality != ranges[j].quality {
			return ranges[i].quality > ranges[j].quality
		}
		return ranges[i].order < ranges[j].order
	})

	for _, candidate := range ranges {
		if candidate.tag == "*" {
			return Default
		}
		if locale, ok := Parse(candidate.tag); ok {
			return locale
		}
	}

	return Default
}
