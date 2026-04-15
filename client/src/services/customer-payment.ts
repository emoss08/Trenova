import { api } from "@/lib/api";
import type { CustomerPayment } from "@/types/customer-payment";

export class CustomerPaymentService {
  async getById(id: string) {
    return api.get<CustomerPayment>(`/accounting/customer-payments/${id}/`);
  }
}
