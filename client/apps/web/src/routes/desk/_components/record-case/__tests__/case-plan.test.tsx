import type { ProposalPreview } from "@/lib/graphql/agent-preview";
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CasePlan, CasePlanDrafting, CasePlanNone } from "../case-plan";
import type { CaseDecision } from "../use-case-decision";
import { caseDecision, deskCase, planStep } from "./case-fixture";

const STEPS = [
  planStep(),
  planStep({
    id: "aprop_2",
    toolName: "customer_send_email",
    arguments: { subject: "New arrival time", body: "SHP-1 arrives about 1:18 PM." },
    planStep: 2,
  }),
];

function decisionState(overrides: Partial<CaseDecision> = {}): CaseDecision {
  return {
    steps: STEPS,
    previews: new Map(),
    stepsLoading: false,
    stepsFailed: false,
    previewLoading: false,
    previewFailed: false,
    stale: false,
    refusedStep: null,
    blocked: false,
    busy: null,
    failure: "",
    approve: vi.fn(async () => undefined),
    decline: vi.fn(async () => undefined),
    undo: vi.fn(async () => undefined),
    doNow: vi.fn(async () => undefined),
    ...overrides,
  };
}

function renderPlan(
  decision = caseDecision(),
  state = decisionState(),
  handlers: { onDecline?: () => void; onChange?: () => void; onAsk?: (q: string) => void } = {},
  canDecide = true,
) {
  return render(
    <MemoryRouter>
      <CasePlan
        data={deskCase({ decision })}
        decision={decision}
        state={state}
        canDecide={canDecide}
        timezone="UTC"
        recordPath="/shipments/shp_1"
        onDecline={handlers.onDecline ?? vi.fn()}
        onChange={handlers.onChange ?? vi.fn()}
        onAsk={handlers.onAsk ?? vi.fn()}
      />
    </MemoryRouter>,
  );
}

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

