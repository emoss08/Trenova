import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { api } from "@/lib/api";
import { dataScopeChoices } from "@/lib/choices";
import {
  addPermission,
  getAvailableResources,
  removePermission,
  updatePermission,
  type ResourceCategory,
  type ResourceDefinition,
} from "@/lib/role-api";
import type {
  AddPermission,
  DataScope,
  Operation,
  ResourcePermission,
  Role,
} from "@/types/role";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { PlusIcon, TrashIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";

type RolePermissionsEditorProps = {
  roleId: string;
  isSystemRole?: boolean;
};

export function RolePermissionsEditor({
  roleId,
  isSystemRole,
}: RolePermissionsEditorProps) {
  const queryClient = useQueryClient();
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const { data: role, isLoading } = useQuery({
    queryKey: ["role", roleId],
    queryFn: () => api.get<Role>(`/roles/${roleId}`),
    select: (response) => response,
  });

  const { data: resourceCategories = [] } = useQuery({
    queryKey: ["permission-resources"],
    queryFn: getAvailableResources,
    staleTime: 1000 * 60 * 30,
  });

  const resourceMap = useMemo(() => {
    const map = new Map<string, ResourceDefinition>();
    for (const category of resourceCategories) {
      for (const resource of category.resources) {
        map.set(resource.resource, resource);
      }
    }
    return map;
  }, [resourceCategories]);

  const permissions = role?.permissions ?? [];

  const handleRemovePermission = useCallback(
    async (permissionId: string) => {
      try {
        await removePermission(roleId, permissionId);
        await queryClient.invalidateQueries({ queryKey: ["role", roleId] });
        toast.success("Permission removed");
      } catch {
        toast.error("Failed to remove permission");
      }
    },
    [roleId, queryClient],
  );

  const handleOperationToggle = useCallback(
    async (permission: ResourcePermission, operation: Operation) => {
      const currentOps = permission.operations;
      const hasOp = currentOps.includes(operation);

      let newOps: Operation[];
      if (hasOp) {
        newOps = currentOps.filter((op) => op !== operation);
      } else {
        newOps = [...currentOps, operation];
      }

      if (newOps.length === 0) {
        toast.error("At least one operation is required");
        return;
      }

      try {
        await updatePermission(roleId, permission.id!, {
          resource: permission.resource,
          operations: newOps,
          dataScope: permission.dataScope,
        });
        await queryClient.invalidateQueries({ queryKey: ["role", roleId] });
      } catch {
        toast.error("Failed to update permission");
      }
    },
    [roleId, queryClient],
  );

  const handleDataScopeChange = useCallback(
    async (permission: ResourcePermission, newScope: DataScope) => {
      try {
        await updatePermission(roleId, permission.id!, {
          resource: permission.resource,
          operations: permission.operations,
          dataScope: newScope,
        });
        await queryClient.invalidateQueries({ queryKey: ["role", roleId] });
      } catch {
        toast.error("Failed to update permission");
      }
    },
    [roleId, queryClient],
  );

  if (isLoading) {
    return (
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <Skeleton className="h-5 w-24" />
          <Skeleton className="h-8 w-20" />
        </div>
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-24 w-full" />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium">Permissions</h3>
        {!isSystemRole && (
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => setAddDialogOpen(true)}
          >
            <PlusIcon className="mr-1 size-3.5" />
            Add
          </Button>
        )}
      </div>

      {permissions.length === 0 ? (
        <div className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
          No permissions configured for this role.
        </div>
      ) : (
        <div className="space-y-3">
          {permissions.map((permission) => (
            <PermissionRow
              key={permission.id}
              permission={permission}
              resourceDef={resourceMap.get(permission.resource)}
              isSystemRole={isSystemRole}
              onOperationToggle={(op) => handleOperationToggle(permission, op)}
              onDataScopeChange={(scope) =>
                handleDataScopeChange(permission, scope)
              }
              onRemove={() => handleRemovePermission(permission.id!)}
            />
          ))}
        </div>
      )}

      <AddPermissionDialog
        open={addDialogOpen}
        onOpenChange={setAddDialogOpen}
        roleId={roleId}
        existingResources={permissions.map((p) => p.resource)}
        resourceCategories={resourceCategories}
        isSubmitting={isSubmitting}
        setIsSubmitting={setIsSubmitting}
      />
    </div>
  );
}

type PermissionRowProps = {
  permission: ResourcePermission;
  resourceDef?: ResourceDefinition;
  isSystemRole?: boolean;
  onOperationToggle: (operation: Operation) => void;
  onDataScopeChange: (scope: DataScope) => void;
  onRemove: () => void;
};

function PermissionRow({
  permission,
  resourceDef,
  isSystemRole,
  onOperationToggle,
  onDataScopeChange,
  onRemove,
}: PermissionRowProps) {
  const resourceLabel = resourceDef?.displayName ?? permission.resource;
  const availableOperations = resourceDef?.operations ?? [];

  return (
    <div className="rounded-md border p-3">
      <div className="flex items-start justify-between gap-2">
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <p className="text-sm font-medium">{resourceLabel}</p>
            {resourceDef?.category && (
              <span className="text-xs text-muted-foreground">
                ({resourceDef.category})
              </span>
            )}
          </div>
          <div className="mt-2 flex flex-wrap gap-3">
            {availableOperations.map((opDef) => (
              <label
                key={opDef.operation}
                className="flex items-center gap-1.5 text-xs text-muted-foreground"
                title={opDef.description}
              >
                <Checkbox
                  checked={permission.operations.includes(
                    opDef.operation as Operation,
                  )}
                  onCheckedChange={() =>
                    onOperationToggle(opDef.operation as Operation)
                  }
                  disabled={isSystemRole}
                />
                {opDef.displayName}
              </label>
            ))}
          </div>
          <div className="mt-3 flex items-center gap-2">
            <span className="text-xs text-muted-foreground">Scope:</span>
            <Select
              value={permission.dataScope}
              onValueChange={(value) => onDataScopeChange(value as DataScope)}
              disabled={isSystemRole}
            >
              <SelectTrigger className="h-7 w-36 text-xs">
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
        {!isSystemRole && (
          <Button
            type="button"
            size="icon-sm"
            variant="ghost"
            className="text-destructive hover:bg-destructive/10 hover:text-destructive"
            onClick={onRemove}
          >
            <TrashIcon className="size-4" />
          </Button>
        )}
      </div>
    </div>
  );
}

type AddPermissionDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  roleId: string;
  existingResources: string[];
  resourceCategories: ResourceCategory[];
  isSubmitting: boolean;
  setIsSubmitting: (submitting: boolean) => void;
};

