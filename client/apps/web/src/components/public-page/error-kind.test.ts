import { describe, expect, it } from "vitest";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { publicLinkErrorKind } from "./error-kind";

function apiError(status: number, type = "about:blank"): ApiRequestError {
  return new ApiRequestError(status, { type, title: "Error", status });
}

describe("publicLinkErrorKind", () => {
  it("maps 429 to throttled", () => {
    expect(publicLinkErrorKind(apiError(429))).toBe("throttled");
  });

  it("maps a 422 validation rejection to invalid", () => {
    expect(publicLinkErrorKind(apiError(422, "problem/validation-error"))).toBe("invalid");
  });

  it("maps any other definitive 4xx to invalid", () => {
    expect(publicLinkErrorKind(apiError(404))).toBe("invalid");
    expect(publicLinkErrorKind(apiError(410))).toBe("invalid");
  });

  it("treats a 5xx as a transient server problem, not a dead link", () => {
    expect(publicLinkErrorKind(apiError(500))).toBe("unavailable");
    expect(publicLinkErrorKind(apiError(503))).toBe("unavailable");
  });

  it("treats network failures and unknown errors as unavailable", () => {
    expect(publicLinkErrorKind(new Error("Failed to fetch"))).toBe("unavailable");
    expect(publicLinkErrorKind(undefined)).toBe("unavailable");
  });
});
