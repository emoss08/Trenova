import { http } from "@/lib/http-client";
import type {
  AutoCompleteLocationResult,
  CheckAPIKeyResponse,
} from "@/types/google-maps";

export class GoogleMapsAPI {
  async locationAutocomplete(input: string) {
    return http.post<AutoCompleteLocationResult>(
      `/integrations/google-maps/autocomplete/`,
      {
        input,
      },
    );
  }

  async checkAPIKey() {
    return http.get<CheckAPIKeyResponse>(`/google-maps/check-api-key/`);
  }
}
