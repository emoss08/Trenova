import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, type OperationType, Resource } from "@trenova/shared/types/permission";
import { useCallback, useEffect } from "react";

export function usePermission(resource: string, operation: OperationType) {
  const hasPermission = usePermissionStore((state) => state.hasPermission);
  const manifest = usePermissionStore((state) => state.manifest);

  return {
    allowed: hasPermission(resource, operation),
    isLoading: !manifest,
  };
}

export type ReviewApproval = {
  /** The person may approve this record now. */
  allowed: boolean;
  /**
   * The person may approve records like it, but submitted this one and cannot approve their own
   * changes, so a second reviewer has to: show the action disabled and say why.
   */
  needsSecondReviewer: boolean;
};

/**
 * Whether the person may approve a record under review. The two-person rule keeps a submitter
 * from approving their own submission unless they hold the resource's approve-own permission.
 * The server also refuses someone who edited the revision without submitting it.
 */
export function useReviewApproval(
  resource: string,
  submittedById: string | null | undefined,
): ReviewApproval {
  const { allowed: canApprove } = usePermission(resource, Operation.Approve);
  const { allowed: canApproveOwn } = usePermission(resource, Operation.ApproveOwn);
  const userId = useAuthStore((state) => state.user?.id ?? null);
  const own = Boolean(userId && submittedById && submittedById === userId);
  const needsSecondReviewer = canApprove && own && !canApproveOwn;

  return { allowed: canApprove && !needsSecondReviewer, needsSecondReviewer };
}

/**
 * Whether the page access panel is offered: to whoever may read roles, and on
 * every screen of a development build.
 */
export function usePageAccessOffered(): boolean {
  return usePermission(Resource.Role, Operation.Read).allowed || import.meta.env.DEV;
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
  const hasAnyPermission = usePermissionStore((state) => state.hasAnyPermission);
  const hasAllPermissions = usePermissionStore((state) => state.hasAllPermissions);
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
