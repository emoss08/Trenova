import type {
  InboundMessageRow,
  InboundSnooze,
  TriageAction,
  TriageExceptionKind,
  TriageReplyPurpose,
  TriageRuleAction,
} from "@/lib/graphql/inbox";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { formatUnixInUserTimezone, formatUnixTime } from "@trenova/shared/lib/date";
import { toSentenceFragment } from "@trenova/shared/lib/utils";
import { attachmentKindLabel } from "./classification";
import { senderFirstName } from "@/lib/triage/message-state";

/**
 * What a message's steps are called, everywhere they are shown.
 *
 * The server works out the steps (`TriageAction`) from the message's state, so
 * the list row, the reader's checklist, the floating action bar and the
 * briefing deck all agree on what is left. This module is the one place that
 * turns a step into words: a title, the line under it, and the label of the
 * button that does it. Nothing else writes copy for a step.
 */
export type ActionCopy = {
  /** "File RateCon_S-104251_signed", "Confirm you'll take it". */
  title: string;
  /** The quiet line under the title: what it is about, or what came of it. */
  detail: string;
  /** The button that does the step: "File", "Have Nova draft it". */
  doLabel: string;
};

/** Formatters the copy needs, passed in so the words follow the reader's time zone. */
export type CopyFormatters = {
  /** A time of day: "4:10 PM". */
  time: (unixSeconds: number) => string;
  /** A day and time within the week: "Thu 8:00 AM". */
  when: (unixSeconds: number) => string;
};

/** The formatters in the signed-in person's time zone and clock. */
export const USER_FORMATTERS: CopyFormatters = {
  time: (unixSeconds) => formatUnixTime(unixSeconds, { fallback: "" }),
  when: (unixSeconds) =>
    formatUnixInUserTimezone(
      unixSeconds,
      { weekday: "short", hour: "numeric", minute: "2-digit" },
      "",
    ),
};

type SenderFields = Pick<InboundMessageRow, "fromName" | "fromAddress">;

const EXTENSION = /\.[a-z0-9]{2,5}$/i;

function withoutExtension(fileName: string): string {
  return fileName.replace(EXTENSION, "");
}

function documentType(t: TranslateFn, action: TriageAction): string {
  const attachment = action.attachment;
  if (attachment === null) {
    return toSentenceFragment(t("Document"));
  }
  const filedAs = attachment.filedAs?.trim() ?? "";

  return toSentenceFragment(filedAs !== "" ? filedAs : attachmentKindLabel(t, attachment.kind));
}

function fileCopy(t: TranslateFn, action: TriageAction): ActionCopy {
  const fileName = action.attachment?.fileName ?? "";
  const type = documentType(t, action);
  const pro = action.shipment?.proNumber ?? "";
  let detail: string;
  if (action.done) {
    detail = pro === "" ? t("Filed as {0}", type) : t("Filed on {0} as {1}", pro, type);
  } else {
    detail = pro === "" ? t("As {0}", type) : t("As {0} on {1}", type, pro);
  }

  return {
    title: fileName === "" ? t("File the attachment") : t("File {0}", withoutExtension(fileName)),
    detail,
    doLabel: t("File"),
  };
}

function loadCopy(t: TranslateFn, action: TriageAction): ActionCopy {
  let detail: string;
  if (action.createdShipment !== null) {
    detail = t("{0} created", action.createdShipment.proNumber);
  } else if (action.loadDrafted) {
    detail = t("Draft ready to check");
  } else if (action.attachment !== null) {
    detail = t("Read from {0}", action.attachment.fileName);
  } else {
    detail = t("From the email text");
  }

  let doLabel: string;
  if (action.loadDrafted) {
    doLabel = t("Review the draft");
  } else if (action.novaCanDo) {
    doLabel = t("Have Nova build it");
  } else {
    doLabel = t("Build it");
  }

  return { title: t("Build the load"), detail, doLabel };
}

function replyTitle(
  t: TranslateFn,
  purpose: TriageReplyPurpose | null,
  first: string,
  refused: string,
): string {
  switch (purpose) {
    case "ConfirmTender":
      return t("Confirm you'll take it");
    case "AcceptTender":
      return t("Accept the tender");
    case "AnswerDispute":
      return t("Answer {0}", first);
    case "ReturnSigned":
      return t("Return it signed");
    case "AskResend":
      return refused === ""
        ? t("Ask {0} to resend the file", first)
        : t("Ask {0} to resend {1}", first, withoutExtension(refused));
    case "SendEta":
      return t("Send {0} the ETA", first);
    case null:
      return t("Reply to {0}", first);
  }
}

