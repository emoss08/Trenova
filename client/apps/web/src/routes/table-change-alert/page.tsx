import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const SubscriptionTable = lazy(() => import("./_components/subscription-table"));

export function TableChangeAlertPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Table change alert"),
        description: t("Get told when the records you watch are created, changed or deleted"),
      }}
    >
      <DataTableLazyComponent>
        <SubscriptionTable />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
