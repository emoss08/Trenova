import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@trenova/shared/components/ui/tabs";
import { useQueryState } from "nuqs";
import { Activity, lazy } from "react";
import {
  USER_ROLES_COLUMN_COUNTS,
  userRolesTabParser,
  type UserRolesTab,
} from "./user-roles-skeleton";

const UserTable = lazy(() => import("./user-table"));
const RoleTable = lazy(() => import("@/routes/admin/roles/_components/role-table"));

export default function UserRolesTable() {
  const t = useT();

  const [activeTab, setActiveTab] = useQueryState("tab", userRolesTabParser);

  return (
    <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as UserRolesTab)}>
      <TabsList variant="underline">
        <TabsTrigger value="users">{t("Users")}</TabsTrigger>
        <TabsTrigger value="roles">{t("Roles & permissions")}</TabsTrigger>
      </TabsList>
      <TabsContent value="users" keepMounted>
        <Activity mode={activeTab === "users" ? "visible" : "hidden"}>
          <DataTableLazyComponent columnCount={USER_ROLES_COLUMN_COUNTS.users}>
            <UserTable />
          </DataTableLazyComponent>
        </Activity>
      </TabsContent>
      <TabsContent value="roles" keepMounted>
        <Activity mode={activeTab === "roles" ? "visible" : "hidden"}>
          <DataTableLazyComponent columnCount={USER_ROLES_COLUMN_COUNTS.roles}>
            <RoleTable />
          </DataTableLazyComponent>
        </Activity>
      </TabsContent>
    </Tabs>
  );
}
