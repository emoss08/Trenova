/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { FormCreateModal } from "@/components/ui/form-create-modal";
import {
  createWorkflowRequestSchema,
  WorkflowStatus,
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
      status: WorkflowStatus.enum.draft,
      name: "",
      description: "",
      triggerType: TriggerType.enum.manual,
      triggerConfig: {},
      timeoutSeconds: 300,
      maxRetries: 3,
      retryDelaySeconds: 60,
      enableLogging: true,
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
