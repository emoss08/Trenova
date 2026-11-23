import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { stubLayout } from "@/test/layout";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { PolicyAcknowledgements } from "../policy-acknowledgements";

const mocks = vi.hoisted(() => ({
  fetchWorkerPolicyAcknowledgements: vi.fn(),
  allowed: true,
}));

vi.mock("@/lib/graphql/self-service", () => ({
  POLICY_ACKNOWLEDGEMENTS_KEY: "worker-policy-acknowledgements",
  fetchWorkerPolicyAcknowledgements: mocks.fetchWorkerPolicyAcknowledgements,
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: mocks.allowed, isLoading: false }),
}));

function ack(overrides: Record<string, unknown>) {
  return {
    id: "ack",
    policyId: "pol",
    versionLabel: "2",
    acknowledgedAt: 1_792_000_000,
    signatureName: "Maria Alvarez",
    documentChecksum: null,
    policy: { id: "pol", title: "Safety handbook", versionLabel: "2" },
    ...overrides,
  };
}

function page(nodes: { id: string }[], hasNextPage = false) {
  return {
    edges: nodes.map((node) => ({ cursor: node.id, node })),
    pageInfo: { hasNextPage, endCursor: hasNextPage ? "next" : null },
  };
}

function renderPanel() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <PolicyAcknowledgements workerId="wrk_1" />
    </QueryClientProvider>,
  );
}

let restoreLayout = () => {};
beforeEach(() => {
  restoreLayout = stubLayout(320);
});

afterEach(() => {
  restoreLayout();
  cleanup();
  vi.clearAllMocks();
  mocks.allowed = true;
});

describe("PolicyAcknowledgements", () => {
  it("lists the policies signed and marks a superseded version, reading the first page", async () => {
    mocks.fetchWorkerPolicyAcknowledgements.mockResolvedValue(
      page([
        ack({ id: "ack_1" }),
        ack({
          id: "ack_2",
          policyId: "pol_b",
          versionLabel: "1",
          policy: { id: "pol_b", title: "Drug policy", versionLabel: "3" },
        }),
      ]),
    );
    renderPanel();

    expect(await screen.findByText("Safety handbook")).toBeTruthy();
    expect(screen.getByText("Superseded by v3")).toBeTruthy();
    expect(mocks.fetchWorkerPolicyAcknowledgements).toHaveBeenCalledWith(
      { workerId: "wrk_1", first: 50, after: null },
      { signal: expect.anything() },
    );
  });

  it("says nothing is signed on an empty page", async () => {
    mocks.fetchWorkerPolicyAcknowledgements.mockResolvedValue(page([]));
    renderPanel();

    expect(await screen.findByText(/Nothing signed yet/)).toBeTruthy();
  });

  it("reads nothing for a reader who may not see worker policies", () => {
    mocks.allowed = false;
    renderPanel();

    expect(screen.queryByText("Policies signed")).toBeNull();
    expect(mocks.fetchWorkerPolicyAcknowledgements).not.toHaveBeenCalled();
  });
});
