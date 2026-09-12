import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const Table = lazy(() => import("./_components/ifta-jurisdiction-mileage-table"));

export function JurisdictionMileagePage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Jurisdiction Mileage"),
        description: t(
          "Miles by state or province that dispatch did not compute — deadhead, yard moves, and units without telematics. They are added to the miles the quarter's IFTA return works out from shipment moves.",
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
