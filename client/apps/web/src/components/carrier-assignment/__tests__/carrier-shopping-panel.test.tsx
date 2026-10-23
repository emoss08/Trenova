import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { shopResultSchema } from "@trenova/shared/types/rate";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CarrierShoppingPanel } from "../carrier-shopping-panel";

const { shop } = vi.hoisted(() => ({ shop: vi.fn() }));

vi.mock("@/services/api", () => ({ apiService: { rateQuoteService: { shop } } }));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

// A shop result as POST /rate-quotes/shop returns it after the client schema: one carrier its
// contract prices, with the quote's trace carrying the buy-side tariff's bill.
const RESULT = shopResultSchema.parse({
  strategy: "LeastCost",
  sellTotal: "2400",
  options: [
    {
      carrierId: "car_1",
      carrierName: "Blue Line",
      rank: 1,
      outcome: "Rated",
      cost: "1800",
      currency: "USD",
      margin: { amount: "600", percent: "25", payPercent: "75" },
      quote: {
        partyType: "Carrier",
        purpose: "Shopping",
        outcome: "Rated",
        status: "Quoted",
        trace: {
          bill: {
            currency: "USD",
            lines: [
              {
                kind: "Charge",
                label: "Line haul, carrier pay",
                freight: true,
                amount: "1800",
                quantity: "900",
                quantityUnit: "mi",
                rate: "2",
                rateUnit: "$/mi",
              },
            ],
            freightTotal: "1800",
            accessorialTotal: "0",
            total: "1800",
          },
        },
      },
    },
    { carrierId: "car_2", carrierName: "No Contract", outcome: "NoRateFound", margin: {} },
  ],
});

describe("CarrierShoppingPanel", () => {
  it("opens what a carrier's contract would bill, line by line", async () => {
    shop.mockResolvedValue(RESULT);
    const user = userEvent.setup();
    render(
      <QueryClientProvider client={new QueryClient()}>
        <CarrierShoppingPanel shipmentId="shp_1" onChoose={() => {}} />
      </QueryClientProvider>,
    );

    await user.click(screen.getByRole("button", { name: "Shop carriers" }));
    await user.click(await screen.findByRole("button", { name: "What Blue Line would bill" }));

    expect(await screen.findByText("Line haul, carrier pay")).toBeInTheDocument();
    expect(screen.getByText("900 mi × $2.00/mi")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "What No Contract would bill" })).toBeNull();
  });
});
