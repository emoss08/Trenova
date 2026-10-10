package iftajobs

import "github.com/emoss08/trenova/internal/core/temporaljobs/schedule"

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider {
	return &ScheduleProvider{}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{}
}
