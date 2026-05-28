import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { queries } from "@/lib/queries";
import { getAvailableOperations, getAvailableResources } from "@/lib/role-api";
import { cn } from "@/lib/utils";
import { apiService } from "@/services/api";
import type { AccessPolicy } from "@/types/iam";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { PencilIcon, PlusIcon, ShieldCheckIcon, ShieldXIcon, Trash2Icon } from "lucide-react";
import { memo, useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { effectFilterOptions } from "./policies/constants";
import { AccessPolicyPanel, type AccessPolicyPanelMode } from "./policies/policy-panel";
import { ConsoleToolbar, EmptyState, ErrorState, RowSkeleton } from "./shared";

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
  const [panelMode, setPanelMode] = useState<AccessPolicyPanelMode>("create");
  const [panelOpen, setPanelOpen] = useState(false);
  const [editingPolicy, setEditingPolicy] = useState<AccessPolicy | null>(null);
  const [search, setSearch] = useState("");
  const [effectFilter, setEffectFilter] = useState("all");
  const selectedOption =
    effectFilterOptions.find((option) => option.value === effectFilter) || null;

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
                {selectedOption?.color ? (
                  <span
                    className="size-2 shrink-0 rounded-full"
                    style={{ backgroundColor: selectedOption?.color }}
                  />
                ) : null}
                <SelectValue />
              </SelectTrigger>
              <SelectContent className="p-1">
                {effectFilterOptions.map((option) => (
                  <SelectItem
                    key={option.value}
                    value={option.value}
                    className="flex w-full flex-row items-center"
                  >
                    {option?.color ? (
                      <span
                        className="mt-1 size-2 shrink-0 rounded-full"
                        style={{ backgroundColor: option?.color }}
                      />
                    ) : null}
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

      <AccessPolicyPanel
        organizationId={organizationId}
        mode={panelMode}
        open={panelOpen}
        policy={editingPolicy}
        resources={resources}
        operations={operations}
        resourcesLoading={resourcesQuery.isLoading}
        operationsLoading={operationsQuery.isLoading}
        onOpenChange={setPanelOpen}
        onSaved={invalidateAccessPolicies}
      />
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
  const isAllow = policy.effect === "allow";
  const EffectIcon = isAllow ? ShieldCheckIcon : ShieldXIcon;
  const effectLabel = isAllow ? "Allow" : "Deny";
  const resourceLabel = resourceName || policy.resource;

  return (
    <div
      className={cn(
        "group grid gap-3 border-b bg-background px-3 py-2.5 transition-colors last:border-b-0 hover:bg-muted/30 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-center",
        !policy.enabled && "bg-muted/20 text-muted-foreground",
      )}
    >
      <div className="flex min-w-0 gap-3">
        <span
          className={cn(
            "mt-0.5 inline-flex size-7 shrink-0 items-center justify-center rounded-md border bg-muted/35 text-foreground/70",
            !policy.enabled && "text-muted-foreground",
          )}
        >
          <EffectIcon
            className={cn(
              "size-3.5",
              isAllow ? "text-emerald-700 dark:text-emerald-400" : "text-red-700 dark:text-red-400",
              !policy.enabled && "text-muted-foreground",
            )}
          />
        </span>
        <div className="min-w-0">
          <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5">
            <span className="truncate text-sm font-medium text-foreground">{policy.name}</span>
            {!policy.enabled && (
              <span className="text-xs font-medium text-muted-foreground">Disabled</span>
            )}
          </div>
          <div className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted-foreground">
            <span
              className={cn(
                "font-medium",
                isAllow
                  ? "text-emerald-700 dark:text-emerald-400"
                  : "text-red-700 dark:text-red-400",
                !policy.enabled && "text-muted-foreground",
              )}
            >
              {effectLabel}
            </span>
            <span aria-hidden="true">/</span>
            <span>Priority {policy.priority}</span>
            <span aria-hidden="true">/</span>
            <span className="text-xs text-muted-foreground">
              {conditionCount === 0
                ? "No conditions"
                : `${conditionCount} condition${conditionCount === 1 ? "" : "s"}`}
            </span>
          </div>
          <div className="mt-2 flex min-w-0 flex-wrap items-center gap-x-1.5 gap-y-1 text-xs">
            <span className="text-muted-foreground">Scope</span>
            <span className="max-w-full truncate font-mono text-[11px] text-foreground/80">
              {resourceLabel}
            </span>
            <span className="text-muted-foreground" aria-hidden="true">
              /
            </span>
            <span className="font-mono text-[11px] text-foreground/80">{policy.operation}</span>
          </div>
        </div>
      </div>
      <div className="flex items-center justify-end gap-1">
        <Button size="sm" variant="ghost" onClick={() => onEditPolicy(policy)}>
          <PencilIcon />
          Edit
        </Button>
        <Button
          size="icon-sm"
          variant="ghost"
          className="text-destructive hover:text-destructive"
          aria-label={`Delete ${policy.name}`}
          onClick={() => onDeletePolicy(policy.id)}
        >
          <Trash2Icon />
        </Button>
      </div>
    </div>
  );
});
