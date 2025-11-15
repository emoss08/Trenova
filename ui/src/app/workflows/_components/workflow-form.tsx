/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { NumberField } from "@/components/ui/number-input";
import {
  type CreateWorkflowRequestSchema,
  TriggerType,
  WorkflowStatus,
} from "@/lib/schemas/workflow-schema";
import { useFormContext } from "react-hook-form";

const workflowStatusChoices = [
  { value: WorkflowStatus.enum.draft, label: "Draft" },
  { value: WorkflowStatus.enum.active, label: "Active" },
  { value: WorkflowStatus.enum.inactive, label: "Inactive" },
  { value: WorkflowStatus.enum.archived, label: "Archived" },
];

const triggerTypeChoices = [
  { value: TriggerType.enum.manual, label: "Manual" },
  { value: TriggerType.enum.scheduled, label: "Scheduled (Cron)" },
  {
    value: TriggerType.enum.shipment_status,
    label: "Shipment Status Change",
  },
  { value: TriggerType.enum.document_uploaded, label: "Document Upload" },
  { value: TriggerType.enum.entity_created, label: "Entity Created" },
  { value: TriggerType.enum.entity_updated, label: "Entity Updated" },
  { value: TriggerType.enum.webhook, label: "Webhook" },
];

export function WorkflowForm() {
  const { control } = useFormContext<CreateWorkflowRequestSchema>();

  return (
    <WorkflowFormOuter>
      <FormGroup cols={2} className="border-b pb-2">
        <FormControl>
          <SelectField
            control={control}
            rules={{ required: true }}
            name="status"
            label="Status"
            placeholder="Select Status"
            description="The current status of the workflow (draft, active, inactive, archived)."
            options={workflowStatusChoices}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="name"
            label="Name"
            placeholder="Workflow Name"
            description="A descriptive name for this workflow."
            maxLength={100}
          />
        </FormControl>
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="description"
            label="Description"
            placeholder="Describe what this workflow does..."
            description="A detailed description of the workflow's purpose and functionality."
          />
        </FormControl>
      </FormGroup>

      <TriggerConfigurationSection />
      <ExecutionSettingsSection />
    </WorkflowFormOuter>
  );
}

function WorkflowFormOuter({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-col">{children}</div>;
}

function TriggerConfigurationSection() {
  const { control } = useFormContext<CreateWorkflowRequestSchema>();

  return (
    <FormSection
      title="Trigger Configuration"
      description="Configure when and how this workflow is triggered"
      className="border-b py-2"
    >
      <FormGroup cols={1}>
        <FormControl>
          <SelectField
            control={control}
            rules={{ required: true }}
            name="triggerType"
            label="Trigger Type"
            placeholder="Select Trigger Type"
            description="How this workflow will be triggered."
            options={triggerTypeChoices}
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}

function ExecutionSettingsSection() {
  const { control } = useFormContext<CreateWorkflowRequestSchema>();

  return (
    <FormSection
      title="Execution Settings"
      description="Configure execution behavior and retry policies"
      className="py-2"
    >
      <FormGroup cols={2}>
        <FormControl>
          <NumberField
            control={control}
            name="timeoutSeconds"
            label="Timeout (seconds)"
            placeholder="300"
            description="Maximum time in seconds before the workflow times out."
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="maxRetries"
            label="Max Retries"
            placeholder="3"
            description="Maximum number of retry attempts on failure."
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="retryDelaySeconds"
            label="Retry Delay (seconds)"
            placeholder="60"
            description="Delay in seconds between retry attempts."
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={control}
            name="enableLogging"
            label="Enable Logging"
            description="Enable detailed execution logging for debugging."
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}
