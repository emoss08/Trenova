/* eslint-disable @typescript-eslint/no-extraneous-class */
import { http, type HttpClientResponse } from "@/lib/http-client";
import type {
  BatchPermissionCheck,
  BatchPermissionResult,
  FieldAccessResponse,
  PermissionManifest,
  Resource,
  SwitchOrganizationRequest,
  SwitchOrganizationResponse,
} from "@/types/permission";

export class PermissionAPI {
  static async getManifest(): Promise<PermissionManifest> {
    const response: HttpClientResponse<PermissionManifest> = await http.get(
      "/permissions/manifest/",
    );
    return response.data;
  }

  static async verify(
    resource: Resource,
    action: string,
  ): Promise<{ allowed: boolean }> {
    const response: HttpClientResponse<{ allowed: boolean }> = await http.post(
      "/permissions/verify/",
      { resource, action },
    );
    return response.data;
  }

  static async checkBatch(
    checks: BatchPermissionCheck[],
  ): Promise<BatchPermissionResult[]> {
    if (checks.length > 100) {
      throw new Error("Batch size cannot exceed 100 checks");
    }

    const response: HttpClientResponse<{ results: BatchPermissionResult[] }> =
      await http.post("/permissions/check-batch/", { checks });
    return response.data.results;
  }

  static async switchOrganization(
    organizationId: string,
  ): Promise<SwitchOrganizationResponse> {
    const request: SwitchOrganizationRequest = { organizationId };
    const response: HttpClientResponse<SwitchOrganizationResponse> =
      await http.post("/auth/switch-org/", request);
    return response.data;
  }

  static async refresh(): Promise<PermissionManifest> {
    const response: HttpClientResponse<PermissionManifest> = await http.post(
      "/permissions/refresh/",
    );
    return response.data;
  }

  static async invalidateCache(): Promise<{ success: boolean }> {
    const response: HttpClientResponse<{ success: boolean }> = await http.post(
      "/permissions/invalidate-cache/",
    );
    return response.data;
  }

  static async getFieldAccess(
    resourceType: Resource,
  ): Promise<FieldAccessResponse> {
    const response: HttpClientResponse<FieldAccessResponse> = await http.get(
      `/permissions/field-access/${resourceType}/`,
    );
    return response.data;
  }
}
