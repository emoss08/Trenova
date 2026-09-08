import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import PoliciesConsole from "../policies-console";

const mocks = vi.hoisted(() => ({
  fetchWorkerPolicies: vi.fn(),
}));

vi.mock("@/lib/graphql/self-service", () => ({
  ...mocks,
  WORKER_POLICIES_KEY: "worker-policies",
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("../policy-dialog", () => ({ PolicyDialog: () => null }));
vi.mock("../policy-compliance-dialog", () => ({ PolicyComplianceDialog: () => null }));

function policy(over: Record<string, unknown> = {}) {
  return {
    id: "pol_1",
    code: "HANDBOOK",
    title: "Driver Handbook",
    summary: "The rules of the road at this carrier.",
    versionLabel: "2026.1",
    appliesTo: "Drivers",
    requiresSignature: true,
    status: "Active",
    effectiveFrom: 1_780_000_000,
    documentId: null,
    ...over,
  };
}

function renderConsole() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <PoliciesConsole />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("PoliciesConsole", () => {
  // While the policies are read the console draws the card grid's own
  // outline, so the loaded cards land in the same cells rather than replacing
  // three grey blocks.
  it("draws the cards' own shape while the policies are still being read", () => {
    mocks.fetchWorkerPolicies.mockReturnValue(new Promise(() => {}));
    renderConsole();

    const loading = screen.getByLabelText("Loading policies");
    expect(loading).toHaveAttribute("aria-busy", "true");
    const cards = within(loading).getAllByRole("listitem", { hidden: true });
    expect(cards.length).toBeGreaterThan(2);
    expect(within(loading).queryAllByRole("listitem")).toHaveLength(0);
  });

  it("replaces the outline with the cards once the policies land", async () => {
    mocks.fetchWorkerPolicies.mockResolvedValue([
      policy(),
      policy({ id: "pol_2", code: "DRUG", title: "Drug and Alcohol Policy" }),
    ]);
    renderConsole();

    expect(await screen.findByText("Driver Handbook")).toBeInTheDocument();
    expect(screen.queryByLabelText("Loading policies")).toBeNull();
    expect(screen.getAllByRole("button", { name: "Who has signed" })).toHaveLength(2);
  });
});
