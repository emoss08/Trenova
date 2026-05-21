import {
  type ApiErrorResponse,
  type ProblemType,
  type ValidationError,
  apiErrorResponseSchema,
} from "@/types/errors";
import { API_BASE_URL } from "./constants";

const CSRF_HEADER_NAME =
  (import.meta.env.VITE_CSRF_HEADER_NAME as string | undefined) ?? "X-CSRF-Token";
const UNSAFE_METHODS = new Set(["POST", "PUT", "PATCH", "DELETE"]);
const CSRF_BOOTSTRAP_ENDPOINT = "/auth/csrf";
const CSRF_EXEMPT_ENDPOINTS = new Set(["/auth/login"]);

let csrfToken: string | null = null;
let csrfHeaderName = CSRF_HEADER_NAME;
let csrfTokenRequest: Promise<string | null> | null = null;

export class ApiRequestError extends Error {
  status: number;
  data: ApiErrorResponse;

  constructor(status: number, data: ApiErrorResponse) {
    super(data.detail || data.title);
    this.name = "ApiRequestError";
    this.status = status;
    this.data = data;
  }

  getProblemType(): ProblemType | null {
    if (!this.data.type) return null;
    const suffix = this.data.type.split("/").pop();
    const validTypes: ProblemType[] = [
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
    return validTypes.includes(suffix as ProblemType) ? (suffix as ProblemType) : null;
  }

  isValidationError(): boolean {
    return this.getProblemType() === "validation-error";
  }

  isBusinessError(): boolean {
    return this.getProblemType() === "business-rule-violation";
  }

  isAuthenticationError(): boolean {
    return this.getProblemType() === "authentication-error";
  }

  isAuthorizationError(): boolean {
    return this.getProblemType() === "authorization-error";
  }

  isNotFoundError(): boolean {
    return this.getProblemType() === "resource-not-found";
  }

  isRateLimitError(): boolean {
    return this.status === 429 || this.getProblemType() === "rate-limit-exceeded";
  }

  isVersionMismatchError(): boolean {
    return (
      this.isValidationError() && this.getFieldErrors().some((e) => e.code === "VERSION_MISMATCH")
    );
  }

  isConflictError(): boolean {
    return this.status === 409 || this.getProblemType() === "resource-conflict";
  }

  getUsageStats(): unknown {
    return this.data.usageStats;
  }

  getFieldErrors(): ValidationError[] {
    return this.data.errors || [];
  }

  getFieldError(field: string): ValidationError | undefined {
    return this.getFieldErrors().find((e) => e.field === field);
  }

  getParams(): Record<string, string> {
    return this.data.params ?? {};
  }
}

function isUnsafeMethod(method: string | undefined): boolean {
  return UNSAFE_METHODS.has((method ?? "GET").toUpperCase());
}

export function setCsrfToken(token: string | null | undefined): void {
  csrfToken = token?.trim() || null;
  if (csrfToken) {
    csrfTokenRequest = null;
  }
}

export function clearCsrfToken(): void {
  csrfToken = null;
  csrfTokenRequest = null;
  csrfHeaderName = CSRF_HEADER_NAME;
}

async function fetchCsrfToken(): Promise<string | null> {
  if (csrfToken) {
    return csrfToken;
  }

  csrfTokenRequest ??= fetch(`${API_BASE_URL}${CSRF_BOOTSTRAP_ENDPOINT}`, {
    credentials: "include",
    headers: { Accept: "application/json" },
  })
    .then(async (response) => {
      if (!response.ok) {
        return null;
      }

      const data = (await response.json()) as {
        csrfToken?: unknown;
        headerName?: unknown;
      };
      if (typeof data.headerName === "string" && data.headerName.trim()) {
        csrfHeaderName = data.headerName;
      }
      if (typeof data.csrfToken !== "string" || !data.csrfToken.trim()) {
        return null;
      }

      csrfToken = data.csrfToken;
      return csrfToken;
    })
    .catch(() => null)
    .finally(() => {
      csrfTokenRequest = null;
    });

  return csrfTokenRequest;
}

function shouldBootstrapCsrf(endpoint: string | undefined): boolean {
  if (!endpoint) {
    return true;
  }

  const path = endpoint.split("?")[0] ?? endpoint;
  return !CSRF_EXEMPT_ENDPOINTS.has(path) && path !== CSRF_BOOTSTRAP_ENDPOINT;
}

export async function withCsrfHeader(
  method: string | undefined,
  headers?: HeadersInit,
  endpoint?: string,
): Promise<Headers> {
  const nextHeaders = new Headers(headers);

  if (
    !isUnsafeMethod(method) ||
    nextHeaders.has(csrfHeaderName) ||
    !shouldBootstrapCsrf(endpoint)
  ) {
    return nextHeaders;
  }

  const token = csrfToken ?? (await fetchCsrfToken());
  if (token) {
    nextHeaders.set(csrfHeaderName, token);
  }

  return nextHeaders;
}

async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const url = `${API_BASE_URL}${endpoint}`;
  const method = options.method ?? "GET";
  const headers = new Headers(options.headers);

