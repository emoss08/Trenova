import {
  ApproveFormulaTemplateDocument,
  BulkUpdateFormulaTemplateStatusDocument,
  CreateFormulaTemplateDocument,
  DescribeFormulaTemplateInputsDocument,
  DuplicateFormulaTemplatesDocument,
  ExplainFormulaTemplateDocument,
  ExplainFormulaTemplateTestDocument,
  ForkFormulaTemplateDocument,
  FormulaTemplateApprovalImpactDocument,
  FormulaTemplateBacktestDocument,
  FormulaTemplateDeclaredInputsDocument,
  FormulaTemplateLineageDocument,
  FormulaTemplatePreviewDocument,
  FormulaTemplateReadinessDocument,
  FormulaTemplateRecentDocument,
  FormulaTemplateRecordDocument,
  FormulaTemplateReviewDiffDocument,
  FormulaTemplateReviewNoteDocument,
  FormulaTemplateReviewersDocument,
  FormulaTemplateReviewsDocument,
  FormulaTemplateScaffoldTestDocument,
  FormulaTemplateScheduledVersionsDocument,
  FormulaTemplateStandardsDocument,
  FormulaTemplateStudioContextDocument,
  FormulaTemplateTableCellsDocument,
  FormulaTemplateUsageDocument,
  FormulaTemplateVersionDiffDocument,
  FormulaTemplateVersionInputsDocument,
  FormulaTemplateVersionsDocument,
  ImportFormulaTemplatesDocument,
  InstallStandardFormulaTemplatesDocument,
  OpenFormulaTemplateAssistantThreadDocument,
  RejectFormulaTemplateDocument,
  RequestFormulaTemplateChangesDocument,
  RollbackFormulaTemplateDocument,
  SubmitFormulaTemplateDocument,
  SuggestFormulaTemplateTestsDocument,
  UpdateFormulaTemplateDocument,
  UpdateFormulaTemplateVersionEffectiveDateDocument,
  UpdateFormulaTemplateVersionTagsDocument,
  type FormulaDraftInput,
  type FormulaVersionTag,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import { isRecord } from "@trenova/shared/lib/utils";
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
  describeInputsResponseSchema,
  explainTestResponseSchema,
  templateUsageResponseSchema,
  versionDiffSchema,
  type BacktestRequest,
  type BacktestResponse,
  type BulkDuplicateFormulaTemplateRequest,
  type BulkUpdateStatusRequest,
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
  type ReadinessResponse,
  type ReviewDiffResponse,
  type ReviewNoteResponse,
  type RollbackRequest,
  type ScaffoldTestRequest,
  type ScaffoldTestResponse,
  type StandardTemplate,
  type SuggestedTestsResponse,
  type DescribeInputsResponse,
  type ExplainTestResponse,
  type TemplateUsageResponse,
  type VersionDiff,
} from "@trenova/shared/types/formula-template";
import {
  tariffStudioContextSchema,
  tariffInputDeclarationsSchema,
  tariffTableCellsSchema,
  type TariffNeed,
  type TariffStudioContext,
  type TariffTableCells,
  type TariffInputDeclaration,
  type TariffInputValues,
} from "@trenova/shared/types/tariff";
import type { GenericLimitOffsetResponse } from "@trenova/shared/types/server";
import { pageThreadSchema, type PageThread } from "@/types/assistant";
import { z } from "zod";

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
  inputs?: TariffInputValues;
  shipmentId?: string;
};

export type ImportTemplatesRequest = {
  exportVersion: string;
  templates: ImportTemplatePayload[];
  onConflict?: "reject" | "rename";
};

type RequestOptions = { signal?: AbortSignal };

/** Raised when the template does not exist or is not the organisation's. */
export class FormulaTemplateNotFoundError extends Error {
  constructor(templateId: string) {
    super(`formula template ${templateId} was not found`);
    this.name = "FormulaTemplateNotFoundError";
  }
}

/**
 * Input declarations as the browser's module states them: `default` is present only when the
 * source declares one. GraphQL says so through `hasDefault`, since its null is also a value.
 */
export function inputDeclarationsFromGraphQL(inputs: readonly unknown[]): unknown[] {
  return inputs.map((input) => {
    if (!isRecord(input)) return input;
    const { hasDefault, default: value, ...declaration } = input;
    return hasDefault ? { ...declaration, default: value } : declaration;
  });
}

