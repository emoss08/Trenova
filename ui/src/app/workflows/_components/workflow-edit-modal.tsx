/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  updateWorkflowRequestSchema,
  type WorkflowSchema,
} from "@/lib/schemas/workflow-schema";
import { type TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { WorkflowForm } from "./workflow-form";

export function EditWorkflowModal({
  open,
  onOpenChange,
  currentRecord,
}: TableSheetProps<WorkflowSchema>) {
  const form = useForm({
    resolver: zodResolver(updateWorkflowRequestSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      open={open}
      onOpenChange={onOpenChange}
      title="Workflow"
      formComponent={<WorkflowForm />}
      form={form}
      url={`/workflows/${currentRecord?.id}/`}
      queryKey="workflow-list"
      className="max-w-[600px]"
    />
  );
}
