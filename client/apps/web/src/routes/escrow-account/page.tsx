import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/escrow-table"));

export function EscrowAccountsPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Escrow Accounts",
        description:
          "Owner-operator maintenance escrow with a full transaction ledger and quarterly interest per 49 CFR 376.12(k).",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