function withDeclaredInputs(record: unknown): unknown {
  if (!isRecord(record) || !Array.isArray(record.inputs)) return record;
  return { ...record, inputs: inputDeclarationsFromGraphQL(record.inputs) };
}

function draftInput(content: FormulaDraftContent): FormulaDraftInput {
  return {
    templateId: content.templateId || null,
    name: content.name,
    description: content.description,
    source: content.source,
  };
}

function parseTemplate(record: unknown, label: string): Promise<FormulaTemplate> {
  return safeParse(formulaTemplateSchema, withDeclaredInputs(record), label);
}

function parseTemplates(records: readonly unknown[], label: string) {
  return safeParse(
    listFormulaTemplateResponseSchema,
    records.map((record) => withDeclaredInputs(record)),
    label,
  );
}

function parseVersion(record: unknown): Promise<FormulaTemplateVersion> {
  return safeParse(
    formulaTemplateVersionSchema,
    withDeclaredInputs(record),
    "Formula Template Version",
  );
}

function parseVersions(records: readonly unknown[]): Promise<FormulaTemplateVersion[]> {
  return safeParse(
    z.array(formulaTemplateVersionSchema),
    records.map((record) => withDeclaredInputs(record)),
    "Formula Template Version",
  );
}

export class FormulaTemplateService {
  /** The organisation's most recently changed templates. */
  public async list(
    params: { limit: number },
    options?: RequestOptions,
  ): Promise<GenericLimitOffsetResponse<FormulaTemplate>> {
    const data = await requestGraphQL({
      document: FormulaTemplateRecentDocument,
      operationName: "FormulaTemplateRecent",
      variables: { first: params.limit },
      signal: options?.signal,
    });
    const results = await safeParse(
      listFormulaTemplateResponseSchema,
      data.formulaTemplates.edges.map((edge) => edge.node),
      "Formula Templates",
    );
    return { results, count: results.length, next: null, prev: null };
  }

  public async get(templateId: string, options?: RequestOptions): Promise<FormulaTemplate> {
    const data = await requestGraphQL({
      document: FormulaTemplateRecordDocument,
      operationName: "FormulaTemplateRecord",
      variables: { id: templateId },
      signal: options?.signal,
    });
    if (!data.formulaTemplate) throw new FormulaTemplateNotFoundError(templateId);
    return parseTemplate(data.formulaTemplate, "Formula Template");
  }

  public async create(request: FormulaTemplateWrite): Promise<FormulaTemplate> {
    const data = await requestGraphQL({
      document: CreateFormulaTemplateDocument,
      operationName: "CreateFormulaTemplate",
      variables: { input: { source: request.source, description: request.description } },
    });
    return parseTemplate(data.createFormulaTemplate, "Formula Template");
  }

  public async update(templateId: string, request: FormulaTemplateWrite): Promise<FormulaTemplate> {
    const data = await requestGraphQL({
      document: UpdateFormulaTemplateDocument,
      operationName: "UpdateFormulaTemplate",
      variables: {
        id: templateId,
        input: {
          source: request.source,
          description: request.description,
          version: request.version ?? 0,
        },
      },
    });
    return parseTemplate(data.updateFormulaTemplate, "Formula Template");
  }

  /**
   * What the browser checks a tariff against: the organisation's settings, its rate tables'
   * shapes, its accessorial charge codes and the sources of the libraries a tariff may import.
   */
  public async getContext(options?: RequestOptions): Promise<TariffStudioContext> {
    const data = await requestGraphQL({
      document: FormulaTemplateStudioContextDocument,
      operationName: "FormulaTemplateStudioContext",
      signal: options?.signal,
    });
    return safeParse(
      tariffStudioContextSchema,
      data.formulaTemplateStudioContext,
      "Tariff Context",
    );
  }

  /**
   * The rate table cells a file's tests read in the browser (the `needs` `tariff.test` answered),
   * or with `whole` the tables those needs name, entire, with the stamp they were read at.
   */
  public async getContextCells(
    request: { needs: readonly TariffNeed[]; whole: boolean },
    options?: RequestOptions,
  ): Promise<TariffTableCells> {
    const data = await requestGraphQL({
      document: FormulaTemplateTableCellsDocument,
      operationName: "FormulaTemplateTableCells",
      variables: {
        input: {
          needs: request.needs.map((need) => ({ table: need.table, keys: [...need.keys] })),
          whole: request.whole,
        },
      },
      signal: options?.signal,
    });
    return safeParse(tariffTableCellsSchema, data.formulaTemplateTableCells, "Tariff Table Cells");
  }

