package analytics

import "github.com/emoss08/trenova/internal/core/ports/services"

// Request represents a request for analytics data
type Request struct {
	Page      services.AnalyticsPage `query:"page"      json:"page"`
	StartDate int64                  `query:"startDate" json:"startDate"`
	EndDate   int64                  `query:"endDate"   json:"endDate"`
	Limit     int                    `query:"limit"     json:"limit"`
}
