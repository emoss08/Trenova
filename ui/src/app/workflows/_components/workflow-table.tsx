import { DataTable } from "@/components/data-table/data-table";
import { type WorkflowSchema } from "@/lib/schemas/workflow-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./workflow-columns";
import { CreateWorkflowModal } from "./workflow-create-modal";
import { EditWorkflowModal } from "./workflow-edit-modal";

export default function WorkflowTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<WorkflowSchema>
      resource={Resource.Workflow}
      name="Workflow"
      link="/workflows/"
      queryKey="workflow-list"
      exportModelName="workflow"
      TableModal={CreateWorkflowModal}
      TableEditModal={EditWorkflowModal}
      columns={columns}
    />
  );
}
