import type { InboundActivity, InboundClassification, TriageRuleAction } from "@/lib/graphql/inbox";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { toSentenceFragment } from "@trenova/shared/lib/utils";
import { ruleActionPhrase, type CopyFormatters } from "../action-copy";
import { classificationLabel } from "../classification";

/**
 * The activity trace in words.
 *
 * The server records each line as a kind with the names it is about (a file,
 * a PRO, a person, an address) and never as prose, so every sentence is
 * written here, once, keyed by `InboundActivityKind`. The reader's trace and
 * anything else that tells a message's story read it from this module.
 */
export type ActivityContext = {
  /** The signed-in person, so their own lines read "You". */
  meId: string | null;
  format: CopyFormatters;
};

const RULE_ACTIONS: ReadonlySet<string> = new Set<TriageRuleAction>([
  "FileOnShipment",
  "FileAndQueuePay",
  "DraftReply",
  "DraftLoad",
  "LinkBySender",
  "Ignore",
  "Assign",
]);

function isRuleAction(value: string): value is TriageRuleAction {
  return RULE_ACTIONS.has(value);
}

/** Who did it, as the sentence names them; null when the system did it on its own. */
function actorName(t: TranslateFn, activity: InboundActivity, meId: string | null): string | null {
  switch (activity.actorKind) {
    case "Nova":
      return t("Nova");
    case "System":
      return null;
    case "Person":
      if (activity.actor === null) {
        return t("A teammate");
      }
      return activity.actor.id === meId ? t("You") : activity.actor.name;
  }
}

function kindFragment(t: TranslateFn, kind: InboundClassification): string {
  return toSentenceFragment(classificationLabel(t, kind));
}

function readAs(t: TranslateFn, kind: InboundClassification | null): string {
  switch (kind) {
    case "Tender":
      return t("Nova read it as a tender");
    case "RateConfirmation":
      return t("Nova read it as a rate confirmation");
    case "ProofOfDelivery":
      return t("Nova read it as a proof of delivery");
    case "Invoice":
      return t("Nova read it as an invoice");
    case "StatusRequest":
      return t("Nova read it as a status request");
    case "DetentionDispute":
      return t("Nova read it as a detention dispute");
    case "Other":
      return t("Nova read it as something else");
    case null:
      return t("Nova read it");
  }
}

function snoozedUntil(t: TranslateFn, activity: InboundActivity, format: CopyFormatters): string {
  switch (activity.subject) {
    case "ShipmentDelivered":
      return t("until the shipment delivers");
    case "DocumentAttached":
      return activity.detail.trim() === ""
        ? t("until a document is attached")
        : t("until a {0} is attached", toSentenceFragment(activity.detail.trim()));
    case "InvoicePaid":
      return t("until the invoice is paid");
    default:
      return activity.until === null ? t("for now") : t("until {0}", format.when(activity.until));
  }
}

function wokenBy(t: TranslateFn, event: string): string {
  switch (event) {
    case "ShipmentDelivered":
      return t("Back: the shipment delivered");
    case "DocumentAttached":
      return t("Back: a document was attached");
    case "InvoicePaid":
      return t("Back: the invoice was paid");
    default:
      return t("Back from snooze");
  }
}

function unlinkedRecord(t: TranslateFn, record: string): string {
  switch (record) {
    case "Customer":
      return t("the customer");
    case "Carrier":
      return t("the carrier");
    default:
      return t("the shipment");
  }
}

function conflictOutcome(t: TranslateFn, actor: string, resolution: string): string {
  switch (resolution) {
    case "ReplaceExisting":
      return t("{0} replaced the document on file", actor);
    case "KeepBoth":
      return t("{0} kept both documents", actor);
    case "SkipDuplicate":
      return t("{0} skipped the load already on the board", actor);
    case "ConfirmNew":
      return t("{0} said it's a new load", actor);
    default:
      return t("{0} settled the conflict", actor);
  }
}

