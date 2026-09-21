import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/distance-override-table"));

export function DistanceOverridesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Distance Overrides"),
        description: t(
          "Override calculated distances between location pairs for routing and billing adjustments",
        ),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
