import { useT } from "@trenova/shared/i18n/use-t";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import {
  createWorkerCredentialType,
  updateWorkerCredentialType,
  WORKER_CREDENTIAL_TYPE_LIST_KEY,
  type WorkerCredentialTypeRow,
} from "@/lib/graphql/worker-credential";
import type { WorkerCredentialTypeInput } from "@trenova/graphql/generated/graphql";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import {
  credentialTypeFormSchema,
  type CredentialTypeFormValues,
} from "@trenova/shared/types/worker-credential";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm, type Resolver } from "react-hook-form";
import { CredentialTypeForm } from "./credential-type-form";

export function buildCredentialTypeDefaults(
  row?: WorkerCredentialTypeRow | null,
): CredentialTypeFormValues {
  if (!row) {
    return {
      code: "",
      name: "",
      description: null,
      category: "Certification",
      status: "Active",
      isRequired: false,
      requiredForDriverTypes: [],
      renewalWindowDays: 30,
      validityMonths: null,
      requiresNumber: false,
      requiresDocument: false,
    };
  }
  return {
    code: row.code,
    name: row.name,
    description: row.description ?? null,
    category: row.category,
    status: row.status === "Inactive" ? "Inactive" : "Active",
    isRequired: row.isRequired,
    requiredForDriverTypes: [...row.requiredForDriverTypes],
    renewalWindowDays: row.renewalWindowDays,
    validityMonths: row.validityMonths ?? null,
    requiresNumber: row.requiresNumber,
    requiresDocument: row.requiresDocument,
  };
}

export function toCredentialTypeInput(
  values: CredentialTypeFormValues,
  version?: number,
): WorkerCredentialTypeInput {
  return {
    code: values.code.toUpperCase(),
    name: values.name,
    description: values.description ?? undefined,
    category: values.category,
    status: values.status,
    isRequired: values.isRequired,
    requiredForDriverTypes: values.isRequired ? values.requiredForDriverTypes : [],
    renewalWindowDays: values.renewalWindowDays,
    validityMonths: values.validityMonths ?? undefined,
    requiresNumber: values.requiresNumber,
    requiresDocument: values.requiresDocument,
    version,
  };
}

export function CredentialTypePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<WorkerCredentialTypeRow>) {
  if (mode === "edit" && row) {
    return <CredentialTypeEditPanel open={open} onOpenChange={onOpenChange} row={row} />;
  }
  return <CredentialTypeCreatePanel open={open} onOpenChange={onOpenChange} />;
}

function CredentialTypeCreatePanel({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();

  const form = useForm<CredentialTypeFormValues>({
    resolver: zodResolver(credentialTypeFormSchema) as Resolver<CredentialTypeFormValues>,
    defaultValues: buildCredentialTypeDefaults(null),
  });

  return (
    <FormCreatePanel<CredentialTypeFormValues, WorkerCredentialTypeRow>
      open={open}
      onOpenChange={onOpenChange}
      title={t("Credential Type")}
      description={t(
        "Add a licence, endorsement or certificate workers can hold, and decide whether it is required.",
      )}
      queryKey={WORKER_CREDENTIAL_TYPE_LIST_KEY}
      form={form}
      size="lg"
      formComponent={<CredentialTypeForm isEdit={false} />}
      mutationFn={async (values) => {
        await createWorkerCredentialType(toCredentialTypeInput(values));
        return values;
      }}
    />
  );
}

function CredentialTypeEditPanel({
  open,
  onOpenChange,
  row,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  row: WorkerCredentialTypeRow;
}) {
  const t = useT();

  const formRow = {
    ...row,
    ...buildCredentialTypeDefaults(row),
  } as unknown as WorkerCredentialTypeRow & Record<string, unknown>;
  const form = useForm<CredentialTypeFormValues>({
    resolver: zodResolver(credentialTypeFormSchema) as Resolver<CredentialTypeFormValues>,
    defaultValues: buildCredentialTypeDefaults(row),
  });

  return (
    <FormEditPanel<CredentialTypeFormValues, WorkerCredentialTypeRow & Record<string, unknown>>
      open={open}
      onOpenChange={onOpenChange}
      row={formRow}
      title={t("Credential Type")}
      fieldKey="code"
      queryKey={WORKER_CREDENTIAL_TYPE_LIST_KEY}
      form={form}
      size="lg"
      formComponent={
        <CredentialTypeForm
          isEdit
          isSystem={row.isSystem}
          profileField={row.profileField}
          activeCredentialCount={row.activeCredentialCount}
        />
      }
      mutationFn={async (values) => {
        await updateWorkerCredentialType(row.id, toCredentialTypeInput(values, row.version));
        return values;
      }}
    />
  );
}
