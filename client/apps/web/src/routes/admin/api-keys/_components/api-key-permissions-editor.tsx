import {
  Accordion,
  AccordionHeader,
  AccordionItem,
  AccordionPanel,
  AccordionTrigger,
} from "@/components/animate-ui/primitives/base/accordion";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { dataScopeChoices } from "@/lib/choices";
import type { ResourceCategory, ResourceDefinition } from "@/lib/role-api";
import { cn, pluralize } from "@/lib/utils";
import { apiService } from "@/services/api";
import type { ApiKeyPermissionInput } from "@/types/api-key";
import type { DataScope } from "@/types/role";
import { useQuery } from "@tanstack/react-query";
import { CheckIcon, ChevronDownIcon, EyeIcon, SearchIcon, XIcon, ZapIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import type { ApiKeyPanelFormValues } from "./api-key-panel";

type BulkMode = "read" | "write" | "full" | "clear";

export function APIKeyPermissionsEditor() {
  const { setValue, control } = useFormContext<ApiKeyPanelFormValues>();
  const permissions = useWatch({ control, name: "permissions" });
  const [searchQuery, setSearchQuery] = useState("");
  const [expandedCategories, setExpandedCategories] = useState<string[]>([]);

  const { data: resourceCategories = [], isLoading } = useQuery({
    queryKey: ["api-key-allowed-resources"],
    queryFn: () => apiService.apiKeyService.getAllowedResources(),
    staleTime: 1000 * 60 * 30,
  });

  const permissionMap = useMemo(() => createPermissionMap(permissions), [permissions]);

  const allResources = useMemo(
    () => resourceCategories.flatMap((category) => category.resources),
    [resourceCategories],
  );

  const filteredCategories = useMemo(() => {
    const query = searchQuery.trim().toLowerCase();

    if (!query) {
      return resourceCategories;
    }

    return resourceCategories
      .map((category) => ({
        ...category,
        resources: category.resources.filter((resource) =>
          [resource.displayName, resource.resource, resource.description].some((value) =>
            value.toLowerCase().includes(query),
          ),
        ),
      }))
      .filter((category) => category.resources.length > 0);
  }, [resourceCategories, searchQuery]);

  const selectionSummary = useMemo(
    () => summarizeSelection(allResources, permissionMap),
    [allResources, permissionMap],
  );

  const updatePermissions = useCallback(
    (nextPermissions: ApiKeyPermissionInput[]) => {
      setValue("permissions", nextPermissions, {
        shouldDirty: true,
        shouldTouch: true,
        shouldValidate: true,
      });
    },
    [setValue],
  );

  const applyBulkPreset = useCallback(
    (resources: ResourceDefinition[], mode: BulkMode) => {
      updatePermissions(buildBulkPermissions(resources, mode));
    },
    [updatePermissions],
  );

  const applyCategoryPreset = useCallback(
    (category: ResourceCategory, mode: BulkMode) => {
      const nextMap = new Map(permissionMap);

      for (const resource of category.resources) {
        nextMap.delete(resource.resource);

        if (mode === "clear") {
          continue;
        }

        const operations = getOperationsForMode(resource, mode);
        if (operations.length > 0) {
          nextMap.set(resource.resource, {
            resource: resource.resource,
            operations,
            dataScope: permissionMap.get(resource.resource)?.dataScope ?? "organization",
          });
        }
      }

      updatePermissions(Array.from(nextMap.values()));
    },
    [permissionMap, updatePermissions],
  );

  const handleToggleResource = useCallback(
    (resource: ResourceDefinition) => {
      const existing = permissionMap.get(resource.resource);
      if (existing) {
        updatePermissions(permissions.filter((p) => p.resource !== resource.resource));
      } else {
        updatePermissions([
          ...permissions,
          {
            resource: resource.resource,
            operations: ["read"],
            dataScope: "organization",
          },
        ]);
      }
    },
    [permissions, permissionMap, updatePermissions],
  );

  const handleToggleOperation = useCallback(
    (resourceKey: string, operation: string) => {
      const existing = permissionMap.get(resourceKey);
      if (!existing) return;

      const hasOp = existing.operations.includes(operation);
      let newOps: string[];

      if (hasOp) {
        newOps = existing.operations.filter((op) => op !== operation);
        if (newOps.length === 0) {
          updatePermissions(permissions.filter((p) => p.resource !== resourceKey));
          return;
        }
      } else {
        newOps = [...existing.operations, operation];
      }

      updatePermissions(
        permissions.map((p) => (p.resource === resourceKey ? { ...p, operations: newOps } : p)),
      );
    },
    [permissions, permissionMap, updatePermissions],
  );

  const handleDataScopeChange = useCallback(
    (resourceKey: string, dataScope: DataScope) => {
      if (!permissionMap.has(resourceKey)) return;

      updatePermissions(
        permissions.map((p) => (p.resource === resourceKey ? { ...p, dataScope } : p)),
      );
    },
    [permissionMap, permissions, updatePermissions],
  );

  const handleQuickAction = useCallback(
    (resource: ResourceDefinition, action: "full" | "view") => {
      const ops =
        action === "view"
          ? resource.operations.filter((op) => op.operation === "read").map((op) => op.operation)
          : resource.operations.map((op) => op.operation);

      if (ops.length === 0) return;

      const existing = permissionMap.get(resource.resource);
      if (existing) {
        updatePermissions(
          permissions.map((p) =>
            p.resource === resource.resource ? { ...p, operations: ops } : p,
          ),
        );
      } else {
        updatePermissions([
          ...permissions,
          {
            resource: resource.resource,
            operations: ops,
            dataScope: "organization",
          },
        ]);
      }
    },
    [permissions, permissionMap, updatePermissions],
  );

  const handleExpandedChange = useCallback((value: unknown) => {
    const v = value as string | string[] | undefined;
    setExpandedCategories(Array.isArray(v) ? v : v ? [v] : []);
  }, []);

  return (
    <section className="space-y-4">
      <div className="flex flex-col gap-3 border-t border-border/70 pt-6">
        <div className="flex flex-col gap-3 xl:flex-row xl:items-start xl:justify-between">
          <div className="space-y-1">
            <h3 className="text-sm font-semibold">Permissions</h3>
            <p className="text-sm text-muted-foreground">
              Apply a global preset, then narrow access by resource where needed.
            </p>
          </div>
          <div className="w-full xl:max-w-sm">
            <div className="relative">
              <SearchIcon className="absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={searchQuery}
                onChange={(event) => setSearchQuery(event.target.value)}
                placeholder="Search resources..."
                className="h-9 pl-9"
              />
              {searchQuery && (
                <button
                  type="button"
                  onClick={() => setSearchQuery("")}
                  className="absolute top-1/2 right-2.5 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                >
                  <XIcon className="size-4" />
                </button>
              )}
            </div>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {(["read", "write", "full"] as const).map((mode) => (
            <Button
              key={mode}
              type="button"
              variant="outline"
              size="sm"
              onClick={() => applyBulkPreset(allResources, mode)}
            >
              {mode === "read" ? "All Read" : mode === "write" ? "All Write" : "Full Access"}
            </Button>
          ))}
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => applyBulkPreset(allResources, "clear")}
            disabled={permissions.length === 0}
          >
            Clear All
          </Button>
          <div className="ml-auto flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
            <Badge variant="secondary">
              {selectionSummary.selectedResources}{" "}
              {pluralize("resource", selectionSummary.selectedResources)}
            </Badge>
            <Badge variant="secondary">
              {selectionSummary.selectedOperations}{" "}
              {pluralize("operation", selectionSummary.selectedOperations)}
            </Badge>
            <Badge variant="outline">{selectionSummary.modeLabel}</Badge>
          </div>
        </div>
      </div>

      {isLoading ? (
        <div className="space-y-4">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-40 w-full" />
          <Skeleton className="h-40 w-full" />
        </div>
      ) : filteredCategories.length === 0 ? (
        <div className="py-8 text-center text-sm text-muted-foreground">
          No resources found matching &ldquo;{searchQuery}&rdquo;
        </div>
      ) : (
        <div className="max-h-[500px] overflow-y-auto rounded-lg border bg-card p-1">
          <Accordion value={expandedCategories} onValueChange={handleExpandedChange} multiple>
            {filteredCategories.map((category) => (
              <CategorySection
                key={category.category}
                category={category}
                permissionMap={permissionMap}
                onToggleResource={handleToggleResource}
                onToggleOperation={handleToggleOperation}
                onDataScopeChange={handleDataScopeChange}
                onQuickAction={handleQuickAction}
                onCategoryPreset={applyCategoryPreset}
              />
            ))}
          </Accordion>
        </div>
      )}
    </section>
  );
}

