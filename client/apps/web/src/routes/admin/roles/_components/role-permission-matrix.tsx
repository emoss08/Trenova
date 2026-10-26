import { useT } from "@trenova/shared/i18n/use-t";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { Tabs, TabsContent, TabsList, TabsTab } from "@trenova/shared/components/ui/tabs";
import type { AddPermission } from "@trenova/shared/types/role";
import { Activity, lazy, useState } from "react";
import { RoleBundlePicker } from "./role-bundle-picker";
import { RoleResourceAccessSkeleton } from "./role-resource-access-skeleton";

const RoleResourceAccess = lazy(() =>
  import("./role-resource-access").then((module) => ({ default: module.RoleResourceAccess })),
);

type PermissionTab = "capabilities" | "advanced";

type RolePermissionMatrixProps = {
  permissions: AddPermission[];
  onPermissionsChange: (permissions: AddPermission[]) => void;
  /** The role's sensitivity ceiling, which decides what its reference views show. */
  maxSensitivity: string;
};

export function RolePermissionMatrix({
  permissions,
  onPermissionsChange,
  maxSensitivity,
}: RolePermissionMatrixProps) {
  const t = useT();
  const [tab, setTab] = useState<PermissionTab>("capabilities");

  return (
    <Tabs
      value={tab}
      onValueChange={(value) => setTab(value as PermissionTab)}
      className="flex flex-col gap-4"
    >
      <TabsList variant="underline">
        <TabsTab value="capabilities">{t("Capabilities")}</TabsTab>
        <TabsTab value="advanced">{t("Advanced")}</TabsTab>
      </TabsList>
      <TabsContent value="capabilities" keepMounted>
        <Activity mode={tab === "capabilities" ? "visible" : "hidden"}>
          <RoleBundlePicker permissions={permissions} onPermissionsChange={onPermissionsChange} />
        </Activity>
      </TabsContent>
      <TabsContent value="advanced" keepMounted>
        <Activity mode={tab === "advanced" ? "visible" : "hidden"}>
          <LazyComponent fallback={<RoleResourceAccessSkeleton label={t("Loading resources")} />}>
            <RoleResourceAccess
              permissions={permissions}
              onPermissionsChange={onPermissionsChange}
              maxSensitivity={maxSensitivity}
            />
          </LazyComponent>
        </Activity>
      </TabsContent>
    </Tabs>
  );
}
