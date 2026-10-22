import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  backtestResponseSchema,
  forkLineageSchema,
  formulaExplainResponseSchema,
  formulaPreviewResponseSchema,
  formulaTemplateSchema,
  formulaTemplateVersionSchema,
  importTemplatesResponseSchema,
  installStandardsResponseSchema,
  listFormulaReviewersResponseSchema,
  listFormulaTemplateResponseSchema,
  listFormulaTemplateReviewsResponseSchema,
  listStandardsResponseSchema,
  readinessResponseSchema,
  reviewDiffResponseSchema,
  reviewNoteResponseSchema,
  scaffoldTestResponseSchema,
  suggestedTestsResponseSchema,
  templateUsageResponseSchema,
  versionDiffSchema,
  type BacktestRequest,
  type BacktestResponse,
  type BulkDuplicateFormulaTemplateRequest,
  type BulkUpdateStatusRequest,
  type CreateVersionRequest,
  type ForkLineage,
  type ForkRequest,
  type FormulaDraftContent,
  type FormulaExplainResponse,
  type FormulaPreviewRequest,
  type FormulaPreviewResponse,
  type FormulaReviewer,
  type FormulaTemplate,
  type FormulaTemplateReview,
  type FormulaTemplateVersion,
  type FormulaTemplateWrite,
  type ImportTemplatesResponse,
  type InstallStandardsResponse,
  type ListFormulaTemplateResponse,
  type ReadinessResponse,
  type ReviewDiffResponse,
  type ReviewNoteResponse,
  type RollbackRequest,
  type ScaffoldTestRequest,
  type ScaffoldTestResponse,
  type StandardTemplate,
  type SuggestedTestsResponse,
  type TemplateUsageResponse,
  type VersionDiff,
} from "@trenova/shared/types/formula-template";
import {
  tariffContextSchema,
  tariffInputDeclarationsSchema,
  type TariffContext,
  type TariffInputDeclaration,
} from "@trenova/shared/types/tariff";
import type { GenericLimitOffsetResponse } from "@trenova/shared/types/server";
import { pageThreadSchema, type PageThread } from "@/types/assistant";
import { z } from "zod";

const declaredInputsSchema = z.object({
  inputs: tariffInputDeclarationsSchema,
  billing: z.object({ versionNumber: z.number().int() }).nullish(),
});

/** A template as an export file carries it: its content and where it was forked from. */
export type ImportTemplatePayload = {
  name: string;
  description?: string;
  source: string;
  sourceTemplateId?: string | null;
  sourceVersionNumber?: number | null;
};

/** Submitting for review: who should look, and the note they read first. */
export type SubmitForReviewRequest = {
  comment?: string;
  reviewerId?: string;
  reviewerNote?: string;
};

/** The load Nova's explanation is priced on: the preview's sample inputs or a saved shipment. */
export type ExplainPreviewLoad = {
  inputs?: Record<string, unknown>;
  shipmentId?: string;
};

export type ImportTemplatesRequest = {
  exportVersion: string;
  templates: ImportTemplatePayload[];
  onConflict?: "reject" | "rename";
};

type RequestOptions = { signal?: AbortSignal };

const BASE = "/formula-templates";

export class FormulaTemplateService {
  public async list(
    params: { limit: number; offset?: number },
    options?: RequestOptions,
  ): Promise<GenericLimitOffsetResponse<FormulaTemplate>> {
    const search = new URLSearchParams({
      limit: String(params.limit),
      offset: String(params.offset ?? 0),
    });
    const response = await api.get<GenericLimitOffsetResponse<FormulaTemplate>>(
      `${BASE}/?${search.toString()}`,
      { signal: options?.signal },
    );

    return {
      ...response,
      results: await safeParse(
        listFormulaTemplateResponseSchema,
        response.results ?? [],
        "Formula Templates",
      ),
    };
  }

  public async get(templateId: string, options?: RequestOptions): Promise<FormulaTemplate> {
    const response = await api.get<unknown>(`${BASE}/${templateId}/`, { signal: options?.signal });
    return safeParse(formulaTemplateSchema, response, "Formula Template");
  }

  public async create(request: FormulaTemplateWrite): Promise<FormulaTemplate> {
    const response = await api.post<unknown>(`${BASE}/`, request);
    return safeParse(formulaTemplateSchema, response, "Formula Template");
  }

