import type { InboundReply } from "@/lib/graphql/inbox";

/**
 * Where the reply composer is, as a state machine: writing, Nova drafting,
 * sending, refused by the provider (with an address to try instead), or
 * sent. The words themselves live in the composer's own state; this only
 * decides what the composer shows and which presses count, so a double press
 * cannot send twice and a refusal cannot be lost behind a retry that errors.
 */

/** What the composer needs to know about a reply the server returned. */
export type ReplyOutcome = Pick<
  InboundReply,
  "id" | "status" | "failureText" | "suggestedAddress" | "toAddress"
>;

/** A provider's refusal: why, where it was going, and where it may go instead. */
export type ComposerFailure = {
  failureText: string;
  /** Another address at the sender likely to accept it; null when none is known. */
  suggestedAddress: string | null;
  attemptedTo: string;
};

export type ComposerState =
  | { phase: "writing"; error: string | null }
  | { phase: "drafting" }
  | {
      phase: "sending";
      /** The address it goes to instead of the sender, on a retry. */
      to: string | null;
      /** The refusal this send retries, so an error puts it back. */
      retrying: ComposerFailure | null;
    }
  | { phase: "failed"; failure: ComposerFailure; error: string | null }
  | { phase: "sent"; replyId: string };

export type ComposerEvent =
  | { type: "draftRequested" }
  | { type: "draftReady" }
  | { type: "draftFailed"; error: string }
  | { type: "send"; to: string | null }
  | { type: "sendSettled"; reply: ReplyOutcome }
  | { type: "sendErrored"; error: string }
  /** The message was read again (a realtime update, a refetch): its latest sent or refused reply. */
  | { type: "synced"; latest: ReplyOutcome | null };

const WRITING: ComposerState = { phase: "writing", error: null };

function failureOf(reply: ReplyOutcome): ComposerFailure {
  return {
    failureText: reply.failureText,
    suggestedAddress: reply.suggestedAddress,
    attemptedTo: reply.toAddress,
  };
}

/** Where a reply the server holds puts the composer. */
function stateOf(reply: ReplyOutcome, retrying: ComposerFailure | null): ComposerState {
  switch (reply.status) {
    case "Sent":
      return { phase: "sent", replyId: reply.id };
    case "Failed":
      return { phase: "failed", failure: failureOf(reply), error: null };
    case "Sending":
      return { phase: "sending", to: reply.toAddress, retrying };
    case "Draft":
      return WRITING;
  }
}

/** The composer as it opens: blank, or where the message's latest reply left it. */
export function initialComposer(latest: ReplyOutcome | null): ComposerState {
  return latest === null ? WRITING : stateOf(latest, null);
}

export function composerReducer(state: ComposerState, event: ComposerEvent): ComposerState {
  switch (event.type) {
    case "draftRequested":
      return state.phase === "writing" || state.phase === "failed" ? { phase: "drafting" } : state;
    case "draftReady":
      return state.phase === "drafting" ? WRITING : state;
    case "draftFailed":
      return state.phase === "drafting" ? { phase: "writing", error: event.error } : state;
    case "send":
      if (state.phase === "writing") {
        return { phase: "sending", to: event.to, retrying: null };
      }
      if (state.phase === "failed") {
        return { phase: "sending", to: event.to, retrying: state.failure };
      }
      return state;
    case "sendSettled":
      return state.phase === "sending"
        ? stateOf(event.reply, state.retrying)
        : state;
    case "sendErrored":
      if (state.phase !== "sending") {
        return state;
      }
      return state.retrying === null
        ? { phase: "writing", error: event.error }
        : { phase: "failed", failure: state.retrying, error: event.error };
    case "synced": {
      const latest = event.latest;
      if (latest === null) {
        return state;
      }
      if (latest.status === "Sent") {
        return state.phase === "sent" && state.replyId === latest.id
          ? state
          : { phase: "sent", replyId: latest.id };
      }
      if (state.phase === "sending" && latest.status === "Failed") {
        return { phase: "failed", failure: failureOf(latest), error: null };
      }
      return state;
    }
  }
}

/** Whether Send does anything: there are words, and the composer is not busy or done. */
export function canSend(state: ComposerState, body: string): boolean {
  return (state.phase === "writing" || state.phase === "failed") && body.trim() !== "";
}
