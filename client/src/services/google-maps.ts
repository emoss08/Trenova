import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  type AutocompleteLocationRequest,
  type AutocompleteLocationResult,
  autocompleteLocationResultSchema,
} from "@/types/google-maps";

export class GoogleMapsService {
  public async getAPIKey() {
    return api.get<Record<string, string>>("/integrations/GoogleMaps/runtime-config/");
  }

  public async autocomplete(req: AutocompleteLocationRequest) {
    const response = await api.post<AutocompleteLocationResult>(
      `/google-maps/autocomplete/`,
      req,
    );

    return safeParse(autocompleteLocationResultSchema, response, "AutocompleteLocationResult");
  }
}
