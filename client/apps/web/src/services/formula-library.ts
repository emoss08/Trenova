import { pageThreadSchema, type PageThread } from "@/types/assistant";
import {
  ApproveFormulaLibraryDocument,
  ArchiveFormulaLibraryDocument,
  CreateFormulaLibraryDocument,
  DuplicateFormulaLibrariesDocument,
  FormulaLibraryReadinessDocument,
  FormulaLibraryReviewDiffDocument,
  FormulaLibraryReviewersDocument,
  FormulaLibraryUsagesDocument,
  FormulaLibraryVersionDiffDocument,
  OpenFormulaLibraryAssistantThreadDocument,
  RejectFormulaLibraryDocument,
  RequestFormulaLibraryChangesDocument,
  RollbackFormulaLibraryDocument,
  SubmitFormulaLibraryDocument,
  UpdateFormulaLibraryDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  formulaLibrarySchema,
  formulaLibraryUsagesSchema,
  type ArchiveFormulaLibraryRequest,
  type FormulaLibrary,
  type FormulaLibraryUsage,
  type FormulaLibraryWrite,
} from "@trenova/shared/types/formula-library";
import {
  listFormulaReviewersResponseSchema,
  readinessResponseSchema,
  reviewDiffResponseSchema,
  versionDiffSchema,
  type FormulaReviewer,
  type ReadinessResponse,
  type ReviewDiffResponse,
  type RollbackRequest,
  type VersionDiff,
} from "@trenova/shared/types/formula-template";
import { z } from "zod";
import type { SubmitForReviewRequest } from "./formula-template";

type RequestOptions = { signal?: AbortSignal };

function parseLibrary(record: unknown): Promise<FormulaLibrary> {
  return safeParse(formulaLibrarySchema, record, "Formula Library");
}

/**
 * Writes to formula libraries, and the reads that answer a question rather than show a record
 * (readiness, review diff, usages, reviewers, a version comparison). Lists, records and their
 * versions are read through `lib/graphql/formula-library.ts`.
 */
export class FormulaLibraryService {
  public async create(request: FormulaLibraryWrite): Promise<FormulaLibrary> {
    const data = await requestGraphQL({
      document: CreateFormulaLibraryDocument,
      operationName: "CreateFormulaLibrary",
      variables: {
        input: {
          name: request.name,
          description: request.description,
          source: request.source,
        },
      },
    });
    return parseLibrary(data.createFormulaLibrary);
  }

  /** A change to the source or the name saves a new version and returns the library to draft. */
  public async update(libraryId: string, request: FormulaLibraryWrite): Promise<FormulaLibrary> {
    const data = await requestGraphQL({
      document: UpdateFormulaLibraryDocument,
      operationName: "UpdateFormulaLibrary",
      variables: {
        id: libraryId,
        input: {
          name: request.name,
          description: request.description,
          source: request.source,
          version: request.version ?? 0,
        },
      },
    });
    return parseLibrary(data.updateFormulaLibrary);
  }

  /** Copies each library as a draft under a free name; returns the copies. */
  public async duplicate(libraryIds: string[]): Promise<FormulaLibrary[]> {
    const data = await requestGraphQL({
      document: DuplicateFormulaLibrariesDocument,
      operationName: "DuplicateFormulaLibraries",
      variables: { libraryIds },
    });
    return safeParse(
      z.array(formulaLibrarySchema),
      data.duplicateFormulaLibraries,
      "Formula Libraries",
    );
  }

  public async archive(
    libraryId: string,
    request: ArchiveFormulaLibraryRequest,
  ): Promise<FormulaLibrary> {
    const data = await requestGraphQL({
      document: ArchiveFormulaLibraryDocument,
      operationName: "ArchiveFormulaLibrary",
      variables: { id: libraryId, confirmInUse: request.confirmInUse },
    });
    return parseLibrary(data.archiveFormulaLibrary);
  }

  public async rollback(libraryId: string, request: RollbackRequest): Promise<FormulaLibrary> {
    const data = await requestGraphQL({
      document: RollbackFormulaLibraryDocument,
      operationName: "RollbackFormulaLibrary",
      variables: {
        id: libraryId,
        input: {
          targetVersion: request.targetVersion,
          changeMessage: request.changeMessage ?? null,
        },
      },
    });
    return parseLibrary(data.rollbackFormulaLibrary);
  }

