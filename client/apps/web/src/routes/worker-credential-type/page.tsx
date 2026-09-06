import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const Table = lazy(() => import("./_components/credential-type-table"));

export function WorkerCredentialTypesPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Credential Types",
        description:
          "The licences, cards, endorsements and certificates workers can hold — which are required, for whom, and how far ahead renewals are flagged.",
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
