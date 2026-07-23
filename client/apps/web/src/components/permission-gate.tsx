import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { OperationType } from "@trenova/shared/types/permission";
import type { ReactNode } from "react";

interface PermissionGateProps {
  resource: string;
  operation: OperationType;
  children: ReactNode;
  fallback?: ReactNode;
}

export function PermissionGate({
  resource,
  operation,
  children,
  fallback = null,
}: PermissionGateProps) {
  const hasPermission = usePermissionStore((state) => state.hasPermission);
  const manifest = usePermissionStore((state) => state.manifest);

  if (!manifest) {
    return null;
  }

  if (hasPermission(resource, operation)) {
    return <>{children}</>;
  }

  return <>{fallback}</>;
}

interface AnyPermissionGateProps {
  resource: string;
  operations: OperationType[];
  children: ReactNode;
  fallback?: ReactNode;
}

export function AnyPermissionGate({
  resource,
  operations,
  children,
  fallback = null,
}: AnyPermissionGateProps) {
  const hasAnyPermission = usePermissionStore(
    (state) => state.hasAnyPermission,
  );
  const manifest = usePermissionStore((state) => state.manifest);

  if (!manifest) {
    return null;
  }

  if (hasAnyPermission(resource, operations)) {
    return <>{children}</>;
  }

  return <>{fallback}</>;
}

interface AllPermissionsGateProps {
  resource: string;
  operations: OperationType[];
  children: ReactNode;
  fallback?: ReactNode;
}

export function AllPermissionsGate({
  resource,
  operations,
  children,
  fallback = null,
}: AllPermissionsGateProps) {
  const hasAllPermissions = usePermissionStore(
    (state) => state.hasAllPermissions,
  );
  const manifest = usePermissionStore((state) => state.manifest);

  if (!manifest) {
    return null;
  }

  if (hasAllPermissions(resource, operations)) {
    return <>{children}</>;
  }

  return <>{fallback}</>;
}
