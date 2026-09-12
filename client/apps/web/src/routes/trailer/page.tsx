import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/trailer-table"));

export function TrailersPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Trailers"),
        description: t("Manage and configure trailers for your organization"),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
