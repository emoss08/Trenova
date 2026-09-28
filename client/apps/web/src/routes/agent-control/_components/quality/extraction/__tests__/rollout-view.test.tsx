import type { ExtractionRollout, ExtractionRolloutReport } from "@/lib/graphql/extraction-rollout";
import { stubLayout } from "@/test/layout";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const permissions = vi.hoisted(() => ({ denied: new Set<string>() }));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string, operation: string) => ({
    allowed: !permissions.denied.has(`${resource}:${operation}`),
    isLoading: false,
  }),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const fetchRollout = vi.fn<() => Promise<ExtractionRollout>>();
const fetchReport = vi.fn<() => Promise<ExtractionRolloutReport>>();
const updateRollout = vi.fn();

vi.mock("@/lib/graphql/extraction-rollout", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/graphql/extraction-rollout")>();
  return {
    ...actual,
    fetchExtractionRollout: () => fetchRollout(),
    fetchExtractionRolloutReport: () => fetchReport(),
    updateExtractionRollout: (input: unknown) => updateRollout(input),
  };
});

const providers = [
  {
    id: "aip_candidate",
    name: "Fine-tuned Qwen",
    model: "trenova-extract",
    enabled: true,
    tasks: ["DocumentExtraction"],
  },
];

vi.mock("@/lib/graphql/ai-provider", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/graphql/ai-provider")>();
  return { ...actual, fetchAIProviders: () => Promise.resolve(providers) };
});

const { RolloutView } = await import("../rollout-view");

function rollout(overrides: Partial<ExtractionRollout> = {}): ExtractionRollout {
  return {
    enabled: true,
    providerId: "aip_candidate",
    percent: 20,
    maxAccuracyDropPoints: 5,
    maxRejectionIncreasePoints: 10,
    serving: true,
    startedAt: 1_790_000_000,
    haltedAt: null,
    haltReason: null,
    haltCandidateRate: 0,
    haltBaselineRate: 0,
    updatedById: null,
    version: 3,
    updatedAt: 1_790_000_000,
    ...overrides,
  } as ExtractionRollout;
}

function arm(overrides: Partial<ExtractionRolloutReport["candidate"]> = {}) {
  return {
    assigned: 40,
    pending: 2,
    accepted: 34,
    rejected: 2,
    failed: 0,
    superseded: 1,
    fellBack: 1,
    rejectionRate: 2 / 36,
    ...overrides,
  };
}

function report(overrides: Partial<ExtractionRolloutReport> = {}): ExtractionRolloutReport {
  return {
    rollout: rollout(),
    providerName: "Fine-tuned Qwen",
    candidate: arm(),
    control: arm({
      assigned: 160,
      accepted: 150,
      rejected: 10,
      fellBack: 0,
      rejectionRate: 0.0625,
    }),
    candidateAccuracy: { scored: 250, correct: 235, accuracy: 0.94 },
    productionAccuracy: { scored: 900, correct: 810, accuracy: 0.9 },
    fields: [
      {
        key: "rate",
        candidateScored: 40,
        candidateCorrect: 38,
        candidateAccuracy: 0.95,
        productionScored: 150,
        productionCorrect: 135,
        productionAccuracy: 0.9,
      },
    ],
    truncated: false,
    minGuardScoredFields: 200,
    minGuardExtractions: 30,
    ...overrides,
  } as ExtractionRolloutReport;
}

let restoreLayout = () => {};

beforeEach(() => {
  restoreLayout = stubLayout();
  permissions.denied.clear();
  fetchRollout.mockReset();
  fetchReport.mockReset();
  updateRollout.mockReset();
});

afterEach(() => {
  cleanup();
  restoreLayout();
});

function renderView() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <RolloutView />
    </QueryClientProvider>,
  );
}

