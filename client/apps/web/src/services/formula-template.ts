import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  backtestResponseSchema,
  listFormulaTemplateReviewsResponseSchema,
  listStandardsResponseSchema,
  type FormulaTemplateReview,
  readinessResponseSchema,
  reviewDiffResponseSchema,
  type StandardTemplate,
  type ReviewDiffResponse,
  type ReadinessResponse,
  explainFormulaResponseSchema,
  forkLineageSchema,
  formulaSchemaResponseSchema,
  formulaTemplateSchema,
  formulaTestCaseSchema,
  formulaTemplateVersionSchema,
  generateFormulaResponseSchema,
  importTemplatesResponseSchema,
  installStandardsResponseSchema,
  listFormulaTemplateResponseSchema,
  runTestCasesResponseSchema,
  templateUsageResponseSchema,
  testExpressionResponseSchema,
  versionDiffSchema,
  type BacktestRequest,
  type BacktestResponse,
  type BulkDuplicateFormulaTemplateRequest,
  type BulkUpdateStatusRequest,
  type CreateVersionRequest,
  type ExplainFormulaResponse,
  type ForkLineage,
  type ForkRequest,
  type FormulaSchemaResponse,
  type FormulaTemplate,
  type FormulaTemplateVersion,
  type FormulaTestCase,
  type FormulaTestCaseInput,
  type RunTestCasesResponse,
  type TestCaseCandidate,
  type GenerateFormulaRequest,
  type GenerateFormulaResponse,
  type ImportTemplatesResponse,
  type InstallStandardsResponse,
  type ListFormulaTemplateResponse,
  type RollbackRequest,
  type TemplateUsageResponse,
  type TestExpressionRequest,
  type TestExpressionResponse,
  type VersionDiff,
} from "@trenova/shared/types/formula-template";
import type { GenericLimitOffsetResponse } from "@trenova/shared/types/server";
import { z } from "zod";

export type ImportTestCasePayload = {
  name: string;
  description?: string;
  variables?: Record<string, unknown>;
  expectedAmount: number;
  tolerance?: number;
};

export type ImportTemplatePayload = {
  name: string;
  description?: string;
  type: FormulaTemplate["type"];
  expression: string;
  schemaId?: string;
  variableDefinitions?: FormulaTemplate["variableDefinitions"];
  breakdownDefinitions?: FormulaTemplate["breakdownDefinitions"];
  minCharge?: FormulaTemplate["minCharge"];
  maxCharge?: FormulaTemplate["maxCharge"];
  roundingMode?: FormulaTemplate["roundingMode"];
  roundingPrecision?: FormulaTemplate["roundingPrecision"];
  metadata?: FormulaTemplate["metadata"];
  sourceTemplateId?: string | null;
  sourceVersionNumber?: number | null;
  testCases?: ImportTestCasePayload[];
};

export type ImportTemplatesRequest = {
  exportVersion: string;
  templates: ImportTemplatePayload[];
  onConflict?: "reject" | "rename";
};

export class FormulaTemplateService {
  public async get(templateId: FormulaTemplate["id"]): Promise<FormulaTemplate> {
    const response = await api.get<FormulaTemplate>(`/formula-templates/${templateId}/`);

    return safeParse(formulaTemplateSchema, response, "Formula Template");
  }

  public async test(request: TestExpressionRequest): Promise<TestExpressionResponse> {
    const response = await api.post<TestExpressionResponse>("/formula-templates/test", request);

    return safeParse(testExpressionResponseSchema, response, "Test Expression");
  }

  public async getSchema(schemaId = "shipment"): Promise<FormulaSchemaResponse> {
    const response = await api.get<FormulaSchemaResponse>(
      `/formula-templates/schema?schemaId=${encodeURIComponent(schemaId)}`,
    );

    return safeParse(formulaSchemaResponseSchema, response, "Formula Schema");
  }

