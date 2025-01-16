import { API_URL } from "@/constants/env";
import { APIError } from "@/types/errors";

export type HttpMethod = "GET" | "POST" | "PUT" | "DELETE" | "PATCH";

export type HttpClientResponse<T> = {
  data: T;
  status: number;
  headers: Headers;
  statusText: string;
};

export interface RequestConfig extends Omit<RequestInit, "method" | "body"> {
  params?: Record<string, string>;
  timeout?: number;
  retries?: number;
  isFormData?: boolean;
}

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
    timeoutId: NodeJS.Timeout | null;
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
    if (attempt >= maxRetries) return false;
    if (error instanceof APIError) {
      return error.status >= 500 || error.message.includes("network");
    }
    return true;
  }

  private delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  private buildUrl(endpoint: string, params?: Record<string, string>): string {
    const url = new URL(`${API_URL}${endpoint}`);
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        url.searchParams.append(key, value);
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
      "X-Request-ID": crypto.randomUUID(),
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
      ...options
    } = config;

    const { controller, timeoutId } = this.createAbortController(timeout);

    try {
      return await this.executeWithRetry(async () => {
        console.debug(
          `%c[Trenova] HTTP ${method} Request: ${endpoint}`,
          "color: #34ebe5; font-weight: bold",
        );

        const response = await fetch(this.buildUrl(endpoint, params), {
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
}

export const http = HttpClient.getInstance();
