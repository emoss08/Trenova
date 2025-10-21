import { http } from "@/lib/http-client";
import {
  AutocompleteLocationResultSchema,
  GetApiKeyResponseSchema,
} from "@/lib/schemas/geocode-schema";

export class GoogleMapsAPI {
  async locationAutocomplete(input: string) {
    return http.post<AutocompleteLocationResultSchema>(
      `/google-maps/autocomplete/`,
      {
        input,
      },
    );
  }

  async getAPIKey() {
    return (await http.get<GetApiKeyResponseSchema>(`/google-maps/api-key/`))
      .data;
  }
}
