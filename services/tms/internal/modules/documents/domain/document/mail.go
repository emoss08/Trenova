package document

func (s Status) Mailable() bool {
	switch s {
	case StatusDraft, StatusActive, StatusPending, StatusPendingApproval:
		return true
	case StatusArchived, StatusExpired, StatusRejected:
		return false
	default:
		return false
	}
}

func (d *Document) Mailable() bool {
	return d.Status.Mailable() && !d.OwnedByConversation() && d.StoragePath != ""
}

func (d *Document) MailFileName() string {
	if d.OriginalName != "" {
		return d.OriginalName
	}

	return d.FileName
}