  /** Rates the source against the sample record, or a real shipment, with the given inputs. */
  public async preview(
    request: FormulaPreviewRequest,
    options?: RequestOptions,
  ): Promise<FormulaPreviewResponse> {
    const data = await requestGraphQL({
      document: FormulaTemplatePreviewDocument,
      operationName: "FormulaTemplatePreview",
      variables: {
        input: {
          source: request.source,
          inputs: request.inputs ?? null,
          shipmentId: request.shipmentId || null,
        },
      },
      signal: options?.signal,
    });
    return safeParse(formulaPreviewResponseSchema, data.formulaTemplatePreview, "Formula Preview");
  }

  /** Writes a `test` block pinning what the tariff bills for a real shipment. */
  public async scaffoldTest(
    request: ScaffoldTestRequest,
    options?: RequestOptions,
  ): Promise<ScaffoldTestResponse> {
    const data = await requestGraphQL({
      document: FormulaTemplateScaffoldTestDocument,
      operationName: "FormulaTemplateScaffoldTest",
      variables: {
        input: {
          source: request.source,
          inputs: request.inputs ?? null,
          shipmentId: request.shipmentId,
          name: request.name,
        },
      },
      signal: options?.signal,
    });
    return safeParse(
      scaffoldTestResponseSchema,
      data.formulaTemplateScaffoldTest,
      "Scaffolded Test",
    );
  }

  public async importTemplates(request: ImportTemplatesRequest): Promise<ImportTemplatesResponse> {
    const data = await requestGraphQL({
      document: ImportFormulaTemplatesDocument,
      operationName: "ImportFormulaTemplates",
      variables: {
        input: {
          exportVersion: request.exportVersion,
          templates: request.templates.map((template) => ({
            name: template.name,
            description: template.description ?? null,
            source: template.source,
            sourceTemplateId: template.sourceTemplateId ?? null,
            sourceVersionNumber: template.sourceVersionNumber ?? null,
          })),
          onConflict:
            request.onConflict === "rename"
              ? "Rename"
              : request.onConflict === "reject"
                ? "Reject"
                : null,
        },
      },
    });
    const result = data.importFormulaTemplates;
    return safeParse(
      importTemplatesResponseSchema,
      { ...result, created: result.created.map((record) => withDeclaredInputs(record)) },
      "Formula Template Import",
    );
  }

  public async installStandards(): Promise<InstallStandardsResponse> {
    const data = await requestGraphQL({
      document: InstallStandardFormulaTemplatesDocument,
      operationName: "InstallStandardFormulaTemplates",
    });
    const result = data.installStandardFormulaTemplates;
    return safeParse(
      installStandardsResponseSchema,
      { ...result, installed: result.installed.map((record) => withDeclaredInputs(record)) },
      "Standard Templates",
    );
  }

  /**
   * Opens the person's conversation with the formula assistant about a saved
   * template, or a fresh one while the template is unsaved.
   */
  public async openAssistantThread(templateId: string | null): Promise<PageThread> {
    const data = await requestGraphQL({
      document: OpenFormulaTemplateAssistantThreadDocument,
      operationName: "OpenFormulaTemplateAssistantThread",
      variables: { templateId: templateId || null },
    });
    return safeParse(
      pageThreadSchema,
      data.openFormulaTemplateAssistantThread,
      "Formula Assistant Conversation",
    );
  }

  public async bulkUpdateStatus(request: BulkUpdateStatusRequest) {
    const data = await requestGraphQL({
      document: BulkUpdateFormulaTemplateStatusDocument,
      operationName: "BulkUpdateFormulaTemplateStatus",
      variables: {
        input: {
          templateIds: request.templateIds,
          status: request.status,
          confirmInUse: request.confirmInUse ?? null,
        },
      },
    });
    return parseTemplates(data.bulkUpdateFormulaTemplateStatus, "Formula Template");
  }

  public async bulkDuplicate(request: BulkDuplicateFormulaTemplateRequest) {
    const data = await requestGraphQL({
      document: DuplicateFormulaTemplatesDocument,
      operationName: "DuplicateFormulaTemplates",
      variables: { templateIds: request.templateIds },
    });
    return parseTemplates(data.duplicateFormulaTemplates, "Formula Template");
  }

