import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const exchangeRate = createQueryKeys("exchangeRate", {
  convert: (from: string, to: string, amount: number, date?: string) => ({
    queryKey: ["convert", from, to, amount, date ?? ""],
    queryFn: () => apiService.exchangeRateService.convert(from, to, amount, date),
  }),
  latest: (base: string) => ({
    queryKey: ["latest", base],
    queryFn: () => apiService.exchangeRateService.latest(base),
  }),
});
