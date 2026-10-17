import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import {
  SegmentedControl,
  type SegmentedControlItem,
} from "@trenova/shared/components/ui/segmented-control";
import { organization } from "@/lib/queries/organization";
import { getAvailableOperations, getAvailableResources } from "@/lib/role-api";
import { cn } from "@trenova/shared/lib/utils";
import { apiService } from "@/services/api";
import type { AccessPolicy } from "@trenova/shared/types/iam";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Edit02Icon,
  PlusIcon,
  ShieldTickIcon,
  ShieldXIcon,
  Trash01Icon,
} from "@trenova/shared/components/icons";
import { memo, useCallback, useMemo, useState } from "react";
import { AccessPolicyPanel, type AccessPolicyPanelMode } from "./policies/policy-panel";
import { EmptyState, ErrorState, PanelSearch, PanelToolbar, RowSkeleton, SecuritySection } from "./shared";
import { RoleActivationSetting } from "./role-activation-setting";
import { useSecurityAccess } from "./use-security-access";

type EffectFilter = "all" | AccessPolicy["effect"];

export function PoliciesTab({ organizationId }: { organizationId: string }) {
  const t = useT();

  const queryClient = useQueryClient();
  const access = useSecurityAccess().accessPolicies;
  const policiesQuery = useQuery(organization.accessPolicies(organizationId));
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
  const [effectFilter, setEffectFilter] = useState<EffectFilter>("all");
  const effectFilterItems: SegmentedControlItem<EffectFilter>[] = [
    { value: "all", label: t("All") },
    { value: "allow", label: t("Allow") },
    { value: "deny", label: t("Deny") },
  ];

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
        queryKey: organization.accessPolicies(organizationId).queryKey,
      }),
    [organizationId, queryClient],
  );

  const { mutate: removeAccessPolicy } = useMutation({
    mutationFn: async (policyId: string) =>
      apiService.organizationService.deleteAccessPolicy(organizationId, policyId),
    onSuccess: async () => {
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
    <div className="flex flex-col gap-8">
      <RoleActivationSetting organizationId={organizationId} />
      <SecuritySection
        title={t("Access policies")}
        icon={<ShieldTickIcon />}
        count={policies.length}
        help={t(
          "Policies are checked in priority order, lowest number first, and a deny wins over an allow at the same priority. They narrow what roles grant; they never grant more.",
        )}
        action={
          access.create ? (
            <Button size="sm" onClick={createPolicy}>
              <PlusIcon />
              {t("New policy")}
            </Button>
          ) : null
        }
      >
        {policies.length > 0 ? (
          <PanelToolbar>
            <SegmentedControl
              aria-label={t("Filter by effect")}
              items={effectFilterItems}
              value={effectFilter}
              onValueChange={setEffectFilter}
            />
            <PanelSearch
              value={search}
              onChange={setSearch}
              placeholder={t("Search policies or resources")}
            />
          </PanelToolbar>
        ) : null}
        {policiesQuery.isLoading ? (
          <RowSkeleton rows={3} label={t("Loading access policies")} />
        ) : policiesQuery.isError ? (
          <ErrorState label={t("Access policies could not be loaded.")} />
        ) : filteredPolicies.length > 0 ? (
          <div className="divide-border-subtle divide-y">
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
            icon={<ShieldTickIcon />}
            label={
              policies.length === 0 ? t("No access policies configured") : t("No policies found")
            }
            description={
              policies.length === 0
                ? t("Create allow and deny policies to control sensitive operations.")
                : t("Adjust filters to find a policy.")
            }
          />
        )}
      </SecuritySection>

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
  const t = useT();
  const access = useSecurityAccess().accessPolicies;

  const conditionCount = Object.keys(policy.conditions).length;
  const isAllow = policy.effect === "allow";
  const EffectIcon = isAllow ? ShieldTickIcon : ShieldXIcon;
  const effectLabel = isAllow ? t("Allow") : t("Deny");
  const resourceLabel = resourceName || policy.resource;

  return (
    <div
      className={cn(
        "group grid gap-3 py-2.5 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-center",
        !policy.enabled && "text-muted-foreground",
      )}
    >
      <div className="flex min-w-0 gap-3">
        <span
          className={cn(
            "bg-muted/35 text-foreground/70 mt-0.5 inline-flex size-7 shrink-0 items-center justify-center rounded-md border",
            !policy.enabled && "text-muted-foreground",
          )}
        >
          <EffectIcon
            className={cn(
              "size-3.5",
              isAllow ? "text-success-foreground" : "text-danger-foreground",
              !policy.enabled && "text-muted-foreground",
            )}
          />
        </span>
        <div className="min-w-0">
          <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5">
            <span className="text-foreground truncate text-sm font-medium">{policy.name}</span>
            {!policy.enabled && (
              <span className="text-muted-foreground text-xs font-medium">{t("Disabled")}</span>
            )}
          </div>
          <div className="text-muted-foreground mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs">
            <span
              className={cn(
                "font-medium",
                isAllow ? "text-success-foreground" : "text-danger-foreground",
                !policy.enabled && "text-muted-foreground",
              )}
            >
              {effectLabel}
            </span>
            <span aria-hidden="true">/</span>
            <span>{t("Priority {0}", policy.priority)}</span>
            <span aria-hidden="true">/</span>
            <span className="text-muted-foreground text-xs">
              {conditionCount === 0
                ? t("No conditions")
                : t("{0, plural, one {# condition} other {# conditions}}", conditionCount)}
            </span>
          </div>
          <div className="mt-2 flex min-w-0 flex-wrap items-center gap-x-1.5 gap-y-1 text-xs">
            <span className="text-muted-foreground">{t("Scope")}</span>
            <span className="text-foreground/80 max-w-full truncate font-mono text-xs">
              {resourceLabel}
            </span>
            <span className="text-muted-foreground" aria-hidden="true">
              /
            </span>
            <span className="text-foreground/80 font-mono text-xs">{policy.operation}</span>
          </div>
        </div>
      </div>
      <div className="flex items-center justify-end gap-1">
        {access.update ? (
          <Button size="sm" variant="ghost" onClick={() => onEditPolicy(policy)}>
            <Edit02Icon />
            {t("Edit")}
          </Button>
        ) : null}
        {access.remove ? (
          <Button
            size="icon-sm"
            variant="ghost"
            className="text-destructive hover:text-destructive"
            aria-label={t("Delete {0}", policy.name)}
            onClick={() => onDeletePolicy(policy.id)}
          >
            <Trash01Icon />
          </Button>
        ) : null}
      </div>
    </div>
  );
});
