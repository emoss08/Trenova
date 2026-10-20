import { fetchDeskFuelCard } from "@/lib/graphql/fuel-desk-cards";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const fuelCard = createQueryKeys("fuelCard", {
  card: (id: string) => ({
    queryKey: ["card", id],
    queryFn: async ({ signal }) => fetchDeskFuelCard(id, { signal }),
  }),
});
