import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/advances-table"));

export function PayAdvancesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Pay Advances"),
        description: t(
          "Cash and money-code advances that are automatically recovered from the driver's next settlement.",
        ),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
