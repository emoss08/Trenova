package values

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

const SchemaVersion = 1

const (
	chartName   = "trenova"
	maxFullname = 40
)

const EscrowSecret = "trenova-escrow"

const (
	DocumentsSecret = "trenova-storage"
	BackupsSecret   = "trenova-backup-storage"
)

type Storage struct {
	Endpoint        string
	PrivateEndpoint string
	Region          string
	DocumentsBucket string
	BackupsPath     string
}

func (s Storage) Internal() string {
	if s.PrivateEndpoint != "" {
		return s.PrivateEndpoint
	}
	return s.Endpoint
}

func (s Storage) Configured() bool {
	return s.DocumentsBucket != "" && s.BackupsPath != "" && s.Endpoint != ""
}

const AdminSetupLink = "link"

const (
	maxAdminName = 100
	maxOrgName   = 100
	maxEmail     = 254
)

const (
	keyPostgres = "postgres"
	keyTMS      = "tms"
	keyRedis    = "redis"
	keyMinio    = "minio"
	keyDocling  = "docling"
	keyClamAV   = "clamav"
	keyMemory   = "memory"
	keyCPU      = "cpu"
)

var (
	ErrSchemaVersion = errors.New("this operator does not know the spec's schema version")
	ErrInvalidSpec   = errors.New("the spec cannot be applied")

	generationPattern = regexp.MustCompile(`^r[0-9a-f]{8}$`)
	emailPattern      = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	versionPattern    = regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`)
	hostnamePattern   = regexp.MustCompile(
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
	storage Storage,
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
	admin, err := bootstrap(&spec.FirstAdministrator)
	if err != nil {
		return Rendered{}, err
	}
	pg, err := database(spec, storage, hibernate)
	if err != nil {
		return Rendered{}, err
	}
	out := deepCopy(base)
	docling := slices.Contains(spec.Components, keyDocling)
	merge(out, map[string]any{
		"domain":        spec.Hostname,
		"storageDomain": StorageHost(spec.Hostname),
		"stopped":       stopped,
		keyTMS:          tmsValues(spec.InstanceId.String()),
		keyDocling:      map[string]any{"enabled": docling},
		keyPostgres:     pg,
		"managed":       managed(operatorNamespace),
		"secrets":       map[string]any{"escrowSecret": EscrowSecret},
		"bootstrap":     admin,
	})
	if storage.Configured() {
		merge(out, objectStorage(storage))
	}
	sized, err := size(sizeInput{
		vcpu:      spec.Plan.Vcpu,
		memoryGiB: spec.Plan.MemoryGiB,
		diskGiB:   spec.Plan.DiskGiB,
		clamav:    enabled(out, keyClamAV),
	})
	if err != nil {
		return Rendered{}, err
	}
	merge(out, sized)
	return Rendered{Version: spec.Release.Version, Values: out}, nil
}

func objectStorage(s Storage) map[string]any {
	internal := s.Internal()
	return map[string]any{
		keyMinio: map[string]any{"enabled": false},
		"storage": map[string]any{
			"bucket":           s.DocumentsBucket,
			"region":           s.Region,
			"autoCreateBucket": false,
			"external": map[string]any{
				"endpoint":       strings.TrimPrefix(strings.TrimPrefix(internal, "https://"), "http://"),
				"useSSL":         !strings.HasPrefix(internal, "http://"),
				"publicEndpoint": s.Endpoint,
				"existingSecret": DocumentsSecret,
			},
		},
		"backup": map[string]any{
			"enabled":         true,
			"destinationPath": s.BackupsPath,
			"endpointURL":     internal,
			"existingSecret":  BackupsSecret,
		},
	}
}

func database(
	spec *instanceapi.InstanceSpec,
	storage Storage,
	hibernate bool,
) (map[string]any, error) {
	pg := map[string]any{"hibernate": hibernate}
	if d := spec.Database; d != nil {
		if !generationPattern.MatchString(d.Generation) || !source(d.SourceGeneration) {
			return nil, fmt.Errorf("%w: database generation %q", ErrInvalidSpec, d.Generation)
		}
		pg["generation"] = d.Generation
		pg["restoredFrom"] = map[string]any{
			"sourceGeneration": d.SourceGeneration,
			"targetTime":       d.TargetTime.UTC().Format(time.RFC3339),
		}
	}
	r := spec.Restore
	if r == nil || (spec.Database != nil && spec.Database.Generation == r.Generation) {
		return pg, nil
	}
	if !storage.Configured() {
		return nil, fmt.Errorf("%w: a restore needs the instance's backups, and it has no backup storage", ErrInvalidSpec)
	}
	if !generationPattern.MatchString(r.Generation) || !source(r.SourceGeneration) {
		return nil, fmt.Errorf("%w: restore generation %q", ErrInvalidSpec, r.Generation)
	}
	pg["restore"] = map[string]any{
		"generation":       r.Generation,
		"sourceGeneration": r.SourceGeneration,
		"targetTime":       r.TargetTime.UTC().Format(time.RFC3339),
	}
	return pg, nil
}

func source(generation string) bool {
	return generation == "" || generationPattern.MatchString(generation)
}

func PostgresCluster(release, generation string) string {
	name := Fullname(release) + "-postgres"
	if generation == "" {
		return name
	}
	return name + "-" + generation
}

func bootstrap(admin *instanceapi.SpecFirstAdministrator) (map[string]any, error) {
	email := string(admin.Email)
	switch {
	case !text(admin.Name, maxAdminName):
		return nil, fmt.Errorf("%w: the first administrator's name", ErrInvalidSpec)
	case !text(admin.OrganizationName, maxOrgName):
		return nil, fmt.Errorf("%w: the first organization's name", ErrInvalidSpec)
	case len(email) > maxEmail || !emailPattern.MatchString(email):
		return nil, fmt.Errorf("%w: the first administrator's email", ErrInvalidSpec)
	}
	return map[string]any{
		"adminSetup": AdminSetupLink,
		"orgName":    admin.OrganizationName,
		"adminName":  admin.Name,
		"adminEmail": email,
	}, nil
}

func text(s string, maxRunes int) bool {
	if strings.TrimSpace(s) == "" || !utf8.ValidString(s) || utf8.RuneCountInString(s) > maxRunes {
		return false
	}
	return !strings.ContainsFunc(s, unicode.IsControl)
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

func tmsValues(instanceID string) map[string]any {
	return map[string]any{
		"instanceId": instanceID,
		"config": map[string]any{"ai": map[string]any{
			"privateNetworkProviders": false,
			"serverCloudCredentials":  false,
		}},
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
		return false
	}
	on, _ := m["enabled"].(bool)
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
