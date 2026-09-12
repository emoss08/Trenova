import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const Table = lazy(() => import("./_components/checklist-template-table"));

export function WorkerChecklistTemplatesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Checklist Templates"),
        description: t(
          "Onboarding and offboarding checklists that start automatically from a hire, rehire or termination — with who owns each step and when it is due.",
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
