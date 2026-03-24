import { getPermissionManifest } from "@/lib/permission-api";
import { usePermissionStore } from "@/stores/permission-store";
import {
  Operation,
  type OperationType,
} from "@/types/permission";
import { useQuery } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";

const QUERY_KEY = "permissions";

export function usePermissionManifest() {
  const { manifest, lastFetched } = usePermissionStore();

  return useQuery({
    queryKey: [QUERY_KEY, "manifest"],
    queryFn: async () => {
      const newManifest = await getPermissionManifest();
      void usePermissionStore.getState().fetchManifest();
      return newManifest;
    },
    initialData: manifest ?? undefined,
    staleTime: 1000 * 60 * 5,
    enabled:
      !manifest ||
      !lastFetched ||
      new Date().getTime() - lastFetched > 1000 * 60 * 5,
  });
}

export function usePermission(
  resource: string,
  operation: OperationType,
) {
  const hasPermission = usePermissionStore((state) => state.hasPermission);
  const manifest = usePermissionStore((state) => state.manifest);

  return {
    allowed: hasPermission(resource, operation),
    isLoading: !manifest,
    isPlatformAdmin: manifest?.isPlatformAdmin ?? false,
    isOrgAdmin: manifest?.isOrgAdmin ?? false,
  };
}

export function usePermissions(resource: string) {
  const manifest = usePermissionStore((state) => state.manifest);
  const hasPermission = usePermissionStore((state) => state.hasPermission);

  return {
    canRead: hasPermission(resource, Operation.Read),
    canCreate: hasPermission(resource, Operation.Create),
    canUpdate: hasPermission(resource, Operation.Update),
    canExport: hasPermission(resource, Operation.Export),
    canImport: hasPermission(resource, Operation.Import),
    isLoading: !manifest,
    isPlatformAdmin: manifest?.isPlatformAdmin ?? false,
    isOrgAdmin: manifest?.isOrgAdmin ?? false,
  };
}

export function useRouteAccess(route: string) {
  const canAccessRoute = usePermissionStore((state) => state.canAccessRoute);
  const manifest = usePermissionStore((state) => state.manifest);

  return {
    allowed: canAccessRoute(route),
    isLoading: !manifest,
  };
}

export function usePermissionCheck() {
  const hasPermission = usePermissionStore((state) => state.hasPermission);
  const hasAnyPermission = usePermissionStore(
    (state) => state.hasAnyPermission,
  );
  const hasAllPermissions = usePermissionStore(
    (state) => state.hasAllPermissions,
  );
  const canAccessRoute = usePermissionStore((state) => state.canAccessRoute);
  const manifest = usePermissionStore((state) => state.manifest);

  const check = useCallback(
    (resource: string, operation: OperationType) => {
      return hasPermission(resource, operation);
    },
    [hasPermission],
  );

  const checkAny = useCallback(
    (resource: string, operations: OperationType[]) => {
      return hasAnyPermission(resource, operations);
    },
    [hasAnyPermission],
  );

  const checkAll = useCallback(
    (resource: string, operations: OperationType[]) => {
      return hasAllPermissions(resource, operations);
    },
    [hasAllPermissions],
  );

  const checkRoute = useCallback(
    (route: string) => {
      return canAccessRoute(route);
    },
    [canAccessRoute],
  );

  return {
    check,
    checkAny,
    checkAll,
    checkRoute,
    isReady: !!manifest,
    isPlatformAdmin: manifest?.isPlatformAdmin ?? false,
    isOrgAdmin: manifest?.isOrgAdmin ?? false,
  };
}

export function usePermissionSync() {
  const { fetchManifest, checkForUpdates, manifest } = usePermissionStore();

  useEffect(() => {
    if (!manifest) {
      fetchManifest().catch(console.error);
    }
  }, [manifest, fetchManifest]);

  useEffect(() => {
    const interval = setInterval(() => {
      checkForUpdates().catch(console.error);
    }, 60 * 1000);

    return () => clearInterval(interval);
  }, [checkForUpdates]);
}
