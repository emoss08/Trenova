package property

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

type DatabaseAction string

const (
	DatabaseActionInsert = DatabaseAction("Insert")
	DatabaseActionUpdate = DatabaseAction("Update")
	DatabaseActionDelete = DatabaseAction("Delete")
	DatabaseActionAll    = DatabaseAction("All")
)

func (o DatabaseAction) String() string {
	return string(o)
}

func (DatabaseAction) Values() []string {
	return []string{"Insert", "Update", "Delete", "All"}
}

func (o DatabaseAction) Value() (driver.Value, error) {
	return string(o), nil
}

func (o *DatabaseAction) Scan(value any) error {
	if value == nil {
		return errors.New("databaseaction: expected a value, got nil")
	}
	switch v := value.(type) {
	case string:
		*o = DatabaseAction(v)
	case []byte:
		*o = DatabaseAction(string(v))
	default:
		return fmt.Errorf("databaseaction: cannot can type %T into DatabaseAction", value)
	}
	return nil
}
