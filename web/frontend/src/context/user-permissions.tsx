/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

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
