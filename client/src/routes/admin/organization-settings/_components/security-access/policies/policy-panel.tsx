import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { OperationDefinition, ResourceDefinition } from "@/lib/role-api";
import { apiService } from "@/services/api";
import {
  accessPolicyFormSchema,
  type AccessPolicy,
  type AccessPolicyFormValues,
} from "@/types/iam";
import { zodResolver } from "@hookform/resolvers/zod";
import { type Resolver, useForm } from "react-hook-form";
import { conditionRowsToRecord, emptyPolicy, recordToConditionRows } from "../utils";
import { accessPolicyPanelQueryKey } from "./constants";
import { AccessPolicyForm } from "./policy-form";

export type AccessPolicyPanelMode = "create" | "edit";

type AccessPolicyRecord = AccessPolicyFormValues & Record<string, unknown>;

type AccessPolicyPanelProps = {
  organizationId: string;
  mode: AccessPolicyPanelMode;
  open: boolean;
  policy: AccessPolicy | null;
  resources: ResourceDefinition[];
  operations: OperationDefinition[];
  resourcesLoading: boolean;
  operationsLoading: boolean;
  onOpenChange: (open: boolean) => void;
  onSaved: () => Promise<void>;
};

function toAccessPolicyFormValues(policy: AccessPolicy): AccessPolicyFormValues {
  return {
    ...policy,
    conditionRows: recordToConditionRows(policy.conditions),
  };
}

function toAccessPolicy(values: AccessPolicyFormValues): AccessPolicy {
  const { conditionRows, ...policyValues } = values;

  return {
    ...emptyPolicy,
    ...policyValues,
    conditions: conditionRowsToRecord(conditionRows),
  };
}

export function AccessPolicyPanel({
  organizationId,
  mode,
  open,
  policy,
  resources,
  operations,
  resourcesLoading,
  operationsLoading,
  onOpenChange,
  onSaved,
}: AccessPolicyPanelProps) {
  const form = useForm<AccessPolicyFormValues>({
    resolver: zodResolver(accessPolicyFormSchema) as Resolver<AccessPolicyFormValues>,
    defaultValues: toAccessPolicyFormValues(emptyPolicy),
    mode: "onChange",
  });
  const queryKey = accessPolicyPanelQueryKey(organizationId);
  const formComponent = (
    <AccessPolicyForm
      resources={resources}
      operations={operations}
      resourcesLoading={resourcesLoading}
      operationsLoading={operationsLoading}
    />
  );

  if (mode === "edit") {
    return (
      <FormEditPanel<AccessPolicyFormValues, AccessPolicyRecord>
        open={open}
        onOpenChange={onOpenChange}
        row={policy ? (toAccessPolicyFormValues(policy) as AccessPolicyRecord) : null}
        form={form}
        queryKey={queryKey}
        title="Access Policy"
        fieldKey="name"
        formComponent={formComponent}
        mutationFn={async (values) => {
          const saved = await apiService.organizationService.updateAccessPolicy(
            organizationId,
            toAccessPolicy(values),
          );
          await onSaved();
          return toAccessPolicyFormValues(saved);
        }}
      />
    );
  }

  return (
    <FormCreatePanel<AccessPolicyFormValues, AccessPolicyRecord>
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      queryKey={queryKey}
      title="Access Policy"
      description="Create a priority-ordered authorization decision for a protected resource."
      formComponent={formComponent}
      mutationFn={async (values) => {
        const saved = await apiService.organizationService.createAccessPolicy(
          organizationId,
          toAccessPolicy(values),
        );
        await onSaved();
        return toAccessPolicyFormValues(saved);
      }}
    />
  );
}
