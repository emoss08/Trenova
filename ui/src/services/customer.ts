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
