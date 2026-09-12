import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/pay-profiles-table"));

export function PayProfilesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Pay Profiles"),
        description: t(
          "Reusable driver pay packages: mileage rates and bands, revenue percentages, accessorial pay, and guarantees.",
        ),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
