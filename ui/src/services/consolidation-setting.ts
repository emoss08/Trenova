/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { http } from "@/lib/http-client";
import { ConsolidationSettingSchema } from "@/lib/schemas/consolidation-setting-schema";

export class ConsolidationSettingsAPI {
  async get() {
    const response = await http.get<ConsolidationSettingSchema>(
      "/consolidation-settings/",
    );
    return response.data;
  }

  async update(data: ConsolidationSettingSchema) {
    const response = await http.put<ConsolidationSettingSchema>(
      "/consolidation-settings/",
      data,
    );

    return response.data;
  }
}
