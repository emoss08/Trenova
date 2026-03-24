import { DataTableLazyComponent } from "@/components/error-boundary";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { parseAsStringLiteral, useQueryState } from "nuqs";
import { lazy } from "react";

const UserTable = lazy(() => import("./user-table"));
const RoleTable = lazy(
  () => import("@/routes/admin/roles/_components/role-table"),
);

const tabValues = ["users", "roles"] as const;

export default function UserRolesTable() {
  const [activeTab, setActiveTab] = useQueryState(
    "tab",
    parseAsStringLiteral(tabValues).withDefault(tabValues[0]),
  );

  return (
    <Tabs
      value={activeTab}
      onValueChange={(value) =>
        setActiveTab(value as (typeof tabValues)[number])
      }
    >
      <TabsList variant="underline">
        <TabsTrigger value="users">Users</TabsTrigger>
        <TabsTrigger value="roles">Roles & Permissions</TabsTrigger>
      </TabsList>
      <TabsContent value="users">
        <DataTableLazyComponent>
          <UserTable />
        </DataTableLazyComponent>
      </TabsContent>
      <TabsContent value="roles">
        <DataTableLazyComponent>
          <RoleTable />
        </DataTableLazyComponent>
      </TabsContent>
    </Tabs>
  );
}
