import { useT } from "@trenova/shared/i18n/use-t";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import {
  createWorkerChecklistTemplate,
  updateWorkerChecklistTemplate,
  WORKER_CHECKLIST_TEMPLATE_LIST_KEY,
  type WorkerChecklistTemplateRow,
} from "@/lib/graphql/worker-checklist";
import type { WorkerChecklistTemplateInput } from "@trenova/graphql/generated/graphql";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import {
  checklistTemplateFormSchema,
  type ChecklistTemplateFormValues,
} from "@trenova/shared/types/worker-checklist";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm, type Resolver } from "react-hook-form";
import { ChecklistTemplateForm } from "./checklist-template-form";

export function buildChecklistTemplateDefaults(
  row?: WorkerChecklistTemplateRow | null,
): ChecklistTemplateFormValues {
  if (!row) {
    return {
      code: "",
      name: "",
      description: null,
      kind: "Onboarding",
      trigger: "Hired",
      status: "Active",
      isDefault: false,
      items: [
        {
          label: "",
          description: null,
          kind: "Task",
          required: true,
          dueOffsetDays: 3,
          owner: "HR",
          credentialTypeId: null,
          documentTypeId: null,
        },
      ],
    };
  }
  return {
    code: row.code,
    name: row.name,
    description: row.description ?? null,
    kind: row.kind,
    trigger: row.trigger,
    status: row.status === "Inactive" ? "Inactive" : "Active",
    isDefault: row.isDefault,
    items: [...row.items]
      .sort((a, b) => a.sortOrder - b.sortOrder)
      .map((item) => ({
        label: item.label,
        description: item.description ?? null,
        kind: item.kind,
        required: item.required,
        dueOffsetDays: item.dueOffsetDays,
        owner: item.owner,
        credentialTypeId: item.credentialTypeId ?? null,
        documentTypeId: item.documentTypeId ?? null,
      })),
  };
}

export function toChecklistTemplateInput(
  values: ChecklistTemplateFormValues,
  version?: number,
): WorkerChecklistTemplateInput {
  return {
    code: values.code.toUpperCase(),
    name: values.name,
    description: values.description ?? undefined,
    kind: values.kind,
    trigger: values.trigger,
    status: values.status,
    isDefault: values.isDefault,
    items: values.items.map((item) => ({
      label: item.label,
      description: item.description ?? undefined,
      kind: item.kind,
      required: item.required,
      dueOffsetDays: item.dueOffsetDays,
      owner: item.owner,
      credentialTypeId:
        item.kind === "Credential" ? (item.credentialTypeId ?? undefined) : undefined,
      documentTypeId: item.kind === "Document" ? (item.documentTypeId ?? undefined) : undefined,
    })),
    version,
  };
}

export function ChecklistTemplatePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<WorkerChecklistTemplateRow>) {
  if (mode === "edit" && row) {
    return <ChecklistTemplateEditPanel open={open} onOpenChange={onOpenChange} row={row} />;
  }
  return <ChecklistTemplateCreatePanel open={open} onOpenChange={onOpenChange} />;
}

function ChecklistTemplateCreatePanel({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();

  const form = useForm<ChecklistTemplateFormValues>({
    resolver: zodResolver(checklistTemplateFormSchema) as Resolver<ChecklistTemplateFormValues>,
    defaultValues: buildChecklistTemplateDefaults(null),
  });

  return (
    <FormCreatePanel<ChecklistTemplateFormValues, WorkerChecklistTemplateRow>
      open={open}
      onOpenChange={onOpenChange}
      title={t("Checklist Template")}
      description={t(
        "Lay out the steps a worker goes through when they join or leave, and who owns each one.",
      )}
      queryKey={WORKER_CHECKLIST_TEMPLATE_LIST_KEY}
      form={form}
      size="lg"
      formComponent={<ChecklistTemplateForm isEdit={false} />}
      mutationFn={async (values) => {
        await createWorkerChecklistTemplate(toChecklistTemplateInput(values));
        return values;
      }}
    />
  );
}

function ChecklistTemplateEditPanel({
  open,
  onOpenChange,
  row,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  row: WorkerChecklistTemplateRow;
}) {
  const t = useT();

  const formRow = {
    ...row,
    ...buildChecklistTemplateDefaults(row),
  } as unknown as WorkerChecklistTemplateRow & Record<string, unknown>;
  const form = useForm<ChecklistTemplateFormValues>({
    resolver: zodResolver(checklistTemplateFormSchema) as Resolver<ChecklistTemplateFormValues>,
    defaultValues: buildChecklistTemplateDefaults(row),
  });

  return (
    <FormEditPanel<
      ChecklistTemplateFormValues,
      WorkerChecklistTemplateRow & Record<string, unknown>
    >
      open={open}
      onOpenChange={onOpenChange}
      row={formRow}
      title={t("Checklist Template")}
      fieldKey="code"
      queryKey={WORKER_CHECKLIST_TEMPLATE_LIST_KEY}
      form={form}
      size="lg"
      formComponent={<ChecklistTemplateForm isEdit openChecklistCount={row.openChecklistCount} />}
      mutationFn={async (values) => {
        await updateWorkerChecklistTemplate(row.id, toChecklistTemplateInput(values, row.version));
        return values;
      }}
    />
  );
}
