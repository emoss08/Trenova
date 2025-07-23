/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { http } from "@/lib/http-client";
import type { PatternConfigSchema } from "@/lib/schemas/pattern-config-schema";

export class PatternConfigAPI {
  async get() {
    const response = await http.get<PatternConfigSchema>("/pattern-config/");
    return response.data;
  }

  async update(data: PatternConfigSchema) {
    const response = await http.put<PatternConfigSchema>(
      `/pattern-config/`,
      data,
    );
    return response.data;
  }
}
