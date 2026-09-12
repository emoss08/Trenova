import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/settlements-table"));

export function CarrierSettlementsPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Carrier Settlement History"),
        description: t(
          "Read-only record of every carrier payable statement across pay periods — process active settlements in the workspace.",
        ),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
