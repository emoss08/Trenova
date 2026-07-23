import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/recurring-shipment-table"));

export function RecurringShipmentsPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Recurring Shipments",
        description:
          "Automatically generate shipments for repeating lanes on a schedule you control",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
