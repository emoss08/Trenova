import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
import {
  getAvailableResources,
  type ResourceCategory,
  type ResourceDefinition,
} from "@/lib/role-api";
import { cn } from "@/lib/utils";
import type { AddPermission, DataScope, Operation } from "@/types/role";
import { useQuery } from "@tanstack/react-query";
import {
  CheckIcon,
  ChevronDownIcon,
  MinusIcon,
  SearchIcon,
  SearchXIcon,
  XIcon,
} from "lucide-react";
import { AnimatePresence, m } from "motion/react";
import { useCallback, useMemo, useState } from "react";

type RolePermissionMatrixProps = {
  permissions: AddPermission[];
  onPermissionsChange: (permissions: AddPermission[]) => void;
};

export function RolePermissionMatrix({
  permissions,
  onPermissionsChange,
}: RolePermissionMatrixProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [expandedCategories, setExpandedCategories] = useState<Set<string>>(new Set());

  const { data: resourceCategories = [], isLoading } = useQuery({
    queryKey: ["permission-resources"],
    queryFn: getAvailableResources,
    staleTime: 1000 * 60 * 30,
  });

  const permissionMap = useMemo(() => {
    const map = new Map<string, AddPermission>();
    for (const perm of permissions) {
      map.set(perm.resource, perm);
    }
    return map;
  }, [permissions]);

  const isSearching = searchQuery.trim().length > 0;

  const filteredCategories = useMemo(() => {
    if (!isSearching) return resourceCategories;

    const query = searchQuery.toLowerCase();
    return resourceCategories
      .map((cat) => ({
        ...cat,
        resources: cat.resources.filter(
          (r) =>
            r.displayName.toLowerCase().includes(query) || r.resource.toLowerCase().includes(query),
        ),
      }))
      .filter((cat) => cat.resources.length > 0);
  }, [resourceCategories, searchQuery, isSearching]);

  const toggleCategory = useCallback((category: string) => {
    setExpandedCategories((prev) => {
      const next = new Set(prev);
      if (next.has(category)) {
        next.delete(category);
      } else {
        next.add(category);
      }
      return next;
    });
  }, []);

  const toggleOperation = useCallback(
    (resource: ResourceDefinition, operation: string) => {
      const existing = permissionMap.get(resource.resource);
      const op = operation as Operation;

      if (existing) {
        const hasOp = existing.operations.includes(op);
        if (hasOp) {
          const newOps = existing.operations.filter((o) => o !== op);
          if (newOps.length === 0) {
            onPermissionsChange(permissions.filter((p) => p.resource !== resource.resource));
          } else {
            onPermissionsChange(
              permissions.map((p) =>
                p.resource === resource.resource ? { ...p, operations: newOps } : p,
              ),
            );
          }
        } else {
          onPermissionsChange(
            permissions.map((p) =>
              p.resource === resource.resource ? { ...p, operations: [...p.operations, op] } : p,
            ),
          );
        }
      } else {
        onPermissionsChange([
          ...permissions,
          {
            resource: resource.resource,
            operations: [op],
            dataScope: "organization" as DataScope,
          },
        ]);
      }
    },
    [permissions, permissionMap, onPermissionsChange],
  );

  const toggleAllForResource = useCallback(
    (resource: ResourceDefinition) => {
      const existing = permissionMap.get(resource.resource);
      const allOps = resource.operations.map((o) => o.operation as Operation);

      if (existing && existing.operations.length === allOps.length) {
        onPermissionsChange(permissions.filter((p) => p.resource !== resource.resource));
      } else {
        if (existing) {
          onPermissionsChange(
            permissions.map((p) =>
              p.resource === resource.resource ? { ...p, operations: allOps } : p,
            ),
          );
        } else {
          onPermissionsChange([
            ...permissions,
            {
              resource: resource.resource,
              operations: allOps,
              dataScope: "organization" as DataScope,
            },
          ]);
        }
      }
    },
    [permissions, permissionMap, onPermissionsChange],
  );

  const toggleAllForCategory = useCallback(
    (category: ResourceCategory, action: "select" | "clear") => {
      if (action === "clear") {
        const resourceNames = new Set(category.resources.map((r) => r.resource));
        onPermissionsChange(permissions.filter((p) => !resourceNames.has(p.resource)));
      } else {
        const existingMap = new Map(permissions.map((p) => [p.resource, p]));
        const newPerms: AddPermission[] = [...permissions];

        for (const resource of category.resources) {
          const allOps = resource.operations.map((o) => o.operation as Operation);
          const existing = existingMap.get(resource.resource);

          if (existing) {
            const idx = newPerms.findIndex((p) => p.resource === resource.resource);
            if (idx !== -1) {
              newPerms[idx] = { ...existing, operations: allOps };
            }
          } else {
            newPerms.push({
              resource: resource.resource,
              operations: allOps,
              dataScope: "organization" as DataScope,
            });
          }
        }

        onPermissionsChange(newPerms);
      }
    },
    [permissions, onPermissionsChange],
  );

  const updateDataScope = useCallback(
    (resource: string, scope: DataScope) => {
      onPermissionsChange(
        permissions.map((p) => (p.resource === resource ? { ...p, dataScope: scope } : p)),
      );
    },
    [permissions, onPermissionsChange],
  );

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col gap-4">
      <div className="flex items-center gap-3">
        <div className="relative flex-1">
          <Input
            leftElement={<SearchIcon className="size-4 text-muted-foreground" />}
            placeholder="Search resources..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            rightElement={
              searchQuery && (
                <Button
                  type="button"
                  variant="ghostInvert"
                  size="icon-xs"
                  onClick={() => setSearchQuery("")}
                  className="cursor-pointer text-muted-foreground hover:text-foreground"
                >
                  <XIcon className="size-3.5" />
                </Button>
              )
            }
            className="h-9 pl-9 text-sm"
          />
        </div>
        <Badge variant="outline" className="shrink-0 text-xs font-normal">
          {permissions.length} selected
        </Badge>
      </div>

      <div className="min-h-0 flex-1 overflow-auto rounded-lg border">
        {filteredCategories.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-center">
            <SearchXIcon className="size-8 text-muted-foreground/50" />
            <p className="mt-3 text-sm font-medium">No resources found</p>
            <p className="mt-1 text-xs text-muted-foreground">
              No resources match &ldquo;{searchQuery}&rdquo;. Try a different search term.
            </p>
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="mt-4"
              onClick={() => setSearchQuery("")}
            >
              Clear search
            </Button>
          </div>
        ) : (
          <div>
            {filteredCategories.map((category, idx) => (
              <CategorySection
                key={category.category}
                category={category}
                isExpanded={isSearching || expandedCategories.has(category.category)}
                isLast={idx === filteredCategories.length - 1}
                onToggle={() => toggleCategory(category.category)}
                permissionMap={permissionMap}
                onToggleOperation={toggleOperation}
                onToggleAllForResource={toggleAllForResource}
                onToggleAllForCategory={toggleAllForCategory}
                onUpdateDataScope={updateDataScope}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

type CategorySectionProps = {
  category: ResourceCategory;
  isExpanded: boolean;
  isLast: boolean;
  onToggle: () => void;
  permissionMap: Map<string, AddPermission>;
  onToggleOperation: (resource: ResourceDefinition, operation: string) => void;
  onToggleAllForResource: (resource: ResourceDefinition) => void;
  onToggleAllForCategory: (category: ResourceCategory, action: "select" | "clear") => void;
  onUpdateDataScope: (resource: string, scope: DataScope) => void;
};

function CategorySection({
  category,
  isExpanded,
  isLast,
  onToggle,
  permissionMap,
  onToggleOperation,
  onToggleAllForResource,
  onToggleAllForCategory,
  onUpdateDataScope,
}: CategorySectionProps) {
  const grantedCount = category.resources.filter((r) => permissionMap.has(r.resource)).length;

  return (
    <div className={cn(!isLast && "border-b")}>
      <span
        onClick={onToggle}
        className="flex w-full items-center justify-between bg-card px-4 py-3 text-left transition-colors hover:bg-muted/50"
      >
        <div className="flex items-center gap-2">
          <m.div animate={{ rotate: isExpanded ? 0 : -90 }} transition={{ duration: 0.15 }}>
            <ChevronDownIcon className="size-4 text-muted-foreground" />
          </m.div>
          <span className="text-sm font-medium">{category.category}</span>
          <span className="text-xs text-muted-foreground">{category.resources.length}</span>
        </div>

        <div className="flex items-center gap-2">
          {grantedCount > 0 && (
            <Badge className="bg-primary/10 text-primary hover:bg-primary/10">{grantedCount}</Badge>
          )}
          <div className="flex gap-1" onClick={(e) => e.stopPropagation()}>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-6 px-2 text-[11px]"
              onClick={() => onToggleAllForCategory(category, "select")}
            >
              All
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-6 px-2 text-[11px]"
              onClick={() => onToggleAllForCategory(category, "clear")}
            >
              None
            </Button>
          </div>
        </div>
      </span>

      <AnimatePresence initial={false}>
        {isExpanded && (
          <m.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.2, ease: "easeInOut" }}
            className="overflow-hidden"
          >
            <div className="border-t bg-muted/30">
              {category.resources.map((resource, idx) => (
                <ResourceRow
                  key={resource.resource}
                  resource={resource}
                  permission={permissionMap.get(resource.resource)}
                  isLast={idx === category.resources.length - 1}
                  onToggleOperation={(op) => onToggleOperation(resource, op)}
                  onToggleAll={() => onToggleAllForResource(resource)}
                  onUpdateDataScope={(scope) => onUpdateDataScope(resource.resource, scope)}
                />
              ))}
            </div>
          </m.div>
        )}
      </AnimatePresence>
    </div>
  );
}

type ResourceRowProps = {
  resource: ResourceDefinition;
  permission?: AddPermission;
  isLast: boolean;
  onToggleOperation: (operation: string) => void;
  onToggleAll: () => void;
  onUpdateDataScope: (scope: DataScope) => void;
};

function ResourceRow({
  resource,
  permission,
  isLast,
  onToggleOperation,
  onToggleAll,
  onUpdateDataScope,
}: ResourceRowProps) {
  const isGranted = !!permission;
  const operationCount = permission?.operations.length || 0;
  const totalOperations = resource.operations.length;
  const isFullAccess = operationCount === totalOperations;
  const isPartial = operationCount > 0 && operationCount < totalOperations;

  return (
    <div
      className={cn(
        "flex items-center gap-4 bg-background px-4 py-2.5 transition-colors",
        isGranted && "bg-primary/5",
        !isLast && "border-b border-border/50",
      )}
    >
      <Tooltip>
        <TooltipTrigger
          render={
            <button
              type="button"
              onClick={onToggleAll}
              className={cn(
                "flex size-5 shrink-0 items-center justify-center rounded border transition-colors",
                isFullAccess
                  ? "border-primary bg-primary text-primary-foreground"
                  : isPartial
                    ? "border-primary bg-primary/20"
                    : "border-muted-foreground/40 hover:border-primary/60",
              )}
            >
              {isFullAccess && <CheckIcon className="size-3" />}
              {isPartial && <MinusIcon className="size-3 text-primary" />}
            </button>
          }
        />
        <TooltipContent side="left" className="text-xs">
          {isFullAccess ? "Remove all" : "Grant all"}
        </TooltipContent>
      </Tooltip>

      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">{resource.displayName}</p>
      </div>

      <div className="flex items-center gap-1.5">
        {resource.operations.map((op) => {
          const isChecked = permission?.operations.includes(op.operation as Operation);
          return (
            <Tooltip key={op.operation}>
              <TooltipTrigger
                render={
                  <button
                    type="button"
                    onClick={() => onToggleOperation(op.operation)}
                    className={cn(
                      "cursor-pointer rounded-md px-2.5 py-1 text-xs font-medium transition-all",
                      isChecked
                        ? "bg-primary text-primary-foreground"
                        : "bg-muted text-muted-foreground hover:bg-muted/80 hover:text-foreground",
                    )}
                  >
                    {op.displayName}
                  </button>
                }
              />
              <TooltipContent className="text-xs">{op.description}</TooltipContent>
            </Tooltip>
          );
        })}
      </div>

      {isGranted && (
        <Select
          value={permission.dataScope}
          onValueChange={(value) => onUpdateDataScope(value as DataScope)}
          items={dataScopeChoices}
        >
          <SelectTrigger className="h-7 w-24 text-xs">
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
      )}
    </div>
  );
}
