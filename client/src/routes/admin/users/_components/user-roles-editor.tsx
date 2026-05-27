import { RoleSelectAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateTimeField } from "@/components/fields/date-field/datetime-field";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { Skeleton } from "@/components/ui/skeleton";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { api } from "@/lib/api";
import { formatToUserTimezone } from "@/lib/date";
import { assignRole, unassignRole } from "@/lib/role-api";
import type { UserRoleAssignment } from "@/types/role";
import { TimeFormat } from "@/types/user";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { CalendarIcon, PlusIcon, TrashIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

const assignRoleFormSchema = z.object({
  roleId: z.string().min(1, "Role is required"),
  expiresAt: z.number().nullable().optional(),
});

type AssignRoleFormValues = z.infer<typeof assignRoleFormSchema>;

const assignRoleDefaultValues: AssignRoleFormValues = {
  roleId: "",
  expiresAt: null,
};

type UserRolesEditorProps = {
  userId: string;
  isDisabled?: boolean;
};

export function UserRolesEditor({ userId, isDisabled = false }: UserRolesEditorProps) {
  const queryClient = useQueryClient();
  const [addDialogOpen, setAddDialogOpen] = useState(false);

  const { data: assignments, isLoading } = useQuery({
    queryKey: ["user-role-assignments", userId],
    queryFn: () => api.get<UserRoleAssignment[]>(`/users/${userId}/role-assignments/`),
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
          disabled={isDisabled}
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
              isDisabled={isDisabled}
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
        isDisabled={isDisabled}
      />
    </div>
  );
}

type RoleAssignmentRowProps = {
  assignment: UserRoleAssignment;
  isDisabled: boolean;
  onUnassign: () => void;
};

function RoleAssignmentRow({ assignment, isDisabled, onUnassign }: RoleAssignmentRowProps) {
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
          <p className="text-sm font-medium">{role?.name ?? "Unknown Role"}</p>
          {role?.description && (
            <p className="mt-1 text-xs text-muted-foreground">{role.description}</p>
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
          disabled={isDisabled}
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
  isDisabled: boolean;
};

function AssignRoleDialog({
  open,
  onOpenChange,
  userId,
  existingRoleIds,
  isDisabled,
}: AssignRoleDialogProps) {
  const queryClient = useQueryClient();
  const form = useForm<AssignRoleFormValues>({
    resolver: zodResolver(assignRoleFormSchema),
    defaultValues: assignRoleDefaultValues,
  });

  const {
    control,
    handleSubmit,
    reset,
    setError,
    formState: { isSubmitting },
  } = form;

  const assignedRoleIDSet = useMemo(() => new Set(existingRoleIds), [existingRoleIds]);
  const filterAvailableRole = useCallback(
    (role: { id?: string | null }) => !!role.id && !assignedRoleIDSet.has(role.id),
    [assignedRoleIDSet],
  );

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset(assignRoleDefaultValues);
  }, [onOpenChange, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    UserRoleAssignment,
    AssignRoleFormValues,
    unknown,
    AssignRoleFormValues
  >({
    mutationFn: (values) =>
      assignRole(values.roleId, {
        userId,
        expiresAt: values.expiresAt ?? null,
      }),
    resourceName: "Role Assignment",
    setFormError: setError,
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ["user-role-assignments", userId],
      });
      toast.success("Role assigned");
    },
  });

  const onSubmit = useCallback(
    async (values: AssignRoleFormValues) => {
      if (isDisabled) return;
      await mutateAsync(values);
      handleClose();
    },
    [handleClose, isDisabled, mutateAsync],
  );

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (nextOpen) {
          onOpenChange(true);
          return;
        }
        handleClose();
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Assign Role</DialogTitle>
        </DialogHeader>
        <Form
          onSubmit={(event) => {
            event.stopPropagation();
            void handleSubmit(onSubmit)(event);
          }}
        >
          <FormGroup cols={1} className="pb-4">
            <FormControl>
              <RoleSelectAutocompleteField<AssignRoleFormValues>
                control={control}
                name="roleId"
                label="Role"
                placeholder="Select role"
                clearable
                disabled={isDisabled}
                filterOption={filterAvailableRole}
                noResultsMessage="No available roles found."
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl>
              <AutoCompleteDateTimeField<AssignRoleFormValues>
                control={control}
                name="expiresAt"
                label="Expires At"
                description="Leave empty for permanent assignment"
                placeholder="No expiration"
                clearable
                disabled={isDisabled}
              />
            </FormControl>
          </FormGroup>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose}>
              Cancel
            </Button>
            <Button
              type="submit"
              isLoading={isSubmitting || isPending}
              loadingText="Assigning..."
              disabled={isDisabled}
            >
              Assign Role
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
