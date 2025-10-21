import { API_URL, APP_ENV } from "@/constants/env";
import { APIError } from "@/types/errors";
import { generateRequestID } from "./pulid";

export type HttpMethod = "GET" | "POST" | "PUT" | "DELETE" | "PATCH";

export type HttpClientResponse<T> = {
  data: T;
  status: number;
  headers: Headers;
  statusText: string;
};

export interface RequestConfig extends Omit<RequestInit, "method" | "body"> {
  params?: Record<string, any>;
  timeout?: number;
  retries?: number;
  isFormData?: boolean;
  responseType?: "blob";
  onProgress?: (progress: number) => void;
}

type DownloadOptions = {
  filename?: string;
  responseType?: "blob";
  onProgress?: (progress: number) => void;
} & Omit<RequestConfig, "responseType">;

const DEFAULT_RETRY_COUNT = 1;
const DEFAULT_TIMEOUT = 30000;

class HttpClient {
  private static instance: HttpClient;
  private constructor() {}

  static getInstance(): HttpClient {
    if (!this.instance) {
      this.instance = new HttpClient();
    }
    return this.instance;
  }

  private async handleResponse<T>(
    response: Response,
  ): Promise<HttpClientResponse<T>> {
    if (!response.ok) {
      const error = await response
        .json()
        .catch(() => ({ error: "Unknown error" }));
      throw new APIError(
        error.error || response.statusText || "Request failed",
        response.status,
        error,
      );
    }

    const contentType = response.headers.get("content-type");
    let data: T;

    if (contentType?.includes("application/json")) {
      data = await response.json();
    } else {
      // Handle non-JSON responses (like blob or text)
      data = (await response.text()) as T;
    }

    return {
      data,
      status: response.status,
      headers: response.headers,
      statusText: response.statusText,
    };
  }

  private createAbortController(timeout?: number): {
    controller: AbortController;
    timeoutId: ReturnType<typeof setTimeout> | null;
  } {
    const controller = new AbortController();
    const timeoutId = timeout
      ? setTimeout(
          () => controller.abort(new Error("Request timeout")),
          timeout,
        )
      : null;
    return { controller, timeoutId };
  }

  private async executeWithRetry<T>(
    requestFn: () => Promise<HttpClientResponse<T>>,
    retries: number = DEFAULT_RETRY_COUNT,
  ): Promise<HttpClientResponse<T>> {
    let lastError: Error | null = null;

    for (let attempt = 0; attempt < retries; attempt++) {
      try {
        return await requestFn();
      } catch (error) {
        lastError = error as Error;
        if (!this.shouldRetry(error as Error, attempt, retries)) {
          throw error;
        }
        await this.delay(Math.pow(2, attempt) * 1000);
      }
    }

    throw lastError;
  }

  private shouldRetry(
    error: Error,
    attempt: number,
    maxRetries: number,
  ): boolean {
    const shouldNotRetyStatuCodes = [500, 404, 429];

    if (attempt >= maxRetries) return false;
    if (error instanceof APIError) {
      return (
        error.message.includes("network") &&
        !shouldNotRetyStatuCodes.includes(error.status)
      );
    }

    return true;
  }

  private delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  /**
   * Builds the URL with query parameters, filtering out undefined values
   */
  private buildUrl(
    endpoint: string,
    params?: Record<string, string | undefined>,
  ): string {
    const url = new URL(`${API_URL}${endpoint}`);

    if (params) {
      // Filter out undefined, null, and empty string values
      Object.entries(params).forEach(([key, value]) => {
        // Only append parameters that have actual values
        if (value !== undefined && value !== null && value !== "") {
          url.searchParams.append(key, value);
        }
      });
    }

    return url.toString();
  }

  private prepareRequestBody(
    data: unknown,
    isFormData: boolean,
  ): BodyInit | undefined {
    if (data === undefined) return undefined;

    if (data instanceof FormData) {
      return data;
    }

    if (isFormData && typeof data === "object") {
      const formData = new FormData();

      Object.entries(data as Record<string, any>).forEach(([key, value]) => {
        if (value instanceof File) {
          formData.append(key, value);
        } else if (Array.isArray(value)) {
          value.forEach((item, index) => {
            formData.append(`${key}[${index}]`, item);
          });
        } else {
          formData.append(key, String(value));
        }
      });
      return formData;
    }

    return JSON.stringify(data);
  }

  private getHeaders(options: RequestConfig, isFormData: boolean): HeadersInit {
    let baseHeaders: HeadersInit = {
      "X-Request-ID": generateRequestID(),
      ...options.headers,
    };

    if (!isFormData) {
      baseHeaders = {
        ...baseHeaders,
        "Content-Type": "application/json",
      };
    }

    return baseHeaders;
  }

