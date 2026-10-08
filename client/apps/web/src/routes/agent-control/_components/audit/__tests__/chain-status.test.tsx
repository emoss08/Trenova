import type { AIAuditChainStatus } from "@/lib/graphql/ai-audit";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const fetchAIAuditChainStatus = vi.fn<() => Promise<AIAuditChainStatus>>();
const verifyAIAuditChain = vi.fn<() => Promise<AIAuditChainStatus>>();

vi.mock("@/lib/graphql/ai-audit", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/graphql/ai-audit")>();
  return {
    ...actual,
    fetchAIAuditChainStatus: () => fetchAIAuditChainStatus(),
    verifyAIAuditChain: () => verifyAIAuditChain(),
  };
});

const { ChainStatusHeader } = await import("../chain-status");
const { queries } = await import("@/lib/queries");

/** A signed chain last verified clean, as aiAuditChainStatus returns it. */
function chainStatus(overrides: Partial<AIAuditChainStatus> = {}): AIAuditChainStatus {
  return {
    signed: true,
    activeKeyId: "k2026",
    firstSeq: 1,
    lastSeq: 1_204,
    lastHash: "ab".repeat(32),
    sealedThroughSeq: 1_200,
    lastVerifiedSeq: 1_150,
    lastVerifiedAt: 1_758_800_000,
    lastVerificationStatus: "Verified",
    failedSeq: null,
    detail: null,
    verifying: false,
    ...overrides,
  };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

let client: QueryClient;

function renderStrip() {
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ChainStatusHeader />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  fetchAIAuditChainStatus.mockReset();
  verifyAIAuditChain.mockReset();
});

afterEach(() => {
  cleanup();
});

describe("ChainStatusHeader", () => {
  it("says whether the chain is signed, how far it is sealed and what its last check found", async () => {
    fetchAIAuditChainStatus.mockResolvedValue(chainStatus());
    renderStrip();

    expect(await screen.findByText("Signed")).toBeInTheDocument();
    expect(screen.getByText("Key k2026")).toBeInTheDocument();
    expect(screen.getByText("#1,200")).toBeInTheDocument();
    expect(screen.getByText("Verified")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /verify now/i })).toBeEnabled();
    expect(screen.getByText("signed chain")).toBeInTheDocument();
    expect(screen.getByText("intact")).toHaveClass("t-k");
    expect(screen.getByText(/through #1,150\./)).toBeInTheDocument();
    expect(
      screen.getByText(/The 4 newest rows are sealed at the next check\./),
    ).toBeInTheDocument();
  });

  it("says every row is sealed when the seal has caught up", async () => {
    fetchAIAuditChainStatus.mockResolvedValue(chainStatus({ sealedThroughSeq: 1_204 }));
    renderStrip();

    expect(await screen.findByText(/Every row is sealed\./)).toBeInTheDocument();
  });

  it("says nothing has been written to an empty trail", async () => {
    fetchAIAuditChainStatus.mockResolvedValue(
      chainStatus({
        lastSeq: 0,
        sealedThroughSeq: 0,
        lastVerifiedAt: null,
        lastVerificationStatus: null,
      }),
    );
    renderStrip();

    expect(
      await screen.findByText("Nothing has been written to the audit trail yet."),
    ).toBeInTheDocument();
    expect(screen.getByText("Nothing recorded yet")).toBeInTheDocument();
  });

  // Unsigned is not an error: the trail is still chained, only with plain
  // SHA-256, and the strip says so rather than claiming a signature.
  it("says when no signing key is configured", async () => {
    fetchAIAuditChainStatus.mockResolvedValue(
      chainStatus({
        signed: false,
        activeKeyId: null,
        lastVerifiedAt: null,
        lastVerificationStatus: null,
      }),
    );
    renderStrip();

    expect(await screen.findByText("Unsigned")).toBeInTheDocument();
    expect(screen.getByText("Plain SHA-256; no signing key is configured")).toBeInTheDocument();
    expect(screen.getByText("Not yet verified")).toBeInTheDocument();
    expect(screen.getByText("Never checked")).toBeInTheDocument();
    expect(screen.getByText(/unsigned because no signing key is configured/)).toBeInTheDocument();
    expect(screen.getByText(/It has not been checked yet\./)).toBeInTheDocument();
  });

  it("holds the button while a check starts, then while it runs, until its result arrives", async () => {
    const user = userEvent.setup();
    fetchAIAuditChainStatus.mockResolvedValue(chainStatus());
    const started = deferred<AIAuditChainStatus>();
    verifyAIAuditChain.mockReturnValue(started.promise);
    renderStrip();

    await user.click(await screen.findByRole("button", { name: /verify now/i }));
    expect(screen.getByRole("button", { name: /starting the check/i })).toBeDisabled();

    await act(async () => {
      started.resolve(chainStatus({ verifying: true }));
      await started.promise;
    });
    expect(await screen.findByRole("button", { name: /verifying/i })).toBeDisabled();
    expect(
      screen.getByText("The result appears here when the check finishes."),
    ).toBeInTheDocument();
    expect(screen.getByText("Checking the chain from the last sealed row…")).toBeInTheDocument();

    // The status read again while the check runs: more rows have been sealed,
    // but no result is stored yet, and the server's status never says it is
    // verifying on its own. The button stays held.
    fetchAIAuditChainStatus.mockResolvedValue(chainStatus({ sealedThroughSeq: 1_204 }));
    await act(async () => {
      await client.invalidateQueries({ queryKey: queries.aiAudit.chainStatus().queryKey });
    });
    expect(await screen.findByText("#1,204")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /verifying/i })).toBeDisabled();

    // The check's result lands as a newer last-verified time.
    fetchAIAuditChainStatus.mockResolvedValue(
      chainStatus({ lastVerifiedAt: 1_758_900_000, lastVerifiedSeq: 1_204 }),
    );
    await act(async () => {
      await client.invalidateQueries({ queryKey: queries.aiAudit.chainStatus().queryKey });
    });
    await waitFor(() => expect(screen.getByRole("button", { name: /verify now/i })).toBeEnabled());
  });

  it("says why a check could not be started and lets it be tried again", async () => {
    const user = userEvent.setup();
    fetchAIAuditChainStatus.mockResolvedValue(chainStatus());
    verifyAIAuditChain.mockRejectedValue(
      new Error("The verification could not be started — try again shortly"),
    );
    renderStrip();

    await user.click(await screen.findByRole("button", { name: /verify now/i }));

    expect(
      await screen.findByText("The verification could not be started — try again shortly"),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /verify now/i })).toBeEnabled();
  });

  it("shows where the chain stopped matching", async () => {
    fetchAIAuditChainStatus.mockResolvedValue(
      chainStatus({
        lastVerificationStatus: "Mismatch",
        failedSeq: 812,
        detail: "hash does not recompute",
      }),
    );
    renderStrip();

    expect(await screen.findByText("Mismatch")).toBeInTheDocument();
    expect(screen.getByText("no longer matches")).toHaveClass("t-d");
    expect(
      screen.getByText(/The trail no longer matches its chain at #812: hash does not recompute/),
    ).toBeInTheDocument();
  });

  it("asks for a missing key back", async () => {
    fetchAIAuditChainStatus.mockResolvedValue(
      chainStatus({ lastVerificationStatus: "KeyMissing" }),
    );
    renderStrip();

    expect(await screen.findByText("Key missing")).toBeInTheDocument();
    expect(screen.getByText(/Put the key back, then verify again/)).toBeInTheDocument();
  });
});
