/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { http } from "@/lib/http-client";
import type { BillingControlSchema } from "@/lib/schemas/billing-schema";

export class BillingControlAPI {
  async get() {
    const response = await http.get<BillingControlSchema>("/billing-controls/");
    return response.data;
  }

  async update(data: BillingControlSchema) {
    const response = await http.put<BillingControlSchema>(
      "/billing-controls/",
      data,
    );
    return response.data;
  }
}
