import {
  PermissionOperationAutocompleteField,
  PermissionResourceAutocompleteField,
} from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { queries } from "@/lib/queries";
import {
  getAvailableOperations,
  getAvailableResources,
  type OperationDefinition,
  type ResourceDefinition,
} from "@/lib/role-api";
import { apiService } from "@/services/api";
import type { SelectOption } from "@/types/fields";
import {
  accessPolicyFormSchema,
  type AccessPolicy,
  type AccessPolicyConditionRow,
  type AccessPolicyFormValues,
} from "@/types/iam";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { PlusIcon, ShieldCheckIcon, Trash2Icon, XIcon } from "lucide-react";
import { memo, useCallback, useMemo, useState } from "react";
import { type Resolver, useFieldArray, useForm, useFormContext, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { ConsoleToolbar, EmptyState, ErrorState, RowSkeleton } from "./shared";
import { conditionRowsToRecord, emptyPolicy, recordToConditionRows } from "./utils";

type AccessPolicyPanelMode = "create" | "edit";
type AccessPolicyRecord = AccessPolicyFormValues & Record<string, unknown>;

const accessPolicyQueryKey = (organizationId: string) => `access-policy-list:${organizationId}`;

const effectFilterOptions = [
  { value: "all", label: "All effects" },
  { value: "allow", label: "Allow" },
  { value: "deny", label: "Deny" },
] as const;

const policyEffectOptions: SelectOption[] = [
  {
    value: "deny",
    label: "Deny",
    description: "Block matching access requests before lower-priority allow policies apply.",
  },
  {
    value: "allow",
    label: "Allow",
    description: "Permit matching access requests when no higher-priority deny policy applies.",
  },
];

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

export function PoliciesTab({ organizationId }: { organizationId: string }) {
  const queryClient = useQueryClient();
  const policiesQuery = useQuery(queries.organization.accessPolicies(organizationId));
  const resourcesQuery = useQuery({
    queryKey: ["permissions", "resources"],
    queryFn: getAvailableResources,
  });
  const operationsQuery = useQuery({
    queryKey: ["permissions", "operations"],
    queryFn: getAvailableOperations,
  });
  const createForm = useForm<AccessPolicyFormValues>({
    resolver: zodResolver(accessPolicyFormSchema) as Resolver<AccessPolicyFormValues>,
    defaultValues: toAccessPolicyFormValues(emptyPolicy),
    mode: "onChange",
  });
  const editForm = useForm<AccessPolicyFormValues>({
    resolver: zodResolver(accessPolicyFormSchema) as Resolver<AccessPolicyFormValues>,
    defaultValues: toAccessPolicyFormValues(emptyPolicy),
    mode: "onChange",
  });
  const [panelMode, setPanelMode] = useState<AccessPolicyPanelMode>("create");
  const [panelOpen, setPanelOpen] = useState(false);
  const [editingPolicy, setEditingPolicy] = useState<AccessPolicy | null>(null);
  const [search, setSearch] = useState("");
  const [effectFilter, setEffectFilter] = useState("all");

  const resources = useMemo(
    () => (resourcesQuery.data ?? []).flatMap((category) => category.resources),
    [resourcesQuery.data],
  );
  const operations = useMemo(() => operationsQuery.data ?? [], [operationsQuery.data]);
  const resourceDisplayNames = useMemo(
    () => new Map(resources.map((resource) => [resource.resource, resource.displayName])),
    [resources],
  );
  const policies = useMemo(
    () => [...(policiesQuery.data ?? [])].sort((left, right) => left.priority - right.priority),
    [policiesQuery.data],
  );
  const filteredPolicies = useMemo(() => {
    const query = search.trim().toLowerCase();
    return policies.filter((item) => {
      const matchesEffect = effectFilter === "all" || item.effect === effectFilter;
      const matchesSearch =
        !query ||
        [item.name, item.resource, item.operation, item.effect]
          .join(" ")
          .toLowerCase()
          .includes(query);
      return matchesEffect && matchesSearch;
    });
  }, [effectFilter, policies, search]);

  const invalidateAccessPolicies = useCallback(
    async () =>
      queryClient.invalidateQueries({
        queryKey: queries.organization.accessPolicies(organizationId).queryKey,
      }),
    [organizationId, queryClient],
  );

  const { mutate: removeAccessPolicy } = useMutation({
    mutationFn: async (policyId: string) =>
      apiService.organizationService.deleteAccessPolicy(organizationId, policyId),
    onSuccess: async () => {
      toast.success("Access policy removed");
      await invalidateAccessPolicies();
    },
  });

  const createPolicy = useCallback(() => {
    setPanelMode("create");
    setEditingPolicy(null);
    setPanelOpen(true);
  }, []);

  const editPolicy = useCallback((policy: AccessPolicy) => {
    setPanelMode("edit");
    setEditingPolicy(policy);
    setPanelOpen(true);
  }, []);

  const deletePolicy = useCallback(
    (policyId: string) => removeAccessPolicy(policyId),
    [removeAccessPolicy],
  );

  return (
    <div className="space-y-3">
      <ConsoleToolbar
        title="Access policies"
        description="Priority-ordered authorization decisions for protected resources."
        search={search}
        onSearchChange={setSearch}
        searchPlaceholder="Search policies or resources"
        action={
          <div className="flex flex-wrap gap-2">
            <Select
              value={effectFilter}
              items={effectFilterOptions}
              onValueChange={(value) => setEffectFilter(value ?? "all")}
            >
              <SelectTrigger className="h-7 w-32 text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {effectFilterOptions.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button size="sm" onClick={createPolicy}>
              <PlusIcon />
              Add policy
            </Button>
          </div>
        }
      />

      {policiesQuery.isLoading ? (
        <RowSkeleton rows={4} />
      ) : policiesQuery.isError ? (
        <ErrorState label="Access policies could not be loaded." />
      ) : filteredPolicies.length > 0 ? (
        <div className="overflow-hidden rounded-lg border bg-background">
          {filteredPolicies.map((policy) => (
            <PolicyRow
              key={policy.id}
              policy={policy}
              resourceName={resourceDisplayNames.get(policy.resource)}
              onEditPolicy={editPolicy}
              onDeletePolicy={deletePolicy}
            />
          ))}
        </div>
      ) : (
        <EmptyState
          icon={<ShieldCheckIcon />}
          label={policies.length === 0 ? "No access policies configured" : "No policies found"}
          description={
            policies.length === 0
              ? "Create allow and deny policies to control sensitive operations."
              : "Adjust filters to find a policy."
          }
        />
      )}

      {panelMode === "edit" ? (
        <FormEditPanel<AccessPolicyFormValues, AccessPolicyRecord>
          open={panelOpen}
          onOpenChange={setPanelOpen}
          row={
            editingPolicy ? (toAccessPolicyFormValues(editingPolicy) as AccessPolicyRecord) : null
          }
          form={editForm}
          queryKey={accessPolicyQueryKey(organizationId)}
          title="Access Policy"
          fieldKey="name"
          size="lg"
          formComponent={
            <AccessPolicyForm
              resources={resources}
              operations={operations}
              resourcesLoading={resourcesQuery.isLoading}
              operationsLoading={operationsQuery.isLoading}
            />
          }
          mutationFn={async (values) => {
            const saved = await apiService.organizationService.updateAccessPolicy(
              organizationId,
              toAccessPolicy(values),
            );
            await invalidateAccessPolicies();
            return toAccessPolicyFormValues(saved);
          }}
        />
      ) : (
        <FormCreatePanel<AccessPolicyFormValues, AccessPolicyRecord>
          open={panelOpen}
          onOpenChange={setPanelOpen}
          form={createForm}
          queryKey={accessPolicyQueryKey(organizationId)}
          title="Access Policy"
          description="Create a priority-ordered authorization decision for a protected resource."
          size="lg"
          formComponent={
            <AccessPolicyForm
              resources={resources}
              operations={operations}
              resourcesLoading={resourcesQuery.isLoading}
              operationsLoading={operationsQuery.isLoading}
            />
          }
          mutationFn={async (values) => {
            const saved = await apiService.organizationService.createAccessPolicy(
              organizationId,
              toAccessPolicy(values),
            );
            await invalidateAccessPolicies();
            return toAccessPolicyFormValues(saved);
          }}
        />
      )}
    </div>
  );
}

const PolicyRow = memo(function PolicyRow({
  policy,
  resourceName,
  onEditPolicy,
  onDeletePolicy,
}: {
  policy: AccessPolicy;
  resourceName?: string;
  onEditPolicy: (policy: AccessPolicy) => void;
  onDeletePolicy: (policyId: string) => void;
}) {
  const conditionCount = Object.keys(policy.conditions).length;

  return (
    <div className="grid gap-3 border-b p-3 last:border-b-0 lg:grid-cols-[minmax(0,1fr)_160px] lg:items-center">
      <div className="min-w-0 space-y-2">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant={policy.effect === "allow" ? "active" : "inactive"}>
            {policy.effect === "allow" ? "Allow" : "Deny"}
          </Badge>
          <span className="font-medium">{policy.name}</span>
          <Badge variant={policy.enabled ? "info" : "outline"}>Priority {policy.priority}</Badge>
          {!policy.enabled && <Badge variant="outline">Disabled</Badge>}
        </div>
        <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
          <span className="rounded bg-muted px-1.5 py-0.5">{resourceName || policy.resource}</span>
          <span className="rounded bg-muted px-1.5 py-0.5">{policy.operation}</span>
          <span>
            {conditionCount} condition{conditionCount === 1 ? "" : "s"}
          </span>
        </div>
      </div>
      <div className="flex justify-end gap-2">
        <Button size="sm" variant="outline" onClick={() => onEditPolicy(policy)}>
          Edit
        </Button>
        <Button size="sm" variant="destructive" onClick={() => onDeletePolicy(policy.id)}>
          <Trash2Icon />
          Delete
        </Button>
      </div>
    </div>
  );
});

function AccessPolicyForm({
  resources,
  operations,
  resourcesLoading,
  operationsLoading,
}: {
  resources: ResourceDefinition[];
  operations: OperationDefinition[];
  resourcesLoading: boolean;
  operationsLoading: boolean;
}) {
  const { control, setValue } = useFormContext<AccessPolicyFormValues>();
  const selectedResourceName = useWatch({ control, name: "resource" });
  const selectedResource = resources.find((resource) => resource.resource === selectedResourceName);
  const availableOperations = selectedResource?.operations.length
    ? selectedResource.operations
    : operations;

  return (
    <>
      <FormSection title="Policy Decision">
        <FormGroup cols={2}>
          <FormControl cols="full">
            <InputField
              control={control}
              rules={{ required: true }}
              name="name"
              label="Policy Name"
              placeholder="Require managed devices for billing exports"
              description="Administrative name that explains when this policy should match."
              maxLength={120}
            />
          </FormControl>
          <FormControl>
            <PermissionResourceAutocompleteField<AccessPolicyFormValues>
              control={control}
              rules={{ required: true }}
              name="resource"
              resources={resources}
              disabled={resourcesLoading}
              onValueChange={() =>
                setValue("operation", "", { shouldDirty: true, shouldValidate: true })
              }
            />
          </FormControl>
          <FormControl>
            <PermissionOperationAutocompleteField<AccessPolicyFormValues>
              control={control}
              rules={{ required: true }}
              name="operation"
              operations={availableOperations}
              disabled={operationsLoading || availableOperations.length === 0}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="effect"
              label="Effect"
              placeholder="Select effect"
              description="Authorization decision returned when this policy matches."
              options={policyEffectOptions}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              rules={{ required: true, min: 0 }}
              name="priority"
              label="Priority"
              placeholder="100"
              description="Lower numbers evaluate first. Use gaps to leave room for future rules."
              min={0}
            />
          </FormControl>
          <FormControl cols="full">
            <SwitchField
              control={control}
              name="enabled"
              label="Enabled"
              description="Evaluate this policy during access decisions."
              outlined
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <PolicyConditionsSection />
    </>
  );
}

function PolicyConditionsSection() {
  const { control } = useFormContext<AccessPolicyFormValues>();
  const { append, fields, remove } = useFieldArray({
    control,
    name: "conditionRows",
    keyName: "fieldId",
  });

  const addCondition = () => {
    append({
      id: `${Date.now()}-${fields.length}`,
      key: "",
      value: "",
    });
  };

  return (
    <FormSection
      title="Conditions"
      description="Optional claim or context key/value checks persisted with the policy."
      className="border-t py-2"
    >
      <div className="space-y-3">
        <div className="flex justify-end">
          <Button type="button" size="sm" variant="outline" onClick={addCondition}>
            <PlusIcon />
            Add condition
          </Button>
        </div>
        {fields.length > 0 ? (
          <div className="space-y-2">
            {fields.map((field, index) => (
              <ConditionRowFields
                key={field.fieldId}
                index={index}
                condition={field}
                onRemove={() => remove(index)}
              />
            ))}
          </div>
        ) : (
          <div className="rounded-md border border-dashed bg-muted/30 p-3 text-xs text-muted-foreground">
            No conditions. The policy applies whenever the resource and operation match.
          </div>
        )}
      </div>
    </FormSection>
  );
}

function ConditionRowFields({
  index,
  condition,
  onRemove,
}: {
  index: number;
  condition: AccessPolicyConditionRow;
  onRemove: () => void;
}) {
  const { control } = useFormContext<AccessPolicyFormValues>();

  return (
    <div className="grid gap-2 sm:grid-cols-[1fr_1fr_32px] items-center">
      <InputField
        control={control}
        name={`conditionRows.${index}.key`}
        label="Condition Key"
        placeholder="Claim"
        description="Claim, signal, or context key to evaluate."
        defaultValue={condition.key}
      />
      <InputField
        control={control}
        name={`conditionRows.${index}.value`}
        label="Condition Value"
        placeholder="Expected value"
        description="Expected value for the configured condition key."
        defaultValue={condition.value}
      />
      <div className="flex items-end pb-0.5">
        <Button type="button" size="icon-sm" variant="ghost" onClick={onRemove}>
          <XIcon />
        </Button>
      </div>
    </div>
  );
}