  public async compareVersions(
    libraryId: string,
    fromVersion: number,
    toVersion: number,
    options?: RequestOptions,
  ): Promise<VersionDiff> {
    const data = await requestGraphQL({
      document: FormulaLibraryVersionDiffDocument,
      operationName: "FormulaLibraryVersionDiff",
      variables: { libraryId, fromVersion, toVersion },
      signal: options?.signal,
    });
    return safeParse(versionDiffSchema, data.formulaLibraryVersionDiff, "Version Diff");
  }

  public async submit(libraryId: string, request: SubmitForReviewRequest): Promise<FormulaLibrary> {
    const data = await requestGraphQL({
      document: SubmitFormulaLibraryDocument,
      operationName: "SubmitFormulaLibrary",
      variables: {
        id: libraryId,
        input: {
          comment: request.comment ?? null,
          reviewerId: request.reviewerId || null,
          reviewerNote: request.reviewerNote ?? null,
        },
      },
    });
    return parseLibrary(data.submitFormulaLibrary);
  }

  public async approve(libraryId: string, comment: string): Promise<FormulaLibrary> {
    const data = await requestGraphQL({
      document: ApproveFormulaLibraryDocument,
      operationName: "ApproveFormulaLibrary",
      variables: { id: libraryId, comment },
    });
    return parseLibrary(data.approveFormulaLibrary);
  }

  public async reject(libraryId: string, comment: string): Promise<FormulaLibrary> {
    const data = await requestGraphQL({
      document: RejectFormulaLibraryDocument,
      operationName: "RejectFormulaLibrary",
      variables: { id: libraryId, comment },
    });
    return parseLibrary(data.rejectFormulaLibrary);
  }

  public async requestChanges(libraryId: string, comment: string): Promise<FormulaLibrary> {
    const data = await requestGraphQL({
      document: RequestFormulaLibraryChangesDocument,
      operationName: "RequestFormulaLibraryChanges",
      variables: { id: libraryId, comment },
    });
    return parseLibrary(data.requestFormulaLibraryChanges);
  }

  public async listReviewers(
    libraryId: string,
    options?: RequestOptions,
  ): Promise<FormulaReviewer[]> {
    const data = await requestGraphQL({
      document: FormulaLibraryReviewersDocument,
      operationName: "FormulaLibraryReviewers",
      variables: { libraryId },
      signal: options?.signal,
    });
    return safeParse(
      listFormulaReviewersResponseSchema,
      data.formulaLibraryReviewers,
      "Formula Library Reviewers",
    );
  }

  public async readiness(libraryId: string, options?: RequestOptions): Promise<ReadinessResponse> {
    const data = await requestGraphQL({
      document: FormulaLibraryReadinessDocument,
      operationName: "FormulaLibraryReadiness",
      variables: { libraryId },
      signal: options?.signal,
    });
    return safeParse(
      readinessResponseSchema,
      data.formulaLibraryReadiness,
      "Formula Library Readiness",
    );
  }

  public async reviewDiff(
    libraryId: string,
    options?: RequestOptions,
  ): Promise<ReviewDiffResponse> {
    const data = await requestGraphQL({
      document: FormulaLibraryReviewDiffDocument,
      operationName: "FormulaLibraryReviewDiff",
      variables: { libraryId },
      signal: options?.signal,
    });
    return safeParse(
      reviewDiffResponseSchema,
      data.formulaLibraryReviewDiff,
      "Formula Library Review Diff",
    );
  }

  /** The template versions that import the library, and the library version each pinned. */
  public async usages(libraryId: string, options?: RequestOptions): Promise<FormulaLibraryUsage[]> {
    const data = await requestGraphQL({
      document: FormulaLibraryUsagesDocument,
      operationName: "FormulaLibraryUsages",
      variables: { libraryId },
      signal: options?.signal,
    });
    return safeParse(
      formulaLibraryUsagesSchema,
      data.formulaLibraryUsages,
      "Formula Library Usages",
    );
  }

  /**
   * Opens the person's conversation with the formula assistant about a saved library, or a
   * fresh one while the library is unsaved.
   */
  public async openAssistantThread(libraryId: string | null): Promise<PageThread> {
    const data = await requestGraphQL({
      document: OpenFormulaLibraryAssistantThreadDocument,
      operationName: "OpenFormulaLibraryAssistantThread",
      variables: { libraryId: libraryId || null },
    });
    return safeParse(
      pageThreadSchema,
      data.openFormulaLibraryAssistantThread,
      "Formula Assistant Conversation",
    );
  }
}