  public async listVersions(
    templateId: string,
    params?: { limit?: number; offset?: number },
    options?: RequestOptions,
  ): Promise<GenericLimitOffsetResponse<FormulaTemplateVersion>> {
    const data = await requestGraphQL({
      document: FormulaTemplateVersionsDocument,
      operationName: "FormulaTemplateVersions",
      variables: {
        templateId,
        limit: params?.limit ?? null,
        offset: params?.offset ?? null,
      },
      signal: options?.signal,
    });
    const page = data.formulaTemplateVersions;
    return {
      results: await parseVersions(page.results),
      count: page.totalCount,
      totalCount: page.totalCount,
      totalCountExact: page.totalCountExact,
      next: null,
      prev: null,
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
      const data = await requestGraphQL({
        document: FormulaTemplateVersionInputsDocument,
        operationName: "FormulaTemplateVersionInputs",
        variables: { templateId, versionNumber },
        signal: options?.signal,
      });
      if (!data.formulaTemplateVersion) throw new FormulaTemplateNotFoundError(templateId);
      return safeParse(
        tariffInputDeclarationsSchema,
        inputDeclarationsFromGraphQL(data.formulaTemplateVersion.inputs as readonly unknown[]),
        "Formula Template Inputs",
      );
    }
    const data = await requestGraphQL({
      document: FormulaTemplateDeclaredInputsDocument,
      operationName: "FormulaTemplateDeclaredInputs",
      variables: { id: templateId },
      signal: options?.signal,
    });
    if (!data.formulaTemplate) throw new FormulaTemplateNotFoundError(templateId);
    const billing = data.formulaTemplate.billing?.versionNumber;
    if (billing) return this.getDeclaredInputs(templateId, billing, options);
    return safeParse(
      tariffInputDeclarationsSchema,
      inputDeclarationsFromGraphQL(data.formulaTemplate.inputs as readonly unknown[]),
      "Formula Template Inputs",
    );
  }

  public async rollback(templateId: string, request: RollbackRequest): Promise<FormulaTemplate> {
    const data = await requestGraphQL({
      document: RollbackFormulaTemplateDocument,
      operationName: "RollbackFormulaTemplate",
      variables: {
        id: templateId,
        input: {
          targetVersion: request.targetVersion,
          changeMessage: request.changeMessage ?? null,
        },
      },
    });
    return parseTemplate(data.rollbackFormulaTemplate, "Formula Template");
  }

  public async fork(templateId: string, request: ForkRequest): Promise<FormulaTemplate> {
    const data = await requestGraphQL({
      document: ForkFormulaTemplateDocument,
      operationName: "ForkFormulaTemplate",
      variables: {
        id: templateId,
        input: {
          newName: request.newName,
          sourceVersion: request.sourceVersion ?? null,
          changeMessage: request.changeMessage ?? null,
        },
      },
    });
    return parseTemplate(data.forkFormulaTemplate, "Formula Template");
  }

  public async compareVersions(
    templateId: string,
    fromVersion: number,
    toVersion: number,
    options?: RequestOptions,
  ): Promise<VersionDiff> {
    const data = await requestGraphQL({
      document: FormulaTemplateVersionDiffDocument,
      operationName: "FormulaTemplateVersionDiff",
      variables: { templateId, fromVersion, toVersion },
      signal: options?.signal,
    });
    return safeParse(versionDiffSchema, data.formulaTemplateVersionDiff, "Version Diff");
  }

  public async getLineage(templateId: string, options?: RequestOptions): Promise<ForkLineage> {
    const data = await requestGraphQL({
      document: FormulaTemplateLineageDocument,
      operationName: "FormulaTemplateLineage",
      variables: { templateId },
      signal: options?.signal,
    });
    return safeParse(forkLineageSchema, data.formulaTemplateLineage, "Fork Lineage");
  }

  public async getUsage(
    templateId: string,
    options?: RequestOptions,
  ): Promise<TemplateUsageResponse> {
    const data = await requestGraphQL({
      document: FormulaTemplateUsageDocument,
      operationName: "FormulaTemplateUsage",
      variables: { templateId },
      signal: options?.signal,
    });
    return safeParse(templateUsageResponseSchema, data.formulaTemplateUsage, "Template Usage");
  }

  public async updateVersionTags(
    templateId: string,
    versionNumber: number,
    tags: FormulaVersionTag[],
  ): Promise<FormulaTemplateVersion> {
    const data = await requestGraphQL({
      document: UpdateFormulaTemplateVersionTagsDocument,
      operationName: "UpdateFormulaTemplateVersionTags",
      variables: { templateId, versionNumber, tags },
    });
    return parseVersion(data.updateFormulaTemplateVersionTags);
  }

