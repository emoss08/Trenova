import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/escrow-table"));

export function EscrowAccountsPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Escrow Accounts"),
        description: t(
          "Owner-operator maintenance escrow with a full transaction ledger and quarterly interest per 49 CFR 376.12(k).",
        ),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