type CategorySectionProps = {
  category: ResourceCategory;
  permissionMap: Map<string, ApiKeyPermissionInput>;
  onToggleResource: (resource: ResourceDefinition) => void;
  onToggleOperation: (resource: string, operation: string) => void;
  onDataScopeChange: (resource: string, scope: DataScope) => void;
  onQuickAction: (resource: ResourceDefinition, action: "full" | "view") => void;
  onCategoryPreset: (category: ResourceCategory, mode: BulkMode) => void;
};

function CategorySection({
  category,
  permissionMap,
  onToggleResource,
  onToggleOperation,
  onDataScopeChange,
  onQuickAction,
  onCategoryPreset,
}: CategorySectionProps) {
  const grantedInCategory = category.resources.filter((r) => permissionMap.has(r.resource)).length;

  return (
    <AccordionItem value={category.category} className="border-none">
      <AccordionHeader>
        <AccordionTrigger className="group flex w-full items-center justify-between rounded-md px-3 py-2 text-sm hover:bg-accent">
          <div className="flex items-center gap-2">
            <ChevronDownIcon className="size-4 -rotate-90 text-muted-foreground transition-transform duration-200 group-data-panel-open:rotate-0" />
            <span className="font-medium">{category.category}</span>
            <span className="text-xs text-muted-foreground">({category.resources.length})</span>
          </div>
          <div className="flex items-center gap-2">
            {grantedInCategory > 0 && (
              <Badge variant="secondary" className="text-xs">
                {grantedInCategory} granted
              </Badge>
            )}
          </div>
        </AccordionTrigger>
      </AccordionHeader>
      <AccordionPanel className="pl-6">
        <div className="mb-2 flex flex-wrap gap-1.5 px-3">
          <Button
            type="button"
            variant="outline"
            size="xs"
            onClick={(e) => {
              e.stopPropagation();
              onCategoryPreset(category, "read");
            }}
          >
            All Read
          </Button>
          <Button
            type="button"
            variant="outline"
            size="xs"
            onClick={(e) => {
              e.stopPropagation();
              onCategoryPreset(category, "write");
            }}
          >
            All Write
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="xs"
            onClick={(e) => {
              e.stopPropagation();
              onCategoryPreset(category, "clear");
            }}
          >
            Clear
          </Button>
        </div>
        <div className="space-y-1 pb-2">
          {category.resources.map((resource) => (
            <ResourceRow
              key={resource.resource}
              resource={resource}
              permission={permissionMap.get(resource.resource)}
              onToggle={() => onToggleResource(resource)}
              onToggleOperation={(op) => onToggleOperation(resource.resource, op)}
              onDataScopeChange={(scope) => onDataScopeChange(resource.resource, scope)}
              onQuickAction={(action) => onQuickAction(resource, action)}
            />
          ))}
        </div>
      </AccordionPanel>
    </AccordionItem>
  );
}

