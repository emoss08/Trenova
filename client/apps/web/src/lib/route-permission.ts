import { planRestrictionOfResource } from "@/lib/resource-plan";
import { translate } from "@trenova/shared/i18n/runtime";
import { PLAN_RESTRICTED_CODE, planRestrictedResponse } from "@trenova/shared/lib/plan-limit";
import { getOrgCapabilities } from "@trenova/shared/hooks/use-org-capabilities";
import { isSessionExpiry } from "@trenova/shared/lib/error-presentation";
import { isPathScopedLoader, pathScopedLoader } from "@trenova/shared/lib/session-loader";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import {
  hasOrganizationCapability,
  organizationCapabilityLabel,
  type OrganizationCapabilityType,
} from "@trenova/shared/types/organization-capability";
import { Operation } from "@trenova/shared/types/permission";
import { redirect, type LoaderFunction } from "react-router";

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

export function createPermissionLoader(
  resource: string,
  operation: number = Operation.Read,
): LoaderFunction {
  return pathScopedLoader(async () => {
    if (!(await ensurePermissionManifest())) {
      return redirect("/login");
    }

    if (usePermissionStore.getState().hasPermission(resource, operation)) {
      return null;
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
        throw planRestrictedResponse(restriction);
      }

      const operationLabel = getOperationLabel(operation);
      throw new Response(`Missing permission: ${resource}:${operationLabel}`, {
        status: 403,
        statusText: `Missing ${resource}:${operationLabel}`,
      });
    }

    return null;
  });
}

/**
 * Keeps a route out of reach when its half of the product is switched off for
 * the organization. This is the route-level twin of the navigation capability
 * filter — decluttering, not authorization: the API behind the route stays open
 * and `createPermissionLoader` remains the access check.
 */
export function createCapabilityLoader(capability: OrganizationCapabilityType): LoaderFunction {
  return async () => {
    if (hasOrganizationCapability(getOrgCapabilities(), capability)) {
      return null;
    }

    const label = organizationCapabilityLabel(capability);
    throw new Response(translate("{0} is not enabled for this organization", label), {
      status: 403,
      statusText: translate("{0} not enabled", label),
    });
  };
}

/**
 * Runs loaders in order and returns the first answer that is not null. The combination is
 * path-scoped when any part is: a route behind the session or permission gate revalidates
 * only when its path changes, and a prefetch beside the gate is only a cache warm-up the
 * mounted page's own queries already cover.
 */
export function combineLoaders(...loaders: LoaderFunction[]): LoaderFunction {
  const combined: LoaderFunction = async (args) => {
    for (const loader of loaders) {
      const result = await loader(args);
      if (result !== null) {
        return result;
      }
    }
    return null;
  };
  return loaders.some(isPathScopedLoader) ? pathScopedLoader(combined) : combined;
}

async function isPlanRefusal(refusal: Response): Promise<boolean> {
  if (!refusal.headers.get("Content-Type")?.includes("json")) {
    return false;
  }
  try {
    const body: unknown = await refusal.clone().json();
    return (
      typeof body === "object" &&
      body !== null &&
      (body as { code?: unknown }).code === PLAN_RESTRICTED_CODE
    );
  } catch {
    return false;
  }
}

/**
 * Lets a person in when any one of the gates does, for a page whose parts answer to
 * different permissions and which shows each person the parts they may read. A gate
 * that sends them elsewhere (to sign in) is obeyed. When every gate refuses, the page is
 * said to be off the plan only when the plan refused every gate (the first plan refusal
 * names the add-on); otherwise the last permission refusal is the answer, the rule the
 * server's RequireAnyPermission follows.
 */
export function anyOfLoaders(...loaders: LoaderFunction[]): LoaderFunction {
  const any: LoaderFunction = async (args) => {
    let planRefusal: Response | null = null;
    let permissionRefusal: Response | null = null;
    for (const loader of loaders) {
      try {
        return await loader(args);
      } catch (error) {
        if (!(error instanceof Response) || error.status !== 403) {
          throw error;
        }
        if (await isPlanRefusal(error)) {
          planRefusal ??= error;
        } else {
          permissionRefusal = error;
        }
      }
    }
    throw permissionRefusal ?? planRefusal;
  };
  return loaders.some(isPathScopedLoader) ? pathScopedLoader(any) : any;
}
