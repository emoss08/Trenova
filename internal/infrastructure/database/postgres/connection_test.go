package postgres_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/database/postgres"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
)

func TestConnectionPerformance(t *testing.T) {
	t.Skip("Skipping test that requires real database connection")
}

func BenchmarkConnectionRetrieval(b *testing.B) {
	ctx := context.Background()
	cfg := &config.Config{}
	mgr := config.NewManager()
	log := logger.NewLogger(cfg)

	params := postgres.ConnectionParams{
		Config: mgr,
		Logger: log,
	}

	conn := postgres.NewConnection(params)

	// Initialize connection
	_, _ = conn.DB(ctx)

	b.Run("ReadDB", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = conn.ReadDB(ctx)
		}
	})

	b.Run("WriteDB", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = conn.WriteDB(ctx)
		}
	})

	b.Run("DB_Legacy", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = conn.DB(ctx)
		}
	})
}
