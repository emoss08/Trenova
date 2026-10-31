import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { BillingQueueOverview } from "../billing-queue-overview";

const mocks = vi.hoisted(() => ({
  assist: vi.fn(),
  brief: vi.fn(),
  generation: vi.fn(),
  stats: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiService: {
    billingQueueService: {
      getAssist: mocks.assist,
      getBrief: mocks.brief,
      getNovaGeneration: mocks.generation,
      getStats: mocks.stats,
    },
  },
}));

function renderOverview() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <BillingQueueOverview activeLanes={[]} onLaneSelect={() => {}} />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  mocks.assist.mockResolvedValue({ review: true, brief: true, search: false });
  mocks.brief.mockResolvedValue({ available: true, text: "", generatedAt: 0 });
  mocks.generation.mockResolvedValue(null);
  mocks.stats.mockResolvedValue({ lanes: [], openCount: 0, openTotal: "0", pulse: {} });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("BillingQueueOverview", () => {
  it("shows Nova reviewing the queue, in shimmering text with how far it has got, where the brief will be", async () => {
    mocks.generation.mockResolvedValue({ phase: "Reviewing", total: 291, done: 100 });

    renderOverview();

    const generating = await screen.findByTestId("billing-queue-nova-generating");
    expect(generating).toHaveAttribute("aria-busy", "true");
    expect(generating).toHaveTextContent("Nova is reviewing 291 items");
    expect(generating).toHaveTextContent(
      "Reviewing 100 of 291 · each review appears on its item as it finishes",
    );
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
  });

  it("says Nova is reading the queue before it knows how many items it has", async () => {
    mocks.generation.mockResolvedValue({ phase: "Reviewing", total: 0, done: 0 });

    renderOverview();

    expect(await screen.findByText("Nova is reading the queue")).toBeInTheDocument();
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
  });

  it("says Nova is writing the brief once the reviews are in", async () => {
    mocks.generation.mockResolvedValue({ phase: "Briefing", total: 291, done: 291 });

    renderOverview();

    expect(await screen.findByText("Nova is writing the brief")).toBeInTheDocument();
    expect(screen.getByText("The brief appears here when it is written")).toBeInTheDocument();
  });

  it("keeps the brief it has and notes the generation under it", async () => {
    const now = Math.floor(Date.now() / 1000);
    mocks.brief.mockResolvedValue({
      available: true,
      text: "Two items need you.",
      generatedAt: now,
    });
    mocks.generation.mockResolvedValue({ phase: "Reviewing", total: 12, done: 3 });

    renderOverview();

    expect(await screen.findByTestId("billing-queue-nova-generating")).toHaveTextContent(
      "Reviewing 3 of 12",
    );
    expect(screen.getByRole("region", { name: "Nova's brief" })).toBeInTheDocument();
  });

  it("says Nova left items it couldn't review, and when it tries them again, where the brief will be", async () => {
    mocks.generation.mockResolvedValue({
      phase: "Left",
      total: 0,
      done: 0,
      retryAt: Math.floor(Date.now() / 1000) + 300,
    });

    renderOverview();

    const left = await screen.findByTestId("billing-queue-nova-left");
    expect(left).toHaveTextContent("Nova couldn't review some items");
    expect(left).toHaveTextContent(/It tries them again after \d{1,2}:\d{2}/u);
    expect(left).not.toHaveAttribute("aria-busy");
    expect(screen.queryByTestId("billing-queue-nova-generating")).not.toBeInTheDocument();
  });

  it("notes what Nova left under the brief it has", async () => {
    mocks.brief.mockResolvedValue({
      available: true,
      text: "Two items need you.",
      generatedAt: Math.floor(Date.now() / 1000),
    });
    mocks.generation.mockResolvedValue({
      phase: "Left",
      total: 0,
      done: 0,
      retryAt: Math.floor(Date.now() / 1000) + 300,
    });

    renderOverview();

    expect(await screen.findByTestId("billing-queue-nova-left")).toHaveTextContent(
      /Couldn't review some items · tries again after \d{1,2}:\d{2}/u,
    );
    expect(screen.getByRole("region", { name: "Nova's brief" })).toBeInTheDocument();
  });

  it("shows nothing of Nova when nothing is generating and there is no brief", async () => {
    renderOverview();

    await vi.waitFor(() => expect(mocks.generation).toHaveBeenCalled());
    expect(screen.queryByTestId("billing-queue-nova-generating")).not.toBeInTheDocument();
  });

  it("asks nothing of Nova where it neither reviews nor briefs", async () => {
    mocks.assist.mockResolvedValue({ review: false, brief: false, search: false });

    renderOverview();

    await vi.waitFor(() => expect(mocks.assist).toHaveBeenCalled());
    expect(mocks.generation).not.toHaveBeenCalled();
  });
});
