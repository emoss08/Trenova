package email

func (s MessageStatus) progress() int {
	switch s {
	case MessageStatusSending:
		return 1
	case MessageStatusSent:
		return 2
	case MessageStatusDelivered:
		return 3
	case MessageStatusOpened:
		return 4
	case MessageStatusClicked:
		return 5
	case MessageStatusQueued,
		MessageStatusFailed,
		MessageStatusBounced,
		MessageStatusComplained,
		MessageStatusSuppressed:
		return 0
	default:
		return 0
	}
}

func (s MessageStatus) IsFailure() bool {
	switch s {
	case MessageStatusFailed,
		MessageStatusBounced,
		MessageStatusComplained,
		MessageStatusSuppressed:
		return true
	case MessageStatusQueued,
		MessageStatusSending,
		MessageStatusSent,
		MessageStatusDelivered,
		MessageStatusOpened,
		MessageStatusClicked:
		return false
	default:
		return false
	}
}

func eventStatus(eventType EventType) (MessageStatus, bool) {
	switch eventType {
	case EventTypeSent:
		return MessageStatusSent, true
	case EventTypeDelivered:
		return MessageStatusDelivered, true
	case EventTypeOpened:
		return MessageStatusOpened, true
	case EventTypeClicked:
		return MessageStatusClicked, true
	case EventTypeBounced:
		return MessageStatusBounced, true
	case EventTypeComplained:
		return MessageStatusComplained, true
	case EventTypeFailed:
		return MessageStatusFailed, true
	default:
		return "", false
	}
}

func NextMessageStatus(current MessageStatus, eventType EventType) (MessageStatus, bool) {
	next, ok := eventStatus(eventType)
	if !ok || next == current {
		return current, false
	}
	switch next {
	case MessageStatusComplained:
		return next, true
	case MessageStatusBounced:
		if current == MessageStatusComplained {
			return current, false
		}
		return next, true
	case MessageStatusFailed:
		if current.IsFailure() ||
			current.progress() >= MessageStatusDelivered.progress() {
			return current, false
		}
		return next, true
	case MessageStatusQueued,
		MessageStatusSending,
		MessageStatusSent,
		MessageStatusDelivered,
		MessageStatusOpened,
		MessageStatusClicked,
		MessageStatusSuppressed:
		return advance(current, next)
	default:
		return advance(current, next)
	}
}

func advance(current, next MessageStatus) (MessageStatus, bool) {
	if current.IsFailure() || next.progress() <= current.progress() {
		return current, false
	}
	return next, true
}
