import type { InboundActivity, InboundActivityKind } from "@/lib/graphql/inbox";
import { translate } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import { activityLine, type ActivityContext } from "../activity-copy";

const DANA = {
  id: "usr_dana",
  name: "Dana Whitfield",
  emailAddress: "dana@trenova.example",
  thumbnailUrl: "",
};
const ME = { id: "usr_me", name: "System Administrator", emailAddress: "me@x", thumbnailUrl: "" };

const context: ActivityContext = {
  meId: ME.id,
  format: { time: (seconds) => `t${seconds}`, when: (seconds) => `w${seconds}` },
};

function activity(overrides: Partial<InboundActivity> = {}): InboundActivity {
  return {
    id: "iact_1",
    kind: "Handled",
    actorKind: "Person",
    actor: DANA,
    subject: "",
    detail: "",
    until: null,
    fromClassification: null,
    toClassification: null,
    recordId: null,
    createdAt: 1_000,
    ...overrides,
  };
}

const line = (overrides: Partial<InboundActivity>) =>
  activityLine(translate, activity(overrides), context);

describe("activityLine", () => {
  it("says who did it: you, a teammate by name, or Nova", () => {
    expect(line({ kind: "Reopened", actor: ME })).toBe("You reopened it");
    expect(line({ kind: "Reopened" })).toBe("Dana Whitfield reopened it");
    expect(line({ kind: "ReplyDrafted", actorKind: "Nova", actor: null })).toBe(
      "Nova drafted a reply",
    );
  });

  it("names a person who is no longer visible as a teammate rather than leaving a blank", () => {
    expect(line({ kind: "Reopened", actor: null })).toBe("A teammate reopened it");
  });

  it("words an assignment by who it went to, and you when it is you", () => {
    expect(line({ kind: "Assigned", subject: "Marcus Lee", recordId: "usr_marcus" })).toBe(
      "Dana Whitfield assigned it to Marcus Lee",
    );
    expect(line({ kind: "Assigned", subject: ME.name, recordId: ME.id })).toBe(
      "Dana Whitfield assigned it to you",
    );
    expect(
      line({ kind: "Assigned", actorKind: "System", actor: null, subject: "Marcus Lee" }),
    ).toBe("Assigned to Marcus Lee");
    expect(line({ kind: "Assigned", actor: ME, subject: ME.name, recordId: ME.id })).toBe(
      "You took it",
    );
    expect(line({ kind: "Assigned", subject: DANA.name, recordId: DANA.id })).toBe(
      "Dana Whitfield took it",
    );
    expect(line({ kind: "Unassigned", subject: "Marcus Lee" })).toBe(
      "Dana Whitfield took it off Marcus Lee",
    );
  });

  it("says what a snooze waits for: a time or an event", () => {
    expect(line({ kind: "Snoozed", until: 5_000 })).toBe("Dana Whitfield snoozed it until w5000");
    expect(line({ kind: "Snoozed", subject: "ShipmentDelivered" })).toBe(
      "Dana Whitfield snoozed it until the shipment delivers",
    );
    expect(
      line({ kind: "Snoozed", subject: "DocumentAttached", detail: "Proof of delivery" }),
    ).toBe("Dana Whitfield snoozed it until a proof of delivery is attached");
    expect(line({ kind: "Snoozed", subject: "DocumentAttached" })).toBe(
      "Dana Whitfield snoozed it until a document is attached",
    );
    expect(line({ kind: "Snoozed", subject: "InvoicePaid" })).toBe(
      "Dana Whitfield snoozed it until the invoice is paid",
    );
  });

  it("says why a snoozed message came back", () => {
    expect(line({ kind: "Woken" })).toBe("Dana Whitfield woke it");
    expect(line({ kind: "Woken", actorKind: "System", actor: null })).toBe("Back from snooze");
    expect(
      line({ kind: "Woken", actorKind: "System", actor: null, subject: "ShipmentDelivered" }),
    ).toBe("Back: the shipment delivered");
    expect(line({ kind: "Woken", actorKind: "System", actor: null, subject: "InvoicePaid" })).toBe(
      "Back: the invoice was paid",
    );
    expect(
      line({ kind: "Woken", actorKind: "System", actor: null, subject: "DocumentAttached" }),
    ).toBe("Back: a document was attached");
  });

  it("follows waiting on the sender from start to finish", () => {
    expect(line({ kind: "WaitingOnReply", subject: "luis@drivers.example" })).toBe(
      "Dana Whitfield is waiting on luis@drivers.example to reply",
    );
    expect(line({ kind: "WaitingOnReply", actor: ME, subject: "luis@drivers.example" })).toBe(
      "You're waiting on luis@drivers.example to reply",
    );
    expect(line({ kind: "StoppedWaiting" })).toBe("Dana Whitfield stopped waiting");
    expect(
      line({
        kind: "StoppedWaiting",
        actorKind: "System",
        actor: null,
        subject: "luis@drivers.example",
        detail: "no reply",
      }),
    ).toBe("No reply from luis@drivers.example — back in the queue");
    expect(
      line({ kind: "TheyReplied", actorKind: "System", actor: null, subject: "Luis Ortega" }),
    ).toBe("Luis Ortega replied");
  });

  it("says where a reply went, and that one did not go", () => {
    expect(line({ kind: "Replied", actor: ME, subject: "intake@trenova.app", detail: "" })).toBe(
      "You replied from intake@trenova.app",
    );
    expect(
      line({
        kind: "Replied",
        actor: ME,
        subject: "billing@trenova.app",
        detail: "ELD_SEED-PAY-002_Oct3.pdf, BOL.pdf",
      }),
    ).toBe("You replied from billing@trenova.app with ELD_SEED-PAY-002_Oct3.pdf, BOL.pdf");
    expect(line({ kind: "Replied", actorKind: "Nova", actor: null, detail: "ELD.pdf" })).toBe(
      "Nova replied with ELD.pdf",
    );
    expect(
      line({ kind: "ReplyFailed", actor: ME, subject: "edi@partner.example", detail: "550" }),
    ).toBe("Your reply to edi@partner.example didn't send");
    expect(line({ kind: "ReplyFailed", subject: "edi@partner.example", detail: "550" })).toBe(
      "Dana Whitfield's reply to edi@partner.example didn't send",
    );
  });

  it("words linking, filing and splitting by the record and the file", () => {
    expect(line({ kind: "Linked", subject: "The PRO in the subject matches" })).toBe(
      "Dana Whitfield linked it: The PRO in the subject matches",
    );
    expect(line({ kind: "Linked" })).toBe("Dana Whitfield linked it");
    expect(line({ kind: "Unlinked", subject: "Shipment" })).toBe(
      "Dana Whitfield unlinked the shipment",
    );
    expect(line({ kind: "Unlinked", subject: "Customer" })).toBe(
      "Dana Whitfield unlinked the customer",
    );
    expect(line({ kind: "Unlinked", subject: "Carrier" })).toBe(
      "Dana Whitfield unlinked the carrier",
    );
    expect(line({ kind: "Filed", subject: "POD.pdf", detail: "Proof of delivery" })).toBe(
      "Dana Whitfield filed POD.pdf as proof of delivery",
    );
    expect(line({ kind: "Filed", subject: "POD.pdf" })).toBe("Dana Whitfield filed POD.pdf");
    expect(line({ kind: "Unfiled", subject: "POD.pdf" })).toBe("Dana Whitfield unfiled POD.pdf");
    expect(line({ kind: "Split", subject: "tender.pdf", detail: "3 documents" })).toBe(
      "Dana Whitfield split tender.pdf into 3 documents",
    );
  });

  it("words a correction by the kinds, and a flagged fact by its label", () => {
    expect(
      line({
        kind: "Corrected",
        fromClassification: "Tender",
        toClassification: "RateConfirmation",
      }),
    ).toBe("Dana Whitfield corrected it: tender → rate confirmation");
    expect(line({ kind: "Corrected", toClassification: "Invoice" })).toBe(
      "Dana Whitfield corrected it to invoice",
    );
    expect(line({ kind: "FactFlagged", subject: "Pickup", detail: "Thu" })).toBe(
      "Dana Whitfield flagged “Pickup” as wrong",
    );
  });

  it("settles with the note when there is one", () => {
    expect(line({ kind: "Handled", actor: ME })).toBe("You marked it handled");
    expect(line({ kind: "Handled", detail: "Called receiving" })).toBe(
      "Dana Whitfield marked it handled — Called receiving",
    );
    expect(line({ kind: "Ignored" })).toBe("Dana Whitfield ignored it");
    expect(line({ kind: "Ignored", subject: "Blocked the sender" })).toBe(
      "Dana Whitfield ignored it — Blocked the sender",
    );
  });

  it("words the work Nova and people did on the load, the exception, the sender and the stops", () => {
    expect(
      line({ kind: "LoadDrafted", actorKind: "Nova", actor: null, subject: "tender_4471.pdf" }),
    ).toBe("Nova drafted the load from tender_4471.pdf");
    expect(line({ kind: "LoadDrafted", actorKind: "Nova", actor: null })).toBe(
      "Nova drafted the load",
    );
    expect(line({ kind: "LoadCreated", subject: "S-104260" })).toBe(
      "Dana Whitfield created S-104260",
    );
    expect(line({ kind: "ExceptionLogged", subject: "2 pallets short" })).toBe(
      "Dana Whitfield logged 2 pallets short",
    );
    expect(line({ kind: "SenderBlocked", subject: "phish.example" })).toBe(
      "Dana Whitfield blocked phish.example",
    );
    expect(line({ kind: "StopsDecided", subject: "Waive", detail: "185.00 USD" })).toBe(
      "Dana Whitfield waived the detention (185.00 USD)",
    );
    expect(line({ kind: "StopsDecided", subject: "Waive" })).toBe(
      "Dana Whitfield waived the detention",
    );
    expect(line({ kind: "StopsDecided", subject: "StandBy" })).toBe(
      "Dana Whitfield stood by the detention charge",
    );
  });

  it("says how a conflict was settled", () => {
    const resolved = (subject: string, detail: string) =>
      line({ kind: "ConflictResolved", subject, detail });
    expect(resolved("DuplicateDocument", "ReplaceExisting")).toBe(
      "Dana Whitfield replaced the document on file",
    );
    expect(resolved("DuplicateDocument", "KeepBoth")).toBe("Dana Whitfield kept both documents");
    expect(resolved("OverlappingLoad", "SkipDuplicate")).toBe(
      "Dana Whitfield skipped the load already on the board",
    );
    expect(resolved("OverlappingLoad", "ConfirmNew")).toBe("Dana Whitfield said it's a new load");
    expect(resolved("OverlappingLoad", "")).toBe("Dana Whitfield settled the conflict");
    expect(line({ kind: "ConflictReopened" })).toBe("Dana Whitfield reopened the conflict");
  });

  it("words hand-offs, rules, due times, tests and Nova's reading", () => {
    expect(
      line({ kind: "HandedOff", actorKind: "System", actor: null, subject: "Marcus Lee" }),
    ).toBe("Handed off to Marcus Lee");
    expect(line({ kind: "HandedOff", subject: "Marcus Lee" })).toBe(
      "Dana Whitfield handed it off to Marcus Lee",
    );
    expect(
      line({ kind: "RuleApplied", actorKind: "Nova", actor: null, subject: "FileOnShipment" }),
    ).toBe("A rule had Nova file it on the shipment");
    expect(line({ kind: "RuleCreated", subject: "Ignore", detail: "news.example" })).toBe(
      "Dana Whitfield made a rule for news.example: Nova should ignore it",
    );
    expect(line({ kind: "RuleCreated", subject: "Ignore" })).toBe(
      "Dana Whitfield made a rule: Nova should ignore it",
    );
    expect(line({ kind: "DueChanged", actorKind: "System", actor: null, until: 9_000 })).toBe(
      "Now due w9000",
    );
    expect(line({ kind: "DueChanged", actorKind: "System", actor: null })).toBe(
      "No longer has a due time",
    );
    expect(
      line({ kind: "TestReceived", actorKind: "System", actor: null, subject: "claims@x" }),
    ).toBe("Test email arrived on claims@x");
    expect(
      line({ kind: "NovaRead", actorKind: "Nova", actor: null, toClassification: "Tender" }),
    ).toBe("Nova read it as a tender");
    expect(line({ kind: "NovaRead", actorKind: "Nova", actor: null })).toBe("Nova read it");
  });

  it("falls back to the raw action name for a rule action it does not know", () => {
    expect(
      line({ kind: "RuleApplied", actorKind: "Nova", actor: null, subject: "SomethingNew" }),
    ).toBe("A rule acted on it");
  });

  it("has a sentence for every kind the schema has", () => {
    const kinds: InboundActivityKind[] = [
      "Assigned",
      "Unassigned",
      "Snoozed",
      "Woken",
      "WaitingOnReply",
      "StoppedWaiting",
      "TheyReplied",
      "ReplyDrafted",
      "Replied",
      "ReplyFailed",
      "Linked",
      "Unlinked",
      "Filed",
      "Unfiled",
      "Split",
      "Corrected",
      "FactFlagged",
      "Handled",
      "Ignored",
      "Reopened",
      "LoadDrafted",
      "LoadCreated",
      "ExceptionLogged",
      "SenderBlocked",
      "StopsDecided",
      "ConflictResolved",
      "ConflictReopened",
      "HandedOff",
      "RuleApplied",
      "RuleCreated",
      "DueChanged",
      "TestReceived",
      "NovaRead",
    ];
    for (const kind of kinds) {
      const text = line({ kind });
      expect(text, kind).not.toBe("");
      expect(text, kind).not.toMatch(/\{\d\}|undefined|null/);
    }
  });
});
