import { describe, expect, it } from "vitest";
import { ApiRequestError } from "../api";
import type { ApiErrorResponse } from "@/types/errors";

function makeError(
  overrides: Partial<ApiErrorResponse> & { status?: number } = {},
): ApiRequestError {
  const { status = 400, ...dataOverrides } = overrides;
  const data: ApiErrorResponse = {
    type: "https://example.com/problems/validation-error",
    title: "Validation Error",
    status,
    ...dataOverrides,
  };
  return new ApiRequestError(status, data);
}

describe("ApiRequestError constructor", () => {
  it("uses detail as message when present", () => {
    const err = makeError({ detail: "Something went wrong" });
    expect(err.message).toBe("Something went wrong");
  });

  it("falls back to title when detail is missing", () => {
    const err = makeError({ detail: undefined });
    expect(err.message).toBe("Validation Error");
  });

  it("sets status and data", () => {
    const err = makeError({ status: 422 });
    expect(err.status).toBe(422);
    expect(err.data.title).toBe("Validation Error");
  });
});

describe("getProblemType", () => {
  it("extracts suffix from URL type", () => {
    const err = makeError({
      type: "https://example.com/problems/validation-error",
    });
    expect(err.getProblemType()).toBe("validation-error");
  });

  it("returns null for unknown suffix", () => {
    const err = makeError({
      type: "https://example.com/problems/unknown-type",
    });
    expect(err.getProblemType()).toBeNull();
  });

  it("returns null when type is missing", () => {
    const err = makeError({ type: undefined as any });
    expect(err.getProblemType()).toBeNull();
  });

  it("recognizes all 9 valid problem types", () => {
    const validTypes = [
      "validation-error",
      "business-rule-violation",
      "database-error",
      "authentication-error",
      "authorization-error",
      "resource-not-found",
      "rate-limit-exceeded",
      "resource-conflict",
      "internal-error",
    ];
    for (const t of validTypes) {
      const err = makeError({ type: `https://api.trenova.app/problems/${t}` });
      expect(err.getProblemType()).toBe(t);
    }
  });

  it("recognizes resource-conflict", () => {
    const err = makeError({
      type: "https://api.trenova.app/problems/resource-conflict",
    });
    expect(err.getProblemType()).toBe("resource-conflict");
  });
});

describe("boolean type checks", () => {
  it("isValidationError returns true for validation-error", () => {
    const err = makeError({
      type: "https://x.com/problems/validation-error",
    });
    expect(err.isValidationError()).toBe(true);
  });

  it("isBusinessError returns true for business-rule-violation", () => {
    const err = makeError({
      type: "https://x.com/problems/business-rule-violation",
    });
    expect(err.isBusinessError()).toBe(true);
  });

  it("isAuthenticationError returns true for authentication-error", () => {
    const err = makeError({
      type: "https://x.com/problems/authentication-error",
    });
    expect(err.isAuthenticationError()).toBe(true);
  });

  it("isAuthorizationError returns true for authorization-error", () => {
    const err = makeError({
      type: "https://x.com/problems/authorization-error",
    });
    expect(err.isAuthorizationError()).toBe(true);
  });

  it("isNotFoundError returns true for resource-not-found", () => {
    const err = makeError({
      type: "https://x.com/problems/resource-not-found",
    });
    expect(err.isNotFoundError()).toBe(true);
  });
});

describe("isRateLimitError", () => {
  it("returns true for status 429", () => {
    const err = makeError({
      status: 429,
      type: "https://x.com/problems/internal-error",
    });
    expect(err.isRateLimitError()).toBe(true);
  });

  it("returns true for rate-limit-exceeded problem type", () => {
    const err = makeError({
      status: 400,
      type: "https://x.com/problems/rate-limit-exceeded",
    });
    expect(err.isRateLimitError()).toBe(true);
  });

  it("returns false when neither condition matches", () => {
    const err = makeError({
      status: 400,
      type: "https://x.com/problems/validation-error",
    });
    expect(err.isRateLimitError()).toBe(false);
  });
});

describe("isVersionMismatchError", () => {
  it("returns true for validation error with VERSION_MISMATCH code", () => {
    const err = makeError({
      type: "https://x.com/problems/validation-error",
      errors: [
        { field: "version", message: "Version mismatch", code: "VERSION_MISMATCH" },
      ],
    });
    expect(err.isVersionMismatchError()).toBe(true);
  });

  it("returns false without VERSION_MISMATCH code", () => {
    const err = makeError({
      type: "https://x.com/problems/validation-error",
      errors: [{ field: "name", message: "Required" }],
    });
    expect(err.isVersionMismatchError()).toBe(false);
  });

  it("returns false for non-validation error", () => {
    const err = makeError({
      type: "https://x.com/problems/internal-error",
      errors: [
        { field: "version", message: "Mismatch", code: "VERSION_MISMATCH" },
      ],
    });
    expect(err.isVersionMismatchError()).toBe(false);
  });
});

describe("isConflictError", () => {
  it("returns true for status 409", () => {
    const err = makeError({ status: 409 });
    expect(err.isConflictError()).toBe(true);
  });

  it("returns false for non-409 status without resource-conflict type", () => {
    const err = makeError({ status: 200 });
    expect(err.isConflictError()).toBe(false);
  });

  it("returns true for resource-conflict problem type", () => {
    const err = makeError({
      status: 200,
      type: "https://x.com/problems/resource-conflict",
    });
    expect(err.isConflictError()).toBe(true);
  });
});

describe("getFieldErrors", () => {
  it("returns errors array", () => {
    const err = makeError({
      errors: [{ field: "name", message: "Required" }],
    });
    expect(err.getFieldErrors()).toEqual([
      { field: "name", message: "Required" },
    ]);
  });

  it("defaults to empty array", () => {
    const err = makeError({ errors: undefined });
    expect(err.getFieldErrors()).toEqual([]);
  });
});

describe("getFieldError", () => {
  it("finds error by field name", () => {
    const err = makeError({
      errors: [
        { field: "name", message: "Required" },
        { field: "email", message: "Invalid" },
      ],
    });
    expect(err.getFieldError("email")).toEqual({
      field: "email",
      message: "Invalid",
    });
  });

  it("returns undefined when field not found", () => {
    const err = makeError({
      errors: [{ field: "name", message: "Required" }],
    });
    expect(err.getFieldError("missing")).toBeUndefined();
  });
});

describe("getParams", () => {
  it("returns params object", () => {
    const err = makeError({ params: { key: "value" } });
    expect(err.getParams()).toEqual({ key: "value" });
  });

  it("defaults to empty object", () => {
    const err = makeError({ params: undefined });
    expect(err.getParams()).toEqual({});
  });
});
