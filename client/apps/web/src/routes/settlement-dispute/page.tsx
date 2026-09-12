import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/disputes-table"));

export function SettlementDisputesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Settlement Disputes"),
        description: t(
          "Driver-submitted questions and challenges against issued settlements — review, resolve with a correcting adjustment, or deny with an explanation.",
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
