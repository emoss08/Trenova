import { BASE_ONLY_UNAVAILABLE, seedPlan, unavailableFeatures } from "@/test/plan";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { BillingQueueRecommendation } from "@trenova/shared/types/billing-queue";
import { Operation, Resource, type UnavailableFeature } from "@trenova/shared/types/permission";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { BillingQueueOverview } from "../billing-queue-overview";
import { BillingQueueRedetectSignaturesAction } from "../billing-queue-redetect-signatures-action";
import { BillingQueueSuggestions } from "../billing-queue-suggestions";

const mocks = vi.hoisted(() => ({
  assist: vi.fn(),
  brief: vi.fn(),
  generation: vi.fn(),
  stats: vi.fn(),
  recommendations: vi.fn(),
}));

vi.mock("@/services/billing-queue", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/services/billing-queue")>()),
  billingQueueService: {
    getAssist: mocks.assist,
    getBrief: mocks.brief,
    getNovaGeneration: mocks.generation,
    getStats: mocks.stats,
    getRecommendations: mocks.recommendations,
  },
}));

const GRANTS = {
  [Resource.BillingQueue]: Operation.Read | Operation.Update,
  [Resource.Document]: Operation.Read | Operation.Update,
};

function wrap(children: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={client}>{children}</QueryClientProvider>);
}

function onPlan(unavailable: readonly UnavailableFeature[]) {
  seedPlan(GRANTS, unavailable);
}

beforeEach(() => {
  mocks.assist.mockResolvedValue({ review: true, brief: true, search: true });
  mocks.brief.mockResolvedValue({
    available: true,
    text: "Three items wait on you.",
    generatedAt: 1,
  });
  mocks.generation.mockResolvedValue({ phase: "Reviewing", total: 4, done: 1 });
  mocks.stats.mockResolvedValue({ lanes: [], openCount: 0, openTotal: "0", pulse: {} });
  mocks.recommendations.mockResolvedValue([
    {
      kind: "ApproveAgentAgreed",
      source: "Agent",
      count: 2,
      total: 2000,
      itemIds: ["bqi_1", "bqi_2"],
    },
    {
      kind: "ApproveReady",
      source: "Rules",
      count: 3,
      total: 3000,
      itemIds: ["bqi_1", "bqi_2", "bqi_3"],
    },
  ] satisfies BillingQueueRecommendation[]);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  usePermissionStore.getState().clearPermissions();
});

describe("billing queue on a base-only plan", () => {
  it("offers no Read POD signatures", () => {
    onPlan(BASE_ONLY_UNAVAILABLE);

    wrap(<BillingQueueRedetectSignaturesAction />);

    expect(screen.queryByRole("button", { name: "Read POD signatures" })).toBeNull();
  });

  it("never asks what Nova does here and draws no Nova mark over the queue", async () => {
    onPlan(BASE_ONLY_UNAVAILABLE);

    const { container } = wrap(<BillingQueueOverview activeLanes={[]} onLaneSelect={() => {}} />);

    expect(await screen.findByText("Across today's reviews")).toBeInTheDocument();
    expect(mocks.assist).not.toHaveBeenCalled();
    expect(mocks.brief).not.toHaveBeenCalled();
    expect(mocks.generation).not.toHaveBeenCalled();
    expect(container.querySelector('[data-slot="nova-badge"]')).toBeNull();
    expect(screen.queryByText(/Nova/)).toBeNull();
  });

  it("drops what Nova's review recommends and keeps the rules' batches", async () => {
    onPlan(BASE_ONLY_UNAVAILABLE);

    const { container } = wrap(
      <BillingQueueSuggestions
        run={vi.fn()}
        onOpenItem={vi.fn()}
        onHoldItems={vi.fn()}
        isStarting={false}
        revertedRunIds={new Set()}
      />,
    );

    expect(await screen.findByText("Approve 3 ready items")).toBeInTheDocument();
    expect(screen.queryByText("Approve the 2 Nova checked")).toBeNull();
    expect(container.querySelector('[data-slot="nova-badge"]')).toBeNull();
  });
});

describe("billing queue with every add-on", () => {
  it("offers Read POD signatures", () => {
    onPlan([]);

    wrap(<BillingQueueRedetectSignaturesAction />);

    expect(screen.getByRole("button", { name: "Read POD signatures" })).toBeInTheDocument();
  });

  it("asks what Nova does and shows it reviewing the queue", async () => {
    onPlan([]);

    wrap(<BillingQueueOverview activeLanes={[]} onLaneSelect={() => {}} />);

    expect(await screen.findByTestId("billing-queue-nova-generating")).toBeInTheDocument();
    expect(mocks.assist).toHaveBeenCalled();
  });
});

describe("billing queue with Trenova AI but without agents", () => {
  it("reads no review or brief, since the review is an agent's work", async () => {
    onPlan(unavailableFeatures("platform.agent_automation"));

    wrap(<BillingQueueOverview activeLanes={[]} onLaneSelect={() => {}} />);

    expect(await screen.findByText("Across today's reviews")).toBeInTheDocument();
    expect(mocks.brief).not.toHaveBeenCalled();
    expect(mocks.generation).not.toHaveBeenCalled();
  });
});

describe("Read POD signatures follows Document Intelligence", () => {
  it("is withheld when only Document Intelligence is off", () => {
    onPlan(unavailableFeatures("document_intelligence.ai"));

    wrap(<BillingQueueRedetectSignaturesAction />);

    expect(screen.queryByRole("button", { name: "Read POD signatures" })).toBeNull();
  });
});
