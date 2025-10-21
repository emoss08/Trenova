/* eslint-disable react-refresh/only-export-components */

import { PermissionAPI } from "@/lib/permissions/permission-api";
import { PermissionClient } from "@/lib/permissions/permission-client";
import type {
  ActionName,
  FieldAccess,
  PermissionManifest,
  Resource,
} from "@/types/permission";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  type ReactNode,
} from "react";

interface PermissionContextValue {
  client: PermissionClient | null;
  manifest: PermissionManifest | null;
  isLoading: boolean;
  isExpired: boolean;
  error: Error | null;
  can: (resource: Resource, action: ActionName) => boolean;
  canAny: (resource: Resource, actions: ActionName[]) => boolean;
  canAll: (resource: Resource, actions: ActionName[]) => boolean;
  canAccessField: (resource: Resource, field: string) => boolean;
  canWriteField: (resource: Resource, field: string) => boolean;
  getFieldAccess: (resource: Resource, field: string) => FieldAccess | null;
  switchOrganization: (organizationId: string) => Promise<void>;
  getCurrentOrganization: () => string | null;
  getAvailableOrganizations: () => string[];
  refresh: () => Promise<void>;
  invalidateCache: () => Promise<void>;
}

const PermissionContext = createContext<PermissionContextValue | undefined>(
  undefined,
);

interface PermissionProviderProps {
  children: ReactNode;
  autoRefresh?: boolean;
  refreshInterval?: number;
}

const PERMISSION_QUERY_KEY = ["permissions", "manifest"];

export function PermissionProvider({
  children,
  autoRefresh = true,
  refreshInterval = 5 * 60 * 1000, // 5 minutes
}: PermissionProviderProps) {
  const queryClient = useQueryClient();

  const {
    data: manifest,
    error,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: PERMISSION_QUERY_KEY,
    queryFn: () => PermissionAPI.getManifest(),
    staleTime: refreshInterval,
    gcTime: refreshInterval * 2,
    refetchInterval: autoRefresh ? refreshInterval : false,
    refetchOnWindowFocus: false,
    retry: 3,
  });

  const client = useMemo(() => {
    if (!manifest) return null;
    return new PermissionClient(manifest);
  }, [manifest]);

  const isExpired = useMemo(() => {
    if (!client) return false;
    return client.isExpired();
  }, [client]);

  const refresh = useCallback(async () => {
    try {
      await PermissionAPI.refresh();
      await queryClient.invalidateQueries({ queryKey: PERMISSION_QUERY_KEY });
      await refetch();
    } catch (err) {
      console.error("Failed to refresh permissions:", err);
      throw err;
    }
  }, [queryClient, refetch]);

  const invalidateCache = useCallback(async () => {
    try {
      await PermissionAPI.invalidateCache();
      await queryClient.invalidateQueries({ queryKey: PERMISSION_QUERY_KEY });
      await refetch();
    } catch (err) {
      console.error("Failed to invalidate cache:", err);
      throw err;
    }
  }, [queryClient, refetch]);

  const switchOrganization = useCallback(
    async (organizationId: string) => {
      try {
        const response = await PermissionAPI.switchOrganization(organizationId);

        queryClient.setQueryData(PERMISSION_QUERY_KEY, response.permissions);
      } catch (err) {
        console.error("Failed to switch organization:", err);
        throw err;
      }
    },
    [queryClient],
  );

  const can = useCallback(
    (resource: Resource, action: ActionName): boolean => {
      if (!client) return false;
      return client.can(resource, action);
    },
    [client],
  );

  const canAny = useCallback(
    (resource: Resource, actions: ActionName[]): boolean => {
      if (!client) return false;
      return client.canAny(resource, actions);
    },
    [client],
  );

  const canAll = useCallback(
    (resource: Resource, actions: ActionName[]): boolean => {
      if (!client) return false;
      return client.canAll(resource, actions);
    },
    [client],
  );

  const canAccessField = useCallback(
    (resource: Resource, field: string): boolean => {
      if (!client) return false;
      return client.canAccessField(resource, field);
    },
    [client],
  );

  const canWriteField = useCallback(
    (resource: Resource, field: string): boolean => {
      if (!client) return false;
      return client.canWriteField(resource, field);
    },
    [client],
  );

  const getFieldAccess = useCallback(
    (resource: Resource, field: string): FieldAccess | null => {
      if (!client) return null;
      return client.getFieldAccess(resource, field);
    },
    [client],
  );

  const getCurrentOrganization = useCallback((): string | null => {
    if (!client) return null;
    return client.getCurrentOrganization();
  }, [client]);

  const getAvailableOrganizations = useCallback((): string[] => {
    if (!client) return [];
    return client.getAvailableOrganizations();
  }, [client]);

  useEffect(() => {
    if (!autoRefresh || !client || !manifest) return;

    const timeUntilExpiration = client.getTimeUntilExpiration();
    const refreshTime = Math.max(0, (timeUntilExpiration - 60) * 1000);

    if (refreshTime > 0) {
      const expirationTimer = setTimeout(() => {
        refresh();
      }, refreshTime);

      return () => clearTimeout(expirationTimer);
    }
  }, [autoRefresh, client, manifest, refresh]);

  const value = useMemo(
    () => ({
      client,
      manifest: manifest ?? null,
      isLoading,
      isExpired,
      error: error as Error | null,
      can,
      canAny,
      canAll,
      canAccessField,
      canWriteField,
      getFieldAccess,
      switchOrganization,
      getCurrentOrganization,
      getAvailableOrganizations,
      refresh,
      invalidateCache,
    }),
    [
      client,
      manifest,
      isLoading,
      isExpired,
      error,
      can,
      canAny,
      canAll,
      canAccessField,
      canWriteField,
      getFieldAccess,
      switchOrganization,
      getCurrentOrganization,
      getAvailableOrganizations,
      refresh,
      invalidateCache,
    ],
  );

  return (
    <PermissionContext.Provider value={value}>
      {children}
    </PermissionContext.Provider>
  );
}

export function usePermissionContext(): PermissionContextValue {
  const context = useContext(PermissionContext);
  if (!context) {
    throw new Error(
      "usePermissionContext must be used within a PermissionProvider",
    );
  }
  return context;
}
