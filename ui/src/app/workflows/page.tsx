import { DataTableLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

// Lazy Loaded Components
const WorkflowTable = lazy(() => import("./_components/workflow-table"));

export default function Workflows() {
  return (
    <>
      <MetaTags
        title="Workflows"
        description="Manage workflow automation for your organization"
      />
      <DataTableLazyComponent>
        <WorkflowTable />
      </DataTableLazyComponent>
    </>
  );
}
