import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import AuditChainStatus from "../audit-chain-status";

const mocks = vi.hoisted(() => ({ status: vi.fn() }));

vi.mock("@/lib/queries/audit", () => ({
  audit: {
    chainStatus: () => ({ queryKey: ["audit", "chain-status"], queryFn: mocks.status }),
  },
}));

/** The auditChainStatus shape of services/tms/internal/api/graphql/schema/audit_log.graphqls. */
function chainStatus(overrides: Record<string, unknown>) {
  return {
    signed: true,
    lastSeq: 12,
    sealedThrough: null,
    lastVerifiedAt: null,
    lastVerificationStatus: null,
    lastVerificationDetail: null,
    ...overrides,
  };
}

function renderStatus() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <AuditChainStatus />
    </QueryClientProvider>,
  );
}

describe("AuditChainStatus", () => {
  it("names what failed when verification found tampering", async () => {
    mocks.status.mockResolvedValue(
      chainStatus({
        lastVerificationStatus: "Broken",
        lastVerificationDetail: "Operational entries sealed by seal 4 have been changed",
      }),
    );
    renderStatus();

    expect(await screen.findByText("The audit log failed verification")).toBeInTheDocument();
    expect(
      screen.getByText("Operational entries sealed by seal 4 have been changed"),
    ).toBeInTheDocument();
  });

  it("warns when a sealing key is missing rather than calling it tampering", async () => {
    mocks.status.mockResolvedValue(chainStatus({ lastVerificationStatus: "KeyMissing" }));
    renderStatus();

    expect(await screen.findByText("A sealing key is missing")).toBeInTheDocument();
    expect(screen.queryByText("The audit log failed verification")).not.toBeInTheDocument();
  });

  it("warns when seals are not signed with a key", async () => {
    mocks.status.mockResolvedValue(chainStatus({ signed: false }));
    renderStatus();

    expect(await screen.findByText("Entries are sealed without a signing key")).toBeInTheDocument();
    expect(
      screen.getByText("Entries are sealed every hour; nothing has been sealed yet."),
    ).toBeInTheDocument();
  });
});
