import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { api } from "@/lib/api";
import { formatToUserTimezone } from "@/lib/date";
import { assignRole, listRoles, unassignRole } from "@/lib/role-api";
import type { Role, UserRoleAssignment } from "@/types/role";
import { TimeFormat } from "@/types/user";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  CalendarIcon,
  LockIcon,
  PlusIcon,
  ShieldCheckIcon,
  TrashIcon,
} from "lucide-react";
import { useCallback, useState } from "react";
import { toast } from "sonner";

type UserRolesEditorProps = {
  userId: string;
};

export function UserRolesEditor({ userId }: UserRolesEditorProps) {
  const queryClient = useQueryClient();
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const { data: assignments, isLoading } = useQuery({
    queryKey: ["user-role-assignments", userId],
    queryFn: () =>
      api.get<UserRoleAssignment[]>(`/users/${userId}/role-assignments`),
    select: (response) => response,
  });

  const handleUnassign = useCallback(
    async (assignmentId: string) => {
      try {
        await unassignRole(assignmentId);
        await queryClient.invalidateQueries({
          queryKey: ["user-role-assignments", userId],
        });
        toast.success("Role unassigned");
      } catch {
        toast.error("Failed to unassign role");
      }
    },
    [userId, queryClient],
  );

  if (isLoading) {
    return (
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <Skeleton className="h-5 w-28" />
          <Skeleton className="h-8 w-20" />
        </div>
        <Skeleton className="h-20 w-full" />
        <Skeleton className="h-20 w-full" />
      </div>
    );
  }

  const roleAssignments = assignments ?? [];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium">Assigned Roles</h3>
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={() => setAddDialogOpen(true)}
        >
          <PlusIcon className="mr-1 size-3.5" />
          Assign Role
        </Button>
      </div>

      {roleAssignments.length === 0 ? (
        <div className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
          No roles assigned to this user.
        </div>
      ) : (
        <div className="space-y-3">
          {roleAssignments.map((assignment) => (
            <RoleAssignmentRow
              key={assignment.id}
              assignment={assignment}
              onUnassign={() => handleUnassign(assignment.id!)}
            />
          ))}
        </div>
      )}

      <AssignRoleDialog
        open={addDialogOpen}
        onOpenChange={setAddDialogOpen}
        userId={userId}
        existingRoleIds={roleAssignments.map((a) => a.roleId)}
        isSubmitting={isSubmitting}
        setIsSubmitting={setIsSubmitting}
      />
    </div>
  );
}

type RoleAssignmentRowProps = {
  assignment: UserRoleAssignment;
  onUnassign: () => void;
};

function RoleAssignmentRow({ assignment, onUnassign }: RoleAssignmentRowProps) {
  const role = assignment.role;

  const expiresText = assignment.expiresAt
    ? `Expires: ${formatToUserTimezone(assignment.expiresAt, {
        timeFormat: TimeFormat.enum["24-hour"],
      })}`
    : "Never expires";

  const assignedText = assignment.assignedAt
    ? `Assigned: ${formatToUserTimezone(assignment.assignedAt, {
        timeFormat: TimeFormat.enum["24-hour"],
      })}`
    : "";

  return (
    <div className="rounded-md border p-3">
      <div className="flex items-start justify-between gap-2">
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <p className="text-sm font-medium">
              {role?.name ?? "Unknown Role"}
            </p>
            {role?.isSystem && (
              <Badge variant="secondary" className="gap-1 text-xs">
                <LockIcon className="size-3" />
                System
              </Badge>
            )}
            {role?.isOrgAdmin && (
              <Badge
                variant="default"
                className="gap-1 bg-amber-600 text-xs hover:bg-amber-700"
              >
                <ShieldCheckIcon className="size-3" />
                Admin
              </Badge>
            )}
          </div>
          {role?.description && (
            <p className="mt-1 text-xs text-muted-foreground">
              {role.description}
            </p>
          )}
          <div className="mt-2 flex items-center gap-3 text-xs text-muted-foreground">
            <span>{assignedText}</span>
            <span className="flex items-center gap-1">
              <CalendarIcon className="size-3" />
              {expiresText}
            </span>
          </div>
        </div>
        <Button
          type="button"
          size="icon-sm"
          variant="ghost"
          className="text-destructive hover:bg-destructive/10 hover:text-destructive"
          onClick={onUnassign}
        >
          <TrashIcon className="size-4" />
        </Button>
      </div>
    </div>
  );
}

type AssignRoleDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  userId: string;
  existingRoleIds: string[];
  isSubmitting: boolean;
  setIsSubmitting: (submitting: boolean) => void;
};

function AssignRoleDialog({
  open,
  onOpenChange,
  userId,
  existingRoleIds,
  isSubmitting,
  setIsSubmitting,
}: AssignRoleDialogProps) {
  const queryClient = useQueryClient();
  const [selectedRoleId, setSelectedRoleId] = useState<string>("");
  const [expiresAt, setExpiresAt] = useState<string>("");

  const { data: rolesResponse } = useQuery({
    queryKey: ["roles-for-assignment"],
    queryFn: () => listRoles({ includeSystem: true }),
    enabled: open,
  });

  const availableRoles = (rolesResponse?.results ?? []).filter(
    (role: Role) => !existingRoleIds.includes(role.id!),
  );

  const handleSubmit = async () => {
    if (!selectedRoleId) {
      toast.error("Please select a role");
      return;
    }

    setIsSubmitting(true);
    const expiresAtTimestamp = expiresAt
      ? new Date(expiresAt).getTime() / 1000
      : null;

    await assignRole(selectedRoleId, {
      userId,
      expiresAt: expiresAtTimestamp,
    })
      .then(async () => {
        await queryClient.invalidateQueries({
          queryKey: ["user-role-assignments", userId],
        });
        toast.success("Role assigned");
        onOpenChange(false);
        setSelectedRoleId("");
        setExpiresAt("");
      })
      .catch(() => {
        toast.error("Failed to assign role");
      })
      .finally(() => {
        setIsSubmitting(false);
      });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Assign Role</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label>Role</Label>
            <Select
              value={selectedRoleId}
              onValueChange={(value) => setSelectedRoleId(value ?? "")}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {availableRoles.map((role: Role) => (
                  <SelectItem key={role.id} value={role.id!}>
                    <div className="flex items-center gap-2">
                      <span>{role.name}</span>
                      {role.isOrgAdmin && (
                        <Badge
                          variant="default"
                          className="bg-amber-600 text-xs hover:bg-amber-700"
                        >
                          Admin
                        </Badge>
                      )}
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label>Expires At (Optional)</Label>
            <Input
              type="datetime-local"
              value={expiresAt}
              onChange={(e) => setExpiresAt(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Leave empty for permanent assignment
            </p>
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
            disabled={isSubmitting || !selectedRoleId}
          >
            Assign Role
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
