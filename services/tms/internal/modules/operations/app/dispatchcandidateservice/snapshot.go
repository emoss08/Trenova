package dispatchcandidateservice

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/operations/domain/driverpositioning"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcost"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/shared/pulid"
)

type EndMarketValuer interface {
	EndMarketCents(latitude, longitude float64, at int64) int64
}

type FleetSnapshot struct {
	Drivers             []*dispatchboardports.BoardDriver
	DriversTruncated    bool
	WorkersByID         map[pulid.ID]*worker.Worker
	TractorByID         map[pulid.ID]*tractor.Tractor
	TrailerByID         map[pulid.ID]*trailer.Trailer
	HOSByWorker         map[pulid.ID]*telematics.WorkerHOSState
	HOSLogsByWorker     map[pulid.ID][]*telematics.WorkerHOSLog
	PosByTractor        map[pulid.ID]*telematics.VehiclePosition
	CommitmentsByWorker map[pulid.ID][]*dispatchboardports.WorkerCommitment
	TimeOffByWorker     map[pulid.ID][]*dispatchboardports.WorkerTimeOff
	LastTimeOffEnd      map[pulid.ID]int64
	WorkloadByWorker    map[pulid.ID]*dispatchboardports.WorkerWorkload
	OnTimeByWorker      map[pulid.ID]*dispatchboardports.WorkerOnTimeStats
	AcceptanceByWorker  map[pulid.ID]*dispatchboardports.WorkerAcceptanceStats
	ViolationsByWorker  map[pulid.ID]*telematicsports.WorkerHOSViolationStats
	NamesByWorker       map[pulid.ID]string
	PositionedByWorker  map[pulid.ID]*driverpositioning.Positioning
	LaneExperience      map[laneKey]int
	Control             *dispatchcontrol.DispatchControl
	TelematicsActive    bool
	Now                 int64

	Model     dispatchcost.Model
	EndMarket EndMarketValuer
	Forecast  *demandforecast.Forecaster

	profiles        map[pulid.ID]*driverProfile
	driverIndex     map[pulid.ID]int
	medianWeekMiles float64
}

type laneKey struct {
	WorkerID   pulid.ID
	CustomerID pulid.ID
}

func (s *FleetSnapshot) DriverName(driver *dispatchboardports.BoardDriver) string {
	if name, ok := s.NamesByWorker[driver.WorkerID]; ok {
		return name
	}
	return driver.FirstName + " " + driver.LastName
}

func (s *FleetSnapshot) Prepare() {
	s.medianWeekMiles = medianWeekMiles(s.WorkloadByWorker)
	s.profiles = make(map[pulid.ID]*driverProfile, len(s.Drivers))
	s.driverIndex = make(map[pulid.ID]int, len(s.Drivers))
	for i, driver := range s.Drivers {
		s.driverIndex[driver.WorkerID] = i
		if profile := buildProfile(s, driver); profile != nil {
			s.profiles[driver.WorkerID] = profile
		}
	}
}

func (s *FleetSnapshot) profile(driver *dispatchboardports.BoardDriver) *driverProfile {
	if s.profiles == nil {
		s.Prepare()
	}

	return s.profiles[driver.WorkerID]
}

func (s *FleetSnapshot) LastHomeAt(driver *dispatchboardports.BoardDriver) int64 {
	if profile := s.profile(driver); profile != nil {
		return profile.lastHomeAt
	}

	return 0
}

func (s *FleetSnapshot) RefreshProfile(workerID pulid.ID) {
	if s.profiles == nil {
		return
	}
	i, ok := s.driverIndex[workerID]
	if !ok {
		return
	}
	if profile := buildProfile(s, s.Drivers[i]); profile != nil {
		s.profiles[workerID] = profile
	}
}

func medianWeekMiles(workload map[pulid.ID]*dispatchboardports.WorkerWorkload) float64 {
	if len(workload) == 0 {
		return 0
	}
	miles := make([]float64, 0, len(workload))
	for _, item := range workload {
		miles = append(miles, item.TotalMiles)
	}
	slices.Sort(miles)

	middle := len(miles) / 2
	if len(miles)%2 == 1 {
		return miles[middle]
	}
	return (miles[middle-1] + miles[middle]) / 2
}
