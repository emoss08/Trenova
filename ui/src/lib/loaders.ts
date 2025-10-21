/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { PermissionAPI } from "@/lib/permissions/permission-api";
import { PermissionClient } from "@/lib/permissions/permission-client";
import { api } from "@/services/api";
import { useAuthStore } from "@/stores/user-store";
import type { PermissionManifest } from "@/types/permission";
import { LoaderFunctionArgs, redirect } from "react-router";

// Cache for permission manifest to avoid fetching on every route change
let permissionManifestCache: {
  manifest: PermissionManifest;
  timestamp: number;
} | null = null;

const MANIFEST_CACHE_TTL = 5 * 60 * 1000; // 5 minutes

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

/**
 * Get or fetch the permission manifest with caching
 */
async function getPermissionManifest() {
  const now = Date.now();

  // Check if cache is valid
  if (
    permissionManifestCache &&
    now - permissionManifestCache.timestamp < MANIFEST_CACHE_TTL
  ) {
    return permissionManifestCache.manifest;
  }

  // Fetch new manifest
  try {
    const manifest = await PermissionAPI.getManifest();
    permissionManifestCache = {
      manifest,
      timestamp: now,
    };
    return manifest;
  } catch (error) {
    console.error("Failed to fetch permission manifest:", error);
    // Fall back to old permission system if new system fails
    return null;
  }
}

/**
 * Create a loader that checks permissions using the v2 permission system
 *
 * @param resource - The resource to check (e.g., 'shipment', 'user')
 * @param action - The action to check (e.g., 'read', 'create', 'update')
 */
export function createPermissionLoader(
  resource: string,
  action: string = "read",
) {
  return async (args: LoaderFunctionArgs) => {
    // First check authentication
    const authResult = await protectedLoader(args);
    if (authResult) {
      return authResult;
    }

    // Check permissions using v2 system
    const manifest = await getPermissionManifest();

    if (!manifest) {
      console.error("Failed to load permission manifest");
      return redirect("/permission-denied");
    }

    const client = new PermissionClient(manifest);

    // Check permission
    if (!client.can(resource, action)) {
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

/**
 * Clear the permission manifest cache (useful after org switch or logout)
 */
export function clearPermissionCache() {
  permissionManifestCache = null;
}
