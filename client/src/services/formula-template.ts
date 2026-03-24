import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  forkLineageSchema,
  formulaTemplateSchema,
  formulaTemplateVersionSchema,
  listFormulaTemplateResponseSchema,
  templateUsageResponseSchema,
  versionDiffSchema,
  type BulkDuplicateFormulaTemplateRequest,
  type BulkUpdateStatusRequest,
  type CreateVersionRequest,
  type ForkLineage,
  type ForkRequest,
  type FormulaTemplate,
  type FormulaTemplateVersion,
  type ListFormulaTemplateResponse,
  type RollbackRequest,
  type TemplateUsageResponse,
  type VersionDiff,
} from "@/types/formula-template";
import type { GenericLimitOffsetResponse } from "@/types/server";
import { z } from "zod";

export class FormulaTemplateService {
  public async bulkUpdateStatus(request: BulkUpdateStatusRequest) {
    const response = await api.post<ListFormulaTemplateResponse>(
      "/formula-templates/bulk-update-status",
      request,
    );

    return safeParse(listFormulaTemplateResponseSchema, response, "Formula Template");
  }

  public async bulkDuplicate(request: BulkDuplicateFormulaTemplateRequest) {
    const response = await api.post<ListFormulaTemplateResponse>(
      "/formula-templates/duplicate",
      request,
    );

    return safeParse(listFormulaTemplateResponseSchema, response, "Formula Template");
  }

  public async listVersions(
    templateId: FormulaTemplate["id"],
    params?: { limit?: number; offset?: number },
  ): Promise<GenericLimitOffsetResponse<FormulaTemplateVersion>> {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.set("limit", String(params.limit));
    if (params?.offset) searchParams.set("offset", String(params.offset));

    const queryString = searchParams.toString();
    const response = await api.get<
      GenericLimitOffsetResponse<FormulaTemplateVersion>
    >(
      `/formula-templates/${templateId}/versions${queryString ? `?${queryString}` : ""}`,
    );

    return {
      ...response,
      results: await safeParse(z.array(formulaTemplateVersionSchema), response.results, "Formula Template Version"),
    };
  }

  public async getVersion(
    templateId: FormulaTemplate["id"],
    versionNumber: number,
  ): Promise<FormulaTemplateVersion> {
    const response = await api.get<FormulaTemplateVersion>(
      `/formula-templates/${templateId}/versions/${versionNumber}`,
    );

    return safeParse(formulaTemplateVersionSchema, response, "Formula Template Version");
  }

  public async createVersion(
    templateId: FormulaTemplate["id"],
    data: CreateVersionRequest,
  ): Promise<FormulaTemplateVersion> {
    const response = await api.post<FormulaTemplateVersion>(
      `/formula-templates/${templateId}/versions`,
      data,
    );

    return safeParse(formulaTemplateVersionSchema, response, "Formula Template Version");
  }

  public async rollback(
    templateId: FormulaTemplate["id"],
    data: RollbackRequest,
  ): Promise<FormulaTemplate> {
    const response = await api.post<FormulaTemplate>(
      `/formula-templates/${templateId}/rollback`,
      data,
    );

    return safeParse(formulaTemplateSchema, response, "Formula Template");
  }

  public async fork(
    templateId: FormulaTemplate["id"],
    data: ForkRequest,
  ): Promise<FormulaTemplate> {
    const response = await api.post<FormulaTemplate>(
      `/formula-templates/${templateId}/fork`,
      data,
    );

    return safeParse(formulaTemplateSchema, response, "Formula Template");
  }

  public async compareVersions(
    templateId: FormulaTemplate["id"],
    fromVersion: number,
    toVersion: number,
  ): Promise<VersionDiff> {
    const response = await api.get<VersionDiff>(
      `/formula-templates/${templateId}/compare?from=${fromVersion}&to=${toVersion}`,
    );

    return safeParse(versionDiffSchema, response, "Version Diff");
  }

  public async getLineage(
    templateId: FormulaTemplate["id"],
  ): Promise<ForkLineage> {
    const response = await api.get<ForkLineage>(
      `/formula-templates/${templateId}/lineage`,
    );

    return safeParse(forkLineageSchema, response, "Fork Lineage");
  }

  public async getUsage(
    templateId: FormulaTemplate["id"],
  ): Promise<TemplateUsageResponse> {
    const response = await api.get<TemplateUsageResponse>(
      `/formula-templates/${templateId}/usage`,
    );

    return safeParse(templateUsageResponseSchema, response, "Template Usage");
  }

  public async updateVersionTags(
    templateId: FormulaTemplate["id"],
    versionNumber: number,
    tags: string[],
  ): Promise<FormulaTemplateVersion> {
    const response = await api.patch<FormulaTemplateVersion>(
      `/formula-templates/${templateId}/versions/${versionNumber}/tags`,
      { tags },
    );

    return safeParse(formulaTemplateVersionSchema, response, "Formula Template Version");
  }
}
