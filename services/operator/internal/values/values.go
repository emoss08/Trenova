package values

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

const SchemaVersion = 1

const (
	chartName   = "trenova"
	maxFullname = 40
)

const (
	keyPostgres = "postgres"
	keyTMS      = "tms"
	keyRedis    = "redis"
	keyMinio    = "minio"
	keyMemory   = "memory"
)

var (
	ErrSchemaVersion = errors.New("this operator does not know the spec's schema version")
	ErrInvalidSpec   = errors.New("the spec cannot be applied")

	versionPattern  = regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`)
	hostnamePattern = regexp.MustCompile(
		`^([a-z0-9]([-a-z0-9]{0,61}[a-z0-9])?\.)+[a-z][-a-z0-9]{0,61}[a-z0-9]$`,
	)
)

type Rendered struct {
	Version string
	Values  map[string]any
}

func Render(
	spec *instanceapi.InstanceSpec,
	base map[string]any,
	operatorNamespace string,
) (Rendered, error) {
	if operatorNamespace == "" {
		return Rendered{}, errors.New("the operator's namespace is needed to admit it to the TMS")
	}
	if int(spec.SchemaVersion) != SchemaVersion {
		return Rendered{}, fmt.Errorf(
			"%w: %d (it knows %d)",
			ErrSchemaVersion,
			spec.SchemaVersion,
			SchemaVersion,
		)
	}
	if !versionPattern.MatchString(spec.Release.Version) {
		return Rendered{}, fmt.Errorf(
			"%w: release version %q",
			ErrInvalidSpec,
			spec.Release.Version,
		)
	}
	if !hostnamePattern.MatchString(spec.Hostname) {
		return Rendered{}, fmt.Errorf("%w: hostname %q", ErrInvalidSpec, spec.Hostname)
	}
	stopped, hibernate, err := lifecycle(spec.Lifecycle)
	if err != nil {
		return Rendered{}, err
	}
	out := deepCopy(base)
	docling := slices.Contains(spec.Components, "docling")
	merge(out, map[string]any{
		"domain":        spec.Hostname,
		"storageDomain": StorageHost(spec.Hostname),
		"stopped":       stopped,
		keyTMS:          map[string]any{"instanceId": spec.InstanceId.String()},
		"docling":       map[string]any{"enabled": docling},
		keyPostgres:     map[string]any{"hibernate": hibernate},
		"managed":       managed(operatorNamespace),
	})
	sized, err := size(sizeInput{
		vcpu:      spec.Plan.Vcpu,
		memoryGiB: spec.Plan.MemoryGiB,
		diskGiB:   spec.Plan.DiskGiB,
		docling:   docling,
		clamav:    enabled(out, "clamav"),
		minio:     enabled(out, keyMinio),
	})
	if err != nil {
		return Rendered{}, err
	}
	merge(out, sized)
	return Rendered{Version: spec.Release.Version, Values: out}, nil
}

func managed(operatorNamespace string) map[string]any {
	return map[string]any{
		"enabled": true,
		"from": []any{
			map[string]any{
				"namespaceSelector": map[string]any{
					"matchLabels": map[string]any{
						"kubernetes.io/metadata.name": operatorNamespace,
					},
				},
			},
		},
	}
}

func Fullname(release string) string {
	name := release + "-" + chartName
	if strings.Contains(release, chartName) {
		name = release
	}
	if len(name) > maxFullname {
		name = name[:maxFullname]
	}
	return strings.TrimSuffix(name, "-")
}

func AppSecret(release string, base map[string]any) string {
	if secrets, ok := base["secrets"].(map[string]any); ok {
		if name, isString := secrets["existingSecret"].(string); isString && name != "" {
			return name
		}
	}
	return Fullname(release) + "-secrets"
}

func APIService(release string) string {
	return Fullname(release) + "-api"
}

func lifecycle(l instanceapi.InstanceSpecLifecycle) (stopped, hibernate bool, err error) {
	switch l {
	case instanceapi.Active:
		return false, false, nil
	case instanceapi.Suspended:
		return true, true, nil
	case instanceapi.Decommissioned:
		return true, false, nil
	default:
		return false, false, fmt.Errorf("%w: lifecycle %q", ErrInvalidSpec, l)
	}
}

func StorageHost(hostname string) string {
	label, rest, _ := strings.Cut(hostname, ".")
	return label + "-files." + rest
}

func enabled(v map[string]any, component string) bool {
	m, ok := v[component].(map[string]any)
	if !ok {
		return component == keyMinio
	}
	on, ok := m["enabled"].(bool)
	if !ok {
		return component == keyMinio
	}
	return on
}

func merge(dst, src map[string]any) {
	for k, sv := range src {
		if sm, ok := sv.(map[string]any); ok {
			if dm, isMap := dst[k].(map[string]any); isMap {
				merge(dm, sm)
				continue
			}
			dst[k] = deepCopy(sm)
			continue
		}
		dst[k] = sv
	}
}

func deepCopy(src map[string]any) map[string]any {
	out := make(map[string]any, len(src))
	for k, v := range src {
		switch t := v.(type) {
		case map[string]any:
			out[k] = deepCopy(t)
		case []any:
			out[k] = slices.Clone(t)
		default:
			out[k] = v
		}
	}
	return out
}

func mib(n float64) string {
	return fmt.Sprintf("%dMi", int64(math.Floor(n)))
}

func millicores(n float64) string {
	return fmt.Sprintf("%dm", int64(math.Max(math.Floor(n), 10)))
}
