/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { http } from "@/lib/http-client";

export class BillingAPI {
  async bulkTransfer() {
    const response = await http.post(`/billing-queue/bulk-transfer`);
    return response.data as Promise<void>;
  }
}
