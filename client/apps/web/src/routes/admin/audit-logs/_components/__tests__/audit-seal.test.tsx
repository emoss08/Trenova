import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { AuditSealAlert, AuditSealChip } from "../audit-seal";

const mocks = vi.hoisted(() => ({ status: vi.fn() }));

vi.mock("@/lib/queries/audit", () => ({
  audit: {
    chainStatus: () => ({ queryKey: ["audit", "chain-status"], queryFn: mocks.status }),
  },
}));

vi.mock("@trenova/shared/hooks/use-now", () => ({ useNow: () => 1_800_000_000 }));

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

function renderWithClient(node: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(<QueryClientProvider client={client}>{node}</QueryClientProvider>);
}

describe("AuditSealChip", () => {
  it("says the seals verified and how long ago", async () => {
    mocks.status.mockResolvedValue(
      chainStatus({ lastVerificationStatus: "Verified", lastVerifiedAt: 1_800_000_000 - 14 * 60 }),
    );
    renderWithClient(<AuditSealChip />);

    expect(await screen.findByText("Sealed · verified 14 minutes ago")).toBeInTheDocument();
  });

  it("says when nothing has been verified yet", async () => {
    mocks.status.mockResolvedValue(chainStatus({}));
    renderWithClient(<AuditSealChip />);

    expect(await screen.findByText("Sealed · not verified yet")).toBeInTheDocument();
  });

  it("turns to a failure when verification found tampering, even without a signing key", async () => {
    mocks.status.mockResolvedValue(
      chainStatus({ signed: false, lastVerificationStatus: "Broken" }),
    );
    renderWithClient(<AuditSealChip />);

    expect(await screen.findByText("Verification failed")).toBeInTheDocument();
  });

  it("names a missing key rather than tampering", async () => {
    mocks.status.mockResolvedValue(chainStatus({ lastVerificationStatus: "KeyMissing" }));
    renderWithClient(<AuditSealChip />);

    expect(await screen.findByText("A sealing key is missing")).toBeInTheDocument();
  });

  it("warns when seals are not signed with a key", async () => {
    mocks.status.mockResolvedValue(chainStatus({ signed: false }));
    renderWithClient(<AuditSealChip />);

    expect(await screen.findByText("Sealed without a signing key")).toBeInTheDocument();
  });
});

describe("AuditSealAlert", () => {
  it("names what failed when verification found tampering", async () => {
    mocks.status.mockResolvedValue(
      chainStatus({
        lastVerificationStatus: "Broken",
        lastVerificationDetail: "Operational entries sealed by seal 4 have been changed",
      }),
    );
    renderWithClient(<AuditSealAlert />);

    expect(await screen.findByText("The audit log failed verification")).toBeInTheDocument();
    expect(
      screen.getByText("Operational entries sealed by seal 4 have been changed"),
    ).toBeInTheDocument();
  });

  it("stays out of the way while the seals verify", async () => {
    mocks.status.mockResolvedValue(chainStatus({ lastVerificationStatus: "Verified" }));
    const { container } = render(
      <QueryClientProvider client={new QueryClient()}>
        <AuditSealAlert />
      </QueryClientProvider>,
    );
    await vi.waitFor(() => expect(mocks.status).toHaveBeenCalled());

    expect(container).toBeEmptyDOMElement();
  });
});
