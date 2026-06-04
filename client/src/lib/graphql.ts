import { withCsrfHeader } from "@/lib/api";
import { API_BASE_URL } from "@/lib/constants";
import type { GraphQLExecutableDocument, TypedGraphQLDocument } from "@/types/graphql";
import type { ResultOf, VariablesOf } from "@graphql-typed-document-node/core";

type GraphQLErrorResponse = {
  message?: unknown;
};

type GraphQLResponse<TData> = {
  data?: TData;
  errors?: GraphQLErrorResponse[];
};

type GraphQLRequestBody<TVariables> = {
  extensions?: {
    persistedQuery: {
      sha256Hash: string;
      version: 1;
    };
  };
  operationName?: string;
  query: string;
  variables?: TVariables;
};

type GraphQLRequestParams<TVariables = Record<string, unknown>> = {
  document: GraphQLExecutableDocument;
  operationName?: string;
  variables?: TVariables;
};

type TypedGraphQLRequestParams<TDocument extends TypedGraphQLDocument<unknown, never>> = {
  document: TDocument;
  operationName?: string;
  variables?: VariablesOf<TDocument>;
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

export async function requestGraphQL<
  TDocument extends TypedGraphQLDocument<unknown, never>,
>(
  params: TypedGraphQLRequestParams<TDocument>,
): Promise<ResultOf<TDocument>>;
export async function requestGraphQL<
  TData,
  TVariables = Record<string, unknown>,
>(params: GraphQLRequestParams<TVariables>): Promise<TData>;
export async function requestGraphQL<
  TData,
  TVariables = Record<string, unknown>,
>({
  document,
  operationName,
  variables,
}: GraphQLRequestParams<TVariables>): Promise<TData> {
  const body: GraphQLRequestBody<TVariables> = {
    query: document.toString(),
    operationName,
    variables,
  };
  const persistedHash = typeof document === "string" ? undefined : document.__meta__?.hash;
  if (persistedHash) {
    body.extensions = {
      persistedQuery: {
        version: 1,
        sha256Hash: persistedHash,
      },
    };
  }

  const response = await fetch(resolveGraphQLURL(), {
    body: JSON.stringify(body),
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
