package benchresult

import (
	"errors"
	"strconv"
)

const (
	DataFixture = "fixture seed"
	DataVolume  = "fixture and volume seeds"
)

var ErrNoFixtureData = errors.New(
	"the database holds no fixture-seed shipment (SEED-…) in the bench user's organization, " +
		"and the scenarios name those records",
)

type DataProfile struct {
	Shape string `json:"shape"`
	Drift string `json:"drift,omitempty"`
}

func DataProfileOf(fixture, volume, writes int64) (DataProfile, error) {
	if fixture == 0 {
		return DataProfile{}, ErrNoFixtureData
	}
	profile := DataProfile{Shape: DataFixture}
	if volume > 0 {
		profile.Shape = DataVolume
	}
	if writes > 0 {
		profile.Drift = strconv.FormatInt(writes, 10) + " changes earlier bench runs approved " +
			"are still in the data"
	}

	return profile, nil
}

func (p DataProfile) String() string {
	if p.Drift == "" {
		return p.Shape
	}

	return p.Shape + "; " + p.Drift
}

func ComparedDataNote(before, after DataProfile) string {
	if before.Shape == "" || before.Shape == after.Shape {
		return ""
	}

	return "the earlier run ran on the " + before.Shape + " and this one on the " +
		after.Shape + ", so a moved pass rate may be the data rather than the change"
}
