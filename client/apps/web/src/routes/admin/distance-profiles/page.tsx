import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/distance-profile-table"));

export function DistanceProfilesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Distance Profiles"),
        description: t("Manage business-unit routing policy used by distance calculations"),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
