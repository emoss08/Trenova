import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const Table = lazy(() => import("./_components/rate-zone-table"));

export function RateZonePage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Rate zones"),
        description: t(
          "Name a market area once and price against it, instead of listing every postal prefix it covers",
        ),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
