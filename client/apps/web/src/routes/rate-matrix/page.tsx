import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const Table = lazy(() => import("./_components/rate-matrix-table"));

export function RateMatrixPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Rate matrices"),
        description: t(
          "Price a tariff the way it was published — a grid of zones, weight breaks and classes — instead of one lane per cell",
        ),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