  public async submit(
    templateId: string,
    request: string | SubmitForReviewRequest,
  ): Promise<FormulaTemplate> {
    const body = typeof request === "string" ? { comment: request } : request;
    const data = await requestGraphQL({
      document: SubmitFormulaTemplateDocument,
      operationName: "SubmitFormulaTemplate",
      variables: {
        id: templateId,
        input: {
          comment: body.comment ?? null,
          reviewerId: body.reviewerId || null,
          reviewerNote: body.reviewerNote ?? null,
        },
      },
    });
    return parseTemplate(data.submitFormulaTemplate, "Formula Template");
  }

  public async listReviewers(
    templateId: string,
    options?: RequestOptions,
  ): Promise<FormulaReviewer[]> {
    const data = await requestGraphQL({
      document: FormulaTemplateReviewersDocument,
      operationName: "FormulaTemplateReviewers",
      variables: { templateId },
      signal: options?.signal,
    });
    return safeParse(
      listFormulaReviewersResponseSchema,
      data.formulaTemplateReviewers,
      "Formula Template Reviewers",
    );
  }

  /** Nova's statement-by-statement reading of a rating of the draft. */
  public async explainFormula(
    content: FormulaDraftContent,
    load: ExplainPreviewLoad,
    options?: RequestOptions,
  ): Promise<FormulaExplainResponse> {
    const data = await requestGraphQL({
      document: ExplainFormulaTemplateDocument,
      operationName: "ExplainFormulaTemplate",
      variables: {
        input: {
          draft: draftInput(content),
          inputs: load.inputs ?? null,
          shipmentId: load.shipmentId || null,
        },
      },
      signal: options?.signal,
    });
    return safeParse(
      formulaExplainResponseSchema,
      data.explainFormulaTemplate,
      "Formula Explanation",
    );
  }

  /** Tests Nova suggests for the draft, as `test` blocks it has run against the draft. */
  public async suggestTests(
    content: FormulaDraftContent,
    options?: RequestOptions,
  ): Promise<SuggestedTestsResponse> {
    const data = await requestGraphQL({
      document: SuggestFormulaTemplateTestsDocument,
      operationName: "SuggestFormulaTemplateTests",
      variables: { input: draftInput(content) },
      signal: options?.signal,
    });
    return safeParse(
      suggestedTestsResponseSchema,
      data.suggestFormulaTemplateTests,
      "Suggested Tests",
    );
  }

  /** Doc comments for the inputs the draft declares without one, by input name. */
  public async describeInputs(
    content: FormulaDraftContent,
    options?: RequestOptions,
  ): Promise<DescribeInputsResponse> {
    const data = await requestGraphQL({
      document: DescribeFormulaTemplateInputsDocument,
      operationName: "DescribeFormulaTemplateInputs",
      variables: { input: draftInput(content) },
      signal: options?.signal,
    });
    return safeParse(
      describeInputsResponseSchema,
      data.describeFormulaTemplateInputs,
      "Input Descriptions",
    );
  }

  /** Why the named test fails on the draft, and whether the tariff or the test is wrong. */
  public async explainTest(
    content: FormulaDraftContent,
    testName: string,
    options?: RequestOptions,
  ): Promise<ExplainTestResponse> {
    const data = await requestGraphQL({
      document: ExplainFormulaTemplateTestDocument,
      operationName: "ExplainFormulaTemplateTest",
      variables: { input: { draft: draftInput(content), testName } },
      signal: options?.signal,
    });
    return safeParse(
      explainTestResponseSchema,
      data.explainFormulaTemplateTest,
      "Test Explanation",
    );
  }

  public async draftReviewNote(
    templateId: string,
    options?: RequestOptions,
  ): Promise<ReviewNoteResponse> {
    const data = await requestGraphQL({
      document: FormulaTemplateReviewNoteDocument,
      operationName: "FormulaTemplateReviewNote",
      variables: { templateId },
      signal: options?.signal,
    });
    return safeParse(reviewNoteResponseSchema, data.formulaTemplateReviewNote, "Reviewer Note");
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
    const data = await requestGraphQL({
      document: ApproveFormulaTemplateDocument,
      operationName: "ApproveFormulaTemplate",
      variables: { id: templateId, input: { comment, effectiveFrom: effectiveFrom ?? null } },
    });
    return parseTemplate(data.approveFormulaTemplate, "Formula Template");
  }

