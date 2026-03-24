package registry

import (
	"reflect"
	"strings"

	"go.uber.org/zap"
)

type ActivityRegistrar interface {
	RegisterActivity(a any)
}

func RegisterActivitiesFromStruct(
	w ActivityRegistrar,
	activities any,
	logger *zap.Logger,
) (int, error) {
	if activities == nil {
		return 0, nil
	}

	v := reflect.ValueOf(activities)
	t := v.Type()

	count := 0
	for i := 0; i < t.NumMethod(); i++ {
		method := t.Method(i)
		if isActivityMethod(&method) {
			methodValue := v.Method(i).Interface()
			w.RegisterActivity(methodValue)
			count++

			if logger != nil {
				logger.Debug("registered activity",
					zap.String("name", method.Name),
				)
			}
		}
	}

	return count, nil
}

func isActivityMethod(method *reflect.Method) bool {
	if !method.IsExported() {
		return false
	}

	name := method.Name

	if strings.HasSuffix(name, "Activity") {
		return true
	}

	activityPrefixes := []string{
		"Do",
		"Process",
		"Execute",
		"Handle",
		"Fetch",
		"Send",
		"Create",
		"Update",
		"Delete",
		"Get",
	}
	for _, prefix := range activityPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	return false
}
