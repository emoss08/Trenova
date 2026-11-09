import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { FactoringFundingCalculator } from "../factoring-funding-calculator";

const mocks = vi.hoisted(() => ({ fetchFactoringFundingPreview: vi.fn() }));

vi.mock("@/lib/graphql/factoring-company-page", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/factoring-company-page")>()),
  fetchFactoringFundingPreview: mocks.fetchFactoringFundingPreview,
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

const TERMS = {
  advanceRateBp: 9000,
  reserveRateBp: 700,
  minimumFeeMinor: 1500,
  feeTiers: [
    { daysFrom: 30, daysTo: null, rateBp: 250, flatMinor: 0 },
    { daysFrom: 0, daysTo: 30, rateBp: 200, flatMinor: 0 },
  ],
};

const PREVIEW = {
  faceMinor: 1_000_000,
  advanceMinor: 900_000,
  feeMinor: 20_000,
  reserveMinor: 70_000,
  restOnReleaseMinor: 80_000,
  days: 30,
  feeRateBp: 250,
};

function renderCalculator() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <FactoringFundingCalculator terms={TERMS} fundingDays={1} />
    </QueryClientProvider>,
  );
}

describe("FactoringFundingCalculator", () => {
  it("asks the server what the invoice pays under these terms and shows its answer", async () => {
    mocks.fetchFactoringFundingPreview.mockResolvedValue({
      faceMinor: 1_000_000,
      advanceMinor: 900_000,
      feeMinor: 25_000,
      reserveMinor: 70_000,
      restOnReleaseMinor: 75_000,
      days: 30,
      feeRateBp: 250,
    });
    renderCalculator();

    await waitFor(() => expect(screen.getByText("$9,000.00")).toBeTruthy());
    expect(mocks.fetchFactoringFundingPreview).toHaveBeenCalledWith(
      {
        faceMinor: 1_000_000,
        days: 30,
        advanceRateBp: 9000,
        reserveRateBp: 700,
        minimumFeeMinor: 1500,
        feeTiers: [
          { daysFrom: 30, daysTo: null, rateBp: 250, flatMinor: 0 },
          { daysFrom: 0, daysTo: 30, rateBp: 200, flatMinor: 0 },
        ],
      },
      expect.objectContaining({ signal: expect.anything() }),
    );
    expect(screen.getByText("$250.00")).toBeTruthy();
    expect(screen.getByText("$750.00")).toBeTruthy();
    expect(screen.getByText("2.5% tier, 30+ days")).toBeTruthy();
  });

  it("sends a typed amount to the server in cents, written with the currency's mark", async () => {
    mocks.fetchFactoringFundingPreview.mockResolvedValue(PREVIEW);
    renderCalculator();

    const amount = screen.getByLabelText("Invoice amount") as HTMLInputElement;
    expect(amount.value).toBe("10,000.00");
    expect(screen.getByText("$")).toBeTruthy();

    fireEvent.change(amount, { target: { value: "25,000.50" } });

    await waitFor(() =>
      expect(mocks.fetchFactoringFundingPreview).toHaveBeenLastCalledWith(
        expect.objectContaining({ faceMinor: 2_500_050 }),
        expect.anything(),
      ),
    );
  });

  it("names the fee tier's days and sets the yearly cost apart in the sentence", async () => {
    mocks.fetchFactoringFundingPreview.mockResolvedValue({ ...PREVIEW, days: 12, feeRateBp: 200 });
    renderCalculator();

    await waitFor(() => expect(screen.getByText("2% tier, days 0–30")).toBeTruthy());
    const yearly = screen.getByText("67.6% a year");
    expect(yearly.tagName).toBe("STRONG");
  });

  it("labels the fee chart at every tier boundary", () => {
    mocks.fetchFactoringFundingPreview.mockResolvedValue(PREVIEW);
    renderCalculator();

    const chart = screen.getByRole("img", { name: "Fee rate by days left on an invoice's terms" });
    const labels = [...chart.querySelectorAll("text")].map((node) => node.textContent);
    expect(labels).toEqual(expect.arrayContaining(["0d", "30d", "90d"]));
    expect(labels).not.toContain("60d");
  });

  it("says the minimum fee set the fee when the server charged exactly the minimum", async () => {
    mocks.fetchFactoringFundingPreview.mockResolvedValue({
      faceMinor: 1_000_000,
      advanceMinor: 900_000,
      feeMinor: 1500,
      reserveMinor: 70_000,
      restOnReleaseMinor: 98_500,
      days: 30,
      feeRateBp: 10,
    });
    renderCalculator();

    await waitFor(() =>
      expect(screen.getByText("The minimum fee; the tier's rate comes to less")).toBeTruthy(),
    );
  });
});
