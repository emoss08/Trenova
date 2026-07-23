import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(
  () => import("./_components/journal-reversal-table"),
);

export function JournalReversalsPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Journal Reversals",
        description: "Request and manage journal entry reversals.",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
