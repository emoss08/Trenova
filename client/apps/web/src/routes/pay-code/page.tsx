import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/pay-codes-table"));

export function PayCodesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Pay Codes"),
        description: t(
          "Your carrier's catalog of earning and deduction codes — behavior flags and GL account mappings drive how each code settles and posts.",
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
