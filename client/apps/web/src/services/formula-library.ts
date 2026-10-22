import { pageThreadSchema, type PageThread } from "@/types/assistant";
import { api } from "@trenova/shared/lib/api";
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

const BASE = "/formula-libraries";

/**
 * Writes to formula libraries, and the reads that answer a question rather than show a record
 * (readiness, review diff, usages, reviewers, a version comparison). Lists, records and their
 * versions are read through GraphQL (`lib/graphql/formula-library.ts`).
 */
export class FormulaLibraryService {
  public async create(request: FormulaLibraryWrite): Promise<FormulaLibrary> {
    const response = await api.post<unknown>(`${BASE}/`, request);
    return safeParse(formulaLibrarySchema, response, "Formula Library");
  }

  /** A change to the source or the name saves a new version and returns the library to draft. */
  public async update(libraryId: string, request: FormulaLibraryWrite): Promise<FormulaLibrary> {
    const response = await api.put<unknown>(`${BASE}/${libraryId}/`, request);
    return safeParse(formulaLibrarySchema, response, "Formula Library");
  }

  /** Copies each library as a draft under a free name; returns the copies. */
  public async duplicate(libraryIds: string[]): Promise<FormulaLibrary[]> {
    const response = await api.post<unknown>(`${BASE}/duplicate`, { libraryIds });
    return safeParse(z.array(formulaLibrarySchema), response, "Formula Libraries");
  }

  public async archive(
    libraryId: string,
    request: ArchiveFormulaLibraryRequest,
  ): Promise<FormulaLibrary> {
    const response = await api.post<unknown>(`${BASE}/${libraryId}/archive`, request);
    return safeParse(formulaLibrarySchema, response, "Formula Library");
  }

  public async rollback(libraryId: string, data: RollbackRequest): Promise<FormulaLibrary> {
    const response = await api.post<unknown>(`${BASE}/${libraryId}/rollback`, data);
    return safeParse(formulaLibrarySchema, response, "Formula Library");
  }

  public async compareVersions(
    libraryId: string,
    fromVersion: number,
    toVersion: number,
    options?: RequestOptions,
  ): Promise<VersionDiff> {
    const response = await api.get<unknown>(
      `${BASE}/${libraryId}/compare?from=${fromVersion}&to=${toVersion}`,
      { signal: options?.signal },
    );
    return safeParse(versionDiffSchema, response, "Version Diff");
  }

  public async submit(libraryId: string, request: SubmitForReviewRequest): Promise<FormulaLibrary> {
    const response = await api.post<unknown>(`${BASE}/${libraryId}/submit`, request);
    return safeParse(formulaLibrarySchema, response, "Formula Library");
  }

  public async approve(libraryId: string, comment: string): Promise<FormulaLibrary> {
    const response = await api.post<unknown>(`${BASE}/${libraryId}/approve`, { comment });
    return safeParse(formulaLibrarySchema, response, "Formula Library");
  }

  public async reject(libraryId: string, comment: string): Promise<FormulaLibrary> {
    const response = await api.post<unknown>(`${BASE}/${libraryId}/reject`, { comment });
    return safeParse(formulaLibrarySchema, response, "Formula Library");
  }

  public async requestChanges(libraryId: string, comment: string): Promise<FormulaLibrary> {
    const response = await api.post<unknown>(`${BASE}/${libraryId}/request-changes`, { comment });
    return safeParse(formulaLibrarySchema, response, "Formula Library");
  }

  public async listReviewers(
    libraryId: string,
    options?: RequestOptions,
  ): Promise<FormulaReviewer[]> {
    const response = await api.get<unknown>(`${BASE}/${libraryId}/reviewers`, {
      signal: options?.signal,
    });
    return safeParse(listFormulaReviewersResponseSchema, response, "Formula Library Reviewers");
  }

  public async readiness(libraryId: string, options?: RequestOptions): Promise<ReadinessResponse> {
    const response = await api.get<unknown>(`${BASE}/${libraryId}/readiness`, {
      signal: options?.signal,
    });
    return safeParse(readinessResponseSchema, response, "Formula Library Readiness");
  }

  public async reviewDiff(
    libraryId: string,
    options?: RequestOptions,
  ): Promise<ReviewDiffResponse> {
    const response = await api.get<unknown>(`${BASE}/${libraryId}/review-diff`, {
      signal: options?.signal,
    });
    return safeParse(reviewDiffResponseSchema, response, "Formula Library Review Diff");
  }

  /** The template versions that import the library, and the library version each pinned. */
  public async usages(libraryId: string, options?: RequestOptions): Promise<FormulaLibraryUsage[]> {
    const response = await api.get<unknown>(`${BASE}/${libraryId}/usages`, {
      signal: options?.signal,
    });
    return safeParse(formulaLibraryUsagesSchema, response, "Formula Library Usages");
  }

  /**
   * Opens the person's conversation with the formula assistant about a saved library, or a
   * fresh one while the library is unsaved.
   */
  public async openAssistantThread(libraryId: string | null): Promise<PageThread> {
    const response = await api.post<unknown>(`${BASE}/ai/thread/`, {
      libraryId: libraryId ?? "",
    });
    return safeParse(pageThreadSchema, response, "Formula Assistant Conversation");
  }
}
