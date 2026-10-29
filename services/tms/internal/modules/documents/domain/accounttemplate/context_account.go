package accounttemplate

import (
	"html/template"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templatecatalog"
)

// PasswordResetContext is the data a password reset email renders against.
//
// Deliberately thin. This message goes to somebody who may not be the account owner
// — anybody can trigger it by typing an address — so it carries nothing about the
// account beyond a name the recipient already knows, and never states whether the
// address belongs to a real user in some other organization.
type PasswordResetContext struct {
	FirstName string
	FullName  string

	CompanyName string

	// ResetURL is the single-use link.
	//
	// Typed template.URL because the system builds it from the configured base URL
	// and a generated token, so it is ours and cannot carry a hostile scheme. Every
	// other field here is text and is escaped normally.
	ResetURL template.URL

	// ExpiresInMinutes is a real field rather than a number written into the copy, so
	// the wording and the actual expiry cannot drift apart.
	ExpiresInMinutes int
	ExpiresAt        string
}

type AccountInviteContext struct {
	FirstName     string
	FullName      string
	Username      string
	EmailAddress  string
	InvitedByName string

	CompanyName string

	SetupURL template.URL

	ExpiresInDays  int
	ExpiresInHours int
	ExpiresAt      string
}

func newAccountInviteSampleContext() any {
	return AccountInviteContext{
		FirstName:     sampleUserFirstName,
		FullName:      sampleUserFullName,
		Username:      "dwhitfield",
		EmailAddress:  "dana.whitfield@example.com",
		InvitedByName: "Morgan Ellis",
		CompanyName:   templatecatalog.SampleCompanyName,
		SetupURL: template.URL(
			"https://app.example.com/auth/reset?purpose=invite&token=sample-invite-link",
		),
		ExpiresInDays:  7,
		ExpiresInHours: 168,
		ExpiresAt:      "2026-07-29 11:00 CDT",
	}
}

func newPasswordResetSampleContext() any {
	return PasswordResetContext{
		FirstName:        sampleUserFirstName,
		FullName:         sampleUserFullName,
		CompanyName:      templatecatalog.SampleCompanyName,
		ResetURL:         template.URL("https://app.example.com/auth/reset?token=8f2c41a9b7e34d05"),
		ExpiresInMinutes: 30,
		ExpiresAt:        "2026-07-22 11:00 CDT",
	}
}

func RegisterAccountKinds(r *templatecatalog.Registry) {
	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:        documenttemplate.KindPasswordResetEmail,
		DisplayName: "Password Reset",
		Description: "Carries the single-use link that lets somebody who has lost their " +
			"password choose a new one. Not customer-scoped: the recipient is a user.",
		Category: "Account",
		Channels: []documenttemplate.Channel{
			documenttemplate.ChannelSubject,
			documenttemplate.ChannelEmailHTML,
			documenttemplate.ChannelEmailText,
		},
		SampleFactory: newPasswordResetSampleContext,
		Variables: []templatecatalog.VariableDefinition{
			{
				Path:        "FirstName",
				Type:        templatecatalog.VariableString,
				Description: "The user's first name.",
			},
			{
				Path:        "FullName",
				Type:        templatecatalog.VariableString,
				Description: "The user's full name.",
			},
			templatecatalog.CompanyNameVariable(),
			{
				Path:     "ResetURL",
				Type:     templatecatalog.VariableString,
				Required: true,
				Description: "The single-use reset link. Required: without it the message " +
					"has no purpose.",
			},
			{
				Path: "ExpiresInMinutes",
				Type: templatecatalog.VariableInt,
				Description: "How many minutes the link is good for. Use this rather than " +
					"writing a number into the copy, or the wording will drift from the " +
					"real expiry.",
			},
			{
				Path:        "ExpiresAt",
				Type:        templatecatalog.VariableDateTime,
				Description: "The exact moment the link stops working.",
			},
		},
	})

	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:        documenttemplate.KindAccountInviteEmail,
		DisplayName: "Account Invitation",
		Description: "Carries the single-use link a new user follows to choose their first " +
			"password and finish setting up the account an administrator created for them. " +
			"Not customer-scoped: the recipient is a user.",
		Category: "Account",
		Channels: []documenttemplate.Channel{
			documenttemplate.ChannelSubject,
			documenttemplate.ChannelEmailHTML,
			documenttemplate.ChannelEmailText,
		},
		SampleFactory: newAccountInviteSampleContext,
		Variables: []templatecatalog.VariableDefinition{
			{
				Path:        "FirstName",
				Type:        templatecatalog.VariableString,
				Description: "The new user's first name.",
			},
			{
				Path:        "FullName",
				Type:        templatecatalog.VariableString,
				Description: "The new user's full name.",
			},
			{
				Path:        "Username",
				Type:        templatecatalog.VariableString,
				Description: "The username the account was created with.",
			},
			{
				Path:        "EmailAddress",
				Type:        templatecatalog.VariableString,
				Description: "The address the user signs in with, which is where this message is sent.",
			},
			{
				Path:        "InvitedByName",
				Type:        templatecatalog.VariableString,
				Description: "The name of the administrator who created the account. Empty when it is not known.",
			},
			templatecatalog.CompanyNameVariable(),
			{
				Path:     "SetupURL",
				Type:     templatecatalog.VariableString,
				Required: true,
				Description: "The single-use link that sets the first password. Required: without it " +
					"the message has no purpose.",
			},
			{
				Path: templatecatalog.ExpiresInDaysPath,
				Type: templatecatalog.VariableInt,
				Description: "How many whole days the link is good for, or zero when the expiry is " +
					"not a whole number of days. Use this rather than writing a number into the copy.",
			},
			{
				Path:        "ExpiresInHours",
				Type:        templatecatalog.VariableInt,
				Description: "How many hours the link is good for.",
			},
			{
				Path:        "ExpiresAt",
				Type:        templatecatalog.VariableDateTime,
				Description: "The exact moment the link stops working.",
			},
		},
	})
}

const (
	sampleUserFirstName = "Dana"
	sampleUserFullName  = "Dana Whitfield"
)
