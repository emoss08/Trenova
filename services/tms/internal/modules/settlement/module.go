package settlement

import (
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/settlement/adapters/postgres/benefitrepository"
	"github.com/emoss08/trenova/internal/modules/settlement/adapters/postgres/carrierledgerrepository"
	"github.com/emoss08/trenova/internal/modules/settlement/adapters/postgres/carriersettlementrepository"
	"github.com/emoss08/trenova/internal/modules/settlement/adapters/postgres/driverpayrepository"
	"github.com/emoss08/trenova/internal/modules/settlement/adapters/postgres/driversettlementrepository"
	"github.com/emoss08/trenova/internal/modules/settlement/adapters/postgres/fuelpurchaserepository"
	"github.com/emoss08/trenova/internal/modules/settlement/adapters/postgres/iftarepository"
	"github.com/emoss08/trenova/internal/modules/settlement/app/carriersettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driverpayservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/fuelpurchaseservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/iftaservice"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/carriersettlementports"
	"go.uber.org/fx"
)

func SharedOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			benefitrepository.New,
			driverpayrepository.NewPayProfile,
			driverpayrepository.NewWorkerPayAssignment,
			driverpayrepository.NewPayCode,
			driverpayrepository.NewRecurringDeduction,
			driverpayrepository.NewRecurringEarning,
			driverpayrepository.NewPayAdvance,
			driverpayrepository.NewEscrowAccount,
			driverpayrepository.NewDriverExpense,
			driversettlementrepository.NewSettlement,
			driversettlementrepository.NewBatch,
			driversettlementrepository.NewPayEvent,
			driversettlementrepository.NewDispute,
			carriersettlementrepository.NewSettlement,
			carriersettlementrepository.NewBatch,
			carriersettlementrepository.NewCostEvent,
			carrierledgerrepository.New,
			fuelpurchaserepository.New,
			iftarepository.New,
			fuelpurchaseservice.NewValidator,
			driverpayservice.New,
			driversettlementservice.New,
			carriersettlementservice.New,
			func(s *carriersettlementservice.Service) carrierports.CarrierCostAccrual { return s },
			func(s *carriersettlementservice.Service) carrierports.CarrierInvoiceAutoMatcher { return s },
			func(s *carriersettlementservice.Service) carriersettlementports.CarrierSettlementPayer { return s },
			func(s *driversettlementservice.Service) carriersettlementports.DriverSettlementPayer { return s },
			fuelpurchaseservice.New,
			iftaservice.New,
		),
	)
}

func APIOptions() fx.Option {
	return fx.Options()
}
