import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";
import { PayFlowExplainer } from "./_components/pay-flow-explainer";

const Table = lazy(() => import("./_components/settlements-table"));

export function DriverSettlementsPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Settlement History"),
        description: t(
          "Read-only record of every driver and owner-operator settlement across pay periods — process active settlements in the workspace.",
        ),
      }}
    >
      <div className="flex flex-col gap-4">
        <PayFlowExplainer />
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
