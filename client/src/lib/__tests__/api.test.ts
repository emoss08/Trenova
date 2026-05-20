import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiRequestError, api } from "../api";
import type { ApiErrorResponse } from "@/types/errors";

function createJsonResponse(data: unknown = { ok: true }): Response {
  return new Response(JSON.stringify(data), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

class MockXMLHttpRequest {
  static instances: MockXMLHttpRequest[] = [];

  headers = new Map<string, string>();
  method = "";
  responseText = JSON.stringify({ ok: true });
  status = 200;
  upload = { addEventListener: vi.fn() };
  url = "";
  withCredentials = false;

  private listeners = new Map<string, EventListenerOrEventListenerObject[]>();

  constructor() {
    MockXMLHttpRequest.instances.push(this);
  }

  abort(): void {}

  addEventListener(type: string, listener: EventListenerOrEventListenerObject): void {
    const listeners = this.listeners.get(type) ?? [];
    listeners.push(listener);
    this.listeners.set(type, listeners);
  }

  open(method: string, url: string): void {
    this.method = method;
    this.url = url;
  }

  send(): void {
    queueMicrotask(() => {
      this.dispatch("load");
    });
  }

  setRequestHeader(name: string, value: string): void {
    this.headers.set(name.toLowerCase(), value);
  }

  private dispatch(type: string): void {
    const event = new Event(type);
    for (const listener of this.listeners.get(type) ?? []) {
      if (typeof listener === "function") {
        listener.call(this, event);
      } else {
        listener.handleEvent(event);
      }
    }
  }
}

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

describe("api csrf headers", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    document.cookie = "csrf_token=; Max-Age=0; path=/";
    MockXMLHttpRequest.instances = [];
    fetchMock = vi.fn(async () => createJsonResponse());
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    document.cookie = "csrf_token=; Max-Age=0; path=/";
    vi.unstubAllGlobals();
  });

  function headersForLastFetch(): Headers {
    const init = fetchMock.mock.calls.at(-1)?.[1] as RequestInit | undefined;
    return new Headers(init?.headers);
  }

  it("adds the CSRF token from the cookie to unsafe JSON requests", async () => {
    document.cookie = "csrf_token=unsafe-token; path=/";

    await api.post("/widgets/", { name: "Widget" });

    expect(headersForLastFetch().get("X-CSRF-Token")).toBe("unsafe-token");
  });

  it("does not add the CSRF token to safe JSON requests", async () => {
    document.cookie = "csrf_token=safe-token; path=/";

    await api.get("/widgets/");

    expect(headersForLastFetch().has("X-CSRF-Token")).toBe(false);
  });

  it("does not add an empty CSRF header when the cookie is missing", async () => {
    await api.post("/widgets/", { name: "Widget" });

    expect(headersForLastFetch().has("X-CSRF-Token")).toBe(false);
  });

  it("preserves an explicit caller-provided CSRF header", async () => {
    document.cookie = "csrf_token=cookie-token; path=/";

    await api.post(
      "/widgets/",
      { name: "Widget" },
      { headers: { "X-CSRF-Token": "explicit-token" } },
    );

    expect(headersForLastFetch().get("X-CSRF-Token")).toBe("explicit-token");
  });

  it("adds the CSRF token to internal multipart uploads", async () => {
    document.cookie = "csrf_token=upload-token; path=/";
    const formData = new FormData();
    formData.append("file", new Blob(["content"]), "document.txt");

    await api.upload("/documents/upload/", formData);

    const headers = headersForLastFetch();
    expect(headers.get("X-CSRF-Token")).toBe("upload-token");
    expect(headers.has("Content-Type")).toBe(false);
  });

  it("adds the CSRF token to internal XHR uploads", async () => {
    document.cookie = "csrf_token=progress-token; path=/";
    vi.stubGlobal("XMLHttpRequest", MockXMLHttpRequest);

    const formData = new FormData();
    formData.append("file", new Blob(["content"]), "document.txt");
    await api.uploadWithProgress("/documents/upload/", formData);

    const xhr = MockXMLHttpRequest.instances.at(-1);
    expect(xhr?.method).toBe("POST");
    expect(xhr?.url).toContain("/documents/upload/");
    expect(xhr?.withCredentials).toBe(true);
    expect(xhr?.headers.get("x-csrf-token")).toBe("progress-token");
  });

  it("does not add the CSRF token to external file upload targets", async () => {
    document.cookie = "csrf_token=external-token; path=/";
    vi.stubGlobal("XMLHttpRequest", MockXMLHttpRequest);

    await api.putFileWithProgress(
      "https://storage.example.com/upload",
      new Blob(["content"]),
      undefined,
      undefined,
      "application/pdf",
    );

    const xhr = MockXMLHttpRequest.instances.at(-1);
    expect(xhr?.method).toBe("PUT");
    expect(xhr?.url).toBe("https://storage.example.com/upload");
    expect(xhr?.withCredentials).toBe(false);
    expect(xhr?.headers.get("content-type")).toBe("application/pdf");
    expect(xhr?.headers.has("x-csrf-token")).toBe(false);
  });
});

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
