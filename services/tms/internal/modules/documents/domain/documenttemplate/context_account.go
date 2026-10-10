package documenttemplate

import "html/template"

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

func newPasswordResetSampleContext() any {
	return PasswordResetContext{
		FirstName:        "Dana",
		FullName:         "Dana Whitfield",
		CompanyName:      sampleCompanyName,
		ResetURL:         template.URL("https://app.example.com/auth/reset?token=8f2c41a9b7e34d05"),
		ExpiresInMinutes: 30,
		ExpiresAt:        "2026-07-22 11:00 CDT",
	}
}

func (r *Registry) registerAccountKinds() {
	_ = r.Register(&KindDefinition{
		Kind:        KindPasswordResetEmail,
		DisplayName: "Password Reset",
		Description: "Carries the single-use link that lets somebody who has lost their " +
			"password choose a new one. Not customer-scoped: the recipient is a user.",
		Category:      "Account",
		Channels:      []Channel{ChannelSubject, ChannelEmailHTML, ChannelEmailText},
		sampleFactory: newPasswordResetSampleContext,
		Variables: []VariableDefinition{
			{Path: "FirstName", Type: VariableString, Description: "The user's first name."},
			{Path: "FullName", Type: VariableString, Description: "The user's full name."},
			companyNameVariable(),
			{
				Path:     "ResetURL",
				Type:     VariableString,
				Required: true,
				Description: "The single-use reset link. Required: without it the message " +
					"has no purpose.",
			},
			{
				Path: "ExpiresInMinutes",
				Type: VariableInt,
				Description: "How many minutes the link is good for. Use this rather than " +
					"writing a number into the copy, or the wording will drift from the " +
					"real expiry.",
			},
			{
				Path:        "ExpiresAt",
				Type:        VariableDateTime,
				Description: "The exact moment the link stops working.",
			},
		},
	})
}
