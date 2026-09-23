package registry

import (
	"context"
	"reflect"
	"sort"
)

var (
	contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType   = reflect.TypeOf((*error)(nil)).Elem()
)

// ActivityNamer is a registry that can say which activities it will register
// before it registers them. WorkerManager uses it to refuse a name collision
// on a shared task queue with an error that says which two workers clashed;
// the SDK's own check is a panic that names only the activity.
type ActivityNamer interface {
	ActivityNames() []string
}

// ActivityNames is every method on the activities struct that Temporal will
// take as an activity: exported, a context first, and an error last. The SDK
// derives the registered name from the method name, so this is the same set
// of keys it will use.
func (r *DomainRegistry) ActivityNames() []string {
	return ActivityNamesOf(r.activities)
}

// ActivityNamesOf is every activity Temporal will register from an activities
// struct, for a registry that registers more than one.
func ActivityNamesOf(activities any) []string {
	structType := reflect.TypeOf(activities)
	if structType == nil {
		return nil
	}

	names := make([]string, 0, structType.NumMethod())
	for i := range structType.NumMethod() {
		method := structType.Method(i)
		if !method.IsExported() || !isActivityFunc(method.Type) {
			continue
		}
		names = append(names, method.Name)
	}
	sort.Strings(names)

	return names
}

// isActivityFunc reports whether a method's signature is one Temporal accepts.
// The receiver counts as the first input on a method obtained from the type,
// so a valid activity has at least two.
func isActivityFunc(fn reflect.Type) bool {
	if fn.NumIn() < 2 || fn.In(1) != contextType {
		return false
	}
	if fn.NumOut() == 0 || fn.NumOut() > 2 {
		return false
	}

	return fn.Out(fn.NumOut()-1) == errorType
}

// conflictingActivities is the names a registry wants that the queue has
// already given to someone else, paired with who holds them.
func conflictingActivities(taken map[string]string, wanted []string) []activityConflict {
	var conflicts []activityConflict
	for _, name := range wanted {
		if holder, exists := taken[name]; exists {
			conflicts = append(conflicts, activityConflict{Activity: name, HeldBy: holder})
		}
	}

	return conflicts
}

type activityConflict struct {
	Activity string
	HeldBy   string
}
