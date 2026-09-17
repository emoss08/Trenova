import type {
  AssistantPageContext,
  AssistantStreamEvent,
  SendMessageResult,
} from "@/types/assistant";

/**
 * Where a turn is, from the reader's side of the wire.
 *
 * - guarding: sent, waiting to hear whether the question is in scope
 * - streaming: reply text is arriving
 * - working: a tool is running, or the model is thinking between tools
 * - done / refused / error: the turn is over and the thread will be refetched
 */
export type TurnStatus = "guarding" | "streaming" | "working" | "done" | "refused" | "error";

export type TextSegment = {
  kind: "text";
  text: string;
  /** Closed text is a message the model finished before asking for a tool. */
  closed: boolean;
};

export type ToolSegment = {
  kind: "tool";
  callId: string;
  name: string;
  arguments: Record<string, unknown>;
  status: "running" | "done" | "failed" | "proposed";
  content: string;
};

export type TurnSegment = TextSegment | ToolSegment;

export type TurnState = {
  status: TurnStatus;
  userContent: string;
  /** The page the question was asked from, shown on the provisional user turn. */
  pageContext: AssistantPageContext | null;
  refusal: { message: string; reason: string; category: string } | null;
  segments: TurnSegment[];
  error: string | null;
  result: SendMessageResult | null;
};

export function initialTurnState(
  userContent: string,
  pageContext: AssistantPageContext | null = null,
): TurnState {
  return {
    status: "guarding",
    userContent,
    pageContext,
    refusal: null,
    segments: [],
    error: null,
    result: null,
  };
}

/**
 * Applies one server event. Pure, so the whole turn can be replayed in a test
 * and the rendering is only ever a function of this state.
 */
export function reduceTurn(state: TurnState, event: AssistantStreamEvent): TurnState {
  switch (event.event) {
    case "accepted":
      return { ...state, status: "working" };

    case "refused":
      // A refusal of the answer arrives after its text has streamed; the
      // reader must not be left holding words the guard withdrew.
      return {
        ...state,
        status: "refused",
        refusal: {
          message: event.data.message,
          reason: event.data.reason,
          category: event.data.category,
        },
        segments: [],
      };

    case "delta": {
      const last = state.segments.at(-1);
      if (last && last.kind === "text" && !last.closed) {
        const segments = state.segments.slice(0, -1);
        segments.push({ ...last, text: last.text + event.data.text });
        return { ...state, status: "streaming", segments };
      }
      return {
        ...state,
        status: "streaming",
        segments: [...state.segments, { kind: "text", text: event.data.text, closed: false }],
      };
    }

    case "message": {
      const last = state.segments.at(-1);
      const segments = [...state.segments];
      if (last && last.kind === "text" && !last.closed) {
        segments[segments.length - 1] = {
          ...last,
          text: event.data.content !== "" ? event.data.content : last.text,
          closed: true,
        };
      } else if (event.data.content !== "") {
        segments.push({ kind: "text", text: event.data.content, closed: true });
      }
      return { ...state, status: "working", segments };
    }

    case "tool_started":
      return {
        ...state,
        status: "working",
        segments: [
          ...state.segments,
          {
            kind: "tool",
            callId: event.data.callId,
            name: event.data.name,
            arguments: event.data.arguments ?? {},
            status: "running",
            content: "",
          },
        ],
      };

    case "tool_finished": {
      const status: ToolSegment["status"] = event.data.failed
        ? "failed"
        : event.data.proposed
          ? "proposed"
          : "done";
      const segments = state.segments.map((segment) =>
        segment.kind === "tool" && segment.callId === event.data.callId
          ? { ...segment, status, content: event.data.content }
          : segment,
      );
      return { ...state, status: "working", segments };
    }

    case "done":
      return { ...state, status: "done", result: event.data };

    case "error":
      return { ...state, status: "error", error: event.data.message };

    default:
      return state;
  }
}

/** Whether the turn is still in flight, for disabling the composer. */
export function isTurnActive(state: TurnState | null): boolean {
  return state !== null && ["guarding", "streaming", "working"].includes(state.status);
}