function replyHint(t: TranslateFn, purpose: TriageReplyPurpose | null, first: string): string {
  switch (purpose) {
    case "ConfirmTender":
      return t("{0} is waiting on a yes", first);
    case "AcceptTender":
      return t("Tell {0} you'll haul it", first);
    case "AnswerDispute":
      return t("Waive it or send the log");
    case "ReturnSigned":
      return t("Sign the rate confirmation and send it back");
    case "AskResend":
      return t("The file couldn't be read");
    case "SendEta":
      return t("Look up the load and answer");
    case null:
      return t("Write back from the source's address");
  }
}

function replyCopy(
  t: TranslateFn,
  action: TriageAction,
  sender: SenderFields,
  format: CopyFormatters,
): ActionCopy {
  const first = senderFirstName(sender);
  const reply = action.reply;
  let detail: string;
  if (reply === null) {
    detail = replyHint(t, action.replyPurpose, first);
  } else {
    switch (reply.status) {
      case "Sent":
        detail = reply.sentAt === null ? t("Sent") : t("Sent {0}", format.time(reply.sentAt));
        break;
      case "Sending":
        detail = t("Sending…");
        break;
      case "Failed":
        detail = t("Didn't send — try another address");
        break;
      case "Draft":
        detail = reply.draftedByNova ? t("Nova's draft is ready") : t("Draft ready");
        break;
    }
  }

  let doLabel: string;
  if (reply?.status === "Draft" || reply?.status === "Failed") {
    doLabel = t("Review reply");
  } else if (action.novaCanDo) {
    doLabel = t("Have Nova draft it");
  } else {
    doLabel = t("Write the reply");
  }

  return {
    title: replyTitle(t, action.replyPurpose, first, action.refusedFileName ?? ""),
    detail,
    doLabel,
  };
}

function stopsCopy(t: TranslateFn, action: TriageAction): ActionCopy {
  let detail: string;
  switch (action.stopsDecision) {
    case "Waive":
      detail = t("Waiving the detention");
      break;
    case "StandBy":
      detail = t("Standing by the charge");
      break;
    case null:
      detail = t("Before you answer the dispute");
      break;
  }

  return { title: t("Check the stop times"), detail, doLabel: t("Check the times") };
}

function exceptionTitle(t: TranslateFn, kind: TriageExceptionKind | null): string {
  switch (kind) {
    case "Shortage":
      return t("Log the shortage");
    case "Overage":
      return t("Log the overage");
    case "Damage":
      return t("Log the damage");
    case null:
      return t("Log an exception");
  }
}

function exceptionCopy(t: TranslateFn, action: TriageAction): ActionCopy {
  const pro = action.shipment?.proNumber ?? "";
  let detail: string;
  if (action.done) {
    detail = pro === "" ? t("Exception logged") : t("Exception on {0}", pro);
  } else {
    detail = t("Noted on the delivery");
  }

  return { title: exceptionTitle(t, action.exceptionKind), detail, doLabel: t("Log exception") };
}

function blockCopy(t: TranslateFn, action: TriageAction): ActionCopy {
  return {
    title: t("Block the sender"),
    detail: action.done
      ? t("Blocked — future mail goes to Ignored")
      : t("Future mail goes straight to Ignored"),
    doLabel: t("Block and ignore"),
  };
}

function linkCopy(t: TranslateFn, action: TriageAction): ActionCopy {
  let detail: string;
  if (!action.done) {
    detail = t("Nova couldn't tell");
  } else if (action.shipment !== null) {
    detail = t("Linked to {0}", action.shipment.proNumber);
  } else {
    detail = t("Linked");
  }

  return { title: t("Say which record it's about"), detail, doLabel: t("Link by hand") };
}

export function actionCopy(
  t: TranslateFn,
  action: TriageAction,
  sender: SenderFields,
  format: CopyFormatters,
): ActionCopy {
  switch (action.kind) {
    case "File":
      return fileCopy(t, action);
    case "BuildLoad":
      return loadCopy(t, action);
    case "Reply":
      return replyCopy(t, action, sender, format);
    case "CheckStops":
      return stopsCopy(t, action);
    case "LogException":
      return exceptionCopy(t, action);
    case "BlockSender":
      return blockCopy(t, action);
    case "Link":
      return linkCopy(t, action);
  }
}

