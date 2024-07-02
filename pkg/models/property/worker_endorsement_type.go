package property

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

type WorkerEndorsement string

const (
	WorkerEndorsementNone         = WorkerEndorsement("None")
	WorkerEndorsementTanker       = WorkerEndorsement("Tanker")
	WorkerEndorsementHazmat       = WorkerEndorsement("Hazmat")
	WorkerEndorsementTankerHazmat = WorkerEndorsement("TankerHazmat")
)

func (o WorkerEndorsement) String() string {
	return string(o)
}

func (WorkerEndorsement) Values() []string {
	return []string{"None", "Tanker", "Hazmat", "TankerHazmat"}
}

func (o WorkerEndorsement) Value() (driver.Value, error) {
	return string(o), nil
}

func (o *WorkerEndorsement) Scan(value any) error {
	if value == nil {
		return errors.New("WorkerEndorsement: expected a value, got nil")
	}
	switch v := value.(type) {
	case string:
		*o = WorkerEndorsement(v)
	case []byte:
		*o = WorkerEndorsement(string(v))
	default:
		return fmt.Errorf("WorkerEndorsement: cannot can type %T into WorkerEndorsement", value)
	}
	return nil
}

func GetWorkerEndorsementList() []any {
	values := WorkerEndorsement("").Values()
	interfaces := make([]any, len(values))
	for i, v := range values {
		interfaces[i] = v
	}

	return interfaces
}
