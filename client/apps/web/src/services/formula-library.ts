import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  formulaLibrarySchema,
  formulaLibraryUsageResponseSchema,
  formulaLibraryVersionSchema,
  type FormulaLibrary,
  type FormulaLibraryUsageResponse,
  type FormulaLibraryVersion,
  type FormulaLibraryWrite,
} from "@trenova/shared/types/formula-library";
import {
  listFormulaReviewersResponseSchema,
  readinessResponseSchema,
  reviewDiffResponseSchema,
  versionDiffSchema,
  type CreateVersionRequest,
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
 * (readiness, review diff, usages, reviewers). Lists and records are read through GraphQL
 * (`lib/graphql/formula-library.ts`).
 */
export class FormulaLibraryService {
  public async create(request: FormulaLibraryWrite): Promise<FormulaLibrary> {
    const response = await api.post<unknown>(`${BASE}/`, request);
    return safeParse(formulaLibrarySchema, response, "Formula Library");
  }

  public async update(libraryId: string, request: FormulaLibraryWrite): Promise<FormulaLibrary> {
    const response = await api.put<unknown>(`${BASE}/${libraryId}/`, request);
    return safeParse(formulaLibrarySchema, response, "Formula Library");
  }

  public async duplicate(libraryId: string, name: string): Promise<FormulaLibrary> {
    const response = await api.post<unknown>(`${BASE}/${libraryId}/duplicate`, { name });
    return safeParse(formulaLibrarySchema, response, "Formula Library");
  }

  public async archive(libraryId: string): Promise<FormulaLibrary> {
    const response = await api.post<unknown>(`${BASE}/${libraryId}/archive`, {});
    return safeParse(formulaLibrarySchema, response, "Formula Library");
  }

  public async createVersion(
    libraryId: string,
    data: CreateVersionRequest,
  ): Promise<FormulaLibraryVersion> {
    const response = await api.post<unknown>(`${BASE}/${libraryId}/versions`, data);
    return safeParse(formulaLibraryVersionSchema, response, "Formula Library Version");
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

  public async getVersion(
    libraryId: string,
    versionNumber: number,
    options?: RequestOptions,
  ): Promise<FormulaLibraryVersion> {
    const response = await api.get<unknown>(`${BASE}/${libraryId}/versions/${versionNumber}`, {
      signal: options?.signal,
    });
    return safeParse(formulaLibraryVersionSchema, response, "Formula Library Version");
  }

  public async listVersions(
    libraryId: string,
    options?: RequestOptions,
  ): Promise<FormulaLibraryVersion[]> {
    const response = await api.get<unknown>(`${BASE}/${libraryId}/versions`, {
      signal: options?.signal,
    });
    return safeParse(z.array(formulaLibraryVersionSchema), response, "Formula Library Versions");
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

  public async reviewDiff(libraryId: string, options?: RequestOptions): Promise<ReviewDiffResponse> {
    const response = await api.get<unknown>(`${BASE}/${libraryId}/review-diff`, {
      signal: options?.signal,
    });
    return safeParse(reviewDiffResponseSchema, response, "Formula Library Review Diff");
  }

  /** The template versions that import the library, and the library version each pinned. */
  public async usage(
    libraryId: string,
    options?: RequestOptions,
  ): Promise<FormulaLibraryUsageResponse> {
    const response = await api.get<unknown>(`${BASE}/${libraryId}/usage`, {
      signal: options?.signal,
    });
    return safeParse(formulaLibraryUsageResponseSchema, response, "Formula Library Usage");
  }
}
