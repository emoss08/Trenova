/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { http } from "@/lib/http-client";
import type { UsStateSchema } from "@/lib/schemas/us-state-schema";
import { SelectOptionResponse } from "@/types/server";

export class UsStateAPI {
  // Get all US states
  async getUsStates() {
    const response = await http.get<UsStateSchema[]>("/us-states");
    return response.data;
  }
  // Get US state select options
  async getUsStateOptions() {
    const response = await http.get<SelectOptionResponse>(
      "/us-states/select-options",
    );
    return response.data;
  }
}