/** One line of a message's activity trace, as a sentence. */
export function activityLine(
  t: TranslateFn,
  activity: InboundActivity,
  { meId, format }: ActivityContext,
): string {
  const actor = actorName(t, activity, meId);
  const subject = activity.subject.trim();
  const detail = activity.detail.trim();
  const by = actor ?? t("Someone");

  switch (activity.kind) {
    case "Assigned": {
      if (
        activity.actorKind === "Person" &&
        activity.actor !== null &&
        activity.actor.id === activity.recordId
      ) {
        return activity.actor.id === meId
          ? t("You took it")
          : t("{0} took it", activity.actor.name);
      }
      const to = activity.recordId !== null && activity.recordId === meId ? t("you") : subject;
      return actor === null ? t("Assigned to {0}", to) : t("{0} assigned it to {1}", actor, to);
    }
    case "Unassigned":
      if (subject === "") {
        return actor === null ? t("Unassigned") : t("{0} unassigned it", actor);
      }
      return actor === null
        ? t("Taken off {0}", subject)
        : t("{0} took it off {1}", actor, subject);
    case "Snoozed":
      return actor === null
        ? t("Snoozed {0}", snoozedUntil(t, activity, format))
        : t("{0} snoozed it {1}", actor, snoozedUntil(t, activity, format));
    case "Woken":
      return actor === null || activity.actorKind !== "Person"
        ? wokenBy(t, subject)
        : t("{0} woke it", actor);
    case "WaitingOnReply":
      if (actor === null) {
        return t("Waiting on {0} to reply", subject);
      }
      return activity.actorKind === "Person" && activity.actor?.id === meId
        ? t("You're waiting on {0} to reply", subject)
        : t("{0} is waiting on {1} to reply", actor, subject);
    case "StoppedWaiting":
      return actor === null
        ? t("No reply from {0} — back in the queue", subject)
        : t("{0} stopped waiting", actor);
    case "TheyReplied":
      return subject === "" ? t("They replied") : t("{0} replied", subject);
    case "ReplyDrafted":
      return t("{0} drafted a reply", by);
    case "Replied":
      if (detail !== "") {
        return subject === ""
          ? t("{0} replied with {1}", by, detail)
          : t("{0} replied from {1} with {2}", by, subject, detail);
      }
      return subject === "" ? t("{0} replied", by) : t("{0} replied from {1}", by, subject);
    case "ReplyFailed":
      if (activity.actorKind === "Person" && activity.actor?.id === meId) {
        return t("Your reply to {0} didn't send", subject);
      }
      return actor === null || activity.actorKind !== "Person"
        ? t("A reply to {0} didn't send", subject)
        : t("{0}'s reply to {1} didn't send", actor, subject);
    case "Linked":
      return subject === "" ? t("{0} linked it", by) : t("{0} linked it: {1}", by, subject);
    case "Unlinked":
      return t("{0} unlinked {1}", by, unlinkedRecord(t, subject));
    case "Filed":
      return detail === ""
        ? t("{0} filed {1}", by, subject)
        : t("{0} filed {1} as {2}", by, subject, toSentenceFragment(detail));
    case "Unfiled":
      return detail === ""
        ? t("{0} unfiled {1}", by, subject)
        : t("{0} unfiled {1}: {2}", by, subject, detail);
    case "Split":
      return t("{0} split {1} into {2}", by, subject, detail);
    case "Corrected": {
      const to = activity.toClassification;
      if (to === null) {
        return t("{0} corrected the kind", by);
      }
      return activity.fromClassification === null
        ? t("{0} corrected it to {1}", by, kindFragment(t, to))
        : t(
            "{0} corrected it: {1} → {2}",
            by,
            kindFragment(t, activity.fromClassification),
            kindFragment(t, to),
          );
    }
    case "FactFlagged":
      return t("{0} flagged “{1}” as wrong", by, subject);
    case "Handled": {
      const note = detail || subject;
      return note === ""
        ? t("{0} marked it handled", by)
        : t("{0} marked it handled — {1}", by, note);
    }
    case "Ignored": {
      const note = detail || subject;
      return note === "" ? t("{0} ignored it", by) : t("{0} ignored it — {1}", by, note);
    }
    case "Reopened":
      return t("{0} reopened it", by);
    case "LoadDrafted":
      return subject === ""
        ? t("{0} drafted the load", by)
        : t("{0} drafted the load from {1}", by, subject);
    case "LoadCreated":
      return t("{0} created {1}", by, subject);
    case "ExceptionLogged":
      return t("{0} logged {1}", by, subject);
    case "SenderBlocked":
      return t("{0} blocked {1}", by, subject);
    case "StopsDecided":
      if (subject === "StandBy") {
        return t("{0} stood by the detention charge", by);
      }
      return detail === ""
        ? t("{0} waived the detention", by)
        : t("{0} waived the detention ({1})", by, detail);
    case "ConflictResolved":
      return conflictOutcome(t, by, detail);
    case "ConflictReopened":
      return t("{0} reopened the conflict", by);
    case "HandedOff":
      return actor === null
        ? t("Handed off to {0}", subject)
        : t("{0} handed it off to {1}", actor, subject);
    case "RuleApplied":
      return isRuleAction(subject)
        ? t("A rule had Nova {0}", ruleActionPhrase(t, subject))
        : t("A rule acted on it");
    case "RuleCreated": {
      if (!isRuleAction(subject)) {
        return t("{0} made a rule", by);
      }
      const phrase = ruleActionPhrase(t, subject);
      return detail === ""
        ? t("{0} made a rule: Nova should {1}", by, phrase)
        : t("{0} made a rule for {1}: Nova should {2}", by, detail, phrase);
    }
    case "DueChanged":
      return activity.until === null
        ? t("No longer has a due time")
        : t("Now due {0}", format.when(activity.until));
    case "TestReceived":
      return subject === "" ? t("Test email arrived") : t("Test email arrived on {0}", subject);
    case "NovaRead":
      return readAs(t, activity.toClassification);
  }
}
