import { Badge } from "@/components/ui/badge";
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
import { apiService } from "@/services/api";
import type { AccessPolicy } from "@/types/iam";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { PlusIcon, ShieldCheckIcon, Trash2Icon } from "lucide-react";
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
