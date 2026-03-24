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
  EyeIcon,
  PencilIcon,
  SearchIcon,
  ShieldIcon,
  SparklesIcon,
  XIcon,
  ZapIcon,
} from "lucide-react";
import { useCallback, useMemo, useState } from "react";

type RoleTemplate = {
  id: string;
  name: string;
  description: string;
  icon: React.ReactNode;
  getPermissions: (resources: ResourceDefinition[]) => AddPermission[];
};

const ROLE_TEMPLATES: RoleTemplate[] = [
  {
    id: "viewer",
    name: "Viewer",
    description: "Read-only access to all resources",
    icon: <EyeIcon className="size-4" />,
    getPermissions: (resources) =>
      resources.map((r) => ({
        resource: r.resource,
        operations: ["read"] as Operation[],
        dataScope: "organization" as DataScope,
      })),
  },
  {
    id: "editor",
    name: "Editor",
    description: "Read and update access",
    icon: <PencilIcon className="size-4" />,
    getPermissions: (resources) =>
      resources
        .filter((r) => r.operations.some((op) => op.operation === "update"))
        .map((r) => ({
          resource: r.resource,
          operations: ["read", "update"].filter((op) =>
            r.operations.some((o) => o.operation === op),
          ) as Operation[],
          dataScope: "organization" as DataScope,
        })),
  },
  {
    id: "manager",
    name: "Manager",
    description: "Full CRUD access",
    icon: <ShieldIcon className="size-4" />,
    getPermissions: (resources) =>
      resources.map((r) => ({
        resource: r.resource,
        operations: ["read", "create", "update"].filter((op) =>
          r.operations.some((o) => o.operation === op),
        ) as Operation[],
        dataScope: "organization" as DataScope,
      })),
  },
  {
    id: "custom",
    name: "Custom",
    description: "Build your own permissions",
    icon: <SparklesIcon className="size-4" />,
    getPermissions: () => [],
  },
];

type RolePermissionBuilderProps = {
  permissions: AddPermission[];
  onPermissionsChange: (permissions: AddPermission[]) => void;
};