  public async update(templateId: string, request: FormulaTemplateWrite): Promise<FormulaTemplate> {
    const response = await api.put<unknown>(`${BASE}/${templateId}/`, request);
    return safeParse(formulaTemplateSchema, response, "Formula Template");
  }

  /**
   * What the browser checks a tariff against: the organisation's settings, its rate tables'
   * shapes, its accessorial charge codes and the sources of the libraries a tariff may import.
   */
  public async getContext(options?: RequestOptions): Promise<TariffContext> {
    const response = await api.get<unknown>(`${BASE}/context`, { signal: options?.signal });
    return safeParse(tariffContextSchema, response, "Tariff Context");
  }

  /** Rates the source against the sample record, or a real shipment, with the given inputs. */
  public async preview(
    request: FormulaPreviewRequest,
    options?: RequestOptions,
  ): Promise<FormulaPreviewResponse> {
    const response = await api.post<unknown>(`${BASE}/test`, request, {
      signal: options?.signal,
    });
    return safeParse(formulaPreviewResponseSchema, response, "Formula Preview");
  }

  /** Writes a `test` block pinning what the tariff bills for a real shipment. */
  public async scaffoldTest(request: ScaffoldTestRequest): Promise<ScaffoldTestResponse> {
    const response = await api.post<unknown>(`${BASE}/scaffold-test`, request);
    return safeParse(scaffoldTestResponseSchema, response, "Scaffolded Test");
  }

  public async importTemplates(request: ImportTemplatesRequest): Promise<ImportTemplatesResponse> {
    const response = await api.post<unknown>(`${BASE}/import`, request);
    return safeParse(importTemplatesResponseSchema, response, "Formula Template Import");
  }

  public async installStandards(): Promise<InstallStandardsResponse> {
    const response = await api.post<unknown>(`${BASE}/install-standards`, {});
    return safeParse(installStandardsResponseSchema, response, "Standard Templates");
  }

  /**
   * Opens the person's conversation with the formula assistant about a saved
   * template, or a fresh one while the template is unsaved.
   */
  public async openAssistantThread(templateId: string | null): Promise<PageThread> {
    const response = await api.post<unknown>(`${BASE}/ai/thread/`, {
      templateId: templateId ?? "",
    });
    return safeParse(pageThreadSchema, response, "Formula Assistant Conversation");
  }

  public async bulkUpdateStatus(request: BulkUpdateStatusRequest) {
    const response = await api.post<ListFormulaTemplateResponse>(
      `${BASE}/bulk-update-status`,
      request,
    );
    return safeParse(listFormulaTemplateResponseSchema, response, "Formula Template");
  }

  public async bulkDuplicate(request: BulkDuplicateFormulaTemplateRequest) {
    const response = await api.post<ListFormulaTemplateResponse>(`${BASE}/duplicate`, request);
    return safeParse(listFormulaTemplateResponseSchema, response, "Formula Template");
  }

