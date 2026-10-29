package aiauditheartbeat

// Heartbeat reports that a long pass is still making progress.
type Heartbeat func(details ...any)

func Beat(heartbeat Heartbeat, details ...any) {
	if heartbeat != nil {
		heartbeat(details...)
	}
}