/** The first step still open, in the server's order: what "do the next step" does. */
export function nextOpenAction(actions: readonly TriageAction[]): TriageAction | null {
  return actions.find((action) => !action.done) ?? null;
}

/** "until S-104198 delivers", "Thu 8:00 AM". */
export function snoozeText(t: TranslateFn, snooze: InboundSnooze, format: CopyFormatters): string {
  switch (snooze.event) {
    case "ShipmentDelivered":
      return snooze.watchShipment === null
        ? t("until the shipment delivers")
        : t("until {0} delivers", snooze.watchShipment.proNumber);
    case "DocumentAttached":
      return snooze.documentTypeName === null || snooze.documentTypeName.trim() === ""
        ? t("until a document is attached")
        : t("until a {0} is attached", toSentenceFragment(snooze.documentTypeName));
    case "InvoicePaid":
      return t("until the invoice is paid");
    case null:
      return snooze.until === null ? t("for now") : format.when(snooze.until);
  }
}

/** How a row's last line reads, and in which voice. */
export type RowLineTone = "muted" | "danger" | "nova";

export type RowLine = { text: string; tone: RowLineTone };

export type RowLineMessage = Pick<
  InboundMessageRow,
  | "status"
  | "needsReview"
  | "novaRead"
  | "snooze"
  | "awaitingReply"
  | "actions"
  | "fromName"
  | "fromAddress"
>;

/**
 * The line under a row's subject in "next step" mode: why the message is
 * where it is, or the next thing to do with it and how many follow. Null for
 * a settled message, whose row shows its first words instead.
 */
export function rowLine(
  t: TranslateFn,
  message: RowLineMessage,
  format: CopyFormatters,
): RowLine | null {
  if (!message.needsReview) {
    return null;
  }
  if (!message.novaRead) {
    return { text: t("Not read — Nova is off"), tone: "muted" };
  }
  if (message.status === "Quarantined") {
    return { text: t("Held back — looks like phishing"), tone: "danger" };
  }
  if (message.awaitingReply !== null) {
    return { text: t("Waiting on {0} to reply", senderFirstName(message)), tone: "muted" };
  }
  if (message.snooze !== null) {
    return { text: t("Snoozed · {0}", snoozeText(t, message.snooze, format)), tone: "muted" };
  }

  const open = message.actions.filter((action) => !action.done);
  if (open.length === 0) {
    return message.actions.length === 0
      ? null
      : { text: t("All done · mark handled"), tone: "muted" };
  }

  const next = open[0];
  const title = actionCopy(t, next, message, format).title;

  return {
    text: open.length > 1 ? t("{0} · +{1}", title, open.length - 1) : title,
    tone: next.novaCanDo ? "nova" : "muted",
  };
}

/**
 * What a rule tells Nova to do, as the end of the sentence "Nova should …":
 * "file it on the shipment". The Rules page, the rule menus and the briefing's
 * suggestion all read it here.
 */
export function ruleActionPhrase(t: TranslateFn, action: TriageRuleAction): string {
  switch (action) {
    case "FileOnShipment":
      return t("file it on the shipment");
    case "FileAndQueuePay":
      return t("file it and queue it for pay");
    case "DraftReply":
      return t("draft a reply for you to check");
    case "DraftLoad":
      return t("draft the load");
    case "LinkBySender":
      return t("link it by sender");
    case "Ignore":
      return t("ignore it");
    case "Assign":
      return t("hand it to someone");
  }
}

/** The same phrase on its own, as a menu row: "File it on the shipment". */
export function ruleActionLabel(t: TranslateFn, action: TriageRuleAction): string {
  switch (action) {
    case "FileOnShipment":
      return t("File it on the shipment");
    case "FileAndQueuePay":
      return t("File it and queue it for pay");
    case "DraftReply":
      return t("Draft a reply for you to check");
    case "DraftLoad":
      return t("Draft the load");
    case "LinkBySender":
      return t("Link it by sender");
    case "Ignore":
      return t("Ignore it");
    case "Assign":
      return t("Hand it to someone");
  }
}
