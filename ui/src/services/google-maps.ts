import { http } from "@/lib/http-client";
import type {
  AutoCompleteLocationResult,
  CheckAPIKeyResponse,
} from "@/types/google-maps";

export async function locationAutocomplete(input: string) {
  return http.post<AutoCompleteLocationResult>(`/google-maps/autocomplete/`, {
    input,
  });
}

export async function checkAPIKey() {
  return http.get<CheckAPIKeyResponse>(`/google-maps/check-api-key/`);
}
