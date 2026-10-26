import { DataScopeSelect } from "@/components/permissions/data-scope-select";
import { VirtualRows, type VirtualRow } from "@/components/virtual-rows";
import type { RoleResourceGuide } from "@/lib/graphql/page-access";
import { grantReads, impliedOperations } from "@/lib/implied-operations";
import {
  presetLevels,
  resourceAccess,
  setAccessLevel,
  setDataScope,
  toggleOperation,
  type AccessLevel,
  type ResourceAccess,
} from "@/lib/role-access";
import { PERMISSION_CATALOG_STALE_TIME, permissionCatalog } from "@/lib/queries/permission-catalog";
import type { OperationDefinition, ResourceDefinition } from "@/lib/role-api";
import { bundleState } from "@/lib/role-bundles";
import {
  AlertTriangleIcon,
  ChevronDownIcon,
  ChevronRightIcon,
  SearchLgIcon,
  SearchXIcon,
  ShieldTickIcon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { AddPermission, DataScope, Operation } from "@trenova/shared/types/role";
import { useQuery } from "@tanstack/react-query";
import { useId, useMemo, useState, type ReactNode } from "react";
import { RoleResourceAccessSkeleton } from "./role-resource-access-skeleton";
import { ResourceGuideLine, useRoleResourceGuides } from "./role-resource-guide";

const ALL_VIEW = "__all__";
const GRANTED_VIEW = "__granted__";

type RoleResourceAccessProps = {
  permissions: AddPermission[];
  onPermissionsChange: (permissions: AddPermission[]) => void;
  /** The role's sensitivity ceiling, which decides what its reference views show. */
  maxSensitivity: string;
};

type ResourceGroup = {
  category: string;
  resources: ResourceDefinition[];
};

/**
 * Per-resource access for a role: categories down the left, and the resources
 * of the one chosen as a windowed list, each with its access level and its
 * operations a click away.
 */
export function RoleResourceAccess({
  permissions,
  onPermissionsChange,
  maxSensitivity,
}: RoleResourceAccessProps) {
  const t = useT();
  const [searchQuery, setSearchQuery] = useState("");
  const [view, setView] = useState(ALL_VIEW);
  const [expanded, setExpanded] = useState<ReadonlySet<string>>(new Set());

  const {
    data: categories = [],
    isLoading,
    isError,
    refetch,
  } = useQuery({
    ...permissionCatalog.resources(),
    staleTime: PERMISSION_CATALOG_STALE_TIME,
  });

  const { data: bundleCatalog } = useQuery({
    ...permissionCatalog.bundles(),
    staleTime: PERMISSION_CATALOG_STALE_TIME,
  });
  const bundleSources = useMemo(() => {
    const sources = new Map<string, string[]>();
    for (const bundle of bundleCatalog?.bundles ?? []) {
      if (bundleState(bundle, permissions) !== "full") {
        continue;
      }
      for (const grant of bundle.grants) {
        const labels = sources.get(grant.resource) ?? [];
        labels.push(bundle.label);
        sources.set(grant.resource, labels);
      }
    }
    return sources;
  }, [bundleCatalog?.bundles, permissions]);

  const permissionMap = useMemo(
    () => new Map(permissions.map((permission) => [permission.resource, permission])),
    [permissions],
  );

  const definitions = useMemo(
    () =>
      new Map(
        categories.flatMap((category) =>
          category.resources.map((resource) => [resource.resource, resource] as const),
        ),
      ),
    [categories],
  );

  const access = useMemo(() => {
    const out = new Map<string, ResourceAccess>();
    for (const [name, resource] of definitions) {
      out.set(name, resourceAccess(resource, permissionMap.get(name)?.operations ?? []));
    }
    return out;
  }, [definitions, permissionMap]);

  // What the role reads, as the engine evaluates it: a grant's read, an
  // operation that implies it, or the read of the resource's parent.
  const readable = useMemo(() => {
    const reads = (resource: string) => {
      const definition = definitions.get(resource);
      const granted = permissionMap.get(resource)?.operations ?? [];
      return definition !== undefined && grantReads(granted, definition.operations);
    };
    const out = new Set<string>();
    for (const [resource, definition] of definitions) {
      if (reads(resource) || (definition.parentResource && reads(definition.parentResource))) {
        out.add(resource);
      }
    }
    return out;
  }, [definitions, permissionMap]);

  const query = searchQuery.trim().toLowerCase();
  const isSearching = query.length > 0;
  const matches = (resource: ResourceDefinition) =>
    !isSearching ||
    resource.displayName.toLowerCase().includes(query) ||
    resource.resource.toLowerCase().includes(query) ||
    resource.description.toLowerCase().includes(query);

  const groups: ResourceGroup[] = categories
    .filter((category) => view === ALL_VIEW || view === GRANTED_VIEW || view === category.category)
    .map((category) => ({
      category: category.category,
      resources: category.resources.filter(
        (resource) =>
          matches(resource) && (view !== GRANTED_VIEW || permissionMap.has(resource.resource)),
      ),
    }))
    .filter((group) => group.resources.length > 0);

  const { guides, isPending: guidesPending } = useRoleResourceGuides(
    groups
      .filter((group) => group.resources.some((resource) => expanded.has(resource.resource)))
      .map((group) => group.resources.map((resource) => resource.resource)),
    maxSensitivity,
  );

  const railCategories = categories.map((category) => ({
    category: category.category,
    total: category.resources.length,
    granted: category.resources.filter((resource) => permissionMap.has(resource.resource)).length,
    matched: category.resources.filter(matches).length,
  }));
  const allCount = railCategories.reduce(
    (sum, item) => sum + (isSearching ? item.matched : item.total),
    0,
  );
  const grantedCount = categories.reduce(
    (sum, category) =>
      sum +
      category.resources.filter(
        (resource) => permissionMap.has(resource.resource) && matches(resource),
      ).length,
    0,
  );

  const setExpandedFor = (resource: string, open: boolean) => {
    setExpanded((previous) => {
      const next = new Set(previous);
      if (open) {
        next.add(resource);
      } else {
        next.delete(resource);
      }
      return next;
    });
  };

  const handleSetLevel = (resources: readonly ResourceDefinition[], level: AccessLevel) => {
    onPermissionsChange(setAccessLevel(permissions, resources, level));
  };

  const rows: VirtualRow[] = groups.flatMap((group) => [
    {
      key: `category:${group.category}`,
      sticky: true,
      render: () => (
        <GroupHeader
          category={group.category}
          granted={
            group.resources.filter((resource) => permissionMap.has(resource.resource)).length
          }
          total={group.resources.length}
          onSetLevel={(level) => handleSetLevel(group.resources, level)}
        />
      ),
    },
    ...group.resources.map((resource) => {
      const permission = permissionMap.get(resource.resource);
      return {
        key: resource.resource,
        render: () => (
          <ResourceRow
            resource={resource}
            permission={permission}
            access={access.get(resource.resource) ?? resourceAccess(resource, [])}
            reads={readable.has(resource.resource)}
            expanded={expanded.has(resource.resource)}
            sources={bundleSources.get(resource.resource)}
            guide={guides.get(resource.resource)}
            guidePending={guidesPending}
            onExpandedChange={(open) => setExpandedFor(resource.resource, open)}
            onSetLevel={(level) => handleSetLevel([resource], level)}
            onToggleOperation={(operation) =>
              onPermissionsChange(toggleOperation(permissions, resource.resource, operation))
            }
            onSetScope={(scope) =>
              onPermissionsChange(setDataScope(permissions, resource.resource, scope))
            }
          />
        ),
      };
    }),
  ]);

  if (isLoading) {
    return <RoleResourceAccessSkeleton label={t("Loading resources")} />;
  }

  if (isError) {
    return (
      <Alert variant="destructive" size="sm">
        <AlertTriangleIcon />
        <AlertDescription className="flex flex-wrap items-center gap-2">
          {t("Resources could not be loaded.")}
          <Button type="button" size="xs" variant="outline" onClick={() => void refetch()}>
            {t("Try again")}
          </Button>
        </AlertDescription>
      </Alert>
    );
  }

  const viewItems = [
    { value: ALL_VIEW, label: t("All resources") },
    { value: GRANTED_VIEW, label: t("Granted") },
    ...categories.map((category) => ({ value: category.category, label: category.category })),
  ];

  return (
    <div className="bg-card flex h-160 flex-col overflow-hidden rounded-lg border md:flex-row">
      <nav
        aria-label={t("Resource categories")}
        className="flex flex-col border-b md:w-56 md:shrink-0 md:border-r md:border-b-0"
      >
        <div className="p-2">
          <Input
            aria-label={t("Search resources")}
            leftElement={<SearchLgIcon className="text-foreground-subtle size-3.5" />}
            placeholder={t("Search resources...")}
            value={searchQuery}
            onChange={(event) => setSearchQuery(event.target.value)}
            rightElement={
              searchQuery ? (
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-xs"
                  aria-label={t("Clear search")}
                  onClick={() => setSearchQuery("")}
                  className="text-foreground-muted hover:text-foreground"
                >
                  <XCloseIcon className="size-3.5" />
                </Button>
              ) : null
            }
          />
        </div>
        <div className="px-2 pb-2 md:hidden">
          <Select value={view} onValueChange={(next) => setView(next as string)} items={viewItems}>
            <SelectTrigger aria-label={t("Resource category")} className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {viewItems.map((item) => (
                <SelectItem key={item.value} value={item.value}>
                  {item.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <ScrollArea maskVariant="card" className="hidden min-h-0 flex-1 md:flex">
          <ul className="flex flex-col gap-px px-2 pb-2">
            <RailItem
              label={t("All resources")}
              selected={view === ALL_VIEW}
              onSelect={() => setView(ALL_VIEW)}
              count={allCount}
            />
            <RailItem
              label={t("Granted")}
              selected={view === GRANTED_VIEW}
              onSelect={() => setView(GRANTED_VIEW)}
              count={
                <span className={cn(grantedCount > 0 && "text-foreground")}>{grantedCount}</span>
              }
            />
            <li role="separator" className="border-border-subtle mx-2 my-1.5 border-t" />
            {railCategories.map((item) => (
              <RailItem
                key={item.category}
                label={item.category}
                selected={view === item.category}
                dimmed={isSearching && item.matched === 0}
                onSelect={() => setView(item.category)}
                count={
                  isSearching ? (
                    item.matched
                  ) : (
                    <>
                      <span aria-hidden>
                        {item.granted > 0 ? (
                          <>
                            <span className="text-foreground">{item.granted}</span>/{item.total}
                          </>
                        ) : (
                          item.total
                        )}
                      </span>
                      <span className="sr-only">
                        {t("{0} of {1} granted", item.granted, item.total)}
                      </span>
                    </>
                  )
                }
              />
            ))}
          </ul>
        </ScrollArea>
      </nav>

      <div className="flex min-h-0 min-w-0 flex-1 flex-col">
        <VirtualRows
          key={view}
          rows={rows}
          estimateSize={50}
          initialHeight={600}
          mask="card"
          maskClassName="top-10"
          className="min-h-0 flex-1"
          aria-label={t("Resources")}
          empty={
            isSearching ? (
              <EmptyPane
                icon={<SearchXIcon className="text-foreground-subtle size-6" />}
                title={t("No resources found")}
                description={t(
                  "No resources match “{0}”. Try a different search term.",
                  searchQuery,
                )}
                action={
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => setSearchQuery("")}
                  >
                    {t("Clear search")}
                  </Button>
                }
              />
            ) : (
              <EmptyPane
                icon={<ShieldTickIcon className="text-foreground-subtle size-6" />}
                title={t("No resources granted yet")}
                description={t("Choose a capability, or set access on a resource in any category.")}
                action={
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => setView(ALL_VIEW)}
                  >
                    {t("Browse all resources")}
                  </Button>
                }
              />
            )
          }
        />
      </div>
    </div>
  );
}

function RailItem({
  label,
  count,
  selected,
  dimmed = false,
  onSelect,
}: {
  label: string;
  count: ReactNode;
  selected: boolean;
  dimmed?: boolean;
  onSelect: () => void;
}) {
  return (
    <li>
      <button
        type="button"
        aria-current={selected || undefined}
        onClick={onSelect}
        className={cn(
          "ui-focus-ring flex h-7 w-full items-center justify-between gap-2 rounded-md px-2 text-left text-sm transition-colors",
          selected ? "bg-surface-selected text-foreground" : "hover:bg-surface-hover",
          !selected && (dimmed ? "text-foreground-subtle" : "text-foreground-muted"),
        )}
      >
        <span className="truncate">{label}</span>
        <span className="text-foreground-subtle shrink-0 font-mono text-2xs tabular-nums">
          {count}
        </span>
      </button>
    </li>
  );
}

function EmptyPane({
  icon,
  title,
  description,
  action,
}: {
  icon: ReactNode;
  title: string;
  description: string;
  action: ReactNode;
}) {
  return (
    <div className="flex h-full flex-col items-center justify-center px-6 py-16 text-center">
      {icon}
      <p className="mt-3 text-sm font-medium">{title}</p>
      <p className="text-foreground-muted mt-1 max-w-sm text-xs">{description}</p>
      <div className="mt-4">{action}</div>
    </div>
  );
}

function GroupHeader({
  category,
  granted,
  total,
  onSetLevel,
}: {
  category: string;
  granted: number;
  total: number;
  onSetLevel: (level: AccessLevel) => void;
}) {
  const t = useT();

  return (
    <div className="bg-card flex h-10 items-center gap-3 border-b px-4">
      <h3 className="truncate text-sm font-semibold">{category}</h3>
      <span className="text-foreground-muted shrink-0 text-xs">
        {t("{0} of {1} granted", granted, total)}
      </span>
      <BulkAccessMenu count={total} onSelect={onSetLevel} />
    </div>
  );
}

function BulkAccessMenu({
  count,
  onSelect,
}: {
  count: number;
  onSelect: (level: AccessLevel) => void;
}) {
  const t = useT();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button type="button" variant="ghost" size="xs" className="ml-auto shrink-0">
            {t("Set all")}
            <ChevronDownIcon className="size-3.5" />
          </Button>
        }
      />
      <DropdownMenuContent side="bottom" align="end">
        <DropdownMenuGroup>
          <DropdownMenuLabel>
            {t("Set {0, plural, one {# resource} other {# resources}} to", count)}
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            title={t("Full access")}
            description={t("Every operation the resource offers")}
            onClick={() => onSelect("full")}
          />
          <DropdownMenuItem
            title={t("View only")}
            description={t("Read records, change nothing")}
            onClick={() => onSelect("view")}
          />
          <DropdownMenuItem
            title={t("No access")}
            description={t("Remove every operation")}
            onClick={() => onSelect("none")}
          />
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

type ResourceRowProps = {
  resource: ResourceDefinition;
  permission?: AddPermission;
  access: ResourceAccess;
  /** Whether the role reads the resource, directly, by implication or through its parent. */
  reads: boolean;
  expanded: boolean;
  sources?: string[];
  guide?: RoleResourceGuide;
  guidePending: boolean;
  onExpandedChange: (open: boolean) => void;
  onSetLevel: (level: AccessLevel) => void;
  onToggleOperation: (operation: Operation) => void;
  onSetScope: (scope: DataScope) => void;
};

function ResourceRow({
  resource,
  permission,
  access,
  reads,
  expanded,
  sources,
  guide,
  guidePending,
  onExpandedChange,
  onSetLevel,
  onToggleOperation,
  onSetScope,
}: ResourceRowProps) {
  const t = useT();
  const detailId = useId();

  return (
    <div className="border-border-subtle border-b" data-resource={resource.resource}>
      <div className="hover:bg-surface-hover flex items-center gap-3 px-4 py-2 transition-colors">
        <button
          type="button"
          aria-expanded={expanded}
          aria-controls={expanded ? detailId : undefined}
          onClick={() => onExpandedChange(!expanded)}
          className="ui-focus-ring flex min-w-0 flex-1 items-center gap-2.5 rounded-sm text-left"
        >
          <ChevronRightIcon
            className={cn(
              "text-foreground-subtle size-3.5 shrink-0 transition-transform",
              expanded && "rotate-90",
            )}
          />
          <span className="min-w-0">
            <span className="block truncate text-sm font-medium">{resource.displayName}</span>
            <span className="text-foreground-muted block truncate text-xs">
              {resource.description}
            </span>
          </span>
        </button>
        {permission ? (
          <DataScopeSelect
            resource={resource}
            value={permission.dataScope}
            onValueChange={onSetScope}
          />
        ) : null}
        <AccessLevelSelect
          resource={resource}
          access={access}
          onSelect={(level) => (level === "custom" ? onExpandedChange(true) : onSetLevel(level))}
        />
      </div>
      {expanded ? (
        <div
          id={detailId}
          className="bg-sunken border-border-subtle space-y-3 border-t py-3 pr-4 pl-10"
        >
          <ResourceGuideLine guide={guide} reads={reads} isPending={guidePending} />
          {sources?.length ? (
            <p className="text-foreground-muted text-xs">{t("From {0}", sources.join(", "))}</p>
          ) : null}
          <OperationList
            resource={resource}
            granted={permission?.operations ?? []}
            onToggle={onToggleOperation}
          />
        </div>
      ) : null}
    </div>
  );
}

function AccessLevelSelect({
  resource,
  access,
  onSelect,
}: {
  resource: ResourceDefinition;
  access: ResourceAccess;
  onSelect: (level: AccessLevel) => void;
}) {
  const t = useT();
  const levels = presetLevels(resource);
  const readOnly = resource.operations.length === 1 && resource.operations[0].operation === "read";
  const labels: Record<AccessLevel, string> = {
    none: t("No access"),
    view: t("View only"),
    full: readOnly ? t("View only") : t("Full access"),
    custom: t("Custom · {0} of {1}", access.reached, access.total),
  };
  const items = (["none", "view", "full", "custom"] as const).map((level) => ({
    value: level,
    label: labels[level],
  }));

  return (
    <Select
      value={access.level}
      onValueChange={(next) => onSelect(next as AccessLevel)}
      items={items}
    >
      <SelectTrigger
        aria-label={t("Access to {0}", resource.displayName)}
        className={cn("w-36 shrink-0", access.level === "none" && "text-foreground-subtle")}
      >
        <SelectValue />
      </SelectTrigger>
      <SelectContent align="end" alignItemWithTrigger={false}>
        {levels.map((level) => (
          <SelectItem key={level} value={level}>
            {labels[level]}
          </SelectItem>
        ))}
        {resource.operations.length > 1 ? (
          <>
            <SelectSeparator />
            <SelectItem value="custom">{t("Choose operations…")}</SelectItem>
          </>
        ) : null}
      </SelectContent>
    </Select>
  );
}

function OperationList({
  resource,
  granted,
  onToggle,
}: {
  resource: ResourceDefinition;
  granted: readonly Operation[];
  onToggle: (operation: Operation) => void;
}) {
  const t = useT();
  const implied = impliedOperations(granted, resource.operations);
  const displayNames = new Map(resource.operations.map((op) => [op.operation, op.displayName]));

  return (
    <fieldset>
      <legend className="sr-only">{t("Operations on {0}", resource.displayName)}</legend>
      <div className="grid gap-x-6 gap-y-2.5 sm:grid-cols-2">
        {resource.operations.map((op) => (
          <OperationOption
            key={op.operation}
            operation={op}
            held={granted.includes(op.operation as Operation)}
            impliedBy={implied
              .get(op.operation)
              ?.map((name) => displayNames.get(name) ?? name)
              .join(", ")}
            onToggle={() => onToggle(op.operation as Operation)}
          />
        ))}
      </div>
    </fieldset>
  );
}

function OperationOption({
  operation,
  held,
  impliedBy,
  onToggle,
}: {
  operation: OperationDefinition;
  held: boolean;
  /** The held operations that bring this one, when it is not held outright. */
  impliedBy?: string;
  onToggle: () => void;
}) {
  const t = useT();
  const included = !held && impliedBy !== undefined;

  return (
    <label
      className={cn("flex items-start gap-2.5", included ? "cursor-default" : "cursor-pointer")}
    >
      <Checkbox
        checked={held || included}
        disabled={included}
        onCheckedChange={onToggle}
        className="mt-0.5"
      />
      <span className="min-w-0">
        <span className="block text-sm font-medium">{operation.displayName}</span>
        <span className="text-foreground-muted block text-xs">
          {included ? t("Included with {0}", impliedBy) : t(operation.description)}
        </span>
      </span>
    </label>
  );
}
