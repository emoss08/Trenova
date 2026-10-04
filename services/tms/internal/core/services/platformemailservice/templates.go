package platformemailservice

import (
	"bytes"
	"embed"
	"fmt"
	htmltemplate "html/template"
	"strings"
	texttemplate "text/template"
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
)

func AllKinds() []Kind {
	return []Kind{
		KindSignupVerification,
		KindSignupExistingAccount,
		KindWelcome,
		KindTrialEnded,
		KindAccountPurged,
	}
}

type templateData struct {
	ProductName       string
	Subject           string
	FirstName         string
	CompanyName       string
	VerifyURL         string
	ExpiresAt         string
	LoginURL          string
	ForgotPasswordURL string
	AppURL            string
	TrialEndsAt       string
	ReadOnlyUntil     string
	SignupURL         string
}

type buttonData struct {
	URL   string
	Label string
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
	kinds map[Kind]kindTemplates
}

func newRenderer() (*renderer, error) {
	funcs := htmltemplate.FuncMap{
		"button": func(url, label string) buttonData { return buttonData{URL: url, Label: label} },
	}

	kinds := make(map[Kind]kindTemplates, len(AllKinds()))
	for _, kind := range AllKinds() {
		subject, err := texttemplate.New(string(kind)+".subject").
			Option("missingkey=error").
			ParseFS(templateFS, "templates/"+string(kind)+".subject")
		if err != nil {
			return nil, fmt.Errorf("parse %s subject: %w", kind, err)
		}

		text, err := texttemplate.New(string(kind)+".txt").
			Option("missingkey=error").
			ParseFS(templateFS, "templates/"+string(kind)+".txt")
		if err != nil {
			return nil, fmt.Errorf("parse %s text: %w", kind, err)
		}

		html, err := htmltemplate.New("layout.html").
			Funcs(funcs).
			Option("missingkey=error").
			ParseFS(
				templateFS,
				"templates/layout.html",
				"templates/button.html",
				"templates/"+string(kind)+".html",
			)
		if err != nil {
			return nil, fmt.Errorf("parse %s html: %w", kind, err)
		}

		kinds[kind] = kindTemplates{subject: subject, text: text, html: html}
	}

	return &renderer{kinds: kinds}, nil
}

func (r *renderer) render(kind Kind, data *templateData) (*renderedEmail, error) {
	tmpl, ok := r.kinds[kind]
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
