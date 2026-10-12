import { fetchDeskOrder } from "@/lib/graphql/order-card";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const order = createQueryKeys("order", {
  card: (orderId: string) => ({
    queryKey: ["card", orderId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) => fetchDeskOrder(orderId, { signal }),
  }),
});
