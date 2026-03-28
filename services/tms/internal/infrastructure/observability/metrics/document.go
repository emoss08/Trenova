package metrics

import (
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type Document struct {
	Base
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
		Base: NewBase(registry, logger, enabled),
	}

	if !enabled {
		return m
	}

	m.extractionTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "document_intelligence",
		Name:      "extraction_total",
		Help:      "Total number of document extraction outcomes",
	}, []string{"status", "source_kind", "reason"})

	m.shipmentDraftTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "document_intelligence",
		Name:      "shipment_draft_total",
		Help:      "Total number of shipment draft decisions by resource type and detected kind",
	}, []string{"status", "resource_type", "kind"})

	m.typeAssociationTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "document_intelligence",
		Name:      "type_association_total",
		Help:      "Total number of inferred document type association outcomes",
	}, []string{"outcome", "kind"})

	m.searchProjectionSync = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "document_intelligence",
		Name:      "search_projection_sync_total",
		Help:      "Total number of document search projection sync attempts",
	}, []string{"status"})

	m.searchQueryTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "document_intelligence",
		Name:      "search_query_total",
		Help:      "Total number of document intelligence search queries by backend and outcome",
	}, []string{"backend", "outcome"})

	m.reconciliationQueueTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "document_intelligence",
		Name:      "reconciliation_queue_total",
		Help:      "Total number of document intelligence reconciliation queue attempts",
	}, []string{"status"})

	m.aiOutcomeTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: "document_intelligence",
		Name:      "ai_outcome_total",
		Help:      "Total number of AI-assisted document intelligence outcomes",
	}, []string{"operation", "status", "outcome"})

	m.mustRegister(
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

func (m *Document) RecordExtraction(status string, sourceKind documentcontent.SourceKind, reason string) {
	m.ifEnabled(func() {
		if sourceKind == "" {
			sourceKind = documentcontent.SourceKind(UnknownValue)
		}
		if reason == "" {
			reason = "none"
		}
		m.extractionTotal.WithLabelValues(status, string(sourceKind), reason).Inc()
	})
}

func (m *Document) RecordShipmentDraft(status, resourceType, kind string) {
	m.ifEnabled(func() {
		if resourceType == "" {
			resourceType = UnknownValue
		}
		if kind == "" {
			kind = UnknownValue
		}
		m.shipmentDraftTotal.WithLabelValues(status, resourceType, kind).Inc()
	})
}

func (m *Document) RecordTypeAssociation(outcome, kind string) {
	m.ifEnabled(func() {
		if kind == "" {
			kind = UnknownValue
		}
		m.typeAssociationTotal.WithLabelValues(outcome, kind).Inc()
	})
}

func (m *Document) RecordSearchProjectionSync(success bool) {
	m.ifEnabled(func() {
		status := "success"
		if !success {
			status = "failure"
		}
		m.searchProjectionSync.WithLabelValues(status).Inc()
	})
}

func (m *Document) RecordSearchQuery(backend, outcome string) {
	m.ifEnabled(func() {
		if backend == "" {
			backend = UnknownValue
		}
		if outcome == "" {
			outcome = UnknownValue
		}
		m.searchQueryTotal.WithLabelValues(backend, outcome).Inc()
	})
}

func (m *Document) RecordReconciliationQueue(success bool) {
	m.ifEnabled(func() {
		status := "success"
		if !success {
			status = "failure"
		}
		m.reconciliationQueueTotal.WithLabelValues(status).Inc()
	})
}

func (m *Document) RecordAIOutcome(operation string, success bool, outcome string) {
	m.ifEnabled(func() {
		if operation == "" {
			operation = UnknownValue
		}
		if outcome == "" {
			outcome = UnknownValue
		}
		status := "success"
		if !success {
			status = "failure"
		}
		m.aiOutcomeTotal.WithLabelValues(operation, status, outcome).Inc()
	})
}
