package property

import "database/sql/driver"

type LogStatus string

const (
	LogStatusAttempted LogStatus = "ATTEMPTED"
	LogStatusSucceeded LogStatus = "SUCCEEDED"
	LogStatusFailed    LogStatus = "FAILED"
)

func (l LogStatus) String() string {
	return string(l)
}

func (LogStatus) Values() []string {
	return []string{"ATTEMPTED", "SUCCEEDED", "FAILED"}
}

func (l LogStatus) Value() (driver.Value, error) {
	return string(l), nil
}

func (l *LogStatus) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*l = LogStatus(v)
	case []byte:
		*l = LogStatus(string(v))
	default:
		return nil
	}
	return nil
}
