import { withCsrfHeader } from "@/lib/api";
import { API_BASE_URL } from "@/lib/constants";

type GraphQLErrorResponse = {
  message?: unknown;
};

type GraphQLResponse<TData> = {
  data?: TData;
  errors?: GraphQLErrorResponse[];
};

type GraphQLRequestParams = {
  document: string;
  operationName?: string;
  variables?: Record<string, unknown>;
};

export function resolveGraphQLURL(apiBaseURL = API_BASE_URL): string {
  if (!apiBaseURL.startsWith("http")) {
    return "/graphql";
  }

  const url = new URL(apiBaseURL);
  url.pathname = "/graphql";
  url.search = "";
  url.hash = "";

  return url.toString();
}

export async function requestGraphQL<TData>({
  document,
  operationName,
  variables,
}: GraphQLRequestParams): Promise<TData> {
  const response = await fetch(resolveGraphQLURL(), {
    body: JSON.stringify({
      query: document,
      operationName,
      variables,
    }),
    credentials: "include",
    headers: await withCsrfHeader("POST", { "Content-Type": "application/json" }, "/graphql"),
    method: "POST",
  });

  const payload = (await response.json().catch(() => ({}))) as GraphQLResponse<TData>;
  const firstError = payload.errors?.[0];
  if (firstError) {
    throw new Error(
      typeof firstError.message === "string" ? firstError.message : "GraphQL request failed",
    );
  }

  if (!response.ok) {
    throw new Error(`GraphQL request failed with HTTP ${response.status}`);
  }

  if (!payload.data) {
    throw new Error("GraphQL response did not include data");
  }

  return payload.data;
}
