/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { http } from "@/lib/http-client";
import { CustomerSchema } from "@/lib/schemas/customer-schema";
import { type CustomerDocumentRequirement } from "@/types/customer";

export class CustomerAPI {
  // Get customer document requirements
  async getDocumentRequirements(customerId: string) {
    const response = await http.get<CustomerDocumentRequirement[]>(
      `/customers/${customerId}/document-requirements`,
    );
    return response.data;
  }

  // Get customer by ID
  async getById(
    customerId: string,
    { includeBillingProfile = false }: { includeBillingProfile?: boolean },
  ) {
    const response = await http.get<CustomerSchema>(
      `/customers/${customerId}`,
      {
        params: { includeBillingProfile: includeBillingProfile.toString() },
      },
    );
    return response.data;
  }
}