  public async importTemplates(request: ImportTemplatesRequest): Promise<ImportTemplatesResponse> {
    const response = await api.post<ImportTemplatesResponse>("/formula-templates/import", request);

    return safeParse(importTemplatesResponseSchema, response, "Formula Template Import");
  }

  public async listTestCases(templateId: FormulaTemplate["id"]): Promise<FormulaTestCase[]> {
    const response = await api.get<FormulaTestCase[]>(
      `/formula-templates/${templateId}/test-cases`,
    );

    return safeParse(z.array(formulaTestCaseSchema), response, "Formula Test Case");
  }

  public async createTestCase(
    templateId: FormulaTemplate["id"],
    input: FormulaTestCaseInput,
  ): Promise<FormulaTestCase> {
    const response = await api.post<FormulaTestCase>(
      `/formula-templates/${templateId}/test-cases`,
      input,
    );

    return safeParse(formulaTestCaseSchema, response, "Formula Test Case");
  }

  public async updateTestCase(
    templateId: FormulaTemplate["id"],
    testCaseId: string,
    input: FormulaTestCaseInput & { version?: number },
  ): Promise<FormulaTestCase> {
    const response = await api.put<FormulaTestCase>(
      `/formula-templates/${templateId}/test-cases/${testCaseId}`,
      input,
    );

    return safeParse(formulaTestCaseSchema, response, "Formula Test Case");
  }

  public async deleteTestCase(
    templateId: FormulaTemplate["id"],
    testCaseId: string,
  ): Promise<void> {
    await api.delete(`/formula-templates/${templateId}/test-cases/${testCaseId}`);
  }

  public async runTestCases(
    templateId: FormulaTemplate["id"],
    candidate?: TestCaseCandidate,
  ): Promise<RunTestCasesResponse> {
    const response = await api.post<RunTestCasesResponse>(
      `/formula-templates/${templateId}/test-cases/run`,
      { candidate: candidate ?? null },
    );

    return safeParse(runTestCasesResponseSchema, response, "Test Case Run");
  }

  public async installStandards(): Promise<InstallStandardsResponse> {
    const response = await api.post<InstallStandardsResponse>(
      "/formula-templates/install-standards",
      {},
    );

    return safeParse(installStandardsResponseSchema, response, "Standard Templates");
  }

  public async generateFormula(request: GenerateFormulaRequest): Promise<GenerateFormulaResponse> {
    const response = await api.post<GenerateFormulaResponse>(
      "/formula-templates/ai/generate",
      request,
    );

    return safeParse(generateFormulaResponseSchema, response, "Formula Generation");
  }

  public async explainFormula(request: {
    expression: string;
    schemaId?: string;
  }): Promise<ExplainFormulaResponse> {
    const response = await api.post<ExplainFormulaResponse>(
      "/formula-templates/ai/explain",
      request,
    );

    return safeParse(explainFormulaResponseSchema, response, "Formula Explanation");
  }

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
    const response = await api.get<GenericLimitOffsetResponse<FormulaTemplateVersion>>(
      `/formula-templates/${templateId}/versions${queryString ? `?${queryString}` : ""}`,
    );

