package aiusagerepository

import "github.com/emoss08/trenova/shared/pulid"

func pulidFrom(raw string) pulid.ID {
	id, err := pulid.Parse(raw)
	if err != nil {
		return pulid.Nil
	}

	return id
}
