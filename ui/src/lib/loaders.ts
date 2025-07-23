/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { api } from "@/services/api";
import { useAuthStore } from "@/stores/user-store";
import type { Resource } from "@/types/audit-entry";
import type { Action } from "@/types/roles-permissions";
import { LoaderFunctionArgs, redirect } from "react-router";

export async function checkAuthStatus() {
  try {
    const { data: sessionData } = await api.auth.validateSession();

    if (!sessionData.valid) {
      return null;
    }

    const { data: userData } = await api.auth.getCurrentUser();
    return userData;
  } catch {
    return null;
  }
}

export async function authLoader() {
  const { isInitialized } = useAuthStore.getState();

  if (!isInitialized) {
    const user = await checkAuthStatus();
    if (user) {
      useAuthStore.getState().setUser(user);
    }
    useAuthStore.getState().setInitialized(true);
  }

  const { user } = useAuthStore.getState();

  if (user) {
    return redirect("/");
  }

  return null;
}

export async function protectedLoader({ request }: LoaderFunctionArgs) {
  const { isInitialized } = useAuthStore.getState();

  if (!isInitialized) {
    const user = await checkAuthStatus();
    if (user) {
      useAuthStore.getState().setUser(user);
    }
    useAuthStore.getState().setInitialized(true);
  }

  const { user, isAuthenticated } = useAuthStore.getState();

  if (!user || !isAuthenticated) {
    const params = new URLSearchParams();
    params.set("from", new URL(request.url).pathname);
    return redirect("/auth?" + params.toString());
  }

  return null;
}

export function createPermissionLoader(
  resource: Resource,
  action: Action = "read" as Action,
) {
  return async (args: LoaderFunctionArgs) => {
    // First check authentication
    const authResult = await protectedLoader(args);
    if (authResult) {
      return authResult;
    }

    // Then check permissions
    const { hasPermission } = useAuthStore.getState();

    if (!hasPermission(resource, action)) {
      // User is authenticated but doesn't have permission
      const params = new URLSearchParams({
        resource,
        action,
      });
      return redirect(`/permission-denied?${params.toString()}`);
    }

    return null;
  };
}
