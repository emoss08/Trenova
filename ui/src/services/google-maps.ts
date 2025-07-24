/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
