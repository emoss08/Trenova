import { DataTable } from "@/components/data-table/data-table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { RoleSchema, UserSchema } from "@/lib/schemas/user-schema";
import { Resource } from "@/types/audit-entry";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useLocation, useNavigate } from "react-router";
import { EditRoleSheet } from "./role-edit-modal";
import { getRoleColumns, getUserColumns } from "./user-columns";
import { CreateUserModal } from "./user-create-modal";
import { EditUserModal } from "./user-edit-modal";

export default function UserRolesTable() {
  const location = useLocation();
  const navigate = useNavigate();

  const [activeTab, setActiveTab] = useState(() => {
    const hashTab = location.hash.slice(1);
    return hashTab === "roles" ? "roles" : "users";
  });

  useEffect(() => {
    const hashTab = location.hash.slice(1);
    const validTab = hashTab === "roles" ? "roles" : "users";
    setActiveTab(validTab);
  }, [location.hash]);

  const handleTabChange = useCallback(
    (value: string) => {
      setActiveTab(value);
      navigate(`${location.pathname}#${value}`, { replace: true });
    },
    [location.pathname, navigate],
  );

  return (
    <Tabs
      value={activeTab}
      onValueChange={handleTabChange}
      className="items-center"
    >
      <TabsList className="h-auto rounded-none border-b gap-4 bg-transparent p-0 w-full justify-start">
        <TabsTrigger
          value="users"
          className="group data-[state=active]:after:bg-primary data-[state=active]:text-primary relative rounded-none py-2 after:absolute after:inset-x-0 after:bottom-0 after:h-0.5 data-[state=active]:bg-transparent data-[state=active]:shadow-none"
        >
          Users
        </TabsTrigger>
        <TabsTrigger
          value="roles"
          className="group data-[state=active]:after:bg-primary data-[state=active]:text-primary relative rounded-none py-2 after:absolute after:inset-x-0 after:bottom-0 after:h-0.5 data-[state=active]:bg-transparent data-[state=active]:shadow-none"
        >
          Roles & Permissions
        </TabsTrigger>
      </TabsList>
      <TabsContent className="w-full" value="users">
        <UserTable />
      </TabsContent>
      <TabsContent className="w-full" value="roles">
        <RoleTable />
      </TabsContent>
    </Tabs>
  );
}

function UserTable() {
  const columns = useMemo(() => getUserColumns(), []);

  return (
    <DataTable<UserSchema>
      resource={Resource.User}
      name="User"
      link="/users/"
      queryKey="user-list"
      exportModelName="user"
      extraSearchParams={{
        includeRoles: true,
      }}
      TableModal={CreateUserModal}
      TableEditModal={EditUserModal}
      columns={columns}
      config={{
        enableFiltering: true,
        enableSorting: true,
        enableMultiSort: true,
        maxFilters: 5,
        maxSorts: 3,
        searchDebounce: 300,
        showFilterUI: true,
        showSortUI: true,
      }}
      useEnhancedBackend={true}
    />
  );
}

function RoleTable() {
  const columns = useMemo(() => getRoleColumns(), []);

  return (
    <DataTable<RoleSchema>
      resource={Resource.Role}
      name="Role"
      link="/roles/"
      queryKey="role-list"
      exportModelName="role"
      columns={columns}
      TableEditModal={EditRoleSheet}
      extraSearchParams={{
        includePermissions: true,
        // * TODO(wolfred): We should probably load the users assigned to the role as well
      }}
    />
  );
}
