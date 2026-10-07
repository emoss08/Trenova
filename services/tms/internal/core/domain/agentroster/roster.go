package agentroster

import "github.com/emoss08/trenova/shared/pulid"

const (
	RunDays      = 14
	DecisionDays = 30
	secondsInDay = 86400
)

type Stat struct {
	AgentID        pulid.ID
	RunsByDay      []int
	Approved       int
	Modified       int
	Rejected       int
	Failed         int
	ShadowRecorded int
}

func NewStat(agentID pulid.ID) *Stat {
	return &Stat{AgentID: agentID, RunsByDay: make([]int, RunDays)}
}

func RunsSince(now int64) int64 {
	return DayStart(now) - (RunDays-1)*secondsInDay
}

func DecisionsSince(now int64) int64 {
	return now - DecisionDays*secondsInDay
}

func DayStart(unix int64) int64 {
	return unix - unix%secondsInDay
}

func (s *Stat) AddRuns(day, count int) {
	if day < 0 || day >= len(s.RunsByDay) || count <= 0 {
		return
	}
	s.RunsByDay[day] += count
}

func (s *Stat) Runs() int {
	total := 0
	for _, count := range s.RunsByDay {
		total += count
	}
	return total
}

func (s *Stat) ApprovalRate() *float64 {
	decided := s.Approved + s.Modified + s.Rejected
	if decided == 0 {
		return nil
	}
	rate := float64(s.Approved+s.Modified) / float64(decided)
	return &rate
}
