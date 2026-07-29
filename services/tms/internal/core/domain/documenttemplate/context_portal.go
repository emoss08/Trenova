package documenttemplate

import "html/template"

// DriverPortalInvitationContext is the data a portal invitation renders against.
type DriverPortalInvitationContext struct {
	FirstName string
	LastName  string
	FullName  string

	CompanyName string

	// InviteURL is the single-use link.
	//
	// Typed template.URL because this system builds it from the configured portal
	// base URL and a generated token, so it is ours and cannot carry a hostile
	// scheme. Everything else on this context is text and is escaped normally —
	// including the driver's name, which the previous hand-rolled escaper handled
	// but whose sibling inviteURL it interpolated raw.
	InviteURL template.URL

	// ExpiresInDays was hardcoded in the old copy. It is a real field now, so the
	// wording and the actual expiry cannot drift apart.
	ExpiresInDays int
	ExpiresAt     string
}

// AgentEmailContext wraps prose an autonomous agent composed.
//
// The agent supplies the subject and body; the organization's template supplies the
// letterhead and signature around them. That inverts the previous arrangement, where
// the agent's text was the entire message and an organization had no say in how it
// looked going out under their name.
type AgentEmailContext struct {
	AgentSubject string
	AgentBody    string

	CompanyName  string
	CustomerName string

	ShipmentProNumber string

	// RequestedDocuments is what the agent is asking for, so a template can render
	// it as a list rather than leaving it buried in prose.
	RequestedDocuments []string

	LogoDataURI template.URL
}

func newDriverPortalInvitationSampleContext() any {
	return DriverPortalInvitationContext{
		FirstName:     "Marcus",
		LastName:      "Dell",
		FullName:      "Marcus Dell",
		CompanyName:   sampleCompanyName,
		InviteURL:     template.URL("https://dash.example.com/invite/8f2c41a9b7e34d05"),
		ExpiresInDays: 7,
		ExpiresAt:     "2026-07-22 11:00 CDT",
	}
}

func newAgentEmailSampleContext() any {
	return AgentEmailContext{
		AgentSubject:      "Missing paperwork for " + sampleProNumber,
		AgentBody:         "We are missing the signed bill of lading and the lumper receipt for this load. Could you send them over so we can invoice?",
		CompanyName:       sampleCompanyName,
		CustomerName:      sampleCustomerName,
		ShipmentProNumber: sampleProNumber,
		RequestedDocuments: []string{
			"Signed bill of lading",
			"Lumper receipt",
		},
		//nolint:gosec // A compile-time constant data: URI; see the field's doc comment.
		LogoDataURI: template.URL(sampleLogoDataURI),
	}
}

func (r *Registry) registerPortalKinds() {
	_ = r.Register(&KindDefinition{
		Kind:        KindDriverPortalInvitationEmail,
		DisplayName: "Driver Portal Invitation",
		Description: "Invites a driver to the portal. Not customer-scoped: the recipient " +
			"is a worker, so a per-customer assignment would never fire.",
		Category:      "Portal",
		Channels:      []Channel{ChannelSubject, ChannelEmailHTML, ChannelEmailText},
		sampleFactory: newDriverPortalInvitationSampleContext,
		Variables: []VariableDefinition{
			{Path: "FirstName", Type: VariableString, Description: "The driver's first name."},
			{Path: "LastName", Type: VariableString, Description: "The driver's last name."},
			{Path: "FullName", Type: VariableString, Description: "The driver's full name."},
			companyNameVariable(),
			{
				Path:        "InviteURL",
				Type:        VariableString,
				Required:    true,
				Description: "The single-use invitation link. Required: without it the message has no purpose.",
			},
			{
				Path:        "ExpiresInDays",
				Type:        VariableInt,
				Description: "How many days the link is good for. Use this rather than writing a number into the copy, or the wording will drift from the real expiry.",
			},
			{
				Path:        "ExpiresAt",
				Type:        VariableDateTime,
				Description: "The exact moment the link stops working.",
			},
		},
	})
}

func (r *Registry) registerAgentKinds() {
	_ = r.Register(&KindDefinition{
		Kind:        KindAgentRequestMissingDocsEmail,
		DisplayName: "Missing Documents Request",
		Description: "Wraps an agent-composed request in your own letterhead and signature, " +
			"so prose written automatically still goes out looking like your organization.",
		Category:       "Agent",
		Channels:       []Channel{ChannelSubject, ChannelEmailHTML, ChannelEmailText},
		CustomerScoped: true,
		sampleFactory:  newAgentEmailSampleContext,
		Variables: []VariableDefinition{
			{
				Path:        "AgentSubject",
				Type:        VariableString,
				Required:    true,
				Description: "The subject the agent composed. Required on the subject channel: replacing it entirely defeats the point of letting the agent describe what it needs.",
			},
			{
				Path:        "AgentBody",
				Type:        VariableString,
				Required:    true,
				Description: "The request the agent composed. Required: this is the substance of the message.",
			},
			companyNameVariable(),
			customerNameVariable(false, "Who is being asked for the documents."),
			proNumberVariable(false, "The shipment the documents belong to."),
			{
				Path:        "RequestedDocuments",
				Type:        VariableStringList,
				Description: "What is being asked for, as a list, so it can be rendered as bullets instead of buried in a paragraph.",
			},
			logoVariable(),
		},
	})
}