function AddPermissionDialog({
  open,
  onOpenChange,
  roleId,
  existingResources,
  resourceCategories,
  isSubmitting,
  setIsSubmitting,
}: AddPermissionDialogProps) {
  const queryClient = useQueryClient();
  const [selectedResource, setSelectedResource] = useState<string>("");
  const [selectedOperations, setSelectedOperations] = useState<Operation[]>([
    "read",
  ]);
  const [selectedScope, setSelectedScope] = useState<DataScope>("organization");

  const selectedResourceDef = useMemo(() => {
    for (const category of resourceCategories) {
      const resource = category.resources.find(
        (r) => r.resource === selectedResource,
      );
      if (resource) return resource;
    }
    return null;
  }, [selectedResource, resourceCategories]);

  const availableOperations = selectedResourceDef?.operations ?? [];

  const handleResourceChange = (value: string) => {
    setSelectedResource(value);
    setSelectedOperations(["read"]);
  };

  const handleSubmit = async () => {
    if (!selectedResource) {
      toast.error("Please select a resource");
      return;
    }

    if (selectedOperations.length === 0) {
      toast.error("Please select at least one operation");
      return;
    }

    setIsSubmitting(true);
    await addPermission(roleId, {
      resource: selectedResource,
      operations: selectedOperations,
      dataScope: selectedScope,
    })
      .then(async () => {
        await queryClient.invalidateQueries({ queryKey: ["role", roleId] });
        toast.success("Permission added");
        onOpenChange(false);
        setSelectedResource("");
        setSelectedOperations(["read"]);
        setSelectedScope("organization");
      })
      .catch(() => {
        toast.error("Failed to add permission");
      })
      .finally(() => {
        setIsSubmitting(false);
      });
  };

  const toggleOperation = (op: Operation) => {
    setSelectedOperations((prev) =>
      prev.includes(op) ? prev.filter((o) => o !== op) : [...prev, op],
    );
  };

  const selectAllOperations = () => {
    setSelectedOperations(
      availableOperations.map((op) => op.operation as Operation),
    );
  };

  const clearAllOperations = () => {
    setSelectedOperations([]);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Add Permission</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label>Resource</Label>
            <Select
              value={selectedResource}
              onValueChange={(value) => handleResourceChange(value ?? "")}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent className="max-h-80">
                {resourceCategories.map((category) => {
                  const availableInCategory = category.resources.filter(
                    (r) => !existingResources.includes(r.resource),
                  );
                  if (availableInCategory.length === 0) return null;

                  return (
                    <SelectGroup key={category.category}>
                      <SelectLabel>{category.category}</SelectLabel>
                      {availableInCategory.map((resource) => (
                        <SelectItem
                          key={resource.resource}
                          value={resource.resource}
                        >
                          {resource.displayName}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  );
                })}
              </SelectContent>
            </Select>
          </div>

          {selectedResource && (
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label>Operations</Label>
                <div className="flex gap-2">
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="h-6 text-xs"
                    onClick={selectAllOperations}
                  >
                    Select All
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="h-6 text-xs"
                    onClick={clearAllOperations}
                  >
                    Clear
                  </Button>
                </div>
              </div>
              <div className="flex flex-wrap gap-3 rounded-md border p-3">
                {availableOperations.map((opDef) => (
                  <label
                    key={opDef.operation}
                    className="flex items-center gap-1.5 text-sm"
                    title={opDef.description}
                  >
                    <Checkbox
                      checked={selectedOperations.includes(
                        opDef.operation as Operation,
                      )}
                      onCheckedChange={() =>
                        toggleOperation(opDef.operation as Operation)
                      }
                    />
                    {opDef.displayName}
                  </label>
                ))}
              </div>
            </div>
          )}

          <div className="space-y-2">
            <Label>Data Scope</Label>
            <Select
              value={selectedScope}
              onValueChange={(value) => setSelectedScope(value as DataScope)}
            >
              <SelectTrigger>
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
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
          >
            Cancel
          </Button>
          <Button
            type="button"
            onClick={handleSubmit}
            disabled={
              isSubmitting ||
              !selectedResource ||
              selectedOperations.length === 0
            }
          >
            Add Permission
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

type CreateRolePermissionsEditorProps = {
  permissions: AddPermission[];
  onAddPermission: (permission: AddPermission) => void;
  onRemovePermission: (index: number) => void;
  onUpdatePermission: (index: number, permission: AddPermission) => void;
};

export function CreateRolePermissionsEditor({
  permissions,
  onAddPermission,
  onRemovePermission,
  onUpdatePermission,
}: CreateRolePermissionsEditorProps) {
  const [addDialogOpen, setAddDialogOpen] = useState(false);

  const { data: resourceCategories = [] } = useQuery({
    queryKey: ["permission-resources"],
    queryFn: getAvailableResources,
    staleTime: 1000 * 60 * 30,
  });

  const resourceMap = useMemo(() => {
    const map = new Map<string, ResourceDefinition>();
    for (const category of resourceCategories) {
      for (const resource of category.resources) {
        map.set(resource.resource, resource);
      }
    }
    return map;
  }, [resourceCategories]);

  const handleOperationToggle = useCallback(
    (index: number, permission: AddPermission, operation: Operation) => {
      const currentOps = permission.operations;
      const hasOp = currentOps.includes(operation);

      let newOps: Operation[];
      if (hasOp) {
        newOps = currentOps.filter((op) => op !== operation);
      } else {
        newOps = [...currentOps, operation];
      }

      if (newOps.length === 0) {
        toast.error("At least one operation is required");
        return;
      }

      onUpdatePermission(index, { ...permission, operations: newOps });
    },
    [onUpdatePermission],
  );

  const handleDataScopeChange = useCallback(
    (index: number, permission: AddPermission, newScope: DataScope) => {
      onUpdatePermission(index, { ...permission, dataScope: newScope });
    },
    [onUpdatePermission],
  );

  const handleAddPermission = useCallback(
    (permission: AddPermission) => {
      onAddPermission(permission);
      setAddDialogOpen(false);
    },
    [onAddPermission],
  );

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium">Permissions</h3>
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={() => setAddDialogOpen(true)}
        >
          <PlusIcon className="mr-1 size-3.5" />
          Add
        </Button>
      </div>

      {permissions.length === 0 ? (
        <div className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
          No permissions configured. Add permissions to define what this role
          can do.
        </div>
      ) : (
        <div className="space-y-3">
          {permissions.map((permission, index) => (
            <CreatePermissionRow
              key={`${permission.resource}-${index}`}
              permission={permission}
              resourceDef={resourceMap.get(permission.resource)}
              onOperationToggle={(op) =>
                handleOperationToggle(index, permission, op)
              }
              onDataScopeChange={(scope) =>
                handleDataScopeChange(index, permission, scope)
              }
              onRemove={() => onRemovePermission(index)}
            />
          ))}
        </div>
      )}

      <CreateAddPermissionDialog
        open={addDialogOpen}
        onOpenChange={setAddDialogOpen}
        existingResources={permissions.map((p) => p.resource)}
        resourceCategories={resourceCategories}
        onAdd={handleAddPermission}
      />
    </div>
  );
}

type CreatePermissionRowProps = {
  permission: AddPermission;
  resourceDef?: ResourceDefinition;
  onOperationToggle: (operation: Operation) => void;
  onDataScopeChange: (scope: DataScope) => void;
  onRemove: () => void;
};

function CreatePermissionRow({
  permission,
  resourceDef,
  onOperationToggle,
  onDataScopeChange,
  onRemove,
}: CreatePermissionRowProps) {
  const resourceLabel = resourceDef?.displayName ?? permission.resource;
  const availableOperations = resourceDef?.operations ?? [];

  return (
    <div className="rounded-md border p-3">
      <div className="flex items-start justify-between gap-2">
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <p className="text-sm font-medium">{resourceLabel}</p>
            {resourceDef?.category && (
              <span className="text-xs text-muted-foreground">
                ({resourceDef.category})
              </span>
            )}
          </div>
          <div className="mt-2 flex flex-wrap gap-3">
            {availableOperations.map((opDef) => (
              <label
                key={opDef.operation}
                className="flex items-center gap-1.5 text-xs text-muted-foreground"
                title={opDef.description}
              >
                <Checkbox
                  checked={permission.operations.includes(
                    opDef.operation as Operation,
                  )}
                  onCheckedChange={() =>
                    onOperationToggle(opDef.operation as Operation)
                  }
                />
                {opDef.displayName}
              </label>
            ))}
          </div>
          <div className="mt-3 flex items-center gap-2">
            <span className="text-xs text-muted-foreground">Scope:</span>
            <Select
              value={permission.dataScope}
              onValueChange={(value) => onDataScopeChange(value as DataScope)}
            >
              <SelectTrigger className="h-7 w-36 text-xs">
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
        <Button
          type="button"
          size="icon-sm"
          variant="ghost"
          className="text-destructive hover:bg-destructive/10 hover:text-destructive"
          onClick={onRemove}
        >
          <TrashIcon className="size-4" />
        </Button>
      </div>
    </div>
  );
}

type CreateAddPermissionDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  existingResources: string[];
  resourceCategories: ResourceCategory[];
  onAdd: (permission: AddPermission) => void;
};

function CreateAddPermissionDialog({
  open,
  onOpenChange,
  existingResources,
  resourceCategories,
  onAdd,
}: CreateAddPermissionDialogProps) {
  const [selectedResource, setSelectedResource] = useState<string>("");
  const [selectedOperations, setSelectedOperations] = useState<Operation[]>([
    "read",
  ]);
  const [selectedScope, setSelectedScope] = useState<DataScope>("organization");

  const selectedResourceDef = useMemo(() => {
    for (const category of resourceCategories) {
      const resource = category.resources.find(
        (r) => r.resource === selectedResource,
      );
      if (resource) return resource;
    }
    return null;
  }, [selectedResource, resourceCategories]);

  const availableOperations = selectedResourceDef?.operations ?? [];

  const handleResourceChange = (value: string) => {
    setSelectedResource(value);
    setSelectedOperations(["read"]);
  };

  const handleSubmit = () => {
    if (!selectedResource) {
      toast.error("Please select a resource");
      return;
    }

    if (selectedOperations.length === 0) {
      toast.error("Please select at least one operation");
      return;
    }

    onAdd({
      resource: selectedResource,
      operations: selectedOperations,
      dataScope: selectedScope,
    });
    setSelectedResource("");
    setSelectedOperations(["read"]);
    setSelectedScope("organization");
  };

  const toggleOperation = (op: Operation) => {
    setSelectedOperations((prev) =>
      prev.includes(op) ? prev.filter((o) => o !== op) : [...prev, op],
    );
  };

  const selectAllOperations = () => {
    setSelectedOperations(
      availableOperations.map((op) => op.operation as Operation),
    );
  };

  const clearAllOperations = () => {
    setSelectedOperations([]);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Add Permission</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label>Resource</Label>
            <Select
              value={selectedResource}
              onValueChange={(value) => handleResourceChange(value ?? "")}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent className="max-h-80">
                {resourceCategories.map((category) => {
                  const availableInCategory = category.resources.filter(
                    (r) => !existingResources.includes(r.resource),
                  );
                  if (availableInCategory.length === 0) return null;

                  return (
                    <SelectGroup key={category.category}>
                      <SelectLabel>{category.category}</SelectLabel>
                      {availableInCategory.map((resource) => (
                        <SelectItem
                          key={resource.resource}
                          value={resource.resource}
                        >
                          {resource.displayName}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  );
                })}
              </SelectContent>
            </Select>
          </div>

          {selectedResource && (
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label>Operations</Label>
                <div className="flex gap-2">
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="h-6 text-xs"
                    onClick={selectAllOperations}
                  >
                    Select All
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="h-6 text-xs"
                    onClick={clearAllOperations}
                  >
                    Clear
                  </Button>
                </div>
              </div>
              <div className="flex flex-wrap gap-3 rounded-md border p-3">
                {availableOperations.map((opDef) => (
                  <label
                    key={opDef.operation}
                    className="flex items-center gap-1.5 text-sm"
                    title={opDef.description}
                  >
                    <Checkbox
                      checked={selectedOperations.includes(
                        opDef.operation as Operation,
                      )}
                      onCheckedChange={() =>
                        toggleOperation(opDef.operation as Operation)
                      }
                    />
                    {opDef.displayName}
                  </label>
                ))}
              </div>
            </div>
          )}

          <div className="space-y-2">
            <Label>Data Scope</Label>
            <Select
              value={selectedScope}
              onValueChange={(value) => setSelectedScope(value as DataScope)}
            >
              <SelectTrigger>
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
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
          >
            Cancel
          </Button>
          <Button
            type="button"
            onClick={handleSubmit}
            disabled={!selectedResource || selectedOperations.length === 0}
          >
            Add Permission
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