describe("the plan on a record's case", () => {
  it("says what waits, each step in plain words, and which cannot be undone", () => {
    renderPlan();

    expect(screen.getByText("Awaiting your approval")).toBeTruthy();
    expect(
      screen.getByText("Plan · 2 steps · runs in order, stops at the first failure"),
    ).toBeTruthy();
    expect(screen.getByText("Move the appointment")).toBeTruthy();
    expect(screen.getByText("Email the customer")).toBeTruthy();
    expect(screen.getAllByText("Can't be undone")).toHaveLength(1);
    expect(screen.getAllByText("Waiting")).toHaveLength(2);
    expect(screen.getByText("Permanent")).toBeTruthy();
  });

  it("approves, declines and asks to change what is shown", () => {
    const state = decisionState();
    const onDecline = vi.fn();
    const onChange = vi.fn();
    renderPlan(caseDecision(), state, { onDecline, onChange });

    fireEvent.click(screen.getByRole("button", { name: /Approve plan/ }));
    fireEvent.click(screen.getByRole("button", { name: /Decline/ }));
    fireEvent.click(screen.getByRole("button", { name: "Change values" }));

    expect(state.approve).toHaveBeenCalledTimes(1);
    expect(onDecline).toHaveBeenCalledTimes(1);
    expect(onChange).toHaveBeenCalledTimes(1);
  });

  it("holds the approval back while the preview is loading or a record moved on", () => {
    renderPlan(caseDecision(), decisionState({ blocked: true, stale: true }));

    const approve = screen.getByRole("button", { name: /Approve plan/ });
    expect((approve as HTMLButtonElement).disabled).toBe(true);
    expect(screen.getByText("Checked: a record changed since this was drafted")).toBeTruthy();
  });

  it("shows the plan and no way to decide it to someone who may not approve", () => {
    renderPlan(caseDecision(), decisionState(), {}, false);

    expect(screen.queryByRole("button", { name: /Approve plan/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /Decline/ })).toBeNull();
    expect(
      screen.getByText("This waits on someone who may approve what agents propose."),
    ).toBeTruthy();
  });

  it("counts down an approval's undo window and offers to undo it or run it now", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date(1_800_000_000_000));
    const state = decisionState();
    renderPlan(caseDecision({ phase: "Undoable", commitsAt: 1_800_000_008 }), state);

    expect(screen.getByText("Approved · nothing has run yet")).toBeTruthy();
    expect(screen.getByText("It starts in 0:08.")).toBeTruthy();
    expect(screen.getAllByText("Approved")).toHaveLength(2);

    fireEvent.click(screen.getByRole("button", { name: /Undo/ }));
    fireEvent.click(screen.getByRole("button", { name: "Do it now" }));

    expect(state.undo).toHaveBeenCalledTimes(1);
    expect(state.doNow).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("button", { name: /Approve plan/ })).toBeNull();
  });

  it("marks only the steps the server confirmed while the plan runs", () => {
    renderPlan(caseDecision({ phase: "Running", completedSteps: 1 }));

    expect(screen.getByText("Running step 2 of 2")).toBeTruthy();
    expect(screen.getByText("It ran")).toBeTruthy();
    expect(screen.getByText("Running…")).toBeTruthy();
  });

  it("says every step ran once the plan is done", () => {
    renderPlan(caseDecision({ phase: "Done", completedSteps: 2, decidedAt: 1_800_000_060 }));

    expect(screen.getAllByText("It ran")).toHaveLength(2);
    expect(screen.getByText("All 2 steps ran and were confirmed.")).toBeTruthy();
  });

  it("warns on the step the check says would be refused, and leaves approving to the person", () => {
    const onAsk = vi.fn();
    const refused = {
      coverage: "Unavailable",
      changes: [],
      stale: false,
      digest: "d1",
      warnings: [
        {
          code: "would_fail",
          message: "This would be refused as it stands: No email profile is set up.",
          args: ["No email profile is set up."],
          reasons: [],
          remedies: [],
        },
      ],
    } as unknown as ProposalPreview;
    const second = STEPS[1];
    if (!second) {
      throw new Error("the fixture plan has two steps");
    }
    renderPlan(
      caseDecision(),
      decisionState({ previews: new Map([[second.id, refused]]), refusedStep: 1 }),
      { onAsk },
    );

    expect(screen.getByText("Checked: step 2 would be refused as things stand")).toBeTruthy();
    expect(screen.getByRole("alert").textContent).toContain("No email profile is set up.");
    expect(screen.getByRole("button", { name: /Approve plan/ })).toBeEnabled();

    fireEvent.click(screen.getByRole("button", { name: "Ask Dispatch desk to redraft" }));
    expect(onAsk).toHaveBeenCalledWith(
      "Step 2 of this plan would be refused as things stand. Redraft it so every step can run.",
    );
  });

  it("stops at the step that failed, says why, and offers the three ways on", () => {
    const onAsk = vi.fn();
    renderPlan(
      caseDecision({
        phase: "Failed",
        completedSteps: 1,
        failedStep: 2,
        failureError: "The customer has no email address on file.",
      }),
      decisionState(),
      { onAsk },
    );

    expect(screen.getByText("Stopped at step 2")).toBeTruthy();
    expect(screen.getByText("It ran")).toBeTruthy();
    expect(screen.getByText("Didn't go through")).toBeTruthy();
    expect(screen.getByText("The customer has no email address on file.")).toBeTruthy();
    expect(screen.getByText("Step 2 didn't go through, so the plan stopped.")).toBeTruthy();
    expect(screen.getByRole("link", { name: "Finish it by hand" }).getAttribute("href")).toBe(
      "/shipments/shp_1",
    );

    fireEvent.click(screen.getByRole("button", { name: "Ask Dispatch desk to redraft" }));
    expect(onAsk).toHaveBeenCalledWith(
      "The plan stopped at step 2: The customer has no email address on file. Redraft it against the record as it is now.",
    );
    fireEvent.click(screen.getByRole("button", { name: "Undo step 1" }));
    expect(onAsk).toHaveBeenCalledTimes(2);
  });

  it("offers no undo of earlier steps when the first step is the one that failed", () => {
    renderPlan(caseDecision({ phase: "Failed", completedSteps: 0, failedStep: 1 }));

    expect(screen.getByText("Stopped at step 1")).toBeTruthy();
    expect(screen.getByText("Step 1 didn't go through, so the plan stopped.")).toBeTruthy();
    expect(screen.queryByText(/Step 1 ran/)).toBeNull();
    expect(screen.queryByRole("button", { name: /Undo step/ })).toBeNull();
    expect(screen.getByText("Not run")).toBeTruthy();
  });

  it("keeps the reason a plan was declined with and marks its steps as never running", () => {
    renderPlan(
      caseDecision({
        phase: "Declined",
        note: "The receiver already agreed to hold the door.",
        decidedBy: { id: "usr_1", name: "Maya Chen" },
      }),
    );

    expect(screen.getByText("Declined by Maya Chen")).toBeTruthy();
    expect(screen.getAllByText("Won't run")).toHaveLength(2);
    expect(
      screen.getByText(/Dispatch desk read the reason: “The receiver already agreed/),
    ).toBeTruthy();
  });

  it("says a server refusal in the server's words above the buttons", () => {
    renderPlan(
      caseDecision(),
      decisionState({ failure: "You may not approve changes to shipments." }),
    );

    const plan = screen.getByRole("region", { name: "Plan" });
    expect(within(plan).getByText("You may not approve changes to shipments.")).toBeTruthy();
    expect(within(plan).getByRole("button", { name: /Approve plan/ })).toBeTruthy();
  });

  it("draws a single change without the plan's words", () => {
    renderPlan(
      caseDecision({ kind: "Proposal", stepCount: 1 }),
      decisionState({ steps: [STEPS[0]!] }),
    );

    expect(screen.getByText("One change")).toBeTruthy();
    expect(screen.getByRole("button", { name: /^Approve/ }).textContent).toContain("Approve");
    expect(screen.queryByText(/runs in order/)).toBeNull();
  });
});

describe("the plan slot with no plan in it", () => {
  it("says which agent is drafting the chosen option", () => {
    render(<CasePlanDrafting agentName="Dispatch desk" />);

    expect(
      screen.getByText(
        "Dispatch desk is drafting this option against your records as they are now…",
      ),
    ).toBeTruthy();
  });

  it("offers the one thing left to do when nothing is drafted", () => {
    const onClick = vi.fn();
    render(
      <CasePlanNone
        title="Nothing changes."
        note="Mark the risk accepted."
        action={{ label: "Accept the risk…", pending: false, onClick }}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Accept the risk…" }));
    expect(onClick).toHaveBeenCalledTimes(1);
    expect(screen.getByText("No plan")).toBeTruthy();
  });
});
