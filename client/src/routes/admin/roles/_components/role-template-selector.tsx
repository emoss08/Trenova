import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { getAvailableResources, type ResourceDefinition } from "@/lib/role-api";
import { cn } from "@/lib/utils";
import type { AddPermission, DataScope, Operation } from "@/types/role";
import { useQuery } from "@tanstack/react-query";
import { EyeIcon, PencilIcon, ShieldIcon, SparklesIcon } from "lucide-react";
import { useMemo } from "react";

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
    icon: <EyeIcon className="size-3.5" />,
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
    icon: <PencilIcon className="size-3.5" />,
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
    description: "Full CRUD access to all resources",
    icon: <ShieldIcon className="size-3.5" />,
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
    description: "Build your own permission set",
    icon: <SparklesIcon className="size-3.5" />,
    getPermissions: () => [],
  },
];

type RoleTemplateSelectorProps = {
  selectedTemplate: string | null;
  onSelectTemplate: (templateId: string, permissions: AddPermission[]) => void;
};

export function RoleTemplateSelector({
  selectedTemplate,
  onSelectTemplate,
}: RoleTemplateSelectorProps) {
  const { data: resourceCategories = [] } = useQuery({
    queryKey: ["permission-resources"],
    queryFn: getAvailableResources,
    staleTime: 1000 * 60 * 30,
  });

  const allResources = useMemo(() => {
    return resourceCategories.flatMap((cat) => cat.resources);
  }, [resourceCategories]);

  const handleSelect = (template: RoleTemplate) => {
    const permissions = template.getPermissions(allResources);
    onSelectTemplate(template.id, permissions);
  };

  return (
    <div className="flex items-center gap-2">
      {ROLE_TEMPLATES.map((template) => {
        const isSelected = selectedTemplate === template.id;
        return (
          <Tooltip key={template.id}>
            <TooltipTrigger
              render={
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => handleSelect(template)}
                  className={cn(
                    "h-8 gap-1.5 rounded-md px-3 text-sm font-medium",
                    isSelected
                      ? "bg-primary text-primary-foreground hover:bg-primary/90 hover:text-primary-foreground"
                      : "border border-border bg-background text-muted-foreground hover:bg-accent hover:text-foreground",
                  )}
                >
                  {template.icon}
                  {template.name}
                </Button>
              }
            />
            <TooltipContent className="text-xs">{template.description}</TooltipContent>
          </Tooltip>
        );
      })}
    </div>
  );
}

export { ROLE_TEMPLATES, type RoleTemplate };
