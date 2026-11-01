import { PlaceAutocompleteDocument } from "@trenova/graphql/generated/graphql";
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
  public async getAPIKey(): Promise<IntegrationRuntimeConfigResponse> {
    const response = await api.get("/integrations/GoogleMaps/runtime-config/");
    return safeParse(
      integrationRuntimeConfigResponseSchema,
      response,
      "Google Maps Runtime Config",
    );
  }

  public async autocomplete(
    req: AutocompleteLocationRequest,
    options?: { signal?: AbortSignal },
  ): Promise<AutocompleteLocationResult> {
    const response = await requestGraphQL({
      document: PlaceAutocompleteDocument,
      operationName: "PlaceAutocomplete",
      variables: { input: { input: req.input, sessionToken: req.sessionToken ?? null } },
      signal: options?.signal,
    });

    return safeParse(
      autocompleteLocationResultSchema,
      response.placeAutocomplete,
      "AutocompleteLocationResult",
    );
  }
}
