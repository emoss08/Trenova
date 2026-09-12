import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/recurring-shipment-table"));

export function RecurringShipmentsPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Recurring Shipments"),
        description: t(
          "Automatically generate shipments for repeating lanes on a schedule you control",
        ),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
