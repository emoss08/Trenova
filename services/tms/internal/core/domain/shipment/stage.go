package shipment

import (
	"strconv"
	"strings"
)

type Stage string

const (
	StageLate          = Stage("Late")
	StageNeedsCoverage = Stage("NeedsCoverage")
	StageMoving        = Stage("Moving")
	StageScheduled     = Stage("Scheduled")
	StageDelivered     = Stage("Delivered")
	StageCanceled      = Stage("Canceled")
)

type stageDefinition struct {
	stage    Stage
	rank     int16
	statuses []Status
}

var stageDefinitions = [...]stageDefinition{
	{stage: StageLate, rank: 1, statuses: []Status{StatusDelayed}},
	{stage: StageNeedsCoverage, rank: 2, statuses: []Status{StatusNew, StatusPartiallyAssigned}},
	{stage: StageMoving, rank: 3, statuses: []Status{StatusInTransit, StatusPartiallyCompleted}},
	{stage: StageScheduled, rank: 4, statuses: []Status{StatusAssigned}},
	{
		stage:    StageDelivered,
		rank:     5,
		statuses: []Status{StatusCompleted, StatusReadyToInvoice, StatusInvoiced},
	},
	{stage: StageCanceled, rank: 6, statuses: []Status{StatusCanceled}},
}

var (
	stageByStatus = buildStageByStatus()
	rankByStage   = buildRankByStage()
)

func buildStageByStatus() map[Status]Stage {
	out := make(map[Status]Stage, 10)
	for _, def := range stageDefinitions {
		for _, status := range def.statuses {
			out[status] = def.stage
		}
	}
	return out
}

func buildRankByStage() map[Stage]int16 {
	out := make(map[Stage]int16, len(stageDefinitions))
	for _, def := range stageDefinitions {
		out[def.stage] = def.rank
	}
	return out
}

func StageOf(status Status) Stage {
	if stage, ok := stageByStatus[status]; ok {
		return stage
	}
	return StageNeedsCoverage
}

func (s Stage) Rank() int16 {
	return rankByStage[s]
}

func StageFromRank(rank int16) (Stage, bool) {
	for _, def := range stageDefinitions {
		if def.rank == rank {
			return def.stage, true
		}
	}
	return "", false
}

func (s Stage) Statuses() []Status {
	for _, def := range stageDefinitions {
		if def.stage == s {
			return append([]Status(nil), def.statuses...)
		}
	}
	return nil
}

func Stages() []Stage {
	out := make([]Stage, 0, len(stageDefinitions))
	for _, def := range stageDefinitions {
		out = append(out, def.stage)
	}
	return out
}

func (s Stage) IsValid() bool {
	_, ok := rankByStage[s]
	return ok
}

func (s *Shipment) Stage() Stage {
	return StageOf(s.Status)
}

func StageRankSQL(statusColumn string) string {
	var b strings.Builder
	b.WriteString("CASE")
	for _, def := range stageDefinitions {
		b.WriteString(" WHEN ")
		b.WriteString(statusColumn)
		b.WriteString(" IN (")
		for i, status := range def.statuses {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString("'")
			b.WriteString(string(status))
			b.WriteString("'")
		}
		b.WriteString(") THEN ")
		b.WriteString(strconv.Itoa(int(def.rank)))
	}
	b.WriteString(" ELSE ")
	b.WriteString(strconv.Itoa(int(StageNeedsCoverage.Rank())))
	b.WriteString(" END")
	return b.String()
}
