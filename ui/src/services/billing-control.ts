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
