import { z } from "zod";

// ErrorCode, ProblemType and the retryable set are generated from the Go declarations —
// see types/generated/error-enums.ts. They are re-exported here so the long-standing
// `@trenova/shared/types/errors` import path stays the one place callers reach for.
import {
  ErrorCode,
  ProblemType,
  RETRYABLE_PROBLEM_TYPES,
  retryableProblemTypes,
} from "@trenova/shared/types/generated/error-enums";

export { ErrorCode, ProblemType, RETRYABLE_PROBLEM_TYPES, retryableProblemTypes };

export const ErrorLocation = z.enum(["body", "business", "rate-limit"]);

export type ErrorLocation = z.infer<typeof ErrorLocation>;

/**
 * Where a server error points: object keys and list indexes, in order
 * (`["moves", 0, "stops", 1, "status"]`).
 */
export const fieldPathSchema = z.array(z.union([z.string(), z.number().int().nonnegative()]));

export type FieldPath = z.infer<typeof fieldPathSchema>;

/** The react-hook-form name of a server path: its segments joined with dots. */
export function fieldNameOf(path: readonly (string | number)[]): string {
  return path.join(".");
}

const errorCodeSchema = ErrorCode.optional().catch(undefined);

export const validationErrorSchema = z
  .object({
    path: fieldPathSchema,
    message: z.string(),
    code: errorCodeSchema,
    location: z.string().optional(),
  })
  .transform((error) => ({ ...error, field: fieldNameOf(error.path) }));

export type ValidationError = z.output<typeof validationErrorSchema>;

export const AdvisorySeverity = z.enum(["Warn", "RequireReview"]);

export type AdvisorySeverity = z.infer<typeof AdvisorySeverity>;

export const advisorySchema = z
  .object({
    path: fieldPathSchema,
    message: z.string(),
    code: errorCodeSchema,
    severity: AdvisorySeverity,
    ruleKey: z.string().optional(),
  })
  .transform((advisory) => ({ ...advisory, field: fieldNameOf(advisory.path) }));

export type Advisory = z.output<typeof advisorySchema>;

function parseEach<T>(value: unknown, schema: z.ZodType<T>): T[] {
  if (!Array.isArray(value)) {
    return [];
  }
  const parsed: T[] = [];
  for (const entry of value) {
    const result = schema.safeParse(entry);
    if (result.success) {
      parsed.push(result.data);
    }
  }
  return parsed;
}

/** Every well-formed field error in a payload; a malformed entry is skipped, not fatal. */
export function parseFieldErrors(value: unknown): ValidationError[] {
  return parseEach(value, validationErrorSchema);
}

/** Every well-formed advisory in a payload. */
export function parseAdvisories(value: unknown): Advisory[] {
  return parseEach(value, advisorySchema);
}

/**
 * What a save that lost a race is told: the version saved in between, who saved it, when,
 * and which settings they changed since the version the editor loaded.
 */
export const editConflictSchema = z.object({
  version: z.number(),
  updatedById: z.string().optional(),
  updatedByName: z.string().optional(),
  updatedAt: z.number(),
  changes: z.array(z.object({ field: z.string(), label: z.string() })),
});

export type EditConflict = z.infer<typeof editConflictSchema>;

export const apiErrorResponseSchema = z.object({
  type: z.string(),
  title: z.string(),
  status: z.number(),
  code: z.string().optional(),
  detail: z.string().optional(),
  instance: z.string().optional(),
  traceId: z.string().optional(),
  errors: z.optional(z.unknown().transform(parseFieldErrors)),
  advisories: z.optional(z.unknown().transform(parseAdvisories)),
  usageStats: z.any().optional(),
  params: z.record(z.string(), z.string()).optional(),
  conflict: editConflictSchema.optional(),
});

export type ApiErrorResponse = z.infer<typeof apiErrorResponseSchema>;

export function parseProblemType(type: string | undefined): ProblemType | null {
  if (!type) {
    return null;
  }
  const suffix = type.split("/").pop();
  const parsed = ProblemType.safeParse(suffix);
  return parsed.success ? parsed.data : null;
}

export type ProblemClassification = {
  problemType: ProblemType | null;
  status: number;
  code?: string;
  fieldErrors: ValidationError[];
};

export type NormalizedApiError = ProblemClassification & {
  advisories: Advisory[];
  message: string;
  title?: string;
  detail?: string;
  traceId?: string;
};

export const apiProblem = {
  isValidationError: (p: ProblemClassification): boolean => p.problemType === "validation-error",
  isBusinessError: (p: ProblemClassification): boolean =>
    p.problemType === "business-rule-violation",
  isAuthenticationError: (p: ProblemClassification): boolean =>
    p.problemType === "authentication-error",
  isAuthorizationError: (p: ProblemClassification): boolean =>
    p.problemType === "authorization-error",
  isNotFoundError: (p: ProblemClassification): boolean => p.problemType === "resource-not-found",
  isRateLimitError: (p: ProblemClassification): boolean =>
    p.status === 429 || p.problemType === "rate-limit-exceeded",
  isConflictError: (p: ProblemClassification): boolean =>
    p.status === 409 || p.problemType === "resource-conflict",
  isTimeoutError: (p: ProblemClassification): boolean =>
    p.status === 504 || p.problemType === "request-timeout",
  isVersionMismatchError: (p: ProblemClassification): boolean => p.code === "VERSION_MISMATCH",
} as const;
