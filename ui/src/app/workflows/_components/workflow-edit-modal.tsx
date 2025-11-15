import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  updateWorkflowRequestSchema,
  type WorkflowSchema,
} from "@/lib/schemas/workflow-schema";
import { EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { WorkflowForm } from "./workflow-form";

export function EditWorkflowModal({
  currentRecord,
}: EditTableSheetProps<WorkflowSchema>) {
  const form = useForm({
    resolver: zodResolver(updateWorkflowRequestSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      title="Workflow"
      formComponent={<WorkflowForm />}
      form={form}
      url="/workflows/"
      queryKey="workflow-list"
      className="max-w-[600px]"
    />
  );
}
