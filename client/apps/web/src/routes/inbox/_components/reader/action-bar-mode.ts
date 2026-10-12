import type {
  InboundClassification,
  InboundMailbox,
  InboundMessageDetail,
  TriageAction,
  TriageRuleAction,
} from "@/lib/graphql/inbox";
import { senderFirstName } from "@/lib/triage/message-state";
import { nextOpenAction } from "../action-copy";
import type { DraftKind } from "./panel-state";

/**
 * What the floating action bar shows for a message, worked out from the
 * message alone: nothing for a settled one; who it is waiting on; held back
 * as phishing; the next open step with the button that does it; or every
 * step done with what can follow. The bar draws whichever mode this returns,
 * so a message's state picks its bar and nothing else can.
 */
export type BarMessage = Pick<
  InboundMessageDetail,
  | "status"
  | "needsReview"
  | "awaitingReply"
  | "classification"
  | "novaRead"
  | "fromName"
  | "fromAddress"
  | "actions"
  | "replyDraft"
  | "loadDraft"
>;

/** The button that does the next step. */
export type PrimaryKind = "doIt" | "novaDoIt" | "reviewReply" | "reviewDraft";

export type ActionBarMode =
  | { kind: "none" }
  | { kind: "waiting"; firstName: string; until: number }
  | { kind: "held" }
  | {
      kind: "next";
      action: TriageAction;
      /** 1-based, among all the message's steps. */
      position: number;
      total: number;
      primary: PrimaryKind;
      /** Nova is drafting this very step. */
      busy: boolean;
      /** Offer "why it's waiting": only until a step is done. */
      explain: boolean;
    }
  | {
      kind: "done";
      total: number;
      /** The step, when the message asked for just one. */
      only: TriageAction | null;
      /** The rule "Do this on its own next time" would turn on. */
      teach: TriageRuleAction | null;
      /** A reply went out and the message is not yet waiting on them. */
      waitForReply: boolean;
    };

function replyInFlight(action: TriageAction): boolean {
  return action.reply?.status === "Failed" || action.reply?.status === "Sending";
}

function primaryFor(action: TriageAction, message: BarMessage): PrimaryKind {
  switch (action.kind) {
    case "Reply":
      if (message.replyDraft !== null || replyInFlight(action)) {
        return "reviewReply";
      }
      return action.novaCanDo ? "novaDoIt" : "doIt";
    case "BuildLoad":
      if (message.loadDraft !== null) {
        return "reviewDraft";
      }
      return action.novaCanDo ? "novaDoIt" : "doIt";
    case "File":
    case "CheckStops":
    case "LogException":
    case "BlockSender":
    case "Link":
      return "doIt";
  }
}

function busyFor(action: TriageAction, drafting: Readonly<Record<DraftKind, boolean>>): boolean {
  return (
    (action.kind === "BuildLoad" && drafting.load) || (action.kind === "Reply" && drafting.reply)
  );
}

/**
 * The rule "Do this on its own next time" turns on, from the first step the
 * message asked for: what a rule can do for messages like it. Nothing for a
 * message Nova never read, or a first step no rule can take.
 */
export function teachFor(
  actions: readonly TriageAction[],
  classification: InboundClassification | null,
  novaRead: boolean,
): TriageRuleAction | null {
  const first = actions[0];
  if (first === undefined || classification === null || !novaRead) {
    return null;
  }
  switch (first.kind) {
    case "File":
      return classification === "Invoice" ? "FileAndQueuePay" : "FileOnShipment";
    case "BuildLoad":
      return "DraftLoad";
    case "Reply":
    case "CheckStops":
      return "DraftReply";
    case "LogException":
    case "BlockSender":
    case "Link":
      return null;
  }
}

export function actionBarMode(
  message: BarMessage,
  drafting: Readonly<Record<DraftKind, boolean>>,
): ActionBarMode {
  if (!message.needsReview) {
    return { kind: "none" };
  }
  if (message.awaitingReply !== null) {
    return {
      kind: "waiting",
      firstName: senderFirstName(message),
      until: message.awaitingReply.until,
    };
  }

  const actions = message.actions;
  const next = nextOpenAction(actions);
  if (next !== null && message.status === "Quarantined") {
    return { kind: "held" };
  }
  if (next !== null) {
    const done = actions.filter((action) => action.done).length;
    return {
      kind: "next",
      action: next,
      position: actions.indexOf(next) + 1,
      total: actions.length,
      primary: primaryFor(next, message),
      busy: busyFor(next, drafting),
      explain: done === 0,
    };
  }

  return {
    kind: "done",
    total: actions.length,
    only: actions.length === 1 ? actions[0] : null,
    teach: teachFor(actions, message.classification, message.novaRead),
    waitForReply: actions.some(
      (action) => action.kind === "Reply" && action.reply?.status === "Sent",
    ),
  };
}

/** Why a message waits for a person, as the "?" beside the next step explains it. */
export type WhyNote =
  | "notRead"
  | "alwaysReview"
  | "loadByHand"
  | "filingWaits"
  | "replyByHand"
  | "moneyByHand";

export type WhyRow =
  | { kind: "failure"; text: string; held: boolean }
  | { kind: "confidence"; confidence: number; bar: number; under: boolean }
  | { kind: "note"; note: WhyNote; mailbox: string }
  | { kind: "match"; text: string };

export type WhyMessage = Pick<
  InboundMessageDetail,
  "failureText" | "status" | "confidence" | "matchReason" | "novaRead"
> & { mailbox: Pick<InboundMailbox, "reviewPolicy" | "minConfidence" | "name"> | null };

function noteFor(message: WhyMessage, next: TriageAction | null): WhyNote | null {
  if (!message.novaRead) {
    return "notRead";
  }
  if (message.mailbox?.reviewPolicy === "AlwaysReview") {
    return "alwaysReview";
  }
  switch (next?.kind) {
    case "BuildLoad":
      return "loadByHand";
    case "File":
      return "filingWaits";
    case "Reply":
      return "replyByHand";
    case "CheckStops":
      return "moneyByHand";
    default:
      return null;
  }
}

/**
 * The rows of the "why it's waiting" popover: why the reading failed, or how
 * sure it was against the mailbox's bar; why the next step is a person's;
 * and how the message was matched to its record.
 */
export function whyRows(message: WhyMessage, next: TriageAction | null): WhyRow[] {
  const rows: WhyRow[] = [];
  const failure = message.failureText.trim();
  if (failure !== "") {
    rows.push({ kind: "failure", text: failure, held: message.status === "Quarantined" });
  } else if (message.novaRead && message.mailbox?.reviewPolicy === "ReviewBelowConfidence") {
    rows.push({
      kind: "confidence",
      confidence: message.confidence,
      bar: message.mailbox.minConfidence,
      under: message.confidence < message.mailbox.minConfidence,
    });
  }
  if (message.mailbox !== null) {
    const note = noteFor(message, next);
    if (note !== null && failure === "") {
      rows.push({ kind: "note", note, mailbox: message.mailbox.name });
    }
  }
  const reason = message.matchReason.trim();
  if (reason !== "") {
    rows.push({ kind: "match", text: reason });
  }

  return rows;
}
