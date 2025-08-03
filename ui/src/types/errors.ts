/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

export interface InvalidParam {
  name: string;
  reason: string;
  code: string;
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
}
