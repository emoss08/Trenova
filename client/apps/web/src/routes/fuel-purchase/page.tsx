import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const Table = lazy(() => import("./_components/fuel-purchase-table"));

export function FuelPurchasesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Fuel Purchases"),
        description: t(
          "Every gallon bought for a tractor, keyed by hand or imported from a fuel card statement. Tax-paid gallons credit the jurisdiction they were bought in on the quarterly IFTA return.",
        ),
      }}
    >
      <div className="flex flex-col gap-4">
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
