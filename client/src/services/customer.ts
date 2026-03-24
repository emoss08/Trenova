import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  bulkUpdateCustomerStatusResponseSchema,
  customerBillingProfileSchema,
  customerSchema,
  type BulkUpdateCustomerStatusRequest,
  type BulkUpdateCustomerStatusResponse,
  type Customer,
  type CustomerBillingProfile,
} from "@/types/customer";

export class CustomerService {
  public async bulkUpdateStatus(request: BulkUpdateCustomerStatusRequest) {
    const response = await api.post<BulkUpdateCustomerStatusResponse>(
      "/customers/bulk-update-status/",
      request,
    );

    return safeParse(
      bulkUpdateCustomerStatusResponseSchema,
      response,
      "Bulk Update Customer Status",
    );
  }

  public async patch(customerId: Customer["id"], data: Partial<Customer>) {
    const response = await api.patch<Customer>(`/customers/${customerId}/`, data);

    return safeParse(customerSchema, response, "Customer");
  }

  public async getBillingProfile(customerId: Customer["id"]) {
    const response = await api.get<CustomerBillingProfile>(
      `/customers/${customerId}/billing-profile/`,
    );

    return safeParse(customerBillingProfileSchema, response, "Get Customer Billing Profile");
  }
}
