package integrationservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	sharedsamsara "github.com/emoss08/trenova/shared/samsara"
	"github.com/emoss08/trenova/shared/samsara/drivers"
)

type connectionTester interface {
	Test(ctx context.Context, config map[string]string) error
}

var connectionTesters = map[integration.Type]connectionTester{
	integration.TypeSamsara: &samsaraConnectionTester{},
}

type samsaraConnectionTester struct{}

func (t *samsaraConnectionTester) Test(ctx context.Context, cfg map[string]string) error {
	client, err := sharedsamsara.New(
		cfg["token"],
		sharedsamsara.WithBaseURL(cfg["baseUrl"]),
		sharedsamsara.WithTimeout(15*time.Second),
	)
	if err != nil {
		return err
	}

	_, err = client.Drivers.List(ctx, drivers.ListParams{Limit: 1})
	return err
}
