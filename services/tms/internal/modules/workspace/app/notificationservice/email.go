package notificationservice

import (
	"bytes"
	"html/template"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/shared/stringutils"
)

type notificationEmail struct {
	RecipientName string
	Title         string
	Message       string
	Link          string
	SettingsURL   string
}

type digestEmail struct {
	RecipientName string
	Items         []digestItem
	SettingsURL   string
	BellURL       string
}

type digestItem struct {
	Title   string
	Message string
	Link    string
}

type renderedEmail struct {
	Subject string
	HTML    string
	Text    string
}

var notificationEmailTemplate = template.Must(template.New("notification").Parse(`<!doctype html>
<html><body style="margin:0;padding:24px;background:#fafafa;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;color:#171717">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:560px;margin:0 auto;background:#ffffff;border:1px solid #ebebeb;border-radius:8px">
<tr><td style="padding:24px">
{{if .RecipientName}}<p style="margin:0 0 12px;font-size:14px;color:#525252">Hi {{.RecipientName}},</p>{{end}}
<h1 style="margin:0 0 8px;font-size:18px;font-weight:600">{{.Title}}</h1>
{{if .Message}}<p style="margin:0 0 20px;font-size:14px;line-height:1.5">{{.Message}}</p>{{end}}
{{if .Link}}<a href="{{.Link}}" style="display:inline-block;padding:8px 14px;border-radius:6px;background:#171717;color:#ffffff;font-size:14px;text-decoration:none">Open in Trenova</a>{{end}}
</td></tr>
<tr><td style="padding:16px 24px;border-top:1px solid #ebebeb;font-size:12px;color:#737373">
You get this email because of your notification settings in Trenova.{{if .SettingsURL}} <a href="{{.SettingsURL}}" style="color:#0068d6">Change what reaches you</a>.{{end}}
</td></tr>
</table>
</body></html>`))

var digestEmailTemplate = template.Must(template.New("digest").Parse(`<!doctype html>
<html><body style="margin:0;padding:24px;background:#fafafa;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;color:#171717">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:560px;margin:0 auto;background:#ffffff;border:1px solid #ebebeb;border-radius:8px">
<tr><td style="padding:24px 24px 8px">
{{if .RecipientName}}<p style="margin:0 0 12px;font-size:14px;color:#525252">Hi {{.RecipientName}},</p>{{end}}
<h1 style="margin:0;font-size:18px;font-weight:600">Your daily digest</h1>
</td></tr>
{{range .Items}}<tr><td style="padding:12px 24px;border-top:1px solid #ebebeb">
<p style="margin:0 0 4px;font-size:14px;font-weight:500">{{if .Link}}<a href="{{.Link}}" style="color:#171717;text-decoration:none">{{.Title}}</a>{{else}}{{.Title}}{{end}}</p>
{{if .Message}}<p style="margin:0;font-size:13px;line-height:1.5;color:#525252">{{.Message}}</p>{{end}}
</td></tr>{{end}}
<tr><td style="padding:16px 24px;border-top:1px solid #ebebeb;font-size:12px;color:#737373">
Urgent notifications still reach you right away.{{if .BellURL}} <a href="{{.BellURL}}" style="color:#0068d6">See everything in Trenova</a>.{{end}}{{if .SettingsURL}} <a href="{{.SettingsURL}}" style="color:#0068d6">Change your digest</a>.{{end}}
</td></tr>
</table>
</body></html>`))

func renderNotificationEmail(data *notificationEmail) (*renderedEmail, error) {
	data.RecipientName = stringutils.FirstName(data.RecipientName)

	var html bytes.Buffer
	if err := notificationEmailTemplate.Execute(&html, data); err != nil {
		return nil, err
	}

	var text strings.Builder
	if data.RecipientName != "" {
		text.WriteString("Hi " + data.RecipientName + ",\n\n")
	}
	text.WriteString(data.Title + "\n")
	if data.Message != "" {
		text.WriteString("\n" + data.Message + "\n")
	}
	if data.Link != "" {
		text.WriteString("\nOpen in Trenova: " + data.Link + "\n")
	}
	if data.SettingsURL != "" {
		text.WriteString("\nChange what reaches you: " + data.SettingsURL + "\n")
	}

	return &renderedEmail{
		Subject: stringutils.TruncateRunes(singleLine(data.Title), maxSubjectLength),
		HTML:    html.String(),
		Text:    text.String(),
	}, nil
}

func renderDigestEmail(data *digestEmail) (*renderedEmail, error) {
	data.RecipientName = stringutils.FirstName(data.RecipientName)

	var html bytes.Buffer
	if err := digestEmailTemplate.Execute(&html, data); err != nil {
		return nil, err
	}

	var text strings.Builder
	if data.RecipientName != "" {
		text.WriteString("Hi " + data.RecipientName + ",\n\n")
	}
	text.WriteString("Your daily digest\n")
	for _, item := range data.Items {
		text.WriteString("\n- " + item.Title)
		if item.Message != "" {
			text.WriteString("\n  " + item.Message)
		}
		if item.Link != "" {
			text.WriteString("\n  " + item.Link)
		}
		text.WriteString("\n")
	}
	text.WriteString("\nUrgent notifications still reach you right away.\n")
	if data.SettingsURL != "" {
		text.WriteString("Change your digest: " + data.SettingsURL + "\n")
	}

	subject := "Your Trenova digest: 1 notification"
	if len(data.Items) != 1 {
		subject = "Your Trenova digest: " + strconv.Itoa(len(data.Items)) + " notifications"
	}

	return &renderedEmail{Subject: subject, HTML: html.String(), Text: text.String()}, nil
}

const maxSubjectLength = 200

func singleLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