  public async listVersions(
    templateId: string,
    params?: { limit?: number; offset?: number },
    options?: RequestOptions,
  ): Promise<GenericLimitOffsetResponse<FormulaTemplateVersion>> {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.set("limit", String(params.limit));
    if (params?.offset) searchParams.set("offset", String(params.offset));

    const queryString = searchParams.toString();
    const response = await api.get<GenericLimitOffsetResponse<unknown>>(
      `${BASE}/${templateId}/versions${queryString ? `?${queryString}` : ""}`,
      { signal: options?.signal },
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

  /**
   * The inputs a template's version declares, for a contract form. With no version it is the
   * version that bills today (the template's billing version), or the template's own content
   * while nothing has been approved.
   */
  public async getDeclaredInputs(
    templateId: string,
    versionNumber: number | null,
    options?: RequestOptions,
  ): Promise<TariffInputDeclaration[]> {
    if (versionNumber !== null) {
      const version = await api.get<unknown>(`${BASE}/${templateId}/versions/${versionNumber}`, {
        signal: options?.signal,
      });
      const parsed = await safeParse(declaredInputsSchema, version, "Formula Template Inputs");
      return parsed.inputs;
    }
    const template = await api.get<unknown>(`${BASE}/${templateId}/`, {
      signal: options?.signal,
    });
    const parsed = await safeParse(declaredInputsSchema, template, "Formula Template Inputs");
    const billing = parsed.billing?.versionNumber;
    return billing ? this.getDeclaredInputs(templateId, billing, options) : parsed.inputs;
  }

  public async getVersion(
    templateId: string,
    versionNumber: number,
    options?: RequestOptions,
  ): Promise<FormulaTemplateVersion> {
    const response = await api.get<unknown>(`${BASE}/${templateId}/versions/${versionNumber}`, {
      signal: options?.signal,
    });
    return safeParse(formulaTemplateVersionSchema, response, "Formula Template Version");
  }

  public async createVersion(
    templateId: string,
    data: CreateVersionRequest,
  ): Promise<FormulaTemplateVersion> {
    const response = await api.post<unknown>(`${BASE}/${templateId}/versions`, data);
    return safeParse(formulaTemplateVersionSchema, response, "Formula Template Version");
  }

  public async rollback(templateId: string, data: RollbackRequest): Promise<FormulaTemplate> {
    const response = await api.post<unknown>(`${BASE}/${templateId}/rollback`, data);
    return safeParse(formulaTemplateSchema, response, "Formula Template");
  }

  public async fork(templateId: string, data: ForkRequest): Promise<FormulaTemplate> {
    const response = await api.post<unknown>(`${BASE}/${templateId}/fork`, data);
    return safeParse(formulaTemplateSchema, response, "Formula Template");
  }

  public async compareVersions(
    templateId: string,
    fromVersion: number,
    toVersion: number,
    options?: RequestOptions,
  ): Promise<VersionDiff> {
    const response = await api.get<unknown>(
      `${BASE}/${templateId}/compare?from=${fromVersion}&to=${toVersion}`,
      { signal: options?.signal },
    );
    return safeParse(versionDiffSchema, response, "Version Diff");
  }

  public async getLineage(templateId: string, options?: RequestOptions): Promise<ForkLineage> {
    const response = await api.get<unknown>(`${BASE}/${templateId}/lineage`, {
      signal: options?.signal,
    });
    return safeParse(forkLineageSchema, response, "Fork Lineage");
  }

  public async getUsage(templateId: string, options?: RequestOptions): Promise<TemplateUsageResponse> {
    const response = await api.get<unknown>(`${BASE}/${templateId}/usage`, {
      signal: options?.signal,
    });
    return safeParse(templateUsageResponseSchema, response, "Template Usage");
  }

  public async updateVersionTags(
    templateId: string,
    versionNumber: number,
    tags: string[],
  ): Promise<FormulaTemplateVersion> {
    const response = await api.patch<unknown>(
      `${BASE}/${templateId}/versions/${versionNumber}/tags`,
      { tags },
    );
    return safeParse(formulaTemplateVersionSchema, response, "Formula Template Version");
  }

  public async submit(
    templateId: string,
    request: string | SubmitForReviewRequest,
  ): Promise<FormulaTemplate> {
    const body = typeof request === "string" ? { comment: request } : request;
    const response = await api.post<unknown>(`${BASE}/${templateId}/submit`, body);
    return safeParse(formulaTemplateSchema, response, "Formula Template");
  }

  public async listReviewers(
    templateId: string,
    options?: RequestOptions,
  ): Promise<FormulaReviewer[]> {
    const response = await api.get<unknown>(`${BASE}/${templateId}/reviewers`, {
      signal: options?.signal,
    });
    return safeParse(listFormulaReviewersResponseSchema, response, "Formula Template Reviewers");
  }

  /** Nova's statement-by-statement reading of a rating of the draft. */
  public async explainFormula(
    content: FormulaDraftContent,
    load: ExplainPreviewLoad,
    options?: RequestOptions,
  ): Promise<FormulaExplainResponse> {
    const response = await api.post<unknown>(
      `${BASE}/ai/explain`,
      { ...content, ...load },
      { signal: options?.signal },
    );
    return safeParse(formulaExplainResponseSchema, response, "Formula Explanation");
  }

  /** Tests Nova suggests for the draft, as `test` blocks it has run against the draft. */
  public async suggestTests(
    content: FormulaDraftContent,
    options?: RequestOptions,
  ): Promise<SuggestedTestsResponse> {
    const response = await api.post<unknown>(`${BASE}/ai/suggest-tests`, content, {
      signal: options?.signal,
    });
    return safeParse(suggestedTestsResponseSchema, response, "Suggested Tests");
  }

  public async draftReviewNote(
    templateId: string,
    options?: RequestOptions,
  ): Promise<ReviewNoteResponse> {
    const response = await api.post<unknown>(
      `${BASE}/${templateId}/review-note`,
      {},
      { signal: options?.signal },
    );
    return safeParse(reviewNoteResponseSchema, response, "Reviewer Note");
  }

  /**
   * Approves the template. With `effectiveFrom` the approved content bills from
   * that moment; until then the previously approved content keeps billing.
   */
  public async approve(
    templateId: string,
    comment: string,
    effectiveFrom?: number | null,
  ): Promise<FormulaTemplate> {
    const response = await api.post<unknown>(`${BASE}/${templateId}/approve`, {
      comment,
      effectiveFrom: effectiveFrom ?? null,
    });
    return safeParse(formulaTemplateSchema, response, "Formula Template");
  }

  public async reject(templateId: string, comment: string): Promise<FormulaTemplate> {
    const response = await api.post<unknown>(`${BASE}/${templateId}/reject`, { comment });
    return safeParse(formulaTemplateSchema, response, "Formula Template");
  }

  public async requestChanges(templateId: string, comment: string): Promise<FormulaTemplate> {
    const response = await api.post<unknown>(`${BASE}/${templateId}/request-changes`, {
      comment,
    });
    return safeParse(formulaTemplateSchema, response, "Formula Template");
  }

  public async listReviews(
    templateId: string,
    options?: RequestOptions,
  ): Promise<FormulaTemplateReview[]> {
    const response = await api.get<unknown>(`${BASE}/${templateId}/reviews`, {
      signal: options?.signal,
    });
    return safeParse(
      listFormulaTemplateReviewsResponseSchema,
      response,
      "Formula Template Reviews",
    );
  }

  public async updateVersionEffectiveDate(
    templateId: string,
    versionNumber: number,
    effectiveFrom: number | null,
  ): Promise<FormulaTemplateVersion> {
    const response = await api.patch<unknown>(
      `${BASE}/${templateId}/versions/${versionNumber}/effective-date`,
      { effectiveFrom },
    );
    return safeParse(formulaTemplateVersionSchema, response, "Formula Template Version");
  }

  public async listScheduledVersions(
    templateId: string,
    options?: RequestOptions,
  ): Promise<FormulaTemplateVersion[]> {
    const response = await api.get<unknown>(`${BASE}/${templateId}/versions/scheduled`, {
      signal: options?.signal,
    });
    return safeParse(z.array(formulaTemplateVersionSchema), response, "Formula Template Version");
  }

  public async backtest(templateId: string, request: BacktestRequest): Promise<BacktestResponse> {
    const response = await api.post<unknown>(`${BASE}/${templateId}/backtest`, request);
    return safeParse(backtestResponseSchema, response, "Backtest");
  }

  public async approvalImpact(templateId: string, limit?: number): Promise<BacktestResponse> {
    const response = await api.post<unknown>(`${BASE}/${templateId}/impact`, {
      limit: limit ?? 0,
    });
    return safeParse(backtestResponseSchema, response, "Approval Impact");
  }

  public async readiness(templateId: string, options?: RequestOptions): Promise<ReadinessResponse> {
    const response = await api.get<unknown>(`${BASE}/${templateId}/readiness`, {
      signal: options?.signal,
    });
    return safeParse(readinessResponseSchema, response, "Formula Template Readiness");
  }

  public async listStandards(options?: RequestOptions): Promise<StandardTemplate[]> {
    const response = await api.get<unknown>(`${BASE}/standards`, { signal: options?.signal });
    return safeParse(listStandardsResponseSchema, response, "Formula Template Standards");
  }

  public async reviewDiff(templateId: string, options?: RequestOptions): Promise<ReviewDiffResponse> {
    const response = await api.get<unknown>(`${BASE}/${templateId}/review-diff`, {
      signal: options?.signal,
    });
    return safeParse(reviewDiffResponseSchema, response, "Formula Template Review Diff");
  }
}
