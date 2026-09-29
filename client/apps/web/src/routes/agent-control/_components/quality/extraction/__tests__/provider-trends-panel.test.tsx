import type {
  ExtractionProviderTrend,
  ExtractionProviderTrends,
  ExtractionWeekAccuracy,
} from "@/lib/graphql/extraction-eval";
import { stubLayout } from "@/test/layout";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const fetchTrends = vi.fn<() => Promise<ExtractionProviderTrends>>();

vi.mock("@/lib/graphql/extraction-eval", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/graphql/extraction-eval")>();
  return { ...actual, fetchExtractionProviderTrends: () => fetchTrends() };
});

const { ProviderTrendsPanel } = await import("../provider-trends-panel");

const WEEK = 7 * 86_400;
const CURRENT = 1_790_380_800;
const weekStarts = Array.from({ length: 12 }, (_, i) => CURRENT - (11 - i) * WEEK);

function week(weekStart: number, scored: number, correct: number): ExtractionWeekAccuracy {
  return {
    weekStart,
    corrections: Math.ceil(scored / 20),
    scored,
    correct,
    accuracy: scored > 0 ? correct / scored : 0,
  } as ExtractionWeekAccuracy;
}

function trend(overrides: Partial<ExtractionProviderTrend> = {}): ExtractionProviderTrend {
  return {
    providerId: "aip_tuned",
    providerName: "Fine-tuned Qwen",
    model: "trenova-extract",
    providerRemoved: false,
    dropPoints: 20,
    comparable: true,
    drifting: true,
    weeks: weekStarts.map((start, i) => week(start, i === 3 ? 0 : 120, i === 10 ? 90 : 114)),
    checked: week(weekStarts[10], 120, 90),
    baseline: week(weekStarts[6], 400, 380),
    ...overrides,
  } as ExtractionProviderTrend;
}

function trends(providers: ExtractionProviderTrend[]): ExtractionProviderTrends {
  return {
    weeks: weekStarts,
    checkedWeek: weekStarts[10],
    baselineStart: weekStarts[6],
    driftPoints: 5,
    minWeekFields: 100,
    minBaselineFields: 200,
    providers,
  } as ExtractionProviderTrends;
}

let restoreLayout = () => {};

beforeEach(() => {
  restoreLayout = stubLayout();
  fetchTrends.mockReset();
});

afterEach(() => {
  cleanup();
  restoreLayout();
});

function renderPanel() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ProviderTrendsPanel />
    </QueryClientProvider>,
  );
}

function rowFor(name: string) {
  const row = screen.getByText(name).closest("tr");
  expect(row).not.toBeNull();
  return within(row as HTMLElement);
}

describe("ProviderTrendsPanel", () => {
  it("flags a provider that read last week worse than its own recent weeks", async () => {
    fetchTrends.mockResolvedValue(trends([trend()]));
    renderPanel();

    await screen.findByText("Fine-tuned Qwen · trenova-extract");
    const row = rowFor("Fine-tuned Qwen · trenova-extract");
    expect(row.getByText("75%")).toBeInTheDocument();
    expect(row.getByText("95%")).toBeInTheDocument();
    expect(row.getByText("-20 pts")).toBeInTheDocument();
    expect(row.getByText("Drifting")).toBeInTheDocument();
  });

  it("calls a provider steady when last week held up", async () => {
    fetchTrends.mockResolvedValue(
      trends([
        trend({
          providerId: "aip_frontier",
          providerName: "Frontier",
          model: "frontier-large",
          drifting: false,
          dropPoints: -2,
          checked: week(weekStarts[10], 120, 114),
          baseline: week(weekStarts[6], 400, 372),
        }),
      ]),
    );
    renderPanel();

    await screen.findByText("Frontier · frontier-large");
    const row = rowFor("Frontier · frontier-large");
    expect(row.getByText("Steady")).toBeInTheDocument();
    expect(row.getByText("+2 pts")).toBeInTheDocument();
  });

  it("waits for enough fields before judging", async () => {
    fetchTrends.mockResolvedValue(
      trends([
        trend({
          comparable: false,
          drifting: false,
          dropPoints: 0,
          checked: week(weekStarts[10], 40, 20),
        }),
      ]),
    );
    renderPanel();

    await screen.findByText("Fine-tuned Qwen · trenova-extract");
    const row = rowFor("Fine-tuned Qwen · trenova-extract");
    expect(row.getByText("Not enough data")).toBeInTheDocument();
    expect(row.getByText("—")).toBeInTheDocument();
  });

  it("names a provider that has since been removed", async () => {
    fetchTrends.mockResolvedValue(
      trends([trend({ providerRemoved: true, providerName: "", model: "" })]),
    );
    renderPanel();

    expect(await screen.findByText("Removed provider")).toBeInTheDocument();
  });

  it("explains an empty window", async () => {
    fetchTrends.mockResolvedValue(trends([]));
    renderPanel();

    expect(
      await screen.findByText(
        "No corrections in the last 12 weeks came from an AI provider. They are recorded when someone creates a shipment from a document's draft.",
      ),
    ).toBeInTheDocument();
  });

  it("says so when the trends cannot be loaded", async () => {
    fetchTrends.mockRejectedValue(new Error("unavailable"));
    renderPanel();

    expect(
      await screen.findByText("Accuracy by provider could not be loaded. Try again shortly."),
    ).toBeInTheDocument();
  });
});
