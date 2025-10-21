/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

export type ValidationPriority = "HIGH" | "MEDIUM" | "LOW";

export interface InvalidParam {
  name: string;
  reason: string;
  code: string;
  priority?: ValidationPriority;
  location?:
    | "body"
    | "query"
    | "path"
    | "header"
    | "authentication"
    | "authorization"
    | "business"
    | string;
}

export interface ApiErrorResponse {
  type:
    | "validation-error"
    | "business-error"
    | "authentication-error"
    | "authorization-error"
    | "not-found-error"
    | "internal-server-error";
  title: string;
  status: number;
  detail: string;
  instance: string;
  invalidParams: InvalidParam[];
  traceId?: string;
}

export type FieldErrors<T extends Record<string, unknown>> = Partial<
  Record<keyof T, string>
>;

export class APIError extends Error {
  constructor(
    message: string,
    public status: number,
    public data?: ApiErrorResponse,
  ) {
    super(message);
    this.name = "APIError";
  }

  isValidationError(): boolean {
    return this.data?.type === "validation-error";
  }

  isVersionMismatchError(): boolean {
    return (
      this.data?.type === "validation-error" &&
      this.data?.invalidParams.some(
        (param) => param.code === "VERSION_MISMATCH",
      )
    );
  }

  isRateLimitError(): boolean {
    return this.status === 429;
  }

  isBusinessError(): boolean {
    return this.data?.type === "business-error";
  }

  isAuthenticationError(): boolean {
    return this.data?.type === "authentication-error";
  }

  isAuthorizationError(): boolean {
    return this.data?.type === "authorization-error";
  }

  getFieldErrors(): InvalidParam[] {
    return this.data?.invalidParams || [];
  }

  getFieldErrorsByPriority(priority: ValidationPriority): InvalidParam[] {
    return this.getFieldErrors().filter((error) => error.priority === priority);
  }

  hasHighPriorityErrors(): boolean {
    return this.getFieldErrors().some((error) => error.priority === "HIGH");
  }

  hasMediumPriorityErrors(): boolean {
    return this.getFieldErrors().some((error) => error.priority === "MEDIUM");
  }

  hasOnlyLowPriorityErrors(): boolean {
    const errors = this.getFieldErrors();
    return (
      errors.length > 0 &&
      errors.every((error) => error.priority === "LOW" || !error.priority)
    );
  }

  hasBlockingErrors(): boolean {
    return this.getFieldErrors().some(
      (error) => error.priority === "HIGH" || error.priority === "MEDIUM",
    );
  }
}
