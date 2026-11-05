import { billingQueueService } from "@/services/billing-queue";
import { createQueryKeys } from "@lukemorales/query-key-factory";

/**
 * Nova's bulk generation for the billing queue. It has a root of its own so the
 * queue's many item changes do not refetch it; its realtime resource,
 * billing-queue-nova, does.
 */
export const billingQueueNova = createQueryKeys("billingQueueNova", {
  generation: () => ({
    queryKey: ["generation"],
    queryFn: async ({ signal }) => billingQueueService.getNovaGeneration({ signal }),
  }),
});
