package permission

func (r *Registry) registerCommunicationResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceDriverMessage.String(),
		DisplayName: "Driver Message",
		Description: "Messages sent to drivers on their phone through the Dash app",
		Category:    "Communications",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View messages sent to drivers"},
			{Operation: OpCreate, DisplayName: "Send", Description: "Send a message to a driver"},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceCustomerCommunication.String(),
		DisplayName: "Customer Communication",
		Description: "Emails and notices sent to customers and other outside parties",
		Category:    "Communications",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "Read",
				Description: "View what was sent to customers",
			},
			{
				Operation:   OpCreate,
				DisplayName: "Send",
				Description: "Send emails and notices to customers and outside parties",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceInboundMessage.String(),
		DisplayName: "Inbound Message",
		Description: "Email that arrived on a monitored address, and what was made of it",
		Category:    "Communications",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "Read",
				Description: "View inbound messages and their attachments",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Review",
				Description: "Link a message to a record, reply to it, or mark it handled",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	// The mailbox is separated from the messages because they are different
	// privileges: reading what a customer sent is ordinary desk work, while
	// creating an address the outside world can post to, and seeing its token,
	// is not.
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceInboundMailbox.String(),
		DisplayName: "Inbound Mailbox",
		Description: "Addresses the system listens on, and how much they are trusted to act alone",
		Category:    "Communications",
		Operations:  standardOpsWithDelete,

		DefaultSensitivity: SensitivityRestricted,
	})
}
