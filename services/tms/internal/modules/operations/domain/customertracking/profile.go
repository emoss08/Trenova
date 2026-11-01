package customertracking

import "github.com/emoss08/trenova/internal/modules/parties/domain/trackingprofile"

func Notifies(profile *trackingprofile.Profile, event Event) bool {
	if profile == nil {
		return false
	}
	switch event {
	case EventPickedUp:
		return profile.NotifyPickedUp
	case EventDelayed:
		return profile.NotifyDelayed
	case EventDelivered:
		return profile.NotifyDelivered
	case EventPodReady:
		return profile.NotifyPodReady
	default:
		return false
	}
}
