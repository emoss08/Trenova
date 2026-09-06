import {
  fetchEIASeriesOptions,
  fetchFuelDashboard,
  fetchFuelPriceHistory,
  fetchFuelProgramCurrentRates,
  fetchFuelSurchargeProgramDetail,
  generateFuelSurchargeTable,
} from "@/lib/graphql/fuel-surcharge";
import type { GenerateFuelTableInput } from "@trenova/graphql/generated/graphql";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const fuelSurcharge = createQueryKeys("fuelSurcharge", {
  dashboard: () => ({
    queryKey: ["dashboard"],
    queryFn: ({ signal }) => fetchFuelDashboard({ signal }),
  }),
  priceHistory: (indexId: string, limit?: number) => ({
    queryKey: ["priceHistory", indexId, limit ?? 0],
    queryFn: async ({ signal }) => fetchFuelPriceHistory(indexId, { limit }, { signal }),
  }),
  currentRates: () => ({
    queryKey: ["currentRates"],
    queryFn: ({ signal }) => fetchFuelProgramCurrentRates({ signal }),
  }),
  programDetail: (id: string) => ({
    queryKey: ["programDetail", id],
    queryFn: async ({ signal }) => fetchFuelSurchargeProgramDetail(id, { signal }),
  }),
  eiaSeriesOptions: () => ({
    queryKey: ["eiaSeriesOptions"],
    queryFn: ({ signal }) => fetchEIASeriesOptions({ signal }),
  }),
  generateTable: (input: GenerateFuelTableInput) => ({
    queryKey: ["generateTable", input],
    queryFn: async ({ signal }) => generateFuelSurchargeTable(input, { signal }),
  }),
});
