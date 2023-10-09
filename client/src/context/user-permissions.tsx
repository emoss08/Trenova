import { useAuthStore, useUserStore } from "@/stores/AuthStore";
import React, { useContext } from "react";

interface UserPermissionsContextType {
  isAuthenticated: boolean;
  isAdmin: boolean;
  permissions: string[];
  userHasPermission: (permission: string) => boolean;
}

const UserPermissionsContext =
  React.createContext<UserPermissionsContextType | null>(null);

export const UserPermissionsProvider: React.FC<{
  children: React.ReactNode;
}> = ({ children }) => {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const isAdmin = useUserStore.get("user").userIsStaff;
  const permissions = useUserStore.get("user").userPermissions || [];

  const userHasPermission = (permission: string) =>
    isAdmin || permissions.includes(permission);

  return (
    <UserPermissionsContext.Provider
      value={{ isAuthenticated, isAdmin, permissions, userHasPermission }}
    >
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
