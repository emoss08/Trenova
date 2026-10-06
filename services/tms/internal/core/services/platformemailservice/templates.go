package platformemailservice

import (
	"bytes"
	"embed"
	"fmt"
	htmltemplate "html/template"
	"strings"
	texttemplate "text/template"

	"github.com/emoss08/trenova/shared/i18n"
)

//go:embed templates/*
var templateFS embed.FS

type Kind string

const (
	KindSignupVerification    Kind = "signup_verification"
	KindSignupExistingAccount Kind = "signup_existing_account"
	KindWelcome               Kind = "welcome"
	KindTrialEnded            Kind = "trial_ended"
	KindAccountPurged         Kind = "account_purged"
	KindPasswordReset         Kind = "password_reset"
)

func AllKinds() []Kind {
	return []Kind{
		KindSignupVerification,
		KindSignupExistingAccount,
		KindWelcome,
		KindTrialEnded,
		KindAccountPurged,
		KindPasswordReset,
	}
}

type templateData struct {
	Locale            i18n.Locale
	ProductName       string
	Eyebrow           string
	LogoURL           string
	HomeURL           string
	HomeLabel         string
	Subject           string
	FirstName         string
	CompanyName       string
	VerifyURL         string
	ExpiresAt         string
	ExpiresInMinutes  int
	LoginURL          string
	ForgotPasswordURL string
	AppURL            string
	TrialEndsAt       string
	ReadOnlyUntil     string
	SignupURL         string
	ResetURL          string
}

type buttonData struct {
	URL   string
	Label string
}

type stepItem struct {
	Number string
	Label  string
	Last   bool
}

type stepsData struct {
	Label string
	Intro string
	Items []stepItem
}

type calloutData struct {
	Label string
	Note  string
	Lines []string
}

type renderedEmail struct {
	Subject string
	HTML    string
	Text    string
}

type kindTemplates struct {
	subject *texttemplate.Template
	text    *texttemplate.Template
	html    *htmltemplate.Template
}

type renderer struct {
	locales map[i18n.Locale]map[Kind]kindTemplates
}

func newRenderer() (*renderer, error) {
	supported := i18n.Supported()
	locales := make(map[i18n.Locale]map[Kind]kindTemplates, len(supported))

	for _, locale := range supported {
		kinds, err := parseLocale(locale)
		if err != nil {
			return nil, err
		}
		locales[locale] = kinds
	}

	return &renderer{locales: locales}, nil
}

func parseLocale(locale i18n.Locale) (map[Kind]kindTemplates, error) {
	translate := func(message string, args ...any) string {
		return i18n.Translate(locale, message, args...)
	}

	textFuncs := texttemplate.FuncMap{"t": translate}
	htmlFuncs := htmltemplate.FuncMap{
		"t":       translate,
		"button":  func(url, label string) buttonData { return buttonData{URL: url, Label: label} },
		"steps":   newSteps,
		"callout": newCallout,
	}

	kinds := make(map[Kind]kindTemplates, len(AllKinds()))
	for _, kind := range AllKinds() {
		subject, err := texttemplate.New(string(kind)+".subject").
			Funcs(textFuncs).
			Option("missingkey=error").
			ParseFS(templateFS, "templates/"+string(kind)+".subject")
		if err != nil {
			return nil, fmt.Errorf("parse %s subject (%s): %w", kind, locale, err)
		}

		text, err := texttemplate.New(string(kind)+".txt").
			Funcs(textFuncs).
			Option("missingkey=error").
			ParseFS(templateFS, "templates/"+string(kind)+".txt")
		if err != nil {
			return nil, fmt.Errorf("parse %s text (%s): %w", kind, locale, err)
		}

		html, err := htmltemplate.New("layout.html").
			Funcs(htmlFuncs).
			Option("missingkey=error").
			ParseFS(
				templateFS,
				"templates/layout.html",
				"templates/components.html",
				"templates/"+string(kind)+".html",
			)
		if err != nil {
			return nil, fmt.Errorf("parse %s html (%s): %w", kind, locale, err)
		}

		kinds[kind] = kindTemplates{subject: subject, text: text, html: html}
	}

	return kinds, nil
}

func (r *renderer) render(kind Kind, data *templateData) (*renderedEmail, error) {
	data.Locale = resolveLocale(data.Locale)

	tmpl, ok := r.locales[data.Locale][kind]
	if !ok {
		return nil, fmt.Errorf("unknown platform email kind %q", kind)
	}

	var buf bytes.Buffer
	if err := tmpl.subject.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("render %s subject: %w", kind, err)
	}
	subject := strings.Join(strings.Fields(buf.String()), " ")
	data.Subject = subject

	buf.Reset()
	if err := tmpl.text.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("render %s text: %w", kind, err)
	}
	text := strings.TrimSpace(buf.String()) + "\n"

	buf.Reset()
	if err := tmpl.html.ExecuteTemplate(&buf, "layout", data); err != nil {
		return nil, fmt.Errorf("render %s html: %w", kind, err)
	}

	return &renderedEmail{Subject: subject, HTML: buf.String(), Text: text}, nil
}

func resolveLocale(locale i18n.Locale) i18n.Locale {
	parsed, _ := i18n.Parse(string(locale))
	return parsed
}

func newSteps(label, intro string, labels ...string) stepsData {
	items := make([]stepItem, len(labels))
	for i, itemLabel := range labels {
		items[i] = stepItem{
			Number: fmt.Sprintf("%02d", i+1),
			Label:  itemLabel,
			Last:   i == len(labels)-1,
		}
	}

	return stepsData{Label: label, Intro: intro, Items: items}
}

func newCallout(label, note string, lines ...string) calloutData {
	return calloutData{Label: label, Note: note, Lines: lines}
}
