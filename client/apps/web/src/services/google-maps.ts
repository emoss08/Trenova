import { IntegrationRuntimeConfigDocument } from "@trenova/graphql/generated/graphql";
import { api } from "@trenova/shared/lib/api";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  integrationRuntimeConfigResponseSchema,
  type IntegrationRuntimeConfigResponse,
} from "@/types/integration";
import {
  type AutocompleteLocationRequest,
  type AutocompleteLocationResult,
  autocompleteLocationResultSchema,
} from "@/types/google-maps";

export class GoogleMapsService {
  public async getAPIKey(
    options: { signal?: AbortSignal } = {},
  ): Promise<IntegrationRuntimeConfigResponse> {
    const data = await requestGraphQL({
      document: IntegrationRuntimeConfigDocument,
      operationName: "IntegrationRuntimeConfig",
      variables: { type: "GoogleMaps" },
      signal: options.signal,
    });
    return safeParse(
      integrationRuntimeConfigResponseSchema,
      data.integrationRuntimeConfig,
      "Google Maps Runtime Config",
    );
  }

  public async autocomplete(req: AutocompleteLocationRequest) {
    const response = await api.post<AutocompleteLocationResult>(`/google-maps/autocomplete/`, req);

    return safeParse(autocompleteLocationResultSchema, response, "AutocompleteLocationResult");
  }
}