  private async request<T>(
    method: HttpMethod,
    endpoint: string,
    data?: unknown,
    config: RequestConfig = {},
  ): Promise<HttpClientResponse<T>> {
    const {
      params,
      timeout = DEFAULT_TIMEOUT,
      retries = DEFAULT_RETRY_COUNT,
      isFormData = data instanceof FormData,
      onProgress,
      ...options
    } = config;

    const { controller, timeoutId } = this.createAbortController(timeout);

    try {
      return await this.executeWithRetry(async () => {
        const requestUrl = this.buildUrl(endpoint, params);

        if (APP_ENV === "development") {
          console.debug(
            `%c[Trenova] Facilitating HTTP ${method} Request: ${requestUrl}`,
            "color: #34ebe5; font-weight: bold",
          );
        }

        // If we have an onProgress callback and it's an upload operation, use ReadableStream
        if (
          onProgress &&
          data &&
          (method === "POST" || method === "PUT" || method === "PATCH")
        ) {
          const body = this.prepareRequestBody(data, isFormData);

          if (body && typeof body !== "string") {
            // Only files/blobs/buffers can be measured for upload progress
            const contentLength =
              body instanceof FormData
                ? await this.calculateFormDataSize(body)
                : body instanceof Blob
                  ? body.size
                  : null;

            if (contentLength) {
              let uploadedBytes = 0;

              const newBody = new ReadableStream({
                start(controller) {
                  const reader =
                    body instanceof FormData
                      ? new Response(body).body!.getReader()
                      : new Response(body as BodyInit).body!.getReader();

                  // sourcery skip: avoid-function-declarations-in-blocks
                  function push() {
                    reader
                      .read()
                      .then(({ done, value }) => {
                        if (done) {
                          controller.close();
                          return;
                        }

                        uploadedBytes += value.byteLength;
                        const progress = Math.round(
                          (uploadedBytes * 100) / (contentLength ?? 0),
                        );
                        onProgress?.(progress);

                        controller.enqueue(value);
                        push();
                      })
                      .catch((err) => {
                        controller.error(err);
                      });
                  }

                  push();
                },
              });

              const response = await fetch(requestUrl, {
                ...options,
                method,
                signal: controller.signal,
                credentials: "include",
                headers: this.getHeaders(options, isFormData),
                body: newBody as any,
                //@ts-expect-error - apparently this isn't a thing
                duplex: "half",
              });

              return this.handleResponse<T>(response);
            }
          }
        }

        // Fall back to standard fetch if we can't track progress
        const response = await fetch(requestUrl, {
          ...options,
          method,
          signal: controller.signal,
          credentials: "include",
          headers: this.getHeaders(options, isFormData),
          body: this.prepareRequestBody(data, isFormData),
        });

        return this.handleResponse<T>(response);
      }, retries);
    } finally {
      if (timeoutId) clearTimeout(timeoutId);
    }
  }

  async get<T>(
    endpoint: string,
    config?: RequestConfig,
  ): Promise<HttpClientResponse<T>> {
    return this.request<T>("GET", endpoint, undefined, config);
  }

  async post<T>(
    endpoint: string,
    data?: unknown,
    config?: RequestConfig,
  ): Promise<HttpClientResponse<T>> {
    return this.request<T>("POST", endpoint, data, config);
  }

  async put<T>(
    endpoint: string,
    data?: unknown,
    config?: RequestConfig,
  ): Promise<HttpClientResponse<T>> {
    return this.request<T>("PUT", endpoint, data, config);
  }

  async delete<T>(
    endpoint: string,
    config?: RequestConfig,
  ): Promise<HttpClientResponse<T>> {
    return this.request<T>("DELETE", endpoint, undefined, config);
  }

  async patch<T>(
    endpoint: string,
    data?: unknown,
    config?: RequestConfig,
  ): Promise<HttpClientResponse<T>> {
    return this.request<T>("PATCH", endpoint, data, config);
  }

  async downloadFile(
    endpoint: string,
    options: DownloadOptions = {},
  ): Promise<void> {
    try {
      const response = await this.get<Blob>(endpoint, {
        ...options,
        responseType: "blob",
      });

      const contentDisposition = response.headers.get("content-disposition");
      const suggestedFilename = contentDisposition
        ? contentDisposition.split("filename=")[1]?.replace(/['"]/g, "")
        : undefined;

      const blob = new Blob([response.data], {
        type:
          response.headers.get("content-type") || "application/octet-stream",
      });

      const url = window.URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = options.filename || suggestedFilename || "download";

      document.body.appendChild(link);
      link.click();

      // Cleanup
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);
    } catch (error) {
      console.error("Download failed:", error);
      throw error;
    }
  }

  // Helper method to calculate FormData size
  private async calculateFormDataSize(formData: FormData): Promise<number> {
    let size = 0;
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    for (const [_, value] of formData.entries()) {
      if (value instanceof Blob) {
        size += value.size;
      } else {
        // For string values, calculate UTF-8 byte length
        size += new TextEncoder().encode(String(value)).length;
      }

      // Add approximate overhead for multipart boundaries and headers
      size += 128; // Rough estimation for boundaries and headers per entry
    }

    return size;
  }
}

export const http = HttpClient.getInstance();