  public async reject(templateId: string, comment: string): Promise<FormulaTemplate> {
    const data = await requestGraphQL({
      document: RejectFormulaTemplateDocument,
      operationName: "RejectFormulaTemplate",
      variables: { id: templateId, comment },
    });
    return parseTemplate(data.rejectFormulaTemplate, "Formula Template");
  }

  public async requestChanges(templateId: string, comment: string): Promise<FormulaTemplate> {
    const data = await requestGraphQL({
      document: RequestFormulaTemplateChangesDocument,
      operationName: "RequestFormulaTemplateChanges",
      variables: { id: templateId, comment },
    });
    return parseTemplate(data.requestFormulaTemplateChanges, "Formula Template");
  }

  public async listReviews(
    templateId: string,
    options?: RequestOptions,
  ): Promise<FormulaTemplateReview[]> {
    const data = await requestGraphQL({
      document: FormulaTemplateReviewsDocument,
      operationName: "FormulaTemplateReviews",
      variables: { templateId },
      signal: options?.signal,
    });
    return safeParse(
      listFormulaTemplateReviewsResponseSchema,
      data.formulaTemplateReviews,
      "Formula Template Reviews",
    );
  }

  public async updateVersionEffectiveDate(
    templateId: string,
    versionNumber: number,
    effectiveFrom: number | null,
  ): Promise<FormulaTemplateVersion> {
    const data = await requestGraphQL({
      document: UpdateFormulaTemplateVersionEffectiveDateDocument,
      operationName: "UpdateFormulaTemplateVersionEffectiveDate",
      variables: { templateId, versionNumber, effectiveFrom },
    });
    return parseVersion(data.updateFormulaTemplateVersionEffectiveDate);
  }

  public async listScheduledVersions(
    templateId: string,
    options?: RequestOptions,
  ): Promise<FormulaTemplateVersion[]> {
    const data = await requestGraphQL({
      document: FormulaTemplateScheduledVersionsDocument,
      operationName: "FormulaTemplateScheduledVersions",
      variables: { templateId },
      signal: options?.signal,
    });
    return parseVersions(data.formulaTemplateScheduledVersions);
  }

  /** Rates recent shipments with the candidate and with what bills today. Saves nothing. */
  public async backtest(
    templateId: string,
    request: BacktestRequest,
    options?: RequestOptions,
  ): Promise<BacktestResponse> {
    const data = await requestGraphQL({
      document: FormulaTemplateBacktestDocument,
      operationName: "FormulaTemplateBacktest",
      variables: {
        templateId,
        input: {
          source: request.source || null,
          versionNumber: request.versionNumber ?? null,
          limit: request.limit ?? null,
        },
      },
      signal: options?.signal,
    });
    return safeParse(backtestResponseSchema, data.formulaTemplateBacktest, "Backtest");
  }

  public async approvalImpact(
    templateId: string,
    limit?: number,
    options?: RequestOptions,
  ): Promise<BacktestResponse> {
    const data = await requestGraphQL({
      document: FormulaTemplateApprovalImpactDocument,
      operationName: "FormulaTemplateApprovalImpact",
      variables: { templateId, limit: limit ?? null },
      signal: options?.signal,
    });
    return safeParse(backtestResponseSchema, data.formulaTemplateApprovalImpact, "Approval Impact");
  }

  public async readiness(templateId: string, options?: RequestOptions): Promise<ReadinessResponse> {
    const data = await requestGraphQL({
      document: FormulaTemplateReadinessDocument,
      operationName: "FormulaTemplateReadiness",
      variables: { templateId },
      signal: options?.signal,
    });
    return safeParse(
      readinessResponseSchema,
      data.formulaTemplateReadiness,
      "Formula Template Readiness",
    );
  }

  public async listStandards(options?: RequestOptions): Promise<StandardTemplate[]> {
    const data = await requestGraphQL({
      document: FormulaTemplateStandardsDocument,
      operationName: "FormulaTemplateStandards",
      signal: options?.signal,
    });
    return safeParse(
      listStandardsResponseSchema,
      data.formulaTemplateStandards,
      "Formula Template Standards",
    );
  }

  public async reviewDiff(
    templateId: string,
    options?: RequestOptions,
  ): Promise<ReviewDiffResponse> {
    const data = await requestGraphQL({
      document: FormulaTemplateReviewDiffDocument,
      operationName: "FormulaTemplateReviewDiff",
      variables: { templateId },
      signal: options?.signal,
    });
    return safeParse(
      reviewDiffResponseSchema,
      data.formulaTemplateReviewDiff,
      "Formula Template Review Diff",
    );
  }
}
