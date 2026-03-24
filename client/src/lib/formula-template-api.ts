import type {
  CreateVersionRequest,
  ForkLineage,
  ForkRequest,
  FormulaTemplate,
  FormulaTemplateStatus,
  FormulaTemplateType,
  FormulaTemplateVersion,
  RollbackRequest,
  TemplateUsageResponse,
  TestExpressionRequest,
  TestExpressionResponse,
  VersionDiff,
} from "@/types/formula-template";
import type { GenericLimitOffsetResponse } from "@/types/server";
import { api } from "./api";

export type ListFormulaTemplatesParams = {
  limit?: number;
  offset?: number;
  query?: string;
  type?: FormulaTemplateType;
  status?: FormulaTemplateStatus;
};

export async function listFormulaTemplates(
  params?: ListFormulaTemplatesParams,
): Promise<GenericLimitOffsetResponse<FormulaTemplate>> {
  const searchParams = new URLSearchParams();

  if (params?.limit) searchParams.set("limit", String(params.limit));
  if (params?.offset) searchParams.set("offset", String(params.offset));
  if (params?.query) searchParams.set("query", params.query);
  if (params?.type) searchParams.set("type", params.type);
  if (params?.status) searchParams.set("status", params.status);

  const queryString = searchParams.toString();
  const endpoint = `/formula-templates/${queryString ? `?${queryString}` : ""}`;

  return api.get(endpoint);
}

export async function getFormulaTemplate(id: string): Promise<FormulaTemplate> {
  return api.get<FormulaTemplate>(`/formula-templates/${id}/`);
}

export async function createFormulaTemplate(
  data: Omit<
    FormulaTemplate,
    "id" | "organizationId" | "businessUnitId" | "createdAt" | "updatedAt"
  >,
): Promise<FormulaTemplate> {
  return api.post<FormulaTemplate>("/formula-templates/", data);
}

export async function updateFormulaTemplate(
  id: string,
  data: Partial<FormulaTemplate>,
): Promise<FormulaTemplate> {
  return api.put<FormulaTemplate>(`/formula-templates/${id}/`, data);
}

export async function patchFormulaTemplate(
  id: string,
  data: Partial<FormulaTemplate>,
): Promise<FormulaTemplate> {
  return api.patch<FormulaTemplate>(`/formula-templates/${id}/`, data);
}

export async function deleteFormulaTemplate(id: string): Promise<void> {
  return api.delete(`/formula-templates/${id}`);
}

export async function testExpression(data: TestExpressionRequest): Promise<TestExpressionResponse> {
  return api.post<TestExpressionResponse>("/formula-templates/test", data);
}

export type ListVersionsParams = {
  limit?: number;
  offset?: number;
};

export async function listVersions(
  templateId: string,
  params?: ListVersionsParams,
): Promise<GenericLimitOffsetResponse<FormulaTemplateVersion>> {
  const searchParams = new URLSearchParams();

  if (params?.limit) searchParams.set("limit", String(params.limit));
  if (params?.offset) searchParams.set("offset", String(params.offset));

  const queryString = searchParams.toString();
  const endpoint = `/formula-templates/${templateId}/versions${queryString ? `?${queryString}` : ""}`;

  return api.get(endpoint);
}

export async function getVersion(
  templateId: string,
  versionNumber: number,
): Promise<FormulaTemplateVersion> {
  return api.get<FormulaTemplateVersion>(
    `/formula-templates/${templateId}/versions/${versionNumber}`,
  );
}

export async function createVersion(
  templateId: string,
  data: CreateVersionRequest,
): Promise<FormulaTemplateVersion> {
  return api.post<FormulaTemplateVersion>(`/formula-templates/${templateId}/versions`, data);
}

export async function rollbackToVersion(
  templateId: string,
  data: RollbackRequest,
): Promise<FormulaTemplate> {
  return api.post<FormulaTemplate>(`/formula-templates/${templateId}/rollback`, data);
}

export async function forkTemplate(
  templateId: string,
  data: ForkRequest,
): Promise<FormulaTemplate> {
  return api.post<FormulaTemplate>(`/formula-templates/${templateId}/fork`, data);
}

export async function compareVersions(
  templateId: string,
  fromVersion: number,
  toVersion: number,
): Promise<VersionDiff> {
  return api.get<VersionDiff>(
    `/formula-templates/${templateId}/compare?from=${fromVersion}&to=${toVersion}`,
  );
}

export async function getLineage(templateId: string): Promise<ForkLineage> {
  return api.get<ForkLineage>(`/formula-templates/${templateId}/lineage`);
}

export async function getTemplateUsage(templateId: string): Promise<TemplateUsageResponse> {
  return api.get<TemplateUsageResponse>(`/formula-templates/${templateId}/usage`);
}
