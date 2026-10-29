package portaltemplate

import (
	"html/template"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templatecatalog"
)

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
		CompanyName:   templatecatalog.SampleCompanyName,
		InviteURL:     template.URL("https://dash.example.com/invite/8f2c41a9b7e34d05"),
		ExpiresInDays: 7,
		ExpiresAt:     "2026-07-22 11:00 CDT",
	}
}

func newAgentEmailSampleContext() any {
	return AgentEmailContext{
		AgentSubject:      "Missing paperwork for " + templatecatalog.SampleProNumber,
		AgentBody:         "We are missing the signed bill of lading and the lumper receipt for this load. Could you send them over so we can invoice?",
		CompanyName:       templatecatalog.SampleCompanyName,
		CustomerName:      templatecatalog.SampleCustomerName,
		ShipmentProNumber: templatecatalog.SampleProNumber,
		RequestedDocuments: []string{
			"Signed bill of lading",
			"Lumper receipt",
		},
		//nolint:gosec // A compile-time constant data: URI; see the field's doc comment.
		LogoDataURI: template.URL(templatecatalog.SampleLogoDataURI),
	}
}

func newAgentInboundReplySampleContext() any {
	return AgentEmailContext{
		AgentSubject:      "Re: Where is " + templatecatalog.SampleProNumber + "?",
		AgentBody:         "The load is on schedule and due at the consignee tomorrow before noon. We will write again if that changes.",
		CompanyName:       templatecatalog.SampleCompanyName,
		CustomerName:      templatecatalog.SampleCustomerName,
		ShipmentProNumber: templatecatalog.SampleProNumber,
		//nolint:gosec // A compile-time constant data: URI; see the field's doc comment.
		LogoDataURI: template.URL(templatecatalog.SampleLogoDataURI),
	}
}

// ProposalReminderLine is one proposal still waiting, as the reminder lists it.
type ProposalReminderLine struct {
	Tool      string
	Rationale string
}

// AgentProposalReminderContext is what the reminder email has to say: which
// agent, how many changes, how long they have waited, what each one is, and
// where to go to decide.
type AgentProposalReminderContext struct {
	RecipientFirstName string
	AgentName          string
	PendingCount       int
	WaitingHours       int
	Proposals          []ProposalReminderLine
	ReviewURL          template.URL
	CompanyName        string
	LogoDataURI        template.URL
}

func newAgentProposalReminderSampleContext() any {
	return AgentProposalReminderContext{
		RecipientFirstName: "Marcus",
		AgentName:          "Dispatch coverage",
		PendingCount:       2,
		WaitingHours:       6,
		Proposals: []ProposalReminderLine{
			{
				Tool:      "assign move",
				Rationale: "Dana Ortiz has 7 hours left and is 12 miles from the pickup.",
			},
			{
				Tool:      "add shipment comment",
				Rationale: "Record that the customer asked for a morning delivery.",
			},
		},
		ReviewURL: template.URL(
			"https://app.example.com/admin/agent-control?tab=activity&activity=proposals",
		),
		CompanyName: templatecatalog.SampleCompanyName,
		//nolint:gosec // A compile-time constant data: URI; see the field's doc comment.
		LogoDataURI: template.URL(templatecatalog.SampleLogoDataURI),
	}
}

func agentProposalReminderVariables() []templatecatalog.VariableDefinition {
	return []templatecatalog.VariableDefinition{
		{
			Path:        "RecipientFirstName",
			Type:        templatecatalog.VariableString,
			Description: "The first name of the person who can decide.",
		},
		{
			Path:        "AgentName",
			Type:        templatecatalog.VariableString,
			Required:    true,
			Description: "The agent whose changes are waiting.",
		},
		{
			Path:        "PendingCount",
			Type:        templatecatalog.VariableInt,
			Description: "How many of its changes are waiting on a decision.",
		},
		{
			Path:        "WaitingHours",
			Type:        templatecatalog.VariableInt,
			Description: "How many hours the oldest of them has waited.",
		},
		{
			Path:        "Proposals",
			Type:        templatecatalog.VariableCollection,
			Description: "Each waiting change. Range over it: Tool is what the agent asked to do and Rationale is why.",
			Fields: []templatecatalog.VariableDefinition{
				{
					Path:        "Tool",
					Type:        templatecatalog.VariableString,
					Description: "What the agent asked to do, in words.",
				},
				{
					Path:        "Rationale",
					Type:        templatecatalog.VariableString,
					Description: "The agent's reason.",
				},
			},
		},
		{
			Path:        "ReviewURL",
			Type:        templatecatalog.VariableString,
			Required:    true,
			Description: "Where to decide: the proposals list in AI Control, already filtered to this run.",
		},
		templatecatalog.CompanyNameVariable(),
		templatecatalog.LogoVariable(),
	}
}

