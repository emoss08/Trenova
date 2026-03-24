package schedule

type Provider interface {
	GetSchedules() []*Schedule
}

type ProviderFunc func() []*Schedule

func (f ProviderFunc) GetSchedules() []*Schedule {
	return f()
}
