package homelayout

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// WidgetConfig carries every option any widget can take. Which fields matter is
// decided by the widget's ConfigKind, not by the caller — a config arriving with
// fields the widget does not use is normalized away rather than rejected, so a
// user who switches a widget's kind never has to clean up after themselves.
type WidgetConfig struct {
	Metric       string   `json:"metric,omitempty"`
	Metrics      []string `json:"metrics,omitempty"`
	DefinitionID pulid.ID `json:"definitionId,omitempty"`
	CannedKey    string   `json:"cannedKey,omitempty"`
	ChartID      string   `json:"chartId,omitempty"`
	ColumnID     string   `json:"columnId,omitempty"`
	DashboardID  pulid.ID `json:"dashboardId,omitempty"`
	Text         string   `json:"text,omitempty"`
	Limit        int      `json:"limit,omitempty"`
	WindowDays   int      `json:"windowDays,omitempty"`
}

func validateWidgetConfig(
	multiErr *errortypes.MultiError,
	kind ConfigKind,
	config WidgetConfig,
	fieldPath string,
) {
	switch kind {
	case ConfigKindNone:
	case ConfigKindQueue:
		validateLimit(multiErr, config.Limit, fieldPath)
	case ConfigKindMetric:
		validateMetric(multiErr, config.Metric, fieldPath+".config.metric")
	case ConfigKindMetricRow:
		validateMetricRow(multiErr, config.Metrics, fieldPath)
	case ConfigKindTrend:
		validateWindowDays(multiErr, config.WindowDays, fieldPath)
	case ConfigKindReport:
		validateReportConfig(multiErr, config, fieldPath)
	case ConfigKindDashboard:
		if config.DashboardID.IsNil() {
			multiErr.Add(fieldPath+".config.dashboardId", errortypes.ErrRequired,
				"Choose the dashboard this widget links to")
		}
	case ConfigKindText:
		validateText(multiErr, config.Text, fieldPath+".config.text")
	default:
		multiErr.Add(fieldPath+".config", errortypes.ErrInvalid,
			fmt.Sprintf("Unknown widget configuration kind %q", kind))
	}
}

func validateLimit(multiErr *errortypes.MultiError, limit int, fieldPath string) {
	if limit < 0 || limit > MaxWidgetLimit {
		multiErr.Add(fieldPath+".config.limit", errortypes.ErrInvalid,
			fmt.Sprintf("Row limit must be between 1 and %d", MaxWidgetLimit))
	}
}

func validateMetric(multiErr *errortypes.MultiError, metric, fieldPath string) {
	switch {
	case metric == "":
		multiErr.Add(fieldPath, errortypes.ErrRequired, "Choose the metric this widget shows")
	case !metricExists(metric):
		multiErr.Add(fieldPath, errortypes.ErrInvalid,
			fmt.Sprintf("Unknown metric: %s", metric))
	}
}

func validateMetricRow(multiErr *errortypes.MultiError, metrics []string, fieldPath string) {
	if len(metrics) == 0 {
		multiErr.Add(fieldPath+".config.metrics", errortypes.ErrRequired,
			"Choose at least one metric")
		return
	}
	if len(metrics) > MaxKPIRowMetrics {
		multiErr.Add(fieldPath+".config.metrics", errortypes.ErrInvalid,
			fmt.Sprintf("A metric strip holds at most %d metrics", MaxKPIRowMetrics))
		return
	}

	seen := make(map[string]struct{}, len(metrics))
	for idx, metric := range metrics {
		metricPath := fmt.Sprintf("%s.config.metrics[%d]", fieldPath, idx)

		if !metricExists(metric) {
			multiErr.Add(metricPath, errortypes.ErrInvalid,
				fmt.Sprintf("Unknown metric: %s", metric))
			continue
		}
		if _, dup := seen[metric]; dup {
			multiErr.Add(metricPath, errortypes.ErrDuplicate,
				fmt.Sprintf("Duplicate metric: %s", metric))
			continue
		}
		seen[metric] = struct{}{}
	}
}

