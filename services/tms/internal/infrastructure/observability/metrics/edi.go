//revive:disable-next-line:var-naming
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const (
	ediSubsystem    = "edi"
	ediLabelMethod  = "method"
	ediLabelPartner = "partner"
	ediLabelStatus  = "status"
)

var (
	EDIAckLatencyBuckets = []float64{
		60,
		300,
		900,
		1800,
		3600,
		4 * 3600,
		8 * 3600,
		24 * 3600,
		48 * 3600,
	}
	EDIMDNRoundTripBuckets = []float64{.5, 1, 2.5, 5, 15, 30, 60, 300, 900, 1800, 3600, 4 * 3600}
)

type EDI struct {
	Base
	deliveryDuration     *prometheus.HistogramVec
	deliveriesTotal      *prometheus.CounterVec
	deadLetteredTotal    *prometheus.CounterVec
	ackLatency           *prometheus.HistogramVec
	mdnRoundTrip         *prometheus.HistogramVec
	inboundFilesTotal    *prometheus.CounterVec
	inboundParseDuration *prometheus.HistogramVec
	inboundOutcomesTotal *prometheus.CounterVec
	inboundPollTotal     *prometheus.CounterVec
}

func NewEDI(registry *prometheus.Registry, logger *zap.Logger, enabled bool) *EDI {
	m := &EDI{
		Base: NewBase(registry, logger, enabled),
	}

	if !enabled {
		return m
	}

	m.deliveryDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: Namespace,
		Subsystem: ediSubsystem,
		Name:      "delivery_duration_seconds",
		Help:      "Duration of outbound EDI transport deliveries in seconds",
		Buckets:   TemporalDurationBuckets,
	}, []string{ediLabelMethod, ediLabelStatus})

	m.deliveriesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: ediSubsystem,
		Name:      "deliveries_total",
		Help:      "Total number of outbound EDI delivery attempts by partner and transaction set",
	}, []string{ediLabelPartner, "transaction_set", ediLabelMethod, ediLabelStatus})

	m.deadLetteredTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: ediSubsystem,
		Name:      "dead_lettered_total",
		Help:      "Total number of outbound EDI messages dead-lettered after exhausting retries",
	}, []string{ediLabelPartner, "transaction_set"})

	m.ackLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: Namespace,
		Subsystem: ediSubsystem,
		Name:      "ack_latency_seconds",
		Help:      "Latency between sending an outbound EDI message and receiving its 997/999 acknowledgment",
		Buckets:   EDIAckLatencyBuckets,
	}, []string{ediLabelStatus})

	m.mdnRoundTrip = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: Namespace,
		Subsystem: ediSubsystem,
		Name:      "as2_mdn_round_trip_seconds",
		Help:      "Latency between an outbound AS2 send and the resolving MDN",
		Buckets:   EDIMDNRoundTripBuckets,
	}, []string{"mode"})

	m.inboundFilesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: ediSubsystem,
		Name:      "inbound_files_total",
		Help:      "Total number of inbound EDI files by partner and staging outcome",
	}, []string{ediLabelPartner, ediLabelMethod, ediLabelStatus})

	m.inboundParseDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: Namespace,
		Subsystem: ediSubsystem,
		Name:      "inbound_parse_duration_seconds",
		Help:      "Duration of inbound EDI file parsing and routing in seconds",
		Buckets:   HTTPDurationBuckets,
	}, []string{ediLabelStatus})

	m.inboundOutcomesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: ediSubsystem,
		Name:      "inbound_outcomes_total",
		Help:      "Total number of processed inbound EDI files by partner and final status",
	}, []string{ediLabelPartner, ediLabelStatus})

	m.inboundPollTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: ediSubsystem,
		Name:      "inbound_poll_total",
		Help:      "Total number of inbound EDI mailbox poll attempts by transport method and outcome",
	}, []string{ediLabelMethod, ediLabelStatus})

	m.mustRegister(
		m.deliveryDuration,
		m.deliveriesTotal,
		m.deadLetteredTotal,
		m.ackLatency,
		m.mdnRoundTrip,
		m.inboundFilesTotal,
		m.inboundParseDuration,
		m.inboundOutcomesTotal,
		m.inboundPollTotal,
	)

	return m
}

func (m *EDI) RecordDelivery(partner, transactionSet, method, status string, seconds float64) {
	m.ifEnabled(func() {
		m.deliveryDuration.WithLabelValues(method, status).Observe(seconds)
		m.deliveriesTotal.WithLabelValues(partner, transactionSet, method, status).Inc()
	})
}

func (m *EDI) RecordDeadLetter(partner, transactionSet string) {
	m.ifEnabled(func() {
		m.deadLetteredTotal.WithLabelValues(partner, transactionSet).Inc()
	})
}

func (m *EDI) RecordAckLatency(status string, seconds float64) {
	m.ifEnabled(func() {
		m.ackLatency.WithLabelValues(status).Observe(seconds)
	})
}

func (m *EDI) RecordMDNRoundTrip(mode string, seconds float64) {
	m.ifEnabled(func() {
		m.mdnRoundTrip.WithLabelValues(mode).Observe(seconds)
	})
}

func (m *EDI) RecordInboundFile(partner, method, status string) {
	m.ifEnabled(func() {
		m.inboundFilesTotal.WithLabelValues(partner, method, status).Inc()
	})
}

func (m *EDI) RecordInboundParse(status string, seconds float64) {
	m.ifEnabled(func() {
		m.inboundParseDuration.WithLabelValues(status).Observe(seconds)
	})
}

func (m *EDI) RecordInboundOutcome(partner, status string) {
	m.ifEnabled(func() {
		m.inboundOutcomesTotal.WithLabelValues(partner, status).Inc()
	})
}

func (m *EDI) RecordInboundPoll(method, status string) {
	m.ifEnabled(func() {
		m.inboundPollTotal.WithLabelValues(method, status).Inc()
	})
}
