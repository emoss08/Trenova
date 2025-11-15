import { FormCreateModal } from "@/components/ui/form-create-modal";
import {
  createWorkflowRequestSchema,
  TriggerType,
} from "@/lib/schemas/workflow-schema";
import { type TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { WorkflowForm } from "./workflow-form";

export function CreateWorkflowModal({ open, onOpenChange }: TableSheetProps) {
  const form = useForm({
    resolver: zodResolver(createWorkflowRequestSchema),
    defaultValues: {
      name: "",
      description: "",
      triggerType: TriggerType.enum.manual,
      triggerConfig: {},
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Workflow"
      formComponent={<WorkflowForm />}
      form={form}
      url="/workflows/"
      queryKey="workflow-list"
      className="max-w-[600px]"
    />
  );
}
