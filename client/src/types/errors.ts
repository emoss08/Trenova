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