type ResourceRowProps = {
  resource: ResourceDefinition;
  permission?: ApiKeyPermissionInput;
  onToggle: () => void;
  onToggleOperation: (operation: string) => void;
  onDataScopeChange: (scope: DataScope) => void;
  onQuickAction: (action: "full" | "view") => void;
};

function ResourceRow({
  resource,
  permission,
  onToggle,
  onToggleOperation,
  onDataScopeChange,
  onQuickAction,
}: ResourceRowProps) {
  const isGranted = !!permission;
  const [showDetails, setShowDetails] = useState(false);

  const operationCount = permission?.operations.length ?? 0;
  const totalOperations = resource.operations.length;
  const isFullAccess = operationCount === totalOperations;
  const isViewOnly = operationCount === 1 && permission?.operations[0] === "read";

  return (
    <div
      className={cn(
        "rounded-md border bg-background transition-all",
        isGranted && "border-primary/30 bg-primary/5",
      )}
    >
      <div className="flex items-center gap-2 px-3 py-2">
        <Checkbox checked={isGranted} onCheckedChange={onToggle} className="shrink-0" />
        <button
          type="button"
          onClick={() => isGranted && setShowDetails(!showDetails)}
          className="flex flex-1 items-center justify-between text-left"
          disabled={!isGranted}
        >
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <span className="truncate text-sm font-medium">{resource.displayName}</span>
              {isGranted && (
                <Badge variant={isFullAccess ? "default" : "secondary"} className="text-[10px]">
                  {isFullAccess
                    ? "Full Access"
                    : isViewOnly
                      ? "View Only"
                      : `${operationCount}/${totalOperations}`}
                </Badge>
              )}
            </div>
            {resource.description && (
              <p className="truncate text-xs text-muted-foreground">{resource.description}</p>
            )}
          </div>
          {isGranted && (
            <ChevronDownIcon
              className={cn(
                "size-4 shrink-0 text-muted-foreground transition-transform duration-200",
                showDetails ? "rotate-0" : "-rotate-90",
              )}
            />
          )}
        </button>
        <div className="flex shrink-0 items-center gap-1">
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-xs"
                  onClick={() => onQuickAction("view")}
                  className={cn("size-7", isViewOnly && "bg-primary/10 text-primary")}
                >
                  <EyeIcon className="size-3.5" />
                </Button>
              }
            />
            <TooltipContent>View Only</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-xs"
                  onClick={() => onQuickAction("full")}
                  className={cn("size-7", isFullAccess && "bg-primary/10 text-primary")}
                >
                  <ZapIcon className="size-3.5" />
                </Button>
              }
            />
            <TooltipContent>Full Access</TooltipContent>
          </Tooltip>
        </div>
      </div>

      {isGranted && showDetails && (
        <div className="border-t px-3 py-2">
          <div className="mb-2 flex flex-wrap gap-1.5">
            {resource.operations.map((opDef) => {
              const isChecked = permission.operations.includes(opDef.operation);
              return (
                <button
                  key={opDef.operation}
                  type="button"
                  onClick={() => onToggleOperation(opDef.operation)}
                  className={cn(
                    "flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs transition-colors",
                    isChecked
                      ? "border-primary/50 bg-primary/10 text-primary"
                      : "border-border bg-muted/50 text-muted-foreground hover:bg-muted",
                  )}
                >
                  {isChecked && <CheckIcon className="size-3" />}
                  {opDef.displayName}
                </button>
              );
            })}
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs text-muted-foreground">Scope:</span>
            <Select
              value={permission.dataScope}
              onValueChange={(value) => onDataScopeChange(value as DataScope)}
            >
              <SelectTrigger className="h-7 w-32 text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {dataScopeChoices.map((choice) => (
                  <SelectItem key={choice.value} value={choice.value}>
                    {choice.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
      )}
    </div>
  );
}

