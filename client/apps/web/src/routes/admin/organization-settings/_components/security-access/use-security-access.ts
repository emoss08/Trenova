import { usePermissionCheck } from "@/hooks/use-permission";
import type { ActivityViewValue, SecurityTabValue } from "@/hooks/use-organization-setting-state";
import { Operation, Resource, type OperationType } from "@trenova/shared/types/permission";
import { useMemo } from "react";

export type ManageAccess = {
  read: boolean;
  create: boolean;
  update: boolean;
  remove: boolean;
};

export type SecurityAccess = {
  identityProviders: ManageAccess;
  scimDirectories: ManageAccess;
  accessPolicies: ManageAccess;
  provisioningAudit: boolean;
  activity: Record<ActivityViewValue, boolean>;
  tabs: Record<SecurityTabValue, boolean>;
};

type Check = (resource: string, operation: OperationType) => boolean;

function manageAccess(check: Check, resource: string): ManageAccess {
  return {
    read: check(resource, Operation.Read),
    create: check(resource, Operation.Create),
    update: check(resource, Operation.Update),
    remove: check(resource, Operation.Delete),
  };
}

export function securityAccess(check: Check): SecurityAccess {
  const identityProviders = manageAccess(check, Resource.IdentityProvider);
  const scimDirectories = manageAccess(check, Resource.SCIMDirectory);
  const accessPolicies = manageAccess(check, Resource.AccessPolicy);
  const provisioningAudit = check(Resource.ProvisioningAudit, Operation.Read);
  const activity: Record<ActivityViewValue, boolean> = {
    auth: check(Resource.AuthEvent, Operation.Read),
    risk: check(Resource.RiskDecision, Operation.Read),
    identities: check(Resource.ExternalIdentity, Operation.Read),
    mfa: check(Resource.MFAAuthenticator, Operation.Read),
  };

  return {
    identityProviders,
    scimDirectories,
    accessPolicies,
    provisioningAudit,
    activity,
    tabs: {
      "sign-in": identityProviders.read,
      provisioning: scimDirectories.read || provisioningAudit,
      policies: accessPolicies.read,
      activity: Object.values(activity).some(Boolean),
    },
  };
}

export function firstAllowed<T extends string>(
  preferred: T,
  order: readonly T[],
  allowed: Record<T, boolean>,
): T | null {
  if (allowed[preferred]) {
    return preferred;
  }
  return order.find((value) => allowed[value]) ?? null;
}

export function useSecurityAccess(): SecurityAccess {
  const { check } = usePermissionCheck();
  return useMemo(() => securityAccess(check), [check]);
}