    return {
      ...response,
      results: await safeParse(
        z.array(formulaTemplateVersionSchema),
        response.results,
        "Formula Template Version",
      ),
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
    const response = await api.post<FormulaTemplate>(`/formula-templates/${templateId}/fork`, data);

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

  public async getLineage(templateId: FormulaTemplate["id"]): Promise<ForkLineage> {
    const response = await api.get<ForkLineage>(`/formula-templates/${templateId}/lineage`);

    return safeParse(forkLineageSchema, response, "Fork Lineage");
  }

  public async getUsage(templateId: FormulaTemplate["id"]): Promise<TemplateUsageResponse> {
    const response = await api.get<TemplateUsageResponse>(`/formula-templates/${templateId}/usage`);

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

  public async submit(
    templateId: FormulaTemplate["id"],
    comment: string,
  ): Promise<FormulaTemplate> {
    const response = await api.post<FormulaTemplate>(`/formula-templates/${templateId}/submit`, {
      comment,
    });

    return safeParse(formulaTemplateSchema, response, "Formula Template");
  }

  public async approve(
    templateId: FormulaTemplate["id"],
    comment: string,
  ): Promise<FormulaTemplate> {
    const response = await api.post<FormulaTemplate>(`/formula-templates/${templateId}/approve`, {
      comment,
    });

    return safeParse(formulaTemplateSchema, response, "Formula Template");
  }

  public async reject(
    templateId: FormulaTemplate["id"],
    comment: string,
  ): Promise<FormulaTemplate> {
    const response = await api.post<FormulaTemplate>(`/formula-templates/${templateId}/reject`, {
      comment,
    });

    return safeParse(formulaTemplateSchema, response, "Formula Template");
  }

  public async requestChanges(
    templateId: FormulaTemplate["id"],
    comment: string,
  ): Promise<FormulaTemplate> {
    const response = await api.post<FormulaTemplate>(
      `/formula-templates/${templateId}/request-changes`,
      { comment },
    );

    return safeParse(formulaTemplateSchema, response, "Formula Template");
  }

  public async listReviews(templateId: FormulaTemplate["id"]): Promise<FormulaTemplateReview[]> {
    const response = await api.get<FormulaTemplateReview[]>(
      `/formula-templates/${templateId}/reviews`,
    );

    return safeParse(
      listFormulaTemplateReviewsResponseSchema,
      response,
      "Formula Template Reviews",
    );
  }

  public async updateVersionEffectiveDate(
    templateId: FormulaTemplate["id"],
    versionNumber: number,
    effectiveFrom: number | null,
  ): Promise<FormulaTemplateVersion> {
    const response = await api.patch<FormulaTemplateVersion>(
      `/formula-templates/${templateId}/versions/${versionNumber}/effective-date`,
      { effectiveFrom },
    );

    return safeParse(formulaTemplateVersionSchema, response, "Formula Template Version");
  }

  public async listScheduledVersions(
    templateId: FormulaTemplate["id"],
  ): Promise<FormulaTemplateVersion[]> {
    const response = await api.get<FormulaTemplateVersion[]>(
      `/formula-templates/${templateId}/versions/scheduled`,
    );

    return safeParse(z.array(formulaTemplateVersionSchema), response, "Formula Template Version");
  }

  public async backtest(
    templateId: FormulaTemplate["id"],
    request: BacktestRequest,
  ): Promise<BacktestResponse> {
    const response = await api.post<BacktestResponse>(
      `/formula-templates/${templateId}/backtest`,
      request,
    );

    return safeParse(backtestResponseSchema, response, "Backtest");
  }

  public async approvalImpact(
    templateId: FormulaTemplate["id"],
    limit?: number,
  ): Promise<BacktestResponse> {
    const response = await api.post<BacktestResponse>(`/formula-templates/${templateId}/impact`, {
      limit: limit ?? 0,
    });

    return safeParse(backtestResponseSchema, response, "Approval Impact");
  }

  public async readiness(templateId: FormulaTemplate["id"]): Promise<ReadinessResponse> {
    const response = await api.get<ReadinessResponse>(`/formula-templates/${templateId}/readiness`);

    return safeParse(readinessResponseSchema, response, "Formula Template Readiness");
  }

  public async listStandards(): Promise<StandardTemplate[]> {
    const response = await api.get<StandardTemplate[]>("/formula-templates/standards");

    return safeParse(listStandardsResponseSchema, response, "Formula Template Standards");
  }

  public async reviewDiff(templateId: FormulaTemplate["id"]): Promise<ReviewDiffResponse> {
    const response = await api.get<ReviewDiffResponse>(
      `/formula-templates/${templateId}/review-diff`,
    );

    return safeParse(reviewDiffResponseSchema, response, "Formula Template Review Diff");
  }
}
