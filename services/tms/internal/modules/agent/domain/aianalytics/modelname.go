package aianalytics

import (
	"regexp"
	"strings"
)

var (
	regionPrefix = regexp.MustCompile(`^(us|eu|apac|global|jp|au)\.`)
	vendorPrefix = regexp.MustCompile(
		`^(anthropic|meta|mistral|amazon|cohere|ai21|deepseek|qwen|openai|google)\.`,
	)
	versionSuffix = regexp.MustCompile(`-v\d+([.-]\d+)*$`)
	dateSuffix    = regexp.MustCompile(`-(\d{8}|\d{4}-\d{2}-\d{2}|\d{4})$`)
	variantSuffix = regexp.MustCompile(
		`-(instruct|versatile|it|latest|hf|fp8|fp16|awq|gptq|int4|int8|q\d(_k(_[ms])?)?|q8_0)$`,
	)
	familyDigits   = regexp.MustCompile(`^([a-z]{3,})(\d)`)
	splitVersion   = regexp.MustCompile(`^([a-z]+)-(\d+)-(\d+)-(\d+(\.\d+)?[bm])`)
	parameterSize  = regexp.MustCompile(`^\d+(\.\d+)?[bm]$`)
	expertSize     = regexp.MustCompile(`^\d+x\d+(\.\d+)?b$`)
	versionToken   = regexp.MustCompile(`^\d+(\.\d+)*$`)
	claudeTiers    = map[string]string{"opus": "Opus", "sonnet": "Sonnet", "haiku": "Haiku"}
	geminiVariants = map[string]string{
		"flash": "Flash",
		"pro":   "Pro",
		"lite":  "Lite",
		"ultra": "Ultra",
		"nano":  "Nano",
	}
)

func CanonicalModel(raw string) string {
	model := strings.ToLower(strings.TrimSpace(raw))
	if model == "" {
		return ""
	}
	if slash := strings.LastIndexByte(model, '/'); slash >= 0 && slash < len(model)-1 {
		model = model[slash+1:]
	}
	model = regionPrefix.ReplaceAllString(model, "")
	model = vendorPrefix.ReplaceAllString(model, "")
	model = strings.TrimSuffix(model, ":latest")
	model = strings.NewReplacer(":", "-", "_", "-", " ", "-").Replace(model)
	model = strings.ReplaceAll(model, "q4-k-m", "q4_k_m")
	for {
		trimmed := versionSuffix.ReplaceAllString(model, "")
		trimmed = dateSuffix.ReplaceAllString(trimmed, "")
		trimmed = variantSuffix.ReplaceAllString(trimmed, "")
		if trimmed == model || trimmed == "" {
			break
		}
		model = trimmed
	}
	model = familyDigits.ReplaceAllString(model, "$1-$2")
	model = splitVersion.ReplaceAllString(model, "$1-$2.$3-$4")

	return model
}

type ModelName struct {
	Label string
	Short string
}

func NameModel(key string) ModelName {
	parts := strings.Split(key, "-")
	if len(parts) == 0 || key == "" {
		return ModelName{Label: key, Short: key}
	}
	switch parts[0] {
	case "claude":
		if name, ok := claudeName(parts[1:]); ok {
			return name
		}
	case "gpt":
		if name, ok := gptName(parts[1:]); ok {
			return name
		}
	case "gemini":
		if name, ok := geminiName(parts[1:]); ok {
			return name
		}
	}
	if len(parts[0]) >= 2 && parts[0][0] == 'o' && versionToken.MatchString(parts[0][1:]) {
		label := strings.Join(parts, " ")
		return ModelName{Label: label, Short: label}
	}

	return genericName(parts)
}

func claudeName(rest []string) (ModelName, bool) {
	var tier string
	versions := make([]string, 0, 2)
	tierFirst := false
	for index, part := range rest {
		if name, ok := claudeTiers[part]; ok {
			tier = name
			tierFirst = index == 0
			continue
		}
		if versionToken.MatchString(part) {
			versions = append(versions, part)
		}
	}
	if tier == "" {
		return ModelName{}, false
	}
	version := strings.Join(versions, ".")
	short := strings.TrimSpace(tier + " " + version)
	if tierFirst || version == "" {
		return ModelName{
			Label: strings.TrimSpace("Claude " + tier + " " + version),
			Short: short,
		}, true
	}

	return ModelName{Label: "Claude " + version + " " + tier, Short: short}, true
}

func gptName(rest []string) (ModelName, bool) {
	if len(rest) == 0 {
		return ModelName{}, false
	}
	version := rest[0]
	variant := strings.Join(rest[1:], " ")
	label := "GPT-" + version
	if variant == "" {
		return ModelName{Label: label, Short: label}, true
	}

	return ModelName{Label: label + " " + variant, Short: version + " " + variant}, true
}

func geminiName(rest []string) (ModelName, bool) {
	words := make([]string, 0, len(rest))
	variants := make([]string, 0, 2)
	for _, part := range rest {
		if name, ok := geminiVariants[part]; ok {
			words = append(words, name)
			variants = append(variants, name)
			continue
		}
		if versionToken.MatchString(part) {
			words = append(words, part)
			continue
		}
		words = append(words, titleWord(part))
	}
	if len(words) == 0 {
		return ModelName{}, false
	}
	label := "Gemini " + strings.Join(words, " ")
	if len(variants) == 0 {
		return ModelName{Label: label, Short: label}, true
	}

	return ModelName{Label: label, Short: "Gem. " + strings.Join(variants, " ")}, true
}

func genericName(parts []string) ModelName {
	words := make([]string, 0, len(parts))
	family := titleWord(parts[0])
	size := ""
	for index, part := range parts {
		switch {
		case parameterSize.MatchString(part) || expertSize.MatchString(part):
			word := strings.ToUpper(part[len(part)-1:])
			word = part[:len(part)-1] + word
			words = append(words, word)
			size = word
		case index == 0:
			words = append(words, family)
		case versionToken.MatchString(part):
			words = append(words, part)
		default:
			words = append(words, titleWord(part))
		}
	}
	label := strings.Join(words, " ")
	if size != "" {
		return ModelName{Label: label, Short: family + " " + size}
	}

	return ModelName{Label: label, Short: label}
}

func titleWord(word string) string {
	if word == "" {
		return word
	}

	return strings.ToUpper(word[:1]) + word[1:]
}
