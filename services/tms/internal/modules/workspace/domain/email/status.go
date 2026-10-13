package email

var messageStatusProgress = map[MessageStatus]int{
	MessageStatusQueued:    0,
	MessageStatusSending:   1,
	MessageStatusSent:      2,
	MessageStatusDelivered: 3,
	MessageStatusOpened:    4,
	MessageStatusClicked:   5,
}

func (s MessageStatus) IsFailure() bool {
	switch s {
	case MessageStatusFailed,
		MessageStatusBounced,
		MessageStatusComplained,
		MessageStatusSuppressed:
		return true
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
		if current.IsFailure() || messageStatusProgress[current] >= messageStatusProgress[MessageStatusDelivered] {
			return current, false
		}
		return next, true
	default:
		if current.IsFailure() || messageStatusProgress[next] <= messageStatusProgress[current] {
			return current, false
		}
		return next, true
	}
}
