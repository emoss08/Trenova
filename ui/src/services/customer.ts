import { http } from "@/lib/http-client";
import { type CustomerDocumentRequirement } from "@/types/customer";

export async function getCustomerDocumentRequirements(
  customerId: string,
): Promise<CustomerDocumentRequirement[]> {
  const response = await http.get<CustomerDocumentRequirement[]>(
    `/customers/${customerId}/document-requirements`,
  );
  return response.data;
}
