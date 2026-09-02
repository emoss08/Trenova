import { ApiRequestError } from "@trenova/shared/lib/api";
import { describe, expect, it } from "vitest";
import { describeApiError } from "../api-error-message";

describe("describeApiError", () => {
  it("joins field-level validation messages", () => {
    const error = new ApiRequestError(400, {
      type: "https://trenova.app/problems/validation-error",
      title: "Validation failed",
      status: 400,
      errors: [
        { field: "approvedById", code: "INVALID", message: "You cannot approve your own template" },
        { field: "testCases", code: "INVALID", message: "2 scenarios fail" },
      ],
    });

    expect(describeApiError(error)).toBe("You cannot approve your own template 2 scenarios fail");
  });

  it("falls back to the problem detail", () => {
    const error = new ApiRequestError(409, {
      type: "https://trenova.app/problems/conflict",
      title: "Conflict",
      status: 409,
      detail: "The template was modified by someone else",
      errors: [],
    });

    expect(describeApiError(error)).toBe("The template was modified by someone else");
  });

  it("uses a plain error message or the fallback", () => {
    expect(describeApiError(new Error("network down"))).toBe("network down");
    expect(describeApiError("??", "nope")).toBe("nope");
  });
});