export function RolePermissionBuilder({
  permissions,
  onPermissionsChange,
}: RolePermissionBuilderProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedTemplate, setSelectedTemplate] = useState<string | null>(null);
  const [expandedCategories, setExpandedCategories] = useState<string[]>([]);

  const { data: resourceCategories = [], isLoading } = useQuery({
    queryKey: ["permission-resources"],
    queryFn: getAvailableResources,
    staleTime: 1000 * 60 * 30,
  });

  const allResources = useMemo(() => {
    return resourceCategories.flatMap((cat) => cat.resources);
  }, [resourceCategories]);

  const permissionMap = useMemo(() => {
    const map = new Map<string, AddPermission>();
    for (const perm of permissions) {
      map.set(perm.resource, perm);
    }
    return map;
  }, [permissions]);

  const filteredCategories = useMemo(() => {
    if (!searchQuery.trim()) return resourceCategories;

    const query = searchQuery.toLowerCase();
    return resourceCategories
      .map((cat) => ({
        ...cat,
        resources: cat.resources.filter(
          (r) =>
            r.displayName.toLowerCase().includes(query) ||
            r.resource.toLowerCase().includes(query) ||
            r.description.toLowerCase().includes(query),
        ),
      }))
      .filter((cat) => cat.resources.length > 0);
  }, [resourceCategories, searchQuery]);

  const handleTemplateSelect = useCallback(
    (templateId: string) => {
      const template = ROLE_TEMPLATES.find((t) => t.id === templateId);
      if (!template) return;

      setSelectedTemplate(templateId);
      const newPermissions = template.getPermissions(allResources);
      onPermissionsChange(newPermissions);

      if (templateId !== "custom") {
        setExpandedCategories([]);
      }
    },
    [allResources, onPermissionsChange],
  );

  const handleToggleResource = useCallback(
    (resource: ResourceDefinition) => {
      const existing = permissionMap.get(resource.resource);
      if (existing) {
        onPermissionsChange(permissions.filter((p) => p.resource !== resource.resource));
      } else {
        onPermissionsChange([
          ...permissions,
          {
            resource: resource.resource,
            operations: ["read"] as Operation[],
            dataScope: "organization" as DataScope,
          },
        ]);
      }
      setSelectedTemplate("custom");
    },
    [permissions, permissionMap, onPermissionsChange],
  );

  const handleToggleOperation = useCallback(
    (resource: string, operation: Operation) => {
      const existing = permissionMap.get(resource);
      if (!existing) return;

      const hasOp = existing.operations.includes(operation);
      let newOps: Operation[];

      if (hasOp) {
        newOps = existing.operations.filter((op) => op !== operation);
        if (newOps.length === 0) {
          onPermissionsChange(permissions.filter((p) => p.resource !== resource));
          setSelectedTemplate("custom");
          return;
        }
      } else {
        newOps = [...existing.operations, operation];
      }

      onPermissionsChange(
        permissions.map((p) => (p.resource === resource ? { ...p, operations: newOps } : p)),
      );
      setSelectedTemplate("custom");
    },
    [permissions, permissionMap, onPermissionsChange],
  );

  const handleDataScopeChange = useCallback(
    (resource: string, scope: DataScope) => {
      onPermissionsChange(
        permissions.map((p) => (p.resource === resource ? { ...p, dataScope: scope } : p)),
      );
      setSelectedTemplate("custom");
    },
    [permissions, onPermissionsChange],
  );

  const handleQuickAction = useCallback(
    (resource: ResourceDefinition, action: "full" | "view" | "remove") => {
      if (action === "remove") {
        onPermissionsChange(permissions.filter((p) => p.resource !== resource.resource));
      } else if (action === "view") {
        const viewOps = resource.operations
          .filter((op) => op.operation === "read")
          .map((op) => op.operation as Operation);
        if (viewOps.length === 0) return;

        const existing = permissionMap.get(resource.resource);
        if (existing) {
          onPermissionsChange(
            permissions.map((p) =>
              p.resource === resource.resource ? { ...p, operations: viewOps } : p,
            ),
          );
        } else {
          onPermissionsChange([
            ...permissions,
            {
              resource: resource.resource,
              operations: viewOps,
              dataScope: "organization" as DataScope,
            },
          ]);
        }
      } else if (action === "full") {
        const allOps = resource.operations.map((op) => op.operation as Operation);
        const existing = permissionMap.get(resource.resource);
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
      setSelectedTemplate("custom");
    },
    [permissions, permissionMap, onPermissionsChange],
  );

  const handleExpandedChange = useCallback((value: unknown) => {
    const v = value as string | string[] | undefined;
    setExpandedCategories(Array.isArray(v) ? v : v ? [v] : []);
  }, []);

  const grantedCount = permissions.length;

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-10 w-full" />
        <div className="grid grid-cols-4 gap-2">
          {[1, 2, 3, 4].map((i) => (
            <Skeleton key={i} className="h-20" />
          ))}
        </div>
        <Skeleton className="h-40 w-full" />
      </div>
    );
  }

  return (
    <div className="space-y-5">
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-medium">Start with a template</h3>
          {grantedCount > 0 && (
            <Badge variant="secondary" className="text-xs">
              {grantedCount} resource{grantedCount !== 1 ? "s" : ""} granted
            </Badge>
          )}
        </div>
        <div className="grid grid-cols-4 gap-2">
          {ROLE_TEMPLATES.map((template) => (
            <button
              key={template.id}
              type="button"
              onClick={() => handleTemplateSelect(template.id)}
              className={cn(
                "flex flex-col items-center gap-1.5 rounded-lg border p-3 text-center transition-all hover:border-primary/50 hover:bg-accent",
                selectedTemplate === template.id &&
                  "border-primary bg-primary/5 ring-1 ring-primary/20",
              )}
            >
              <div
                className={cn(
                  "flex size-8 items-center justify-center rounded-full bg-muted",
                  selectedTemplate === template.id && "bg-primary/10 text-primary",
                )}
              >
                {template.icon}
              </div>
              <span className="text-xs font-medium">{template.name}</span>
              <span className="text-[10px] leading-tight text-muted-foreground">
                {template.description}
              </span>
            </button>
          ))}
        </div>
      </div>

      <div className="space-y-3">
        <div className="flex items-center gap-2">
          <div className="relative flex-1">
            <SearchIcon className="absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Search resources..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
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

        <div className="max-h-[400px] overflow-y-auto rounded-lg border bg-card p-1">
          {filteredCategories.length === 0 ? (
            <div className="py-8 text-center text-sm text-muted-foreground">
              No resources found matching &ldquo;{searchQuery}&rdquo;
            </div>
          ) : (
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
                />
              ))}
            </Accordion>
          )}
        </div>
      </div>
    </div>
  );
}

type CategorySectionProps = {
  category: ResourceCategory;
  permissionMap: Map<string, AddPermission>;
  onToggleResource: (resource: ResourceDefinition) => void;
  onToggleOperation: (resource: string, operation: Operation) => void;
  onDataScopeChange: (resource: string, scope: DataScope) => void;
  onQuickAction: (resource: ResourceDefinition, action: "full" | "view" | "remove") => void;
};

function CategorySection({
  category,
  permissionMap,
  onToggleResource,
  onToggleOperation,
  onDataScopeChange,
  onQuickAction,
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
          {grantedInCategory > 0 && (
            <Badge variant="secondary" className="text-xs">
              {grantedInCategory} granted
            </Badge>
          )}
        </AccordionTrigger>
      </AccordionHeader>
      <AccordionPanel className="pl-6">
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
  permission?: AddPermission;
  onToggle: () => void;
  onToggleOperation: (operation: Operation) => void;
  onDataScopeChange: (scope: DataScope) => void;
  onQuickAction: (action: "full" | "view" | "remove") => void;
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
              const isChecked = permission.operations.includes(opDef.operation as Operation);
              return (
                <button
                  key={opDef.operation}
                  type="button"
                  onClick={() => onToggleOperation(opDef.operation as Operation)}
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
