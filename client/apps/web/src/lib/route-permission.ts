import { getOrgCapabilities } from "@trenova/shared/hooks/use-org-capabilities";
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

async function ensurePermissionManifest() {
  const { manifest, fetchManifest } = usePermissionStore.getState();
  if (manifest) {
    return true;
  }

  try {
    await fetchManifest();
    return true;
  } catch {
    return false;
  }
}

export function createPermissionLoader(
  resource: string,
  operation: number = Operation.Read,
): LoaderFunction {
  return async () => {
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
      const operationLabel = getOperationLabel(operation);
      throw new Response(`Missing permission: ${resource}:${operationLabel}`, {
        status: 403,
        statusText: `Missing ${resource}:${operationLabel}`,
      });
    }

    return null;
  };
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
    throw new Response(`${label} is not enabled for this organization`, {
      status: 403,
      statusText: `${label} not enabled`,
    });
  };
}

export function combineLoaders(...loaders: LoaderFunction[]): LoaderFunction {
  return async (args) => {
    for (const loader of loaders) {
      const result = await loader(args);
      if (result !== null) {
        return result;
      }
    }
    return null;
  };
}