func RegisterPortalKinds(r *templatecatalog.Registry) {
	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:        documenttemplate.KindDriverPortalInvitationEmail,
		DisplayName: "Driver Portal Invitation",
		Description: "Invites a driver to the portal. Not customer-scoped: the recipient " +
			"is a worker, so a per-customer assignment would never fire.",
		Category: "Portal",
		Channels: []documenttemplate.Channel{
			documenttemplate.ChannelSubject,
			documenttemplate.ChannelEmailHTML,
			documenttemplate.ChannelEmailText,
		},
		SampleFactory: newDriverPortalInvitationSampleContext,
		Variables: []templatecatalog.VariableDefinition{
			{
				Path:        "FirstName",
				Type:        templatecatalog.VariableString,
				Description: "The driver's first name.",
			},
			{
				Path:        "LastName",
				Type:        templatecatalog.VariableString,
				Description: "The driver's last name.",
			},
			{
				Path:        "FullName",
				Type:        templatecatalog.VariableString,
				Description: "The driver's full name.",
			},
			templatecatalog.CompanyNameVariable(),
			{
				Path:        "InviteURL",
				Type:        templatecatalog.VariableString,
				Required:    true,
				Description: "The single-use invitation link. Required: without it the message has no purpose.",
			},
			{
				Path:        templatecatalog.ExpiresInDaysPath,
				Type:        templatecatalog.VariableInt,
				Description: "How many days the link is good for. Use this rather than writing a number into the copy, or the wording will drift from the real expiry.",
			},
			{
				Path:        "ExpiresAt",
				Type:        templatecatalog.VariableDateTime,
				Description: "The exact moment the link stops working.",
			},
		},
	})
}

func RegisterAgentKinds(r *templatecatalog.Registry) {
	_ = r.Register(agentCustomerUpdateKind())
	_ = r.Register(agentInboundReplyKind())
	_ = r.Register(agentProposalReminderKind())
	_ = r.Register(agentRequestMissingDocsKind())
}

func agentCustomerUpdateKind() *templatecatalog.KindDefinition {
	return &templatecatalog.KindDefinition{
		Kind:        documenttemplate.KindAgentCustomerUpdateEmail,
		DisplayName: "Customer Update",
		Description: "Wraps a status update an agent composed for a customer — a delay, a " +
			"revised arrival, a delivery confirmation — in your own letterhead and signature.",
		Category: agentCategory,
		Channels: []documenttemplate.Channel{
			documenttemplate.ChannelSubject,
			documenttemplate.ChannelEmailHTML,
			documenttemplate.ChannelEmailText,
		},
		CustomerScoped: true,
		SampleFactory:  newAgentEmailSampleContext,
		Variables: []templatecatalog.VariableDefinition{
			{
				Path:        agentSubjectPath,
				Type:        templatecatalog.VariableString,
				Required:    true,
				Description: "The subject the agent composed. Required on the subject channel.",
			},
			{
				Path:        agentBodyPath,
				Type:        templatecatalog.VariableString,
				Required:    true,
				Description: "The update the agent composed. Required: this is the substance of the message.",
			},
			templatecatalog.CompanyNameVariable(),
			templatecatalog.CustomerNameVariable(false, "Who the update is for."),
			templatecatalog.ProNumberVariable(false, "The shipment the update is about."),
			{
				Path:        requestedDocumentsPath,
				Type:        templatecatalog.VariableStringList,
				Description: "Optional points to list under the update, such as documents still needed. Usually empty.",
			},
			templatecatalog.LogoVariable(),
		},
	}
}

