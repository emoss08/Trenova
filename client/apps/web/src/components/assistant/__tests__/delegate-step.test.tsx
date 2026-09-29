import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it } from "vitest";
import {
  assistantDelegateFinishedEventSchema,
  assistantMessageSchema,
  type AssistantMessage,
} from "@/types/assistant";
import type { ToolStep } from "../activity";
import { DelegateStep } from "../delegate-step";

afterEach(cleanup);

const CALL = "call_hand";
const AGENT = "agdef_shipment_desk";
let sequence = 0;

function saved(fields: Record<string, unknown>): AssistantMessage {
  sequence += 1;
  return assistantMessageSchema.parse({
    id: `amsg_${sequence}`,
    threadId: "athr_1",
    sequence,
    role: "User",
    kind: "Delegated",
    agentId: AGENT,
    agentName: "Shipment Desk",
    delegateCallId: CALL,
    content: "",
    createdAt: 1_700_000_000 + sequence,
    ...fields,
  });
}

function handOff(): ToolStep {
  return {
    id: CALL,
    name: "delegate_task",
    arguments: { agentId: AGENT, task: "Find where PRO-1001 is." },
    status: "done",
    content: "",
    effect: "delegate",
    summary: "Shipment Desk",
    durationSeconds: 4,
    delegate: {
      kind: "saved",
      messages: [
        saved({ role: "User", content: "Find where PRO-1001 is." }),
        saved({
          role: "Assistant",
          toolCalls: [{ id: "call_look", name: "get_shipment", arguments: {} }],
        }),
        saved({
          role: "Tool",
          toolCallId: "call_look",
          toolName: "get_shipment",
          content: "{}",
          summary: "PRO-1001",
        }),
        saved({ role: "Assistant", content: "PRO-1001 is in Memphis." }),
      ],
      report: assistantDelegateFinishedEventSchema.parse({
        delegateCallId: CALL,
        agentId: AGENT,
        agentName: "Shipment Desk",
        status: "completed",
        reply: "PRO-1001 is in Memphis.",
        toolCallsUsed: 1,
      }),
    },
  };
}

function renderStep(running: boolean) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ul>
          <DelegateStep step={handOff()} live={false} running={running} />
        </ul>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

/**
 * #628: after a reload, the other agent's work looked gone, because a settled
 * hand-off was drawn closed and its chevron only appeared on hover. It now
 * opens onto the task, the steps and the answer, and the chevron is always
 * there to close it.
 */
describe("DelegateStep", () => {
  it("opens a settled hand-off onto what the other agent did", () => {
    renderStep(false);

    expect(screen.getByText("What it did")).toBeTruthy();
    expect(screen.getByText("Its answer")).toBeTruthy();
    expect(screen.getByRole("button", { name: /Shipment Desk/, expanded: true })).toBeTruthy();
  });

  it("always shows the chevron that opens and closes it", () => {
    const { container } = renderStep(false);

    const chevron = container.querySelector("svg.lucide-chevron-right");
    expect(chevron).not.toBeNull();
    expect(chevron?.getAttribute("class")).not.toContain("opacity-0");
  });

  it("closes when the person closes it", async () => {
    renderStep(false);

    await userEvent.click(screen.getByRole("button", { name: /Shipment Desk/, expanded: true }));

    expect(screen.getByRole("button", { name: /Shipment Desk/, expanded: false })).toBeTruthy();
  });

  it("keeps a hand-off still under way to its progress", () => {
    renderStep(true);

    expect(screen.queryByText("What it did")).toBeNull();
    expect(screen.getByRole("button", { name: /Shipment Desk/, expanded: false })).toBeTruthy();
  });
});
