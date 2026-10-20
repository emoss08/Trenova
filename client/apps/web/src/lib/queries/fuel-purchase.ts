import { fetchDeskFuelPurchase } from "@/lib/graphql/fuel-desk-cards";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const fuelPurchase = createQueryKeys("fuelPurchase", {
  card: (id: string) => ({
    queryKey: ["card", id],
    queryFn: async ({ signal }) => fetchDeskFuelPurchase(id, { signal }),
  }),
});
