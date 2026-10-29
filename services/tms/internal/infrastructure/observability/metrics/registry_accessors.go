package metrics

import (
	"sync"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aimetrics"
	"go.uber.org/zap"
)

// AssistantFrom is the registry's assistant collector, or a disabled one when
// there is no registry, which a test does and an install with metrics switched
// off does too. Every method on the result is safe.
func AssistantFrom(registry *Registry) *aimetrics.Assistant {
	if registry == nil || registry.Assistant == nil {
		return aimetrics.NewAssistant(nil, zap.NewNop(), false)
	}

	return registry.Assistant
}

type genAIOnce struct {
	once  sync.Once
	genAI *aimetrics.GenAI
}

func (m *Registry) GenAI() *aimetrics.GenAI {
	if m == nil {
		return nil
	}

	m.genAI.once.Do(func() {
		provider, err := m.MeterProvider()
		if err != nil {
			m.logger.Warn("gen_ai client metrics are unavailable", zap.Error(err))
			return
		}

		genAI, err := aimetrics.NewGenAI(provider)
		if err != nil {
			m.logger.Warn("gen_ai client metrics could not be created", zap.Error(err))
			return
		}
		m.genAI.genAI = genAI
	})

	return m.genAI.genAI
}
