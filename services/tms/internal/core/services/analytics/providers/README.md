<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->
# Analytics Providers

This directory contains analytics providers for different pages in the application. Each provider is responsible for collecting and processing analytics data specific to its domain.

## How to Add a New Analytics Provider

1. **Create a new file** for your provider in this directory, named after the domain it covers (e.g., `billing.go`, `operations.go`).

2. **Implement the AnalyticsPageProvider interface**:

```go
type MyDomainProvider struct {
    l *zerolog.Logger
    // Add any repositories or services needed to collect data
}

// NewMyDomainProvider creates a new provider
func NewMyDomainProvider(logger *logger.Logger, ...) *MyDomainProvider {
    log := logger.With().
        Str("provider", "my_domain_analytics").
        Logger()

    return &MyDomainProvider{
        l: &log,
        // Initialize dependencies
    }
}

// GetPage returns the page identifier
func (p *MyDomainProvider) GetPage() services.AnalyticsPage {
    return services.MyDomainAnalyticsPage // Define this in analytics.go
}

// GetAnalyticsData returns the analytics data
func (p *MyDomainProvider) GetAnalyticsData(ctx context.Context, opts *services.AnalyticsRequestOptions) (services.AnalyticsData, error) {
    // Collect and process data
    return services.AnalyticsData{
        "key1": value1,
        "key2": value2,
        // Add your metrics
    }, nil
}
```

3. **Register your provider** in `internal/core/services/analytics/module.go`:

```go
func RegisterProviders(p RegisterProvidersParams) {
    // ...
    
    // Create and register your provider
    myDomainProvider := providers.NewMyDomainProvider(
        p.Logger,
        // Pass any other dependencies
    )
    p.Service.GetRegistry().RegisterProvider(myDomainProvider)
}
```

4. **Add your page constant** in `internal/core/ports/services/analytics.go`:

```go
const (
    // ...
    MyDomainAnalyticsPage AnalyticsPage = "my-domain"
)
```

That's it! Your analytics provider will now be available through the analytics API. 