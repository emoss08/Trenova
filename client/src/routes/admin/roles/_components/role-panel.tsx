import { ComponentLoader } from "@/components/component-loader";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { Button } from "@/components/ui/button";
import { Form } from "@/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { api } from "@/lib/api";
import { formatToUserTimezone } from "@/lib/date";
import type { DataTablePanelProps } from "@/types/data-table";
import type { AddPermission, CreateRole, Role } from "@/types/role";
import { createRoleSchema, roleSchema } from "@/types/role";
import { TimeFormat } from "@/types/user";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { AlertTriangleIcon } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { RoleForm } from "./role-form";
import { RolePermissionBuilder } from "./role-permission-builder";
import { RolePermissionsEditor } from "./role-permissions-editor";

export function RolePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<Role>) {
  if (mode === "edit") {
    return <RoleEditPanel open={open} onOpenChange={onOpenChange} row={row} />;
  }

  return <RoleCreatePanel open={open} onOpenChange={onOpenChange} />;
}

type RoleCreatePanelProps = Pick<
  DataTablePanelProps<Role>,
  "open" | "onOpenChange"
>;

function RoleCreatePanel({ open, onOpenChange }: RoleCreatePanelProps) {
  const queryClient = useQueryClient();
  const [permissions, setPermissions] = useState<AddPermission[]>([]);

  const form = useForm<CreateRole>({
    resolver: zodResolver(createRoleSchema),
    defaultValues: {
      name: "",
      description: "",
      maxSensitivity: "internal",
      isOrgAdmin: false,
      isBusinessUnitAdmin: false,
      permissions: [],
    },
  });

  const {
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset();
    setPermissions([]);
  }, [onOpenChange, reset]);

  const handleOpenChange = useCallback(
    (isOpen: boolean) => {
      if (isOpen) {
        reset();
        setPermissions([]);
      }
      onOpenChange(isOpen);
    },
    [onOpenChange, reset],
  );

  const { mutateAsync } = useApiMutation<CreateRole, Role, unknown, CreateRole>(
    {
      mutationFn: async (values: CreateRole) => {
        const payload = { ...values, permissions };
        const response = await api.post<Role>("/roles/", payload);
        return response;
      },
      onSuccess: () => {
        toast.success("Changes have been saved", {
          description: "Role created successfully",
        });
        reset();
        setPermissions([]);
        onOpenChange(false);
        void queryClient.invalidateQueries({ queryKey: ["role-list"] });
      },
      setFormError: setError,
      resourceName: "Role",
    },
  );

  const onSubmit = useCallback(
    async (values: CreateRole) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (
        open &&
        (event.ctrlKey || event.metaKey) &&
        event.key === "Enter" &&
        !isSubmitting
      ) {
        event.preventDefault();
        void handleSubmit(onSubmit)();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open, isSubmitting, handleSubmit, onSubmit]);

  const handlePermissionsChange = useCallback(
    (newPermissions: AddPermission[]) => {
      setPermissions(newPermissions);
    },
    [],
  );

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={handleOpenChange}
      title="Add New Role"
      description="Define the role and configure what it can access."
      size="xl"
      footer={
        <>
          <Button type="button" variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button
            type="submit"
            form="role-create-form"
            isLoading={isSubmitting}
            loadingText="Creating..."
          >
            Create Role
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-6">
        <FormProvider {...form}>
          <Form id="role-create-form" onSubmit={handleSubmit(onSubmit)}>
            <RoleForm />
          </Form>
        </FormProvider>
        <div className="border-t pt-5">
          <RolePermissionBuilder
            permissions={permissions}
            onPermissionsChange={handlePermissionsChange}
          />
        </div>
      </div>
    </DataTablePanelContainer>
  );
}

type RoleEditPanelProps = Pick<
  DataTablePanelProps<Role>,
  "open" | "onOpenChange" | "row"
>;

function RoleEditPanel({ open, onOpenChange, row }: RoleEditPanelProps) {
  const queryClient = useQueryClient();

  const form = useForm<Role>({
    resolver: zodResolver(roleSchema),
    defaultValues: {
      name: "",
      description: "",
      maxSensitivity: "internal",
      isOrgAdmin: false,
      isBusinessUnitAdmin: false,
    },
  });

  const {
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  const handleClose = () => {
    onOpenChange(false);
    reset();
  };

  useEffect(() => {
    if (open && row) {
      reset(row as Role, { keepDefaultValues: true });
    }
  }, [open, row?.id, reset, row]);

  const { mutateAsync } = useApiMutation<Role, Role, unknown, Role>({
    mutationFn: async (values: Role) => {
      const response = await api.put<Role>(`/roles/${row?.id}`, values);
      return response;
    },
    onMutate: async (newValues) => {
      await queryClient.cancelQueries({ queryKey: ["role-list"] });
      const previousRecord = queryClient.getQueryData(["role-list"]);
      queryClient.setQueryData(["role-list"], newValues);
      return { previousRecord, newValues };
    },
    onSuccess: () => {
      toast.success("Changes have been saved", {
        description: "Role updated successfully",
      });
      reset();
      onOpenChange(false);
      void queryClient.invalidateQueries({ queryKey: ["role-list"] });
    },
    setFormError: setError,
    resourceName: "Role",
  });

  const onSubmit = useCallback(
    async (values: Role) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (
        open &&
        (event.ctrlKey || event.metaKey) &&
        event.key === "Enter" &&
        !isSubmitting
      ) {
        event.preventDefault();
        void handleSubmit(onSubmit)();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open, isSubmitting, handleSubmit, onSubmit]);

  const panelDescription = row?.updatedAt
    ? `Last updated on ${formatToUserTimezone(row.updatedAt as number, {
        timeFormat: TimeFormat.enum["24-hour"],
      })}`
    : undefined;

  const isSystemRole = row?.isSystem ?? false;

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={row?.name ?? "Role"}
      description={panelDescription}
      size="lg"
      footer={
        <>
          <Button type="button" variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button
            type="submit"
            form="role-edit-form"
            isLoading={isSubmitting}
            loadingText="Saving..."
            disabled={isSystemRole}
          >
            Save
          </Button>
        </>
      }
    >
      {!row ? (
        <ComponentLoader message="Loading Role..." />
      ) : (
        <div className="flex flex-col gap-6">
          {isSystemRole && (
            <div className="flex items-center gap-2 rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-900 dark:bg-amber-950 dark:text-amber-200">
              <AlertTriangleIcon className="size-4 shrink-0" />
              <span>This is a system role and cannot be modified.</span>
            </div>
          )}
          <FormProvider {...form}>
            <Form id="role-edit-form" onSubmit={handleSubmit(onSubmit)}>
              <RoleForm isSystemRole={isSystemRole} />
            </Form>
          </FormProvider>
          <div className="border-t pt-4">
            <RolePermissionsEditor
              roleId={row.id!}
              isSystemRole={isSystemRole}
            />
          </div>
        </div>
      )}
    </DataTablePanelContainer>
  );
}
