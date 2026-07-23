import { fetchCustomerPaymentDetail } from "@/lib/graphql/customer-payment";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const customerPayment = createQueryKeys("customerPayment", {
  detail: (id: string) => ({
    queryKey: ["detail", id],
    queryFn: async () => fetchCustomerPaymentDetail(id),
  }),
});
