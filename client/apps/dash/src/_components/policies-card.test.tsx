import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { PoliciesCard } from "./policies-card";

const { fetchMyPolicies } = vi.hoisted(() => ({ fetchMyPolicies: vi.fn() }));

vi.mock("@trenova/shared/lib/graphql/driver-portal", () => ({
  fetchMyPolicies,
  fetchMyPolicyDocumentUrl: vi.fn(),
  acknowledgeMyPolicy: vi.fn(),
}));

vi.mock("./dash-layout", () => ({
  useDashProfile: () => ({ data: { firstName: "Ada", lastName: "Byron" } }),
}));

function makePolicy(overrides: Record<string, unknown> = {}) {
  return {
    id: "pol_1",
    code: "HANDBOOK",
    title: "Driver handbook",
    summary: "How we work",
    body: null,
    hasDocument: true,
    versionLabel: "2026.1",
    requiresSignature: true,
    effectiveFrom: 1_800_000_000,
    acknowledgedAt: null,
    signatureName: null,
    ...overrides,
  };
}

function renderCard(policies: unknown[]) {
  fetchMyPolicies.mockResolvedValue(policies);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <PoliciesCard />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("PoliciesCard", () => {
  // The driver opens the card to find out what is asked of them, so what still
  // needs a signature comes first however the carrier happened to order the
  // rows, and what is already done sits underneath it.
  it("lists what still needs doing before what is already signed", async () => {
    renderCard([
      makePolicy({ id: "signed", title: "Signed already", acknowledgedAt: 1_800_000_000 }),
      makePolicy({ id: "todo", title: "Still to sign" }),
    ]);

    const rows = await screen.findAllByTestId("policy-row");
    expect(rows.map((row) => row.getAttribute("data-policy-id"))).toEqual(["todo", "signed"]);
  });

  it("says how many are outstanding in the header", async () => {
    renderCard([makePolicy({ id: "a" }), makePolicy({ id: "b", requiresSignature: false })]);

    expect(await screen.findByText("2 to read and sign.")).toBeInTheDocument();
    expect(screen.getByTestId("policies-outstanding-count")).toHaveTextContent("2");
  });

  it("renders nothing when the carrier has published no policies", async () => {
    renderCard([]);

    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(screen.queryByTestId("policies-card")).not.toBeInTheDocument();
  });
});
