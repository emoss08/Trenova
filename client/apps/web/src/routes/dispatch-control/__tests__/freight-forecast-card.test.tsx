import type { DispatchForecastQuality } from "@/lib/graphql/dispatch-console";
import type { DispatchControl } from "@/types/dispatch-control";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import type { ReactNode } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { FreightForecastCard } from "../_components/freight-forecast-card";

const mocks = vi.hoisted(() => ({ fetch: vi.fn() }));

vi.mock("@/lib/graphql/dispatch-console", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/dispatch-console")>()),
  getDispatchForecastQualityGraphQL: mocks.fetch,
}));

const FITTED_AT = 1_793_577_600;

// DispatchForecastQuality as dispatch_console.graphqls defines it.
const ready: DispatchForecastQuality = {
  status: "Ready",
  historyWeeks: 26,
  historyLoads: 980,
  weeksTested: 8,
  loadsTested: 310,
  intervalCoverage: 0.82,
  meanAbsoluteError: 1.42,
  naiveMeanAbsoluteError: 2.1,
  skill: 0.324,
  fittedAt: FITTED_AT,
};

const neverFitted: DispatchForecastQuality = {
  status: "InsufficientHistory",
  historyWeeks: 0,
  historyLoads: 0,
  weeksTested: 0,
  loadsTested: 0,
  intervalCoverage: 0,
  meanAbsoluteError: 0,
  naiveMeanAbsoluteError: 0,
  skill: 0,
  fittedAt: null,
};

function Form({
  anticipateFreight,
  children,
}: {
  anticipateFreight: boolean;
  children: ReactNode;
}) {
  const form = useForm<DispatchControl>({
    defaultValues: { anticipateFreight } as DispatchControl,
  });
  return <FormProvider {...form}>{children}</FormProvider>;
}

function renderCard({ anticipateFreight = false } = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <Form anticipateFreight={anticipateFreight}>
        <FreightForecastCard />
      </Form>
    </QueryClientProvider>,
  );
}

describe("FreightForecastCard", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("draws its own outline while the forecast's test loads", () => {
    mocks.fetch.mockImplementation(() => new Promise<never>(() => {}));

    renderCard();

    expect(screen.getByLabelText("Loading the freight forecast")).toHaveAttribute(
      "aria-busy",
      "true",
    );
    expect(mocks.fetch).toHaveBeenCalledWith({ signal: expect.any(AbortSignal) });
  });

  it("says a forecast that was never fitted will be fitted overnight, with no figures", async () => {
    mocks.fetch.mockResolvedValue(neverFitted);

    renderCard();

    expect(await screen.findByText("Not fitted yet")).toBeInTheDocument();
    expect(
      screen.getByText("The forecast is fitted every night from your shipment history."),
    ).toBeInTheDocument();
    expect(screen.queryByText("Last fitted")).toBeNull();
    expect(screen.queryByText("Not enough history yet")).toBeNull();
  });

  it("says how much history is missing when there is not enough to test", async () => {
    mocks.fetch.mockResolvedValue({
      ...neverFitted,
      historyWeeks: 3,
      historyLoads: 1,
      fittedAt: FITTED_AT,
    });

    renderCard();

    expect(await screen.findByText("Not enough history yet")).toBeInTheDocument();
    expect(
      screen.getByText(
        "The forecast needs 8 weeks and 200 loads of history; it has 3 weeks and 1 load.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByText(formatUnixDateTimeMedium(FITTED_AT))).toBeInTheDocument();
  });

  it("says a forecast no better than last week is ignored by plans", async () => {
    mocks.fetch.mockResolvedValue({
      ...ready,
      status: "NoSkill",
      meanAbsoluteError: 2.04,
      naiveMeanAbsoluteError: 2.1,
      skill: 0.03,
      weeksTested: 1,
    });

    renderCard();

    expect(await screen.findByText("Not better than last week yet")).toBeInTheDocument();
    expect(
      screen.getByText(
        "Tested on your last week, it missed by 2.0 loads per area and slot, against 2.1 for repeating last week. Plans ignore it until it does better; it is tested again every night.",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByText("Average miss")).toBeNull();
  });

  it("shows how a ready forecast did on the weeks it was tested on", async () => {
    mocks.fetch.mockResolvedValue(ready);

    renderCard();

    expect(await screen.findByText("Ready")).toBeInTheDocument();
    expect(
      screen.getByText(
        "Tested on your last 8 weeks: 82% of actual counts fell inside the forecast range; 32% more accurate than repeating last week.",
      ),
    ).toBeInTheDocument();
    const figures = screen.getByLabelText("Forecast figures");
    expect(within(figures).getByText("26 weeks, 980 loads")).toBeInTheDocument();
    expect(
      within(figures).getByText("1.4 loads per area and slot, against 2.1 repeating last week"),
    ).toBeInTheDocument();
    expect(within(figures).getByText(formatUnixDateTimeMedium(FITTED_AT))).toBeInTheDocument();
  });

  it("warns that Anticipate freight does nothing while the forecast is not ready", async () => {
    mocks.fetch.mockResolvedValue({ ...ready, status: "NoSkill" });

    renderCard({ anticipateFreight: true });

    expect(
      await screen.findByText(
        "Anticipate freight is on, but plans ignore the forecast until it is Ready.",
      ),
    ).toBeInTheDocument();
  });

  it("raises no warning once the forecast is ready", async () => {
    mocks.fetch.mockResolvedValue(ready);

    renderCard({ anticipateFreight: true });

    await screen.findByText("Ready");
    expect(screen.queryByText(/Anticipate freight is on/)).toBeNull();
  });

  it("offers to try again when the forecast cannot be loaded", async () => {
    const user = userEvent.setup();
    mocks.fetch.mockRejectedValueOnce(new Error("boom")).mockResolvedValueOnce(ready);

    renderCard();

    expect(
      await screen.findByText("The freight forecast could not be loaded."),
    ).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Try again" }));

    expect(await screen.findByText("Ready")).toBeInTheDocument();
    expect(mocks.fetch).toHaveBeenCalledTimes(2);
  });
});
