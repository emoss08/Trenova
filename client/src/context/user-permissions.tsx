import { useAuthStore, useUserStore } from "@/stores/AuthStore";
import React, { useContext, useMemo } from "react";

export type UserPermissionsContextType = {
  isAuthenticated: boolean;
  isAdmin: boolean;
  permissions: string[];
  userHasPermission: (permission: string) => boolean;
};

const UserPermissionsContext =
  React.createContext<UserPermissionsContextType | null>(null);

export const UserPermissionsProvider: React.FC<{
  children: React.ReactNode;
}> = ({ children }) => {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const user = useUserStore.get("user");

  // User is admin or is super admin
  const isAdmin = user?.isAdmin || user?.isSuperAdmin;
  // User can have multiple roles. We need to get the name of each permission
  const permissions =
    user?.edges.roles
      ?.map((role) => role.edges.permissions)
      .flat()
      .map((permission) => permission.name) ?? [];

  const userHasPermission = useMemo(
    () => (permission: string) => isAdmin || permissions.includes(permission),
    [isAdmin, permissions],
  );

  const contextValue = useMemo(
    () => ({
      isAuthenticated,
      isAdmin,
      permissions,
      userHasPermission,
    }),
    [isAuthenticated, isAdmin, permissions, userHasPermission],
  );

  return (
    <UserPermissionsContext.Provider value={contextValue}>
      {children}
    </UserPermissionsContext.Provider>
  );
};

export const useUserPermissions = (): UserPermissionsContextType => {
  const context = useContext(UserPermissionsContext);
  if (!context) {
    throw new Error(
      "useUserPermissions must be used within a UserPermissionsProvider",
    );
  }
  return context;
};