describe("RolloutView", () => {
  it("shows the candidate serving its share beside production", async () => {
    fetchRollout.mockResolvedValue(rollout());
    fetchReport.mockResolvedValue(report());
    renderView();

    expect(await screen.findByText("Fine-tuned Qwen · trenova-extract")).toBeInTheDocument();
    expect(screen.getByText("Serving")).toBeInTheDocument();
    expect(await screen.findByText("94%")).toBeInTheDocument();
    expect(screen.getByText("4 pts above production")).toBeInTheDocument();
    expect(screen.getByText("1 fell back to production, 2 pending")).toBeInTheDocument();
    expect(screen.getByText("+5 pts")).toBeInTheDocument();
  });

  it("says how much evidence each guard still needs", async () => {
    fetchRollout.mockResolvedValue(rollout());
    fetchReport.mockResolvedValue(
      report({
        candidateAccuracy: { scored: 120, correct: 110, accuracy: 110 / 120 },
        candidate: arm({ accepted: 10, rejected: 1 }),
      }),
    );
    renderView();

    const guards = await screen.findByRole("region", { name: "Guards" });
    expect(
      within(guards).getByText("120 of 200 confirmed fields on each side before it can act"),
    ).toBeInTheDocument();
    expect(
      within(guards).getByText("11 of 30 finished extractions on each side before it can act"),
    ).toBeInTheDocument();
  });

  it("explains why a guard stopped the rollout", async () => {
    const halted = rollout({
      serving: false,
      haltedAt: 1_790_000_500,
      haltReason: "AccuracyDrop",
      haltCandidateRate: 0.8,
      haltBaselineRate: 0.9,
    });
    fetchRollout.mockResolvedValue(halted);
    fetchReport.mockResolvedValue(report({ rollout: halted }));
    renderView();

    expect(await screen.findByText("A guard stopped the rollout")).toBeInTheDocument();
    expect(
      screen.getByText(
        "The candidate read 80% of confirmed fields correctly against 90% for production, more than 5 points below.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByText("Stopped by a guard")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Stop rollout" })).not.toBeInTheDocument();
  });

  it("stops the rollout with one switch", async () => {
    const user = userEvent.setup();
    const current = rollout();
    fetchRollout.mockResolvedValue(current);
    fetchReport.mockResolvedValue(report());
    updateRollout.mockResolvedValue(rollout({ enabled: false, serving: false, version: 4 }));
    renderView();

    await user.click(await screen.findByRole("button", { name: "Stop rollout" }));

    await waitFor(() => expect(updateRollout).toHaveBeenCalledTimes(1));
    expect(updateRollout).toHaveBeenCalledWith({
      enabled: false,
      providerId: "aip_candidate",
      percent: 20,
      maxAccuracyDropPoints: 5,
      maxRejectionIncreasePoints: 10,
      version: 3,
    });
  });

  it("hides the controls from someone who cannot change which provider serves", async () => {
    permissions.denied.add(`${Resource.AIProvider}:${Operation.Update}`);
    fetchRollout.mockResolvedValue(rollout());
    fetchReport.mockResolvedValue(report());
    renderView();

    await screen.findByText("Fine-tuned Qwen · trenova-extract");
    expect(screen.queryByRole("button", { name: "Stop rollout" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Edit settings" })).not.toBeInTheDocument();
  });

  it("keeps the controls for someone who can change which provider serves", async () => {
    permissions.denied.add(`${Resource.AgentEvalSuite}:${Operation.Update}`);
    fetchRollout.mockResolvedValue(rollout());
    fetchReport.mockResolvedValue(report());
    renderView();

    expect(await screen.findByRole("button", { name: "Stop rollout" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Edit settings" })).toBeInTheDocument();
  });

  it("asks for a candidate before there is anything to compare", async () => {
    const unset = rollout({ enabled: false, serving: false, providerId: null, startedAt: null });
    fetchRollout.mockResolvedValue(unset);
    fetchReport.mockResolvedValue(report({ rollout: unset }));
    renderView();

    expect(
      await screen.findByText(
        "Choose a candidate and turn the rollout on to send it a share of real documents.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByText("Not set up")).toBeInTheDocument();
  });

  it("saves the settings with the version it read", async () => {
    const user = userEvent.setup();
    fetchRollout.mockResolvedValue(rollout());
    fetchReport.mockResolvedValue(report());
    updateRollout.mockResolvedValue(rollout({ percent: 50, version: 4 }));
    renderView();

    await user.click(await screen.findByRole("button", { name: "Edit settings" }));
    const dialog = await screen.findByRole("dialog");
    const share = within(dialog).getByLabelText("Share of documents (%)");
    await user.clear(share);
    await user.type(share, "50");
    await user.click(within(dialog).getByRole("button", { name: "Save" }));

    await waitFor(() => expect(updateRollout).toHaveBeenCalledTimes(1));
    expect(updateRollout).toHaveBeenCalledWith(
      expect.objectContaining({ percent: 50, version: 3, enabled: true }),
    );
  });

  it("will not save a share the server would refuse", async () => {
    const user = userEvent.setup();
    fetchRollout.mockResolvedValue(rollout());
    fetchReport.mockResolvedValue(report());
    renderView();

    await user.click(await screen.findByRole("button", { name: "Edit settings" }));
    const dialog = await screen.findByRole("dialog");
    const share = within(dialog).getByLabelText("Share of documents (%)");
    await user.clear(share);
    await user.type(share, "0");

    expect(
      within(dialog).getByText("Send between 1 and 100 percent of documents to the candidate"),
    ).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Save" })).toBeDisabled();
  });

  it("warns that the guards cannot act at a full share", async () => {
    const user = userEvent.setup();
    fetchRollout.mockResolvedValue(rollout());
    fetchReport.mockResolvedValue(report());
    renderView();

    await user.click(await screen.findByRole("button", { name: "Edit settings" }));
    const dialog = await screen.findByRole("dialog");
    const share = within(dialog).getByLabelText("Share of documents (%)");
    await user.clear(share);
    await user.type(share, "100");

    expect(
      within(dialog).getByText(
        "At 100 percent no document is left on production to compare against, so the guards cannot stop the rollout.",
      ),
    ).toBeInTheDocument();
  });

  it("shows when the rollout cannot be loaded", async () => {
    fetchRollout.mockRejectedValue(new Error("unavailable"));
    fetchReport.mockRejectedValue(new Error("unavailable"));
    renderView();

    expect(await screen.findByText("The rollout could not be loaded.")).toBeInTheDocument();
    expect(
      await screen.findByText("The rollout comparison could not be loaded. Try again shortly."),
    ).toBeInTheDocument();
  });
});
