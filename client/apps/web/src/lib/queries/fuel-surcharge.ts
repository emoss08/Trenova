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
    queryFn: fetchFuelDashboard,
  }),
  priceHistory: (indexId: string, limit?: number) => ({
    queryKey: ["priceHistory", indexId, limit ?? 0],
    queryFn: async () => fetchFuelPriceHistory(indexId, { limit }),
  }),
  currentRates: () => ({
    queryKey: ["currentRates"],
    queryFn: fetchFuelProgramCurrentRates,
  }),
  programDetail: (id: string) => ({
    queryKey: ["programDetail", id],
    queryFn: async () => fetchFuelSurchargeProgramDetail(id),
  }),
  eiaSeriesOptions: () => ({
    queryKey: ["eiaSeriesOptions"],
    queryFn: fetchEIASeriesOptions,
  }),
  generateTable: (input: GenerateFuelTableInput) => ({
    queryKey: ["generateTable", input],
    queryFn: async () => generateFuelSurchargeTable(input),
  }),
});
