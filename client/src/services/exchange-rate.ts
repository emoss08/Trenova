import { api } from "@/lib/api";

export interface RateConversionResult {
  fromCurrency: string;
  toCurrency: string;
  amount: number;
  rate: number;
  converted: number;
  date: string;
}

export interface LatestRatesResult {
  baseCurrency: string;
  date: string;
  rates: Record<string, number>;
}

export class ExchangeRateService {
  async convert(from: string, to: string, amount: number, date?: string) {
    const searchParams = new URLSearchParams({ from, to, amount: String(amount) });
    if (date) searchParams.set("date", date);
    return api.get<RateConversionResult>(`/exchange-rates/convert/?${searchParams}`);
  }

  async latest(base: string) {
    const searchParams = new URLSearchParams({ base });
    return api.get<LatestRatesResult>(`/exchange-rates/latest/?${searchParams}`);
  }

  async refresh(base: string) {
    const searchParams = new URLSearchParams({ base });
    return api.post<{ status: string }>(`/exchange-rates/refresh/?${searchParams}`);
  }
}
