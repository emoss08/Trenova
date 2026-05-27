import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { queries } from "@/lib/queries";
import {
  getAvailableOperations,
  getAvailableResources,
  type OperationDefinition,
  type ResourceDefinition,
} from "@/lib/role-api";
import { apiService } from "@/services/api";
import type { AccessPolicy } from "@/types/iam";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { PlusIcon, SaveIcon, SearchIcon, ShieldCheckIcon, Trash2Icon, XIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { ConsoleToolbar, EmptyState, ErrorState, Field, RowSkeleton, ToggleRow } from "./shared";
import {
  conditionRowsToRecord,
  emptyPolicy,
  recordToConditionRows,
  type ConditionRow,
} from "./utils";

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
  const [policy, setPolicy] = useState<AccessPolicy>(emptyPolicy);
  const [conditions, setConditions] = useState<ConditionRow[]>([]);
  const [sheetOpen, setSheetOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [effectFilter, setEffectFilter] = useState("all");

  const resources = useMemo(
    () => (resourcesQuery.data ?? []).flatMap((category) => category.resources),
    [resourcesQuery.data],
  );
  const selectedResource = resources.find((resource) => resource.resource === policy.resource);
  const availableOperations = selectedResource?.operations.length
    ? selectedResource.operations
    : (operationsQuery.data ?? []);
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

  const saveMutation = useMutation({
    mutationFn: async (value: AccessPolicy) =>
      value.id
        ? apiService.organizationService.updateAccessPolicy(organizationId, value)
        : apiService.organizationService.createAccessPolicy(organizationId, value),
    onSuccess: async () => {
      toast.success("Access policy saved");
      setPolicy(emptyPolicy);
      setConditions([]);
      setSheetOpen(false);
      await queryClient.invalidateQueries({
        queryKey: queries.organization.accessPolicies(organizationId).queryKey,
      });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (policyId: string) =>
      apiService.organizationService.deleteAccessPolicy(organizationId, policyId),
    onSuccess: async () => {
      toast.success("Access policy removed");
      await queryClient.invalidateQueries({
        queryKey: queries.organization.accessPolicies(organizationId).queryKey,
      });
    },
  });

  const editPolicy = (item: AccessPolicy) => {
    setPolicy(item);
    setConditions(recordToConditionRows(item.conditions));
    setSheetOpen(true);
  };

  const createPolicy = () => {
    setPolicy(emptyPolicy);
    setConditions([]);
    setSheetOpen(true);
  };

  const savePolicy = () => {
    saveMutation.mutate({ ...policy, conditions: conditionRowsToRecord(conditions) });
  };

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
            <Select value={effectFilter} onValueChange={(value) => setEffectFilter(value ?? "all")}>
              <SelectTrigger className="h-8 w-32 bg-background text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All effects</SelectItem>
                <SelectItem value="allow">Allow</SelectItem>
                <SelectItem value="deny">Deny</SelectItem>
              </SelectContent>
            </Select>
            <Button onClick={createPolicy}>
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
          {filteredPolicies.map((item) => (
            <PolicyRow
              key={item.id}
              policy={item}
              resource={resources.find((resource) => resource.resource === item.resource)}
              onEdit={() => editPolicy(item)}
              onDelete={() => deleteMutation.mutate(item.id)}
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

      <Sheet open={sheetOpen} onOpenChange={setSheetOpen}>
        <SheetContent className="w-[calc(100vw-1rem)] overflow-hidden sm:max-w-xl">
          <SheetHeader className="border-b">
            <SheetTitle>{policy.id ? "Edit access policy" : "Add access policy"}</SheetTitle>
            <SheetDescription>
              Select a resource and operation, then define effect, priority, and optional
              conditions.
            </SheetDescription>
          </SheetHeader>
          <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-4">
            <PolicyForm
              policy={policy}
              resources={resources}
              operations={availableOperations}
              conditions={conditions}
              resourcesLoading={resourcesQuery.isLoading}
              operationsLoading={operationsQuery.isLoading}
              onPolicyChange={setPolicy}
              onConditionsChange={setConditions}
            />
          </div>
          <SheetFooter className="border-t">
            <Button
              onClick={savePolicy}
              isLoading={saveMutation.isPending}
              loadingText="Saving..."
              disabled={!policy.name.trim() || !policy.resource || !policy.operation}
            >
              <SaveIcon />
              Save policy
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </div>
  );
}

function PolicyRow({
  policy,
  resource,
  onEdit,
  onDelete,
}: {
  policy: AccessPolicy;
  resource?: ResourceDefinition;
  onEdit: () => void;
  onDelete: () => void;
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
          <span className="rounded bg-muted px-1.5 py-0.5">
            {resource?.displayName || policy.resource}
          </span>
          <span className="rounded bg-muted px-1.5 py-0.5">{policy.operation}</span>
          <span>
            {conditionCount} condition{conditionCount === 1 ? "" : "s"}
          </span>
        </div>
      </div>
      <div className="flex justify-end gap-2">
        <Button variant="outline" onClick={onEdit}>
          Edit
        </Button>
        <Button variant="destructive" onClick={onDelete}>
          <Trash2Icon />
          Delete
        </Button>
      </div>
    </div>
  );
}

function PolicyForm({
  policy,
  resources,
  operations,
  conditions,
  resourcesLoading,
  operationsLoading,
  onPolicyChange,
  onConditionsChange,
}: {
  policy: AccessPolicy;
  resources: ResourceDefinition[];
  operations: OperationDefinition[];
  conditions: ConditionRow[];
  resourcesLoading: boolean;
  operationsLoading: boolean;
  onPolicyChange: (policy: AccessPolicy) => void;
  onConditionsChange: (conditions: ConditionRow[]) => void;
}) {
  const [resourceSearch, setResourceSearch] = useState("");
  const [operationSearch, setOperationSearch] = useState("");
  const filteredResources = useMemo(() => {
    const query = resourceSearch.trim().toLowerCase();
    if (!query) return resources;
    return resources.filter((resource) =>
      [resource.displayName, resource.resource, resource.category, resource.description]
        .join(" ")
        .toLowerCase()
        .includes(query),
    );
  }, [resourceSearch, resources]);
  const filteredOperations = useMemo(() => {
    const query = operationSearch.trim().toLowerCase();
    if (!query) return operations;
    return operations.filter((operation) =>
      [operation.displayName, operation.operation, operation.description]
        .join(" ")
        .toLowerCase()
        .includes(query),
    );
  }, [operationSearch, operations]);

  return (
    <div className="space-y-4 py-4">
      <Field label="Name">
        <Input
          value={policy.name}
          placeholder="Require managed devices for billing exports"
          onChange={(event) => onPolicyChange({ ...policy, name: event.target.value })}
        />
      </Field>
      <div className="grid gap-3 sm:grid-cols-2">
        <SearchablePolicySelect
          label="Resource"
          value={policy.resource}
          search={resourceSearch}
          searchPlaceholder="Search resources"
          selectPlaceholder="Select resource"
          disabled={resourcesLoading}
          options={filteredResources.map((resource) => ({
            value: resource.resource,
            label: resource.displayName,
          }))}
          onSearchChange={setResourceSearch}
          onValueChange={(value) => onPolicyChange({ ...policy, resource: value, operation: "" })}
        />
        <SearchablePolicySelect
          label="Operation"
          value={policy.operation}
          search={operationSearch}
          searchPlaceholder="Search operations"
          selectPlaceholder="Select operation"
          disabled={operationsLoading || operations.length === 0}
          options={filteredOperations.map((operation) => ({
            value: operation.operation,
            label: operation.displayName || operation.operation,
          }))}
          onSearchChange={setOperationSearch}
          onValueChange={(value) => onPolicyChange({ ...policy, operation: value })}
        />
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        <Field label="Effect">
          <Select
            value={policy.effect}
            onValueChange={(value) =>
              onPolicyChange({ ...policy, effect: (value ?? "deny") as AccessPolicy["effect"] })
            }
          >
            <SelectTrigger className="w-full bg-background">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="deny">Deny</SelectItem>
              <SelectItem value="allow">Allow</SelectItem>
            </SelectContent>
          </Select>
        </Field>
        <Field label="Priority">
          <Input
            type="number"
            value={policy.priority}
            onChange={(event) =>
              onPolicyChange({ ...policy, priority: Number(event.target.value || 0) })
            }
          />
        </Field>
      </div>
      <ToggleRow
        label="Enabled"
        description="Evaluate this policy during access decisions."
        checked={policy.enabled}
        onCheckedChange={(enabled) => onPolicyChange({ ...policy, enabled })}
      />
      <ConditionsEditor conditions={conditions} onChange={onConditionsChange} />
    </div>
  );
}

function ConditionsEditor({
  conditions,
  onChange,
}: {
  conditions: ConditionRow[];
  onChange: (conditions: ConditionRow[]) => void;
}) {
  const addCondition = () => {
    onChange([...conditions, { id: `${Date.now()}-${conditions.length}`, key: "", value: "" }]);
  };

  return (
    <div className="space-y-2 rounded-lg border bg-muted/20 p-3">
      <div className="flex items-center justify-between gap-3">
        <div>
          <div className="text-sm font-medium">Conditions</div>
          <div className="text-xs text-muted-foreground">
            Optional key/value checks persisted with the policy.
          </div>
        </div>
        <Button size="sm" variant="outline" onClick={addCondition}>
          <PlusIcon />
          Add
        </Button>
      </div>
      {conditions.length > 0 ? (
        <div className="space-y-2">
          {conditions.map((condition, index) => (
            <div key={condition.id} className="grid gap-2 sm:grid-cols-[1fr_1fr_32px]">
              <Input
                value={condition.key}
                placeholder="claim"
                onChange={(event) => {
                  const next = [...conditions];
                  next[index] = { ...condition, key: event.target.value };
                  onChange(next);
                }}
              />
              <Input
                value={condition.value}
                placeholder="expected value"
                onChange={(event) => {
                  const next = [...conditions];
                  next[index] = { ...condition, value: event.target.value };
                  onChange(next);
                }}
              />
              <Button
                size="icon"
                variant="ghost"
                onClick={() => onChange(conditions.filter((item) => item.id !== condition.id))}
              >
                <XIcon />
              </Button>
            </div>
          ))}
        </div>
      ) : (
        <div className="rounded-md border border-dashed bg-background p-3 text-xs text-muted-foreground">
          No conditions. The policy applies whenever the resource and operation match.
        </div>
      )}
    </div>
  );
}

function SearchablePolicySelect({
  label,
  value,
  search,
  searchPlaceholder,
  selectPlaceholder,
  disabled,
  options,
  onSearchChange,
  onValueChange,
}: {
  label: string;
  value: string;
  search: string;
  searchPlaceholder: string;
  selectPlaceholder: string;
  disabled: boolean;
  options: Array<{ value: string; label: string }>;
  onSearchChange: (value: string) => void;
  onValueChange: (value: string) => void;
}) {
  return (
    <div className="grid gap-1 text-sm">
      <span className="font-medium">{label}</span>
      <div className="relative">
        <SearchIcon className="pointer-events-none absolute top-1/2 left-2 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          value={search}
          placeholder={searchPlaceholder}
          className="pl-8"
          disabled={disabled}
          onChange={(event) => onSearchChange(event.target.value)}
        />
      </div>
      <Select
        value={value}
        onValueChange={(nextValue) => onValueChange(nextValue ?? "")}
        disabled={disabled || options.length === 0}
      >
        <SelectTrigger className="w-full bg-background">
          <SelectValue placeholder={selectPlaceholder} />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            {options.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
    </div>
  );
}
