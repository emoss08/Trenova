import { exchangeRateService } from "@/services/exchange-rate";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const exchangeRate = createQueryKeys("exchangeRate", {
  convert: (from: string, to: string, amount: number, date?: string) => ({
    queryKey: ["convert", from, to, amount, date ?? ""],
    queryFn: () => exchangeRateService.convert(from, to, amount, date),
  }),
  latest: (base: string) => ({
    queryKey: ["latest", base],
    queryFn: () => exchangeRateService.latest(base),
  }),
});
