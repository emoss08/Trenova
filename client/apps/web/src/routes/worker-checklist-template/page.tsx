import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const Table = lazy(() => import("./_components/checklist-template-table"));

export function WorkerChecklistTemplatesPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Checklist Templates",
        description:
          "Onboarding and offboarding checklists that start automatically from a hire, rehire or termination — with who owns each step and when it is due.",
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
