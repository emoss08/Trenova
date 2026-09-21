import type {
  AssistantPageContext,
  AssistantStreamEvent,
  SendMessageResult,
  RetryKind,
} from "@/types/assistant";

/**
 * Where a turn is, from the reader's side of the wire.
 *
 * - guarding: sent, waiting to hear whether the question is in scope
 * - streaming: reply text, or the thinking before it, is arriving
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

/**
 * What the model thought before the text that follows. A heavy model can sit
 * silent for a minute before its first word; this is what that minute shows.
 */
export type ReasoningSegment = {
  kind: "reasoning";
  text: string;
  /** Closed once the reply or a tool call begins; the thought is then finished. */
  closed: boolean;
};

export type TurnSegment = TextSegment | ToolSegment | ReasoningSegment;

/** Closes an open reasoning segment, if the last one is open. */
function closeReasoning(segments: TurnSegment[]): TurnSegment[] {
  const last = segments.at(-1);
  if (last && last.kind === "reasoning" && !last.closed) {
    return [...segments.slice(0, -1), { ...last, closed: true }];
  }
  return segments;
}

export type TurnState = {
  status: TurnStatus;
  userContent: string;
  /** The page the question was asked from, shown on the provisional user turn. */
  pageContext: AssistantPageContext | null;
  refusal: { message: string; reason: string; category: string } | null;
  segments: TurnSegment[];
  error: string | null;
  result: SendMessageResult | null;
  /** Set while the reply is starting over after a model died partway. */
  retrying: { attempt: number; provider: string; kind: RetryKind; waitSeconds: number } | null;
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
    retrying: null,
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

    case "reasoning": {
      const last = state.segments.at(-1);
      if (last && last.kind === "reasoning" && !last.closed) {
        const segments = state.segments.slice(0, -1);
        segments.push({ ...last, text: last.text + event.data.text });
        return { ...state, status: "streaming", segments };
      }
      return {
        ...state,
        status: "streaming",
        retrying: null,
        segments: [...state.segments, { kind: "reasoning", text: event.data.text, closed: false }],
      };
    }

    case "delta": {
      const segments = closeReasoning(state.segments);
      const last = segments.at(-1);
      if (last && last.kind === "text" && !last.closed) {
        const rest = segments.slice(0, -1);
        rest.push({ ...last, text: last.text + event.data.text });
        return { ...state, status: "streaming", retrying: null, segments: rest };
      }
      return {
        ...state,
        status: "streaming",
        retrying: null,
        segments: [...segments, { kind: "text", text: event.data.text, closed: false }],
      };
    }

    case "message": {
      const segments = closeReasoning(state.segments);
      const last = segments.at(-1);
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
          ...closeReasoning(state.segments),
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

    case "retrying":
      // A restart withdraws the words that arrived, so the reader is not
      // left holding half of one answer under another; the tools that ran
      // are kept, because they did run. A busy retry happened before any
      // word arrived, so there is nothing to withdraw.
      return {
        ...state,
        status: "working",
        retrying: {
          attempt: event.data.attempt,
          provider: event.data.provider,
          kind: event.data.kind,
          waitSeconds: event.data.waitSeconds,
        },
        segments:
          event.data.kind === "busy"
            ? state.segments
            : state.segments.filter((segment) => segment.kind === "tool"),
      };

    case "done":
      // A refusal is complete in itself; done after it only says the turn
      // was saved, and must not turn the refusal back into a reply.
      if (state.status === "refused") {
        return { ...state, result: event.data };
      }
      return { ...state, status: "done", result: event.data };

    case "error":
      return { ...state, status: "error", error: event.data.message };

    default:
      return state;
  }
}

/** Why a turn stopped, from the reader's side. */
export type TurnFailureCause = "stopped" | "failed" | "ended";

/**
 * What the reader lost, which is what the failure copy has to say. A turn
 * with nothing on screen lost nothing but the answer; one with words or a
 * lookup on screen was cut off, and what is shown is what had happened.
 */
export type TurnFailureKind =
  | "stopped-before-start"
  | "stopped"
  | "failed-before-start"
  | "cut-off";

export function describeTurnFailure(state: TurnState, cause: TurnFailureCause): TurnFailureKind {
  const underway = state.segments.length > 0;
  if (cause === "stopped") {
    return underway ? "stopped" : "stopped-before-start";
  }
  return underway ? "cut-off" : "failed-before-start";
}

/** Whether the turn is still in flight, for disabling the composer. */
export function isTurnActive(state: TurnState | null): boolean {
  return state !== null && ["guarding", "streaming", "working"].includes(state.status);
}
