import { withSessionHeaders } from "@trenova/shared/lib/api";
import { API_BASE_URL } from "@trenova/shared/lib/constants";
import { readEventStream } from "@trenova/shared/lib/sse";

/**
 * What the agent working a case is writing, read as it is written. The
 * server follows the run the case itself says is working, so a reader names
 * the case and nothing else. It resolves when the run says it is done, when
 * the stream closes, or quietly on an abort.
 */
export async function followDeskCase(
  kind: string,
  id: string,
  onMessage: (event: string, data: string) => void,
  signal?: AbortSignal,
): Promise<void> {
  const path = `/desk-cases/${encodeURIComponent(kind)}/${encodeURIComponent(id)}/stream/`;
  const headers = await withSessionHeaders("GET", { Accept: "text/event-stream" }, path);
  const response = await fetch(`${API_BASE_URL}${path}`, {
    method: "GET",
    headers,
    credentials: "include",
    signal,
  });
  if (!response.ok || !response.body) {
    throw new Error(`case stream ${response.status}`);
  }

  await readEventStream(response.body, (message) => onMessage(message.event, message.data), signal);
}

/** A change the agent has drafted so far: the tool that would make it, and with what. */
export type DraftedStep = {
  name: string;
  args: Record<string, unknown> | null;
};

/**
 * A plan taking shape. What the agent says to itself while it works (what it
 * will look up next, what it found) is its working, not the plan, and is not
 * kept: the page shows the changes it has drafted, how much it has looked
 * at, and the one thing it is doing now.
 */
export type CaseWriting = {
  /** The changes drafted so far, in the order they were drafted. */
  steps: readonly DraftedStep[];
  /** How many things it has read or checked. */
  checked: number;
  /** The tool it is calling now, by name; "" while it thinks. */
  tool: string;
  /** The run said it ended. */
  ended: boolean;
};

export const NOTHING_WRITTEN: CaseWriting = { steps: [], checked: 0, tool: "", ended: false };

type ToolEvent = {
  callId: string;
  name: string;
  args: Record<string, unknown> | null;
  proposed: boolean;
  failed: boolean;
  delegated: boolean;
};

function toolEvent(data: string): ToolEvent | null {
  try {
    const parsed = JSON.parse(data) as Record<string, unknown> | null;
    if (parsed === null || typeof parsed.name !== "string") {
      return null;
    }
    const args = parsed.arguments;

    return {
      callId: typeof parsed.callId === "string" ? parsed.callId : "",
      name: parsed.name,
      args:
        typeof args === "object" && args !== null && !Array.isArray(args)
          ? (args as Record<string, unknown>)
          : null,
      proposed: parsed.proposed === true,
      failed: parsed.failed === true,
      delegated: typeof parsed.delegateCallId === "string" && parsed.delegateCallId !== "",
    };
  } catch {
    return null;
  }
}

/**
 * What each call in flight was started with, by call id, so the change a
 * finished call drafted can be named with the values it was given. Kept
 * beside the state and not in it: it is bookkeeping, not something drawn.
 */
export type CallLedger = Map<string, Record<string, unknown> | null>;

/**
 * One event of a case run's stream folded into what the page shows. A call
 * that drafts a change adds a step to the plan; drafting the same kind of
 * change again replaces the earlier draft, since an agent that thinks better
 * of a change withdraws it and drafts another. Any other call is something it
 * looked at. A model's reply is its working and is not kept. An event from an
 * agent it delegated to is that agent's, and is left out.
 */
export function writeCaseEvent(
  state: CaseWriting,
  event: string,
  data: string,
  calls: CallLedger = new Map(),
): CaseWriting {
  switch (event) {
    case "tool_started": {
      const call = toolEvent(data);
      if (call === null || call.delegated) {
        return state;
      }
      calls.set(call.callId, call.args);

      return { ...state, tool: call.name };
    }
    case "tool_finished": {
      const call = toolEvent(data);
      if (call === null || call.delegated) {
        return state;
      }
      const args = calls.get(call.callId) ?? null;
      calls.delete(call.callId);
      if (call.failed) {
        return { ...state, tool: "" };
      }
      if (!call.proposed) {
        return { ...state, tool: "", checked: state.checked + 1 };
      }

      return {
        ...state,
        tool: "",
        steps: [
          ...state.steps.filter((step) => step.name !== call.name),
          { name: call.name, args },
        ],
      };
    }
    case "done":
    case "error":
      return { ...state, tool: "", ended: true };
    default:
      return state;
  }
}
