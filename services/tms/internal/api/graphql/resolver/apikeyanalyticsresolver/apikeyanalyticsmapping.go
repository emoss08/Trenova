package apikeyanalyticsresolver

import (
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

const defaultWindowDays = 7

func apiKeyAnalyticsToModel(data kernelports.AnalyticsData) (*gqlmodel.APIKeyAnalytics, error) {
	raw, err := sonic.Marshal(data)
	if err != nil {
		return nil, err
	}

	out := &gqlmodel.APIKeyAnalytics{
		TotalKeys:   &gqlmodel.APIKeyTotalCount{},
		ActiveKeys:  &gqlmodel.APIKeyStateCount{},
		RevokedKeys: &gqlmodel.APIKeyStateCount{},
		Requests30d: &gqlmodel.APIKeyRequestVolume{Sparkline: []*gqlmodel.APIKeyRequestDay{}},
	}
	if err = sonic.Unmarshal(raw, out); err != nil {
		return nil, err
	}

	return out, nil
}
