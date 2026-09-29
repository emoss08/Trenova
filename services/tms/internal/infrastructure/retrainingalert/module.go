package retrainingalert

import "go.uber.org/fx"

var Module = fx.Module("retraining-alert", fx.Provide(New, AsAlerter))