func agentInboundReplyKind() *templatecatalog.KindDefinition {
	return &templatecatalog.KindDefinition{
		Kind:        documenttemplate.KindAgentInboundReplyEmail,
		DisplayName: "Inbox Reply",
		Description: "Wraps an answer an agent composed to a message that arrived on one of " +
			"your monitored addresses — a status question, a missing detail — in your own " +
			"letterhead, sent back to whoever wrote in the same thread.",
		Category: agentCategory,
		Channels: []documenttemplate.Channel{
			documenttemplate.ChannelSubject,
			documenttemplate.ChannelEmailHTML,
			documenttemplate.ChannelEmailText,
		},
		CustomerScoped: true,
		SampleFactory:  newAgentInboundReplySampleContext,
		Variables: []templatecatalog.VariableDefinition{
			{
				Path:        agentSubjectPath,
				Type:        templatecatalog.VariableString,
				Required:    true,
				Description: "The reply's subject, the original's with Re: in front. Required on the subject channel: a changed subject starts a new thread in the sender's mail.",
			},
			{
				Path:        agentBodyPath,
				Type:        templatecatalog.VariableString,
				Required:    true,
				Description: "The answer the agent composed. Required: this is the substance of the message.",
			},
			templatecatalog.CompanyNameVariable(),
			templatecatalog.CustomerNameVariable(
				false,
				"The customer the sender belongs to, when the message was matched to one.",
			),
			templatecatalog.ProNumberVariable(
				false,
				"The shipment the message is about, when it was matched to one.",
			),
			{
				Path:        requestedDocumentsPath,
				Type:        templatecatalog.VariableStringList,
				Description: "Optional points to list under the answer. The inbox reply leaves it empty; it is there for a template shared with the other agent emails.",
			},
			templatecatalog.LogoVariable(),
		},
	}
}

func agentProposalReminderKind() *templatecatalog.KindDefinition {
	return &templatecatalog.KindDefinition{
		Kind:        documenttemplate.KindAgentProposalReminderEmail,
		DisplayName: "Proposal Reminder",
		Description: "Reminds the people who can decide an agent's proposals that some have been " +
			"waiting for hours, and takes them to the list to decide.",
		Category: agentCategory,
		Channels: []documenttemplate.Channel{
			documenttemplate.ChannelSubject,
			documenttemplate.ChannelEmailHTML,
			documenttemplate.ChannelEmailText,
		},
		SampleFactory: newAgentProposalReminderSampleContext,
		Variables:     agentProposalReminderVariables(),
	}
}

func agentRequestMissingDocsKind() *templatecatalog.KindDefinition {
	return &templatecatalog.KindDefinition{
		Kind:        documenttemplate.KindAgentRequestMissingDocsEmail,
		DisplayName: "Missing Documents Request",
		Description: "Wraps an agent-composed request in your own letterhead and signature, " +
			"so prose written automatically still goes out looking like your organization.",
		Category: agentCategory,
		Channels: []documenttemplate.Channel{
			documenttemplate.ChannelSubject,
			documenttemplate.ChannelEmailHTML,
			documenttemplate.ChannelEmailText,
		},
		CustomerScoped: true,
		SampleFactory:  newAgentEmailSampleContext,
		Variables: []templatecatalog.VariableDefinition{
			{
				Path:        agentSubjectPath,
				Type:        templatecatalog.VariableString,
				Required:    true,
				Description: "The subject the agent composed. Required on the subject channel: replacing it entirely defeats the point of letting the agent describe what it needs.",
			},
			{
				Path:        agentBodyPath,
				Type:        templatecatalog.VariableString,
				Required:    true,
				Description: "The request the agent composed. Required: this is the substance of the message.",
			},
			templatecatalog.CompanyNameVariable(),
			templatecatalog.CustomerNameVariable(false, "Who is being asked for the documents."),
			templatecatalog.ProNumberVariable(false, "The shipment the documents belong to."),
			{
				Path:        requestedDocumentsPath,
				Type:        templatecatalog.VariableStringList,
				Description: "What is being asked for, as a list, so it can be rendered as bullets instead of buried in a paragraph.",
			},
			templatecatalog.LogoVariable(),
		},
	}
}

const (
	agentCategory          = "Agent"
	agentSubjectPath       = "AgentSubject"
	agentBodyPath          = "AgentBody"
	requestedDocumentsPath = "RequestedDocuments"
)
