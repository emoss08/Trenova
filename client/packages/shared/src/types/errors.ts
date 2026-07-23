import { z } from "zod";

export const ErrorCode = z.enum([
  "REQUIRED",
  "INVALID",
  "DUPLICATE",
  "NOT_FOUND",
  "BUSINESS_LOGIC",
  "UNAUTHORIZED",
  "FORBIDDEN",
  "INVALID_FORMAT",
  "INVALID_LENGTH",
  "INVALID_REFERENCE",
  "INVALID_OPERATION",
  "SYSTEM_ERROR",
  "ALREADY_EXISTS",
  "ALREADY_CLEARED",
  "VERSION_MISMATCH",
  "TOO_MANY_REQUESTS",
  "COMPLIANCE_VIOLATION",
  "RESOURCE_IN_USE",
  "BREAKING_CHANGE",
]);

export type ErrorCode = z.infer<typeof ErrorCode>;

export const ErrorLocation = z.enum(["body", "business", "rate-limit"]);

export type ErrorLocation = z.infer<typeof ErrorLocation>;

export const ProblemType = z.enum([
  "validation-error",
  "business-rule-violation",
  "database-error",
  "authentication-error",
  "authorization-error",
  "resource-not-found",
  "rate-limit-exceeded",
  "resource-conflict",
  "request-timeout",
  "internal-error",
]);

export type ProblemType = z.infer<typeof ProblemType>;

export const validationErrorSchema = z.object({
  field: z.string(),
  message: z.string(),
  code: z.string().optional(),
  location: z.string().optional(),
});

export type ValidationError = z.infer<typeof validationErrorSchema>;

export const apiErrorResponseSchema = z.object({
  type: z.string(),
  title: z.string(),
  status: z.number(),
  detail: z.string().optional(),
  instance: z.string().optional(),
  traceId: z.string().optional(),
  errors: z.array(validationErrorSchema).optional(),
  usageStats: z.any().optional(),
  params: z.record(z.string(), z.string()).optional(),
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

export function parseFieldErrors(value: unknown): ValidationError[] {
  const parsed = z.array(validationErrorSchema).safeParse(value);
  return parsed.success ? parsed.data : [];
}

export type ProblemClassification = {
  problemType: ProblemType | null;
  status: number;
  fieldErrors: ValidationError[];
};

export type NormalizedApiError = ProblemClassification & {
  message: string;
  title?: string;
  detail?: string;
  traceId?: string;
};

export const apiProblem = {
  isValidationError: (p: ProblemClassification): boolean =>
    p.problemType === "validation-error",
  isBusinessError: (p: ProblemClassification): boolean =>
    p.problemType === "business-rule-violation",
  isAuthenticationError: (p: ProblemClassification): boolean =>
    p.problemType === "authentication-error",
  isAuthorizationError: (p: ProblemClassification): boolean =>
    p.problemType === "authorization-error",
  isNotFoundError: (p: ProblemClassification): boolean =>
    p.problemType === "resource-not-found",
  isRateLimitError: (p: ProblemClassification): boolean =>
    p.status === 429 || p.problemType === "rate-limit-exceeded",
  isConflictError: (p: ProblemClassification): boolean =>
    p.status === 409 || p.problemType === "resource-conflict",
  isTimeoutError: (p: ProblemClassification): boolean =>
    p.status === 504 || p.problemType === "request-timeout",
  isVersionMismatchError: (p: ProblemClassification): boolean =>
    apiProblem.isValidationError(p) &&
    p.fieldErrors.some((e) => e.code === "VERSION_MISMATCH"),
} as const;
