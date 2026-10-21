import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import type { ShipmentAnalyticsData } from "@/lib/shipment-analytics";
import { CustomerMix } from "../customer-mix";

vi.mock("@/lib/graphql/shipment", () => ({
  getShipmentPageAnalyticsGraphQL: vi.fn(),
}));

type CustomerMixCard = ShipmentAnalyticsData["customerMix"];

function renderMix(customerMix: CustomerMixCard) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <NuqsTestingAdapter>
        <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
      </NuqsTestingAdapter>
    );
  }

  render(
    <CustomerMix
      customerMix={customerMix}
      tomorrowsPickups={{ date: "2026-05-08", pickups: [] }}
      enabled={false}
    />,
    { wrapper: Wrapper },
  );
}

// The server keeps a customer's name, which is in its reference view, and withholds the
// figures about it (revenue, share, loads, trend) from a shipment reader who cannot read
// customers. The row says so instead of drawing zero revenue.
describe("customer mix", () => {
  it("draws a customer's withheld figures as hidden", () => {
    renderMix({
      windowDays: 30,
      entries: [
        {
          customerId: "cus_1",
          name: "Acme Foods",
          revenue: null,
          share: null,
          loads: null,
          trend: null,
        },
      ],
    });

    const row = screen.getByText("Acme Foods").closest("li");
    expect(row).not.toBeNull();
    expect(within(row as HTMLElement).getByText("Hidden")).toBeInTheDocument();
    expect(within(row as HTMLElement).queryByText(/\$/)).toBeNull();
    expect(within(row as HTMLElement).queryByText(/▲|▼|–/)).toBeNull();
  });

  it("draws a customer reader's figures", () => {
    renderMix({
      windowDays: 30,
      entries: [
        {
          customerId: "cus_1",
          name: "Acme Foods",
          revenue: 40000,
          share: 80,
          loads: 10,
          trend: 12.5,
        },
      ],
    });

    const row = screen.getByText("Acme Foods").closest("li");
    expect(row).not.toBeNull();
    expect(within(row as HTMLElement).queryByText("Hidden")).toBeNull();
    expect(within(row as HTMLElement).getByText("▲12.5")).toBeInTheDocument();
  });
});
