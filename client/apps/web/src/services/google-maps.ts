import { api } from "@trenova/shared/lib/api";
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

  public async autocomplete(req: AutocompleteLocationRequest) {
    const response = await api.post<AutocompleteLocationResult>(`/google-maps/autocomplete/`, req);

    return safeParse(autocompleteLocationResultSchema, response, "AutocompleteLocationResult");
  }
}
