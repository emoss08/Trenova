package permissioncatalog

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

func (r *Registry) registerCommunicationResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceDriverMessage.String(),
		DisplayName: "Driver Message",
		Description: "Messages sent to drivers on their phone through the Dash app",
		Category:    "Dispatch",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View messages sent to drivers",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Send",
				Description: "Send a message to a driver",
			},
		},
		DefaultSensitivity: permission.SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceCustomerCommunication.String(),
		DisplayName: "Customer Communication",
		Description: "Emails and notices sent to customers and other outside parties",
		Category:    "Triage",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View what was sent to customers",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Send",
				Description: "Send emails and notices to customers and outside parties",
			},
		},
		DefaultSensitivity: permission.SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceInboundMessage.String(),
		DisplayName: "Triage",
		Description: "Email that arrived on a monitored address, and what was made of it",
		Category:    "Triage",
		Operations: []OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "View inbound messages and their attachments",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Review",
				Description: "Link a message to a record, reply to it or mark it handled, and resolve or reopen messages and conflicts",
			},
			{
				Operation:   permission.OpAssign,
				DisplayName: operationDefinitions[permission.OpAssign].DisplayName,
				Description: "Assign messages to other people and hand the queue off",
			},
			{
				Operation:   permission.OpManage,
				DisplayName: operationDefinitions[permission.OpManage].DisplayName,
				Description: "Set the rules Nova follows, reply templates and response targets",
			},
		},
		DefaultSensitivity: permission.SensitivityInternal,
	})

	// The mailbox is separated from the messages because they are different
	// privileges: reading what a customer sent is ordinary desk work, while
	// creating an address the outside world can post to, and seeing its token,
	// is not.
	_ = r.Register(&ResourceDefinition{
		Resource:      permission.ResourceInboundMailbox.String(),
		DisplayName:   "Inbound Mailbox",
		Description:   "Addresses the system listens on, and how much they are trusted to act alone",
		Category:      "Triage",
		ReferenceView: []string{"name", "address", "provider", "status"},
		Operations: relabel(
			standardOpsWithDelete,
			permission.OpUpdate,
			"Update",
			"Modify mailboxes and send test messages",
		),

		DefaultSensitivity: permission.SensitivityRestricted,
	})
}
