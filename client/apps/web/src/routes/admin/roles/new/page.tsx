import { useT } from "@trenova/shared/i18n/use-t";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { Form } from "@trenova/shared/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useBreadcrumbLabel } from "@/hooks/use-breadcrumb-label";
import { api } from "@trenova/shared/lib/api";
import type { AddPermission, CreateRole, Role } from "@trenova/shared/types/role";
import { createRoleSchema } from "@trenova/shared/types/role";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import { RolePageLayout } from "../_components/role-builder-layout";
import { RoleForm } from "../_components/role-form";
import { RolePermissionMatrix } from "../_components/role-permission-matrix";
import { RoleTemplateSelector } from "../_components/role-template-selector";

export function RoleCreatePage() {
  const t = useT();

  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [permissions, setPermissions] = useState<AddPermission[]>([]);
  const [selectedTemplate, setSelectedTemplate] = useState<string | null>(null);
  useBreadcrumbLabel("New Role");

  const form = useForm<CreateRole>({
    resolver: zodResolver(createRoleSchema),
    defaultValues: {
      name: "",
      description: "",
      maxSensitivity: "internal",
      permissions: [],
    },
  });

  const {
    formState: { isSubmitting },
    handleSubmit,
  } = form;

  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: CreateRole) => {
      const payload = { ...values, permissions };
      const response = await api.post<Role>("/roles/", payload);
      return response;
    },
    onSuccess: () => {
      toast.success(t("Role created successfully"));
      void queryClient.invalidateQueries({ queryKey: ["role-list"] });
      void navigate("/admin/roles");
    },
    form,
    resourceName: "Role",
  });

  const onSubmit = handleSubmit(async (values) => {
    await mutateAsync(values);
  });

  const handleTemplateSelect = useCallback(
    (templateId: string, templatePermissions: AddPermission[]) => {
      setSelectedTemplate(templateId);
      setPermissions(templatePermissions);
    },
    [],
  );

  const handleCancel = useCallback(() => {
    void navigate("/admin/roles");
  }, [navigate]);

  return (
    <FormProvider {...form}>
      <Form onSubmit={onSubmit}>
        <RolePageLayout
          title={t("Create Role")}
          isSubmitting={isSubmitting}
          submitLabel={t("Create Role")}
          onSubmit={onSubmit}
          onCancel={handleCancel}
          permissionCount={permissions.length}
        >
          <Card>
            <CardHeader>
              <CardTitle>{t("Role Details")}</CardTitle>
              <CardDescription>{t("Basic information for this role")}</CardDescription>
            </CardHeader>
            <CardContent>
              <RoleForm />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t("Permissions")}</CardTitle>
              <CardDescription>{t("Configure resource access and operations")}</CardDescription>
              <CardAction>
                <RoleTemplateSelector
                  selectedTemplate={selectedTemplate}
                  onSelectTemplate={handleTemplateSelect}
                />
              </CardAction>
            </CardHeader>
            <CardContent>
              <RolePermissionMatrix
                permissions={permissions}
                onPermissionsChange={setPermissions}
              />
            </CardContent>
          </Card>
        </RolePageLayout>
      </Form>
    </FormProvider>
  );
}