  if (!headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(url, {
    ...options,
    credentials: "include",
    headers: await withCsrfHeader(method, headers, endpoint),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({
      type: "internal-error",
      title: "Request failed",
      detail: `HTTP ${response.status}`,
      status: response.status,
    }));
    const parsed = apiErrorResponseSchema.safeParse(errorData);
    const validatedData = parsed.success ? parsed.data : errorData;
    throw new ApiRequestError(response.status, validatedData);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json();
}

async function uploadRequest<T>(
  endpoint: string,
  formData: FormData,
  options: RequestInit = {},
): Promise<T> {
  const url = `${API_BASE_URL}${endpoint}`;
  const method = "POST";

  const response = await fetch(url, {
    ...options,
    method,
    credentials: "include",
    body: formData,
    headers: await withCsrfHeader(method, options.headers, endpoint),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({
      type: "internal-error",
      title: "Upload failed",
      detail: `HTTP ${response.status}`,
      status: response.status,
    }));
    const parsed = apiErrorResponseSchema.safeParse(errorData);
    const validatedData = parsed.success ? parsed.data : errorData;
    throw new ApiRequestError(response.status, validatedData);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json();
}

async function uploadWithProgress<T>(
  endpoint: string,
  formData: FormData,
  onProgress?: (percent: number) => void,
  signal?: AbortSignal,
): Promise<T> {
  const csrfHeaders = await withCsrfHeader("POST", undefined, endpoint);

  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    const url = `${API_BASE_URL}${endpoint}`;

    if (signal) {
      signal.addEventListener("abort", () => {
        xhr.abort();
        reject(new DOMException("Upload aborted", "AbortError"));
      });
    }

    xhr.upload.addEventListener("progress", (event) => {
      if (event.lengthComputable && onProgress) {
        const percent = Math.round((event.loaded / event.total) * 100);
        onProgress(percent);
      }
    });

    xhr.addEventListener("load", () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        if (xhr.status === 204) {
          resolve(undefined as T);
          return;
        }
        try {
          const response = JSON.parse(xhr.responseText);
          resolve(response);
        } catch {
          resolve(xhr.responseText as T);
        }
      } else {
        let errorData;
        try {
          errorData = JSON.parse(xhr.responseText);
        } catch {
          errorData = {
            type: "internal-error",
            title: "Upload failed",
            detail: `HTTP ${xhr.status}`,
            status: xhr.status,
          };
        }
        const parsed = apiErrorResponseSchema.safeParse(errorData);
        const validatedData = parsed.success ? parsed.data : errorData;
        reject(new ApiRequestError(xhr.status, validatedData));
      }
    });

    xhr.addEventListener("error", () => {
      reject(
        new ApiRequestError(0, {
          type: "internal-error",
          title: "Network error",
          detail: "Failed to connect to server",
          status: 0,
        }),
      );
    });

    xhr.open("POST", url);
    xhr.withCredentials = true;
    csrfHeaders.forEach((value, key) => {
      xhr.setRequestHeader(key, value);
    });
    xhr.send(formData);
  });
}

function putFileWithProgress(
  url: string,
  file: Blob,
  onProgress?: (percent: number) => void,
  signal?: AbortSignal,
  contentType?: string,
): Promise<void> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();

    if (signal) {
      signal.addEventListener("abort", () => {
        xhr.abort();
        reject(new DOMException("Upload aborted", "AbortError"));
      });
    }

    xhr.upload.addEventListener("progress", (event) => {
      if (event.lengthComputable && onProgress) {
        const percent = Math.round((event.loaded / event.total) * 100);
        onProgress(percent);
      }
    });

    xhr.addEventListener("load", () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve();
        return;
      }

      reject(
        new ApiRequestError(xhr.status, {
          type: "internal-error",
          title: "Upload failed",
          detail: `HTTP ${xhr.status}`,
          status: xhr.status,
        }),
      );
    });

    xhr.addEventListener("error", () => {
      reject(
        new ApiRequestError(0, {
          type: "internal-error",
          title: "Network error",
          detail: "Failed to connect to upload target",
          status: 0,
        }),
      );
    });

    xhr.open("PUT", url);
    if (contentType) {
      xhr.setRequestHeader("Content-Type", contentType);
    }
    xhr.send(file);
  });
}

export const api = {
  get: async <T>(endpoint: string, options?: RequestInit) =>
    request<T>(endpoint, { ...options, method: "GET" }),

  post: async <T>(endpoint: string, data?: unknown, options?: RequestInit) =>
    request<T>(endpoint, {
      ...options,
      method: "POST",
      body: data ? JSON.stringify(data) : undefined,
    }),

  put: async <T>(endpoint: string, data?: unknown, options?: RequestInit) =>
    request<T>(endpoint, {
      ...options,
      method: "PUT",
      body: data ? JSON.stringify(data) : undefined,
    }),

  patch: async <T>(endpoint: string, data?: unknown, options?: RequestInit) =>
    request<T>(endpoint, {
      ...options,
      method: "PATCH",
      body: data ? JSON.stringify(data) : undefined,
    }),

  delete: async <T>(endpoint: string, options?: RequestInit) =>
    request<T>(endpoint, { ...options, method: "DELETE" }),

  upload: async <T>(endpoint: string, formData: FormData, options?: RequestInit) =>
    uploadRequest<T>(endpoint, formData, options),

  uploadWithProgress: <T>(
    endpoint: string,
    formData: FormData,
    onProgress?: (percent: number) => void,
    signal?: AbortSignal,
  ) => uploadWithProgress<T>(endpoint, formData, onProgress, signal),

  putFileWithProgress: (
    url: string,
    file: Blob,
    onProgress?: (percent: number) => void,
    signal?: AbortSignal,
    contentType?: string,
  ) => putFileWithProgress(url, file, onProgress, signal, contentType),
};
