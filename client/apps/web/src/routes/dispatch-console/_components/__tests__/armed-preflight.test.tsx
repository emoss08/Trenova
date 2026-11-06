import type {
  DispatchAssignmentPreview,
  DispatchBoardMove,
  DispatchCandidate,
} from "@/lib/graphql/dispatch-console";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ArmedPreflight } from "../armed-preflight";

const preview = vi.hoisted(() => ({ read: vi.fn() }));

vi.mock("@/lib/queries/dispatch-console", () => ({
  dispatchConsoleQueries: {
    assignmentPreview: (input: { moveId: string; workerId: string }) => ({
      queryKey: ["test-preview", input] as const,
      queryFn: () => preview.read(input),
    }),
  },
}));

const move = {
  moveId: "sm_1",
  proNumber: "1048790",
  isCovered: false,
  assignedWorkerName: "",
  assignedTractorCode: "",
} as DispatchBoardMove;

const candidate = {
  workerId: "wrk_marcus",
  workerName: "Marcus Bell",
  tractorId: "trc_207",
} as DispatchCandidate;

// DispatchAssignmentPreview as dispatch_console.graphqls defines it, a clean pairing.
const cleanPreview = {
  tractorId: "trc_207",
  trailerId: "trl_9",
  requiresOverride: false,
  blocked: false,
  candidate: {
    verdict: "feasible",
    findings: [],
    hosStrategy: "currentClocks",
    hosRestStartDeadline: 0,
    deadheadMiles: 38,
    driveRemainingMs: 24_000_000,
    minutesOfSlack: 22,
    slackKnown: true,
    hosKnown: true,
    costCents: 8_300,
    costLines: [{ component: "emptyMiles", cents: 7_600, soft: false }],
  },
} as unknown as DispatchAssignmentPreview;

function renderPreflight(target = move, onConfirm = vi.fn().mockResolvedValue(undefined)) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <ArmedPreflight
        move={target}
        candidate={candidate}
        isAssigning={false}
        onConfirm={onConfirm}
        onCancel={vi.fn()}
      />
    </QueryClientProvider>,
  );
  return onConfirm;
}

afterEach(cleanup);

describe("ArmedPreflight", () => {
  it("keeps confirm off until the pairing has been checked", async () => {
    let resolve: (value: DispatchAssignmentPreview) => void = () => {};
    preview.read.mockReturnValue(new Promise((done) => (resolve = done)));
    renderPreflight();

    expect(screen.getByRole("button", { name: /Assign driver/ })).toBeDisabled();

    resolve(cleanPreview);

    await waitFor(() =>
      expect(screen.getByRole("button", { name: /Assign driver/ })).toBeEnabled(),
    );
  });

  it("waits on confirm once checked, so Enter commits the pairing it shows", async () => {
    preview.read.mockResolvedValue(cleanPreview);
    const onConfirm = renderPreflight();

    await waitFor(() =>
      expect(screen.getByRole("button", { name: /Assign driver/ })).toHaveFocus(),
    );
    await userEvent.keyboard("{Enter}");

    expect(onConfirm).toHaveBeenCalledWith({ tractorId: "trc_207", trailerId: "trl_9" });
  });

  it("names who a reassign replaces", async () => {
    preview.read.mockResolvedValue(cleanPreview);
    renderPreflight({
      ...move,
      isCovered: true,
      assignedWorkerName: "Dana Ortiz",
      assignedTractorCode: "T-114",
    } as DispatchBoardMove);

    expect(screen.getByText("Reassign 1048790 to Marcus Bell")).toBeInTheDocument();
    expect(screen.getByText("Replaces Dana Ortiz · T-114")).toBeInTheDocument();
    expect(await screen.findByRole("button", { name: /Reassign driver/ })).toBeInTheDocument();
  });

  it("refuses a pairing the organization blocks by policy", async () => {
    preview.read.mockResolvedValue({ ...cleanPreview, requiresOverride: true });
    renderPreflight();

    expect(
      await screen.findByText(
        "Your organization enforces this rule as a block, so this pairing can't be assigned.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Assign driver/ })).toBeDisabled();
  });

  it("says the check failed and offers it again rather than allowing the assign", async () => {
    preview.read.mockRejectedValue(new Error("upstream timeout"));
    renderPreflight();

    expect(await screen.findByText("Couldn't check this pairing")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Assign driver/ })).toBeDisabled();
  });
});
