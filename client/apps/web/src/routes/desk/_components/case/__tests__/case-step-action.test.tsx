import type { DeskDock } from "@/components/desk-chat/desk-thread";
import type { AssistantThread, CaseChecklist, CaseRecord, StepAbility } from "@/types/assistant";
import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import { CaseStepAction, onlyPeopleCanTake } from "../case-step-action";

const record: CaseRecord = {
  type: "Shipment",
  id: "shp_1",
  label: "SEED-PAY-001",
  status: "Completed",
  closed: false,
  closedAs: "",
  invoiceId: "",
  customerId: "",
  carrierIds: [],
};

const thread = { id: "athr_1", canContinue: true } as AssistantThread;

function renderStep(ability: StepAbility | undefined, busy = false) {
  const dock: DeskDock = { ask: vi.fn(), busy };
  render(
    <MemoryRouter>
      <CaseStepAction
        primary
        step="mark_ready"
        ability={ability}
        record={record}
        checklist={null}
        thread={thread}
        dock={dock}
      />
    </MemoryRouter>,
  );

  return { dock };
}

/**
 * A step is never sent to an agent that can only say it cannot, and the
 * button is the step's own whoever takes it: the conversation's agent is
 * asked when it can, the same press hands the step to another agent that can
 * from inside this conversation, and the page where the person does it opens
 * when no agent they may use can.
 */
describe("CaseStepAction", () => {
  it("asks the conversation's agent when it can take the step", () => {
    const { dock } = renderStep({ via: "Agent", agentId: "", agentName: "" });

    fireEvent.click(screen.getByRole("button", { name: "Mark ready to invoice" }));

    expect(dock.ask).toHaveBeenCalledWith("Mark shipment SEED-PAY-001 ready to invoice.", undefined);
  });

  it("asks the conversation's agent when nothing was worked out about the step", () => {
    const { dock } = renderStep(undefined);

    fireEvent.click(screen.getByRole("button", { name: "Mark ready to invoice" }));

    expect(dock.ask).toHaveBeenCalledWith("Mark shipment SEED-PAY-001 ready to invoice.", undefined);
  });

  it("hands the step to the agent that can with the same button, in this conversation", () => {
    const { dock } = renderStep({
      via: "Ask",
      agentId: "agd_billing",
      agentName: "Billing Assistant",
    });

    const button = screen.getByRole("button", { name: "Mark ready to invoice" });
    expect(button).toHaveAttribute("title", "Billing Assistant takes this step");
    fireEvent.click(button);

    expect(dock.ask).toHaveBeenCalledTimes(1);
    expect(dock.ask).toHaveBeenCalledWith("Mark shipment SEED-PAY-001 ready to invoice.", {
      directedAgentId: "agd_billing",
    });
  });

  it("asks the conversation's agent when another agent is named without an id", () => {
    const { dock } = renderStep({ via: "Ask", agentId: "", agentName: "Billing Assistant" });

    fireEvent.click(screen.getByRole("button", { name: "Mark ready to invoice" }));

    expect(dock.ask).toHaveBeenCalledWith("Mark shipment SEED-PAY-001 ready to invoice.", undefined);
  });

  it("waits for the reply under way before handing the step on", () => {
    const { dock } = renderStep(
      { via: "Ask", agentId: "agd_billing", agentName: "Billing Assistant" },
      true,
    );

    const button = screen.getByRole("button", { name: "Mark ready to invoice" });
    expect(button).toBeDisabled();
    expect(button).toHaveAttribute("title", "Waits for the reply under way");
    fireEvent.click(button);
    expect(dock.ask).not.toHaveBeenCalled();
  });

  it("opens the page where the person takes it when no agent can", () => {
    const { dock } = renderStep({ via: "Person", agentId: "", agentName: "" });

    expect(screen.getByRole("link", { name: "Open the billing queue" })).toHaveAttribute(
      "href",
      "/billing/queue",
    );
    expect(screen.queryByRole("button")).toBeNull();
    expect(dock.ask).not.toHaveBeenCalled();
  });
});

describe("onlyPeopleCanTake", () => {
  const checklist: CaseChecklist = {
    kind: "ReadyToBill",
    ready: false,
    next: "mark_ready",
    items: [],
  };
  const person: StepAbility = { via: "Person", agentId: "", agentName: "" };

  it("says so when no agent the person may use can take any step offered", () => {
    expect(onlyPeopleCanTake(checklist, { mark_ready: person })).toBe(true);
  });

  it("stays quiet when an agent can take a step, here or handed it", () => {
    expect(
      onlyPeopleCanTake(checklist, { mark_ready: { via: "Agent", agentId: "", agentName: "" } }),
    ).toBe(false);
    expect(
      onlyPeopleCanTake(checklist, {
        mark_ready: { via: "Ask", agentId: "agd_billing", agentName: "Billing Assistant" },
      }),
    ).toBe(false);
  });

  it("stays quiet when nothing was worked out or nothing is offered", () => {
    expect(onlyPeopleCanTake(checklist, {})).toBe(false);
    expect(onlyPeopleCanTake({ ...checklist, next: "" }, { mark_ready: person })).toBe(false);
  });

  it("weighs every step offered, not only the next", () => {
    const withBlocked: CaseChecklist = {
      ...checklist,
      items: [
        {
          key: "pod",
          state: "Blocked",
          codes: [],
          names: [],
          count: 0,
          step: "request_pod",
          optional: false,
          label: "",
          stepLabel: "",
          prompt: "",
          manual: false,
          tickedBy: "",
        },
      ],
    };

    expect(
      onlyPeopleCanTake(withBlocked, {
        mark_ready: person,
        request_pod: { via: "Agent", agentId: "", agentName: "" },
      }),
    ).toBe(false);
    expect(onlyPeopleCanTake(withBlocked, { mark_ready: person, request_pod: person })).toBe(true);
  });
});
