package documentmetrics

import (
	"github.com/emoss08/trenova/internal/infrastructure/observability/metricbase"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const documentSubsystem = "document_intelligence"

type Document struct {
	metricbase.Base
	extractionTotal          *prometheus.CounterVec
	shipmentDraftTotal       *prometheus.CounterVec
	typeAssociationTotal     *prometheus.CounterVec
	searchProjectionSync     *prometheus.CounterVec
	searchQueryTotal         *prometheus.CounterVec
	reconciliationQueueTotal *prometheus.CounterVec
	aiOutcomeTotal           *prometheus.CounterVec
}

func NewDocument(registry *prometheus.Registry, logger *zap.Logger, enabled bool) *Document {
	m := &Document{
		Base: metricbase.NewBase(registry, logger, enabled),
	}

	if !enabled {
		return m
	}

	m.extractionTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: documentSubsystem,
		Name:      "extraction_total",
		Help:      "Total number of document extraction outcomes",
	}, []string{metricbase.LabelStatus, "source_kind", metricbase.LabelReason})

	m.shipmentDraftTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: documentSubsystem,
		Name:      "shipment_draft_total",
		Help:      "Total number of shipment draft decisions by resource type and detected kind",
	}, []string{metricbase.LabelStatus, "resource_type", metricbase.LabelKind})

	m.typeAssociationTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: documentSubsystem,
		Name:      "type_association_total",
		Help:      "Total number of inferred document type association outcomes",
	}, []string{metricbase.LabelOutcome, metricbase.LabelKind})

	m.searchProjectionSync = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: documentSubsystem,
		Name:      "search_projection_sync_total",
		Help:      "Total number of document search projection sync attempts",
	}, []string{metricbase.LabelStatus})

	m.searchQueryTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: documentSubsystem,
		Name:      "search_query_total",
		Help:      "Total number of document intelligence search queries by backend and outcome",
	}, []string{"backend", metricbase.LabelOutcome})

	m.reconciliationQueueTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: documentSubsystem,
		Name:      "reconciliation_queue_total",
		Help:      "Total number of document intelligence reconciliation queue attempts",
	}, []string{metricbase.LabelStatus})

	m.aiOutcomeTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricbase.Namespace,
		Subsystem: documentSubsystem,
		Name:      "ai_outcome_total",
		Help:      "Total number of AI-assisted document intelligence outcomes",
	}, []string{metricbase.LabelOperation, metricbase.LabelStatus, metricbase.LabelOutcome})

	m.MustRegister(
		m.extractionTotal,
		m.shipmentDraftTotal,
		m.typeAssociationTotal,
		m.searchProjectionSync,
		m.searchQueryTotal,
		m.reconciliationQueueTotal,
		m.aiOutcomeTotal,
	)

	return m
}

func (m *Document) RecordExtraction(
	status string,
	sourceKind documentcontent.SourceKind,
	reason string,
) {
	m.IfEnabled(func() {
		if sourceKind == "" {
			sourceKind = documentcontent.SourceKind(metricbase.UnknownValue)
		}
		if reason == "" {
			reason = "none"
		}
		m.extractionTotal.WithLabelValues(status, string(sourceKind), reason).Inc()
	})
}

func (m *Document) RecordShipmentDraft(status, resourceType, kind string) {
	m.IfEnabled(func() {
		if resourceType == "" {
			resourceType = metricbase.UnknownValue
		}
		if kind == "" {
			kind = metricbase.UnknownValue
		}
		m.shipmentDraftTotal.WithLabelValues(status, resourceType, kind).Inc()
	})
}

func (m *Document) RecordTypeAssociation(outcome, kind string) {
	m.IfEnabled(func() {
		if kind == "" {
			kind = metricbase.UnknownValue
		}
		m.typeAssociationTotal.WithLabelValues(outcome, kind).Inc()
	})
}

func (m *Document) RecordSearchProjectionSync(success bool) {
	m.IfEnabled(func() {
		status := metricbase.StatusSuccess
		if !success {
			status = metricbase.StatusFailure
		}
		m.searchProjectionSync.WithLabelValues(status).Inc()
	})
}

func (m *Document) RecordSearchQuery(backend, outcome string) {
	m.IfEnabled(func() {
		if backend == "" {
			backend = metricbase.UnknownValue
		}
		if outcome == "" {
			outcome = metricbase.UnknownValue
		}
		m.searchQueryTotal.WithLabelValues(backend, outcome).Inc()
	})
}

func (m *Document) RecordReconciliationQueue(success bool) {
	m.IfEnabled(func() {
		status := metricbase.StatusSuccess
		if !success {
			status = metricbase.StatusFailure
		}
		m.reconciliationQueueTotal.WithLabelValues(status).Inc()
	})
}

func (m *Document) RecordAIOutcome(operation string, success bool, outcome string) {
	m.IfEnabled(func() {
		if operation == "" {
			operation = metricbase.UnknownValue
		}
		if outcome == "" {
			outcome = metricbase.UnknownValue
		}
		status := metricbase.StatusSuccess
		if !success {
			status = metricbase.StatusFailure
		}
		m.aiOutcomeTotal.WithLabelValues(operation, status, outcome).Inc()
	})
}
