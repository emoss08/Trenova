package mailtemplate

import (
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templatecatalog"
)

const mailCategory = "Mail"

type TriageReplyEmailContext struct {
	Subject     string
	Body        string
	CompanyName string
}

type MailboxTestEmailContext struct {
	MailboxAddress string
	Token          string
}

type EmailProfileTestEmailContext struct {
	ProfileName  string
	ProviderName string
	SenderEmail  string
}

func newTriageReplySampleContext() any {
	return TriageReplyEmailContext{
		Subject: "Re: Appointment for load " + templatecatalog.SampleProNumber,
		Body: "Thanks for the heads-up. We have moved the delivery to Thursday at 9 AM " +
			"and told the driver.\n\nThe updated rate confirmation is attached.",
		CompanyName: templatecatalog.SampleCompanyName,
	}
}

//nolint:gosec // G101: a sample test code, not a credential
func newMailboxTestSampleContext() any {
	return MailboxTestEmailContext{
		MailboxAddress: "dispatch@example.com",
		Token:          "trv-test-4f1c9a2b7d3e8a10",
	}
}

func newEmailProfileTestSampleContext() any {
	return EmailProfileTestEmailContext{
		ProfileName:  "Billing",
		ProviderName: "Postmark",
		SenderEmail:  "billing@example.com",
	}
}

func channels() []documenttemplate.Channel {
	return []documenttemplate.Channel{
		documenttemplate.ChannelSubject,
		documenttemplate.ChannelEmailHTML,
		documenttemplate.ChannelEmailText,
	}
}

func RegisterMailKinds(r *templatecatalog.Registry) {
	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:          documenttemplate.KindTriageReplyEmail,
		DisplayName:   "Triage Reply",
		Description:   "Carries a reply someone wrote in Triage back to the sender of the message it answers, in the same thread.",
		Category:      mailCategory,
		Channels:      channels(),
		SampleFactory: newTriageReplySampleContext,
		Variables: []templatecatalog.VariableDefinition{
			{
				Path:        "Subject",
				Type:        templatecatalog.VariableString,
				Required:    true,
				Description: "The reply's subject, which keeps the thread together. Use it as the subject unchanged.",
			},
			{
				Path:        "Body",
				Type:        templatecatalog.VariableString,
				Required:    true,
				Description: "What the person wrote. Print it whole: it is the reply.",
			},
			templatecatalog.CompanyNameVariable(),
		},
	})

	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:          documenttemplate.KindMailboxTestEmail,
		DisplayName:   "Mailbox Test",
		Description:   "Checks that a monitored address receives mail. It goes to the address itself and lands in Triage.",
		Category:      mailCategory,
		Channels:      channels(),
		SampleFactory: newMailboxTestSampleContext,
		Variables: []templatecatalog.VariableDefinition{
			{
				Path:        "MailboxAddress",
				Type:        templatecatalog.VariableString,
				Description: "The address being tested.",
			},
			{
				Path:        "Token",
				Type:        templatecatalog.VariableString,
				Required:    true,
				Description: "The code that tells the test apart from other mail. Keep it in the subject: the arriving message is matched by it.",
			},
		},
	})

	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:          documenttemplate.KindEmailProfileTestEmail,
		DisplayName:   "Email Profile Test",
		Description:   "Checks that an email profile can send, from the email settings.",
		Category:      mailCategory,
		Channels:      channels(),
		SampleFactory: newEmailProfileTestSampleContext,
		Variables: []templatecatalog.VariableDefinition{
			{
				Path:        "ProfileName",
				Type:        templatecatalog.VariableString,
				Required:    true,
				Description: "The name of the email profile being tested. Required: the test is about it.",
			},
			{
				Path:        "ProviderName",
				Type:        templatecatalog.VariableString,
				Description: "The email provider the profile sends through.",
			},
			{
				Path:        "SenderEmail",
				Type:        templatecatalog.VariableString,
				Description: "The address the test was sent from.",
			},
		},
	})
}