func validateWindowDays(multiErr *errortypes.MultiError, windowDays int, fieldPath string) {
	if windowDays < 0 || windowDays > MaxWindowDays {
		multiErr.Add(fieldPath+".config.windowDays", errortypes.ErrInvalid,
			fmt.Sprintf("Window must be between 1 and %d days", MaxWindowDays))
	}
}

// validateReportConfig mirrors the rule report dashboards already enforce: a
// tile reads from a saved report or a canned report, never both.
func validateReportConfig(
	multiErr *errortypes.MultiError,
	config WidgetConfig,
	fieldPath string,
) {
	hasDefinition := !config.DefinitionID.IsNil()
	hasCanned := config.CannedKey != ""

	switch {
	case !hasDefinition && !hasCanned:
		multiErr.Add(fieldPath+".config", errortypes.ErrRequired,
			"Choose the report this widget shows")
	case hasDefinition && hasCanned:
		multiErr.Add(fieldPath+".config", errortypes.ErrInvalid,
			"A widget reads from a saved report or a canned report, not both")
	}

	validateLimit(multiErr, config.Limit, fieldPath)
}

func validateText(multiErr *errortypes.MultiError, text, fieldPath string) {
	switch {
	case strings.TrimSpace(text) == "":
		multiErr.Add(fieldPath, errortypes.ErrRequired, "Write the announcement")
	case len(text) > MaxAnnouncementLength:
		multiErr.Add(fieldPath, errortypes.ErrInvalidLength,
			fmt.Sprintf("An announcement is at most %d characters", MaxAnnouncementLength))
	}
}

// normalizeConfig strips every field the widget's kind does not read, so a
// stored layout never carries configuration that cannot take effect.
func normalizeConfig(kind ConfigKind, config WidgetConfig) WidgetConfig {
	switch kind {
	case ConfigKindQueue:
		return WidgetConfig{Limit: clampLimit(config.Limit)}
	case ConfigKindMetric:
		return WidgetConfig{Metric: config.Metric}
	case ConfigKindMetricRow:
		return WidgetConfig{Metrics: normalizeMetrics(config.Metrics)}
	case ConfigKindTrend:
		return WidgetConfig{WindowDays: clampWindowDays(config.WindowDays)}
	case ConfigKindReport:
		return normalizeReportConfig(config)
	case ConfigKindDashboard:
		return WidgetConfig{DashboardID: config.DashboardID}
	case ConfigKindText:
		return WidgetConfig{Text: config.Text}
	case ConfigKindNone:
		return WidgetConfig{}
	default:
		return WidgetConfig{}
	}
}

func normalizeReportConfig(config WidgetConfig) WidgetConfig {
	normalized := WidgetConfig{
		ChartID:  config.ChartID,
		ColumnID: config.ColumnID,
		Limit:    clampLimit(config.Limit),
	}

	// A definition wins over a canned key so a config carrying both resolves the
	// same way the validator describes rather than failing open on the library.
	if !config.DefinitionID.IsNil() {
		normalized.DefinitionID = config.DefinitionID
		return normalized
	}

	normalized.CannedKey = config.CannedKey
	return normalized
}

func normalizeMetrics(metrics []string) []string {
	normalized := make([]string, 0, min(len(metrics), MaxKPIRowMetrics))
	seen := make(map[string]struct{}, len(metrics))

	for _, metric := range metrics {
		if len(normalized) == MaxKPIRowMetrics {
			break
		}
		if !metricExists(metric) {
			continue
		}
		if _, dup := seen[metric]; dup {
			continue
		}
		seen[metric] = struct{}{}
		normalized = append(normalized, metric)
	}

	return normalized
}

func clampLimit(limit int) int {
	if limit < 0 {
		return 0
	}
	return min(limit, MaxWidgetLimit)
}

func clampWindowDays(windowDays int) int {
	if windowDays < 0 {
		return 0
	}
	return min(windowDays, MaxWindowDays)
}
