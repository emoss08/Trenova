import type { PermissionManifest, PermissionVersion } from "@/types/permission";
import { api } from "./api";

export async function getPermissionManifest(): Promise<PermissionManifest> {
  return api.get<PermissionManifest>("/me/permissions");
}

export async function getPermissionVersion(): Promise<PermissionVersion> {
  return api.get<PermissionVersion>("/me/permissions/version");
}

export interface BatchCheckRequest {
  checks: Array<{
    resource: string;
    operation: string;
  }>;
}

export interface BatchCheckResult {
  results: Record<
    string,
    {
      allowed: boolean;
      reason: string;
    }
  >;
}

export async function checkPermissionsBatch(
  request: BatchCheckRequest,
): Promise<BatchCheckResult> {
  return api.post<BatchCheckResult>("/me/permissions/check", request);
}