function createPermissionMap(permissions: ApiKeyPermissionInput[]) {
  return new Map(permissions.map((permission) => [permission.resource, permission]));
}

function buildBulkPermissions(
  resources: ResourceDefinition[],
  mode: BulkMode,
): ApiKeyPermissionInput[] {
  if (mode === "clear") {
    return [];
  }

  const nextPermissions: ApiKeyPermissionInput[] = [];

  for (const resource of resources) {
    const operations = getOperationsForMode(resource, mode);

    if (operations.length === 0) {
      continue;
    }

    nextPermissions.push({
      resource: resource.resource,
      operations,
      dataScope: "organization",
    });
  }

  return nextPermissions;
}

function getOperationsForMode(resource: ResourceDefinition, mode: Exclude<BulkMode, "clear">) {
  if (mode === "full") {
    return resource.operations.map((operation) => operation.operation);
  }

  const readOperations = resource.operations
    .filter((operation) => operation.operation === "read")
    .map((operation) => operation.operation);

  if (mode === "read") {
    return readOperations;
  }

  const writeOperations = resource.operations
    .filter((operation) => operation.operation !== "read")
    .map((operation) => operation.operation);

  return Array.from(new Set([...readOperations, ...writeOperations]));
}

function summarizeSelection(
  allResources: ResourceDefinition[],
  permissionMap: Map<string, ApiKeyPermissionInput>,
) {
  const selectedResources = permissionMap.size;
  const selectedOperations = Array.from(permissionMap.values()).reduce(
    (total, permission) => total + permission.operations.length,
    0,
  );

  const totalResources = allResources.length;
  const hasGlobalRead =
    totalResources > 0 &&
    allResources.every((resource) => {
      const readOperations = getOperationsForMode(resource, "read");
      if (readOperations.length === 0) {
        return !permissionMap.has(resource.resource);
      }

      const selected = permissionMap.get(resource.resource)?.operations ?? [];
      return readOperations.every((operation) => selected.includes(operation));
    });

  const hasGlobalWrite =
    totalResources > 0 &&
    allResources.every((resource) => {
      const writeOperations = getOperationsForMode(resource, "write");
      if (writeOperations.length === 0) {
        return !permissionMap.has(resource.resource);
      }

      const selected = permissionMap.get(resource.resource)?.operations ?? [];
      return writeOperations.every((operation) => selected.includes(operation));
    });

  const hasFullAccess =
    totalResources > 0 &&
    allResources.every((resource) => {
      const selected = permissionMap.get(resource.resource)?.operations ?? [];
      return resource.operations.every((operation) => selected.includes(operation.operation));
    });

  const modeLabel = hasFullAccess
    ? "Full access"
    : hasGlobalWrite
      ? "Global write"
      : hasGlobalRead
        ? "Global read"
        : selectedResources === 0
          ? "No access"
          : "Custom";

  return {
    selectedResources,
    selectedOperations,
    modeLabel,
  };
}
