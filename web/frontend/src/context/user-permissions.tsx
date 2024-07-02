import { useAuthStore, useUserStore } from "@/stores/AuthStore";
import React, { useContext, useMemo } from "react";

export type UserPermissionsContextType = {
  isAuthenticated: boolean;
  isAdmin: boolean;
  permissions: string[];
  userHasPermission: (permission: string) => boolean;
};

export const UserPermissionsContext =
  React.createContext<UserPermissionsContextType | null>(null);

export const UserPermissionsProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const user = useUserStore.get("user");

  const isAdmin = user?.isAdmin;

  // User is admin or is super admin
  const permissions = useMemo(
    () =>
      user?.role
        ?.map((role) => role.permissions)
        .flat()
        .map((permission) => permission.codename) ?? [],
    [user],
  );

  const contextValue = useMemo(() => {
    const userHasPermission = (permission: string) =>
      isAdmin || permissions.includes(permission);

    return {
      isAuthenticated,
      isAdmin,
      permissions,
      userHasPermission,
    };
  }, [isAuthenticated, isAdmin, permissions]);

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
