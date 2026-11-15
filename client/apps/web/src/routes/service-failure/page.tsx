import { useQueryParams } from "@trenova/shared/hooks/use-query-params";
import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/service-failure-table"));

export function ServiceFailuresPage() {
  const t = useT();

  const [searchParams] = useQueryParams();
  const shipmentId = searchParams.get("shipmentId") ?? undefined;

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Service failures"),
        description: t("Review unresolved pickup and delivery service failures"),
      }}
    >
      <DataTableLazyComponent>
        <Table shipmentId={shipmentId} />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
