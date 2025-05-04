import { http } from "@/lib/http-client";
import { CustomerSchema } from "@/lib/schemas/customer-schema";
import { type CustomerDocumentRequirement } from "@/types/customer";

export async function getCustomerDocumentRequirements(
  customerId: string,
): Promise<CustomerDocumentRequirement[]> {
  const response = await http.get<CustomerDocumentRequirement[]>(
    `/customers/${customerId}/document-requirements`,
  );
  return response.data;
}

export async function getCustomerById(
  customerId: string,
  { includeBillingProfile = false }: { includeBillingProfile?: boolean },
): Promise<CustomerSchema> {
  const response = await http.get<CustomerSchema>(`/customers/${customerId}`, {
    params: {
      includeBillingProfile: includeBillingProfile.toString(),
    },
  });
  return response.data;
}
