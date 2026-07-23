import { api } from "@/lib/api";

export interface RateConversionResult {
  fromCurrency: string;
  toCurrency: string;
  amount: string;
  rate: string;
  converted: string;
  date: string;
  provider?: string;
  rateType?: "bid" | "ask" | "mid";
  sourceTimestamp?: string;
  fetchedAt?: string;
  settlementEligible: boolean;
  settlementQuoteId?: string;
}

export interface LatestRatesResult {
  baseCurrency: string;
  date: string;
  provider: string;
  rateType: "bid" | "ask" | "mid";
  rates: Record<string, string>;
}

export interface SettlementQuoteRequest {
  fromCurrency: string;
  toCurrency: string;
  amount: string;
  rateType?: "bid" | "ask" | "mid";
  date?: string;
}

export interface SettlementQuote {
  id: string;
  provider: string;
  fromCurrency: string;
  toCurrency: string;
  amount: string;
  rate: string;
  convertedAmount: string;
  rateType: "bid" | "ask" | "mid";
  sourceTimestamp: string;
  fetchedAt: string;
  expiresAt: string;
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

  async createSettlementQuote(payload: SettlementQuoteRequest) {
    return api.post<SettlementQuote>("/exchange-rates/settlement-quotes/", payload);
  }
}
