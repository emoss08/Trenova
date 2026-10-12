import type { ConversationWaits } from "@/components/assistant/use-conversation-waits";
import { waitIdOfNote } from "@/components/assistant/use-conversation-waits";
import type { AgentWait } from "@/types/assistant";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DeskWaitNote } from "../../conversation/desk-wait-note";
import { DeskWaits } from "../desk-waits";

afterEach(cleanup);

function wait(overrides: Partial<AgentWait>): AgentWait {
  return {
    id: "awt_1",
    kind: "StopArrival",
    description: "Truck 2214 to reach Kroger DC",
    nextStep: "Tell Acme it is at the dock",
    threadId: "athr_1",
    status: "Waiting",
    dueAt: null,
    expiresAt: 1_800_086_400,
    resolvedAt: null,
    outcome: "",
    resumedTurnId: "",
    createdAt: 1_800_000_000,
    ...overrides,
  };
}

function waitsOf(open: AgentWait[]): ConversationWaits {
  return {
    open,
    byId: new Map(open.map((item) => [item.id, item])),
    cancel: vi.fn(async () => undefined),
  };
}

describe("DeskWaits", () => {
  it("shows nothing while nothing waits", () => {
    const { container } = render(<DeskWaits waits={waitsOf([])} timezone="UTC" />);

    expect(container).toBeEmptyDOMElement();
  });

  it("lists each wait in the agent's words, when it gives up, and cancels one", () => {
    const first = wait({});
    const waits = waitsOf([
      first,
      wait({ id: "awt_2", kind: "Time", description: "Call Acme back", dueAt: 1_800_003_600 }),
    ]);
    render(<DeskWaits waits={waits} timezone="UTC" />);

    expect(screen.getByText("Waiting on 2 things")).toBeInTheDocument();
    expect(screen.getByText("Truck 2214 to reach Kroger DC")).toBeInTheDocument();
    expect(screen.getByText("An arrival")).toBeInTheDocument();
    expect(screen.getByText(/^Gives up /u)).toBeInTheDocument();
    expect(screen.getByText(/^Due /u)).toBeInTheDocument();

    fireEvent.click(screen.getAllByRole("button", { name: "Cancel the wait" })[0]);
    expect(waits.cancel).toHaveBeenCalledWith(first);
  });
});

describe("DeskWaitNote", () => {
  it("names the wait the note picked up, from the id the server ends it with", () => {
    expect(waitIdOfNote("…what you did. Wait id: awt_01J9ABC")).toBe("awt_01J9ABC");
    expect(waitIdOfNote("no id here")).toBeNull();
  });

  it("draws the wait it records and what came of it, not the agent's note", () => {
    const met = wait({
      status: "Met",
      outcome: "Arrived at stop stp_1.",
      resolvedAt: 1_800_001_000,
    });
    render(
      <DeskWaitNote
        content="[Notice from the system…] Pick the work up. Wait id: awt_1"
        waits={new Map([[met.id, met]])}
        timezone="UTC"
      />,
    );

    expect(screen.getByText("Picked up after a wait")).toBeInTheDocument();
    expect(screen.getByText("Truck 2214 to reach Kroger DC")).toBeInTheDocument();
    expect(screen.getByText("Arrived at stop stp_1.")).toBeInTheDocument();
    expect(screen.queryByText(/Notice from the system/u)).toBeNull();
  });

  it("says a wait that ran out ran out", () => {
    const ranOut = wait({ status: "TimedOut" });
    render(
      <DeskWaitNote
        content="Wait id: awt_1"
        waits={new Map([[ranOut.id, ranOut]])}
        timezone="UTC"
      />,
    );

    expect(screen.getByText("Picked up after the wait ran out")).toBeInTheDocument();
  });
});
