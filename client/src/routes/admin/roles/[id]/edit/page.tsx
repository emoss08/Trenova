import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { Skeleton } from "@/components/ui/skeleton";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { fieldSensitivityChoices } from "@/lib/choices";
import { getRole, updateRole } from "@/lib/role-api";
import type { AddPermission, CreateRole, Role } from "@/types/role";
import { createRoleSchema } from "@/types/role";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangleIcon } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef } from "react";
import { FormProvider, useForm, useWatch } from "react-hook-form";
import { useNavigate, useParams } from "react-router";
import { toast } from "sonner";
import { RolePageLayout } from "../../_components/role-builder-layout";
import { RolePermissionMatrix } from "../../_components/role-permission-matrix";

export function RoleEditPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const isInitializedRef = useRef(false);

  const { data: role, isLoading } = useQuery({
    queryKey: ["role", id],
    queryFn: () => getRole(id!),
    enabled: !!id,
  });

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
    control,
    setError,
    setValue,
    reset,
    formState: { isSubmitting },
    handleSubmit,
  } = form;
  const watchedPermissions = useWatch({ control, name: "permissions" });
  const permissions = useMemo(() => watchedPermissions ?? [], [watchedPermissions]);

  useEffect(() => {
    if (role && !isInitializedRef.current) {
      const mappedPermissions: AddPermission[] = (role.permissions ?? []).map((p) => ({
        resource: p.resource,
        operations: p.operations,
        dataScope: p.dataScope,
      }));

      reset({
        name: role.name,
        description: role.description ?? "",
        maxSensitivity: role.maxSensitivity,
        isOrgAdmin: role.isOrgAdmin ?? false,
        isBusinessUnitAdmin: role.isBusinessUnitAdmin ?? false,
        permissions: mappedPermissions,
      });
      isInitializedRef.current = true;
    }
  }, [role, reset]);

  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: Partial<Role>) => {
      const payload = { ...values, permissions };
      const response = await updateRole(id!, payload);
      return response;
    },
    onSuccess: () => {
      toast.success("Role updated successfully");
      void queryClient.invalidateQueries({ queryKey: ["role-list"] });
      void queryClient.invalidateQueries({ queryKey: ["role", id] });
      void navigate("/admin/roles");
    },
    setFormError: setError,
    resourceName: "Role",
  });

  const onSubmit = handleSubmit(async (values) => {
    await mutateAsync(values);
  });

  const handleCancel = useCallback(() => {
    void navigate("/admin/roles");
  }, [navigate]);

  if (isLoading) {
    return (
      <div className="flex flex-col overflow-hidden rounded-md border border-border bg-background">
        <header className="shrink-0 border-b bg-card/95 px-6 py-3">
          <Skeleton className="h-8 w-48" />
        </header>
        <div className="mx-auto w-full max-w-5xl space-y-6 px-6 py-8">
          <Skeleton className="h-48 w-full rounded-xl" />
          <Skeleton className="h-96 w-full rounded-xl" />
        </div>
      </div>
    );
  }

  if (!role) {
    return (
      <div className="flex h-screen items-center justify-center bg-background">
        <p className="text-muted-foreground">Role not found</p>
      </div>
    );
  }

  const systemRoleBanner = role.isSystem ? (
    <div className="mx-auto w-full max-w-5xl px-6 pt-8">
      <div className="flex items-start gap-3 rounded-lg border border-amber-500/30 bg-amber-500/10 p-3">
        <AlertTriangleIcon className="size-4 shrink-0 text-amber-500" />
        <p className="text-xs text-amber-600 dark:text-amber-400">
          This is a system role. Some properties may be restricted.
        </p>
      </div>
    </div>
  ) : undefined;

  return (
    <FormProvider {...form}>
      <Form onSubmit={onSubmit}>
        <RolePageLayout
          title={`Edit ${role.name}`}
          isSubmitting={isSubmitting}
          submitLabel="Save Changes"
          onSubmit={onSubmit}
          onCancel={handleCancel}
          permissionCount={permissions.length}
          banner={systemRoleBanner}
        >
          <Card>
            <CardHeader>
              <CardTitle>Role Details</CardTitle>
              <CardDescription>Basic information for this role</CardDescription>
            </CardHeader>
            <CardContent>
              <FormGroup cols={2}>
                <FormControl>
                  <InputField
                    control={control}
                    rules={{ required: true }}
                    name="name"
                    label="Name"
                    placeholder="e.g., Dispatcher, Billing Clerk"
                    disabled={role.isSystem}
                  />
                </FormControl>
                <FormControl>
                  <SelectField
                    control={control}
                    rules={{ required: true }}
                    name="maxSensitivity"
                    label="Max Sensitivity"
                    description="Highest sensitivity level this role can access"
                    options={fieldSensitivityChoices}
                    isReadOnly={role.isSystem}
                  />
                </FormControl>
                <FormControl cols="full">
                  <TextareaField
                    control={control}
                    name="description"
                    label="Description"
                    placeholder="Describe what this role is for..."
                    disabled={role.isSystem}
                  />
                </FormControl>
                <FormControl cols="full">
                  <SwitchField
                    control={control}
                    name="isBusinessUnitAdmin"
                    label="Business Unit Administrator"
                    description="Grants admin-level access across all organizations in this business unit."
                    disabled={role.isSystem}
                    outlined
                    position="left"
                  />
                </FormControl>
              </FormGroup>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Permissions</CardTitle>
              <CardDescription>Configure resource access and operations</CardDescription>
            </CardHeader>
            <CardContent>
              <RolePermissionMatrix
                permissions={permissions}
                onPermissionsChange={(nextPermissions) =>
                  setValue("permissions", nextPermissions, {
                    shouldDirty: true,
                  })
                }
              />
            </CardContent>
          </Card>
        </RolePageLayout>
      </Form>
    </FormProvider>
  );
}
