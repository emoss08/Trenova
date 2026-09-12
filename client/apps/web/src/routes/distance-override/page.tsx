import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const Table = lazy(() => import("./_components/distance-override-table"));

export function DistanceOverridesPage() {
  const t = useT();

  return (
    <AdminPageLayout>
      <PageHeader
        title={t("Distance Overrides")}
        description={t("Override calculated distances between location pairs for routing and billing adjustments")}
      />
      <div className="p-4">
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
      </div>
    </AdminPageLayout>
  );
}
