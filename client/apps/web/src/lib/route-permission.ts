import { planRestrictionOfResource } from "@/lib/resource-plan";
import { translate } from "@trenova/shared/i18n/runtime";
import { PLAN_RESTRICTED_CODE, planRestrictedRefusal } from "@trenova/shared/lib/plan-limit";
import { getOrgCapabilities } from "@trenova/shared/hooks/use-org-capabilities";
import { isSessionExpiry } from "@trenova/shared/lib/error-presentation";
import type { RouteGate } from "@trenova/shared/lib/router/gates";
import { RouteRefusal, isRouteRefusal } from "@trenova/shared/lib/router/refusal";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import {
  hasOrganizationCapability,
  organizationCapabilityLabel,
  type OrganizationCapabilityType,
} from "@trenova/shared/types/organization-capability";
import { Operation } from "@trenova/shared/types/permission";
import { redirect } from "@tanstack/react-router";

function getOperationLabel(operation: number): string {
  return (
    Object.entries(Operation).find(([, value]) => value === operation)?.[0] ?? String(operation)
  );
}

/**
 * Loads the permission manifest when none is held. Resolves false only when the server
 * refused the session, which is the caller's cue to send the person to sign in. Any other
 * failure — a rate limit, a server error, a dropped connection — says nothing about the
 * session and is rethrown, so a route presents it with a retry instead of the login form.
 */
export async function ensurePermissionManifest(): Promise<boolean> {
  const { manifest, fetchManifest } = usePermissionStore.getState();
  if (manifest) {
    return true;
  }

  try {
    await fetchManifest();
    return true;
  } catch (error) {
    if (isSessionExpiry(error)) {
      return false;
    }
    throw error;
  }
}

/** Lets the person in when their roles grant `operation` on `resource`. */
export function requirePermission(resource: string, operation: number = Operation.Read): RouteGate {
  return async () => {
    if (!(await ensurePermissionManifest())) {
      throw redirect({ href: "/login" });
    }

    if (usePermissionStore.getState().hasPermission(resource, operation)) {
      return;
    }

    try {
      await usePermissionStore.getState().fetchManifest();
    } catch {
      // Keep the existing manifest and fall through to a clear authorization error.
    }

    if (!usePermissionStore.getState().hasPermission(resource, operation)) {
      // A resource the plan leaves out is missing from the manifest too: say which add-on
      // the page is part of instead of blaming the person's roles.
      const restriction = await planRestrictionOfResource(resource);
      if (restriction) {
        throw planRestrictedRefusal(restriction);
      }

      const operationLabel = getOperationLabel(operation);
      throw new RouteRefusal({
        status: 403,
        statusText: `Missing ${resource}:${operationLabel}`,
        data: `Missing permission: ${resource}:${operationLabel}`,
      });
    }
  };
}

/**
 * Keeps a route out of reach when its half of the product is switched off for
 * the organization. This is the route-level twin of the navigation capability
 * filter — decluttering, not authorization: the API behind the route stays open
 * and `requirePermission` remains the access check.
 */
export function requireCapability(capability: OrganizationCapabilityType): RouteGate {
  return () => {
    if (hasOrganizationCapability(getOrgCapabilities(), capability)) {
      return;
    }

    const label = organizationCapabilityLabel(capability);
    throw new RouteRefusal({
      status: 403,
      statusText: translate("{0} not enabled", label),
      data: translate("{0} is not enabled for this organization", label),
    });
  };
}

function isPlanRefusal(refusal: RouteRefusal): boolean {
  const body = refusal.data;
  return (
    typeof body === "object" &&
    body !== null &&
    (body as { code?: unknown }).code === PLAN_RESTRICTED_CODE
  );
}

/**
 * Lets a person in when any one of the gates does, for a page whose parts answer to
 * different permissions and which shows each person the parts they may read. A gate
 * that sends them elsewhere (to sign in) is obeyed. When every gate refuses, the page is
 * said to be off the plan only when the plan refused every gate (the first plan refusal
 * names the add-on); otherwise the last permission refusal is the answer, the rule the
 * server's RequireAnyPermission follows.
 */
export function anyOfGates(...list: readonly RouteGate[]): RouteGate {
  return async (context) => {
    let planRefusal: RouteRefusal | null = null;
    let permissionRefusal: RouteRefusal | null = null;
    for (const gate of list) {
      try {
        await gate(context);
        return;
      } catch (error) {
        if (!isRouteRefusal(error) || error.status !== 403) {
          throw error;
        }
        if (isPlanRefusal(error)) {
          planRefusal ??= error;
        } else {
          permissionRefusal = error;
        }
      }
    }
    throw permissionRefusal ?? planRefusal;
  };
}
