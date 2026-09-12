import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const Table = lazy(() => import("./_components/credential-type-table"));

export function WorkerCredentialTypesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Credential Types"),
        description: t(
          "The licences, cards, endorsements and certificates workers can hold — which are required, for whom, and how far ahead renewals are flagged.",
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
