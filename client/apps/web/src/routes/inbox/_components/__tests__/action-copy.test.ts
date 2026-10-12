import { action, messageRow, SHIPMENT } from "@/lib/triage/__tests__/message-fixture";
import { translate } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import {
  actionCopy,
  nextOpenAction,
  ruleActionPhrase,
  rowLine,
  snoozeText,
  type CopyFormatters,
} from "../action-copy";

const format: CopyFormatters = {
  time: (seconds) => `t${seconds}`,
  when: (seconds) => `w${seconds}`,
};
const sender = { fromName: "Luis Ortega", fromAddress: "luis@drivers.example" };
const attachment = {
  id: "att_1",
  fileName: "IMG_4410.jpg",
  kind: "ProofOfDelivery" as const,
  documentId: "doc_1",
  filedShipmentId: null,
  filedAs: null,
  pageCount: 1,
};

describe("actionCopy", () => {
  it("names a file step by the file, and says where it goes until it is filed", () => {
    const open = actionCopy(translate, action({ id: "file:att_1", kind: "File", attachment, shipment: SHIPMENT }), sender, format);
    expect(open).toEqual({ title: "File IMG_4410", detail: "As proof of delivery on S-104198", doLabel: "File" });

    const filed = actionCopy(
      translate,
      action({ kind: "File", done: true, attachment: { ...attachment, filedAs: "Signed POD" }, shipment: SHIPMENT }),
      sender,
      format,
    );
    expect(filed.detail).toBe("Filed on S-104198 as signed POD");
  });

  it("words a reply by what it is for, by name, and by where it stands", () => {
    expect(actionCopy(translate, action({ kind: "Reply", replyPurpose: "SendEta", novaCanDo: true }), sender, format)).toEqual({
      title: "Send Luis the ETA",
      detail: "Look up the load and answer",
      doLabel: "Have Nova draft it",
    });
    expect(
      actionCopy(translate, action({ kind: "Reply", replyPurpose: "AskResend", refusedFileName: "IMG_4411.jpg" }), sender, format)
        .title,
    ).toBe("Ask Luis to resend IMG_4411");
    expect(
      actionCopy(
        translate,
        action({ kind: "Reply", replyPurpose: "ConfirmTender", reply: { id: "r", status: "Sent", sentAt: 5, draftedByNova: true } }),
        sender,
        format,
      ).detail,
    ).toBe("Sent t5");
    expect(
      actionCopy(
        translate,
        action({ kind: "Reply", replyPurpose: null, reply: { id: "r", status: "Draft", sentAt: null, draftedByNova: true } }),
        sender,
        format,
      ),
    ).toEqual({ title: "Reply to Luis", detail: "Nova's draft is ready", doLabel: "Review reply" });
  });

  it("says what came of a load, a stops check, an exception, a block and a link", () => {
    expect(actionCopy(translate, action({ kind: "BuildLoad", loadDrafted: true }), sender, format)).toMatchObject({
      detail: "Draft ready to check",
      doLabel: "Review the draft",
    });
    expect(actionCopy(translate, action({ kind: "BuildLoad", createdShipment: SHIPMENT }), sender, format).detail).toBe(
      "S-104198 created",
    );
    expect(actionCopy(translate, action({ kind: "CheckStops", stopsDecision: "Waive" }), sender, format).detail).toBe(
      "Waiving the detention",
    );
    expect(
      actionCopy(translate, action({ kind: "LogException", exceptionKind: "Shortage", done: true, shipment: SHIPMENT }), sender, format),
    ).toMatchObject({ title: "Log the shortage", detail: "Exception on S-104198" });
    expect(actionCopy(translate, action({ kind: "BlockSender" }), sender, format).doLabel).toBe("Block and ignore");
    expect(actionCopy(translate, action({ kind: "Link", done: true, shipment: SHIPMENT }), sender, format).detail).toBe(
      "Linked to S-104198",
    );
  });
});

describe("rowLine", () => {
  it("leads with the first open step and counts the rest", () => {
    const message = messageRow({
      ...sender,
      actions: [
        action({ id: "file", kind: "File", done: true, attachment }),
        action({ id: "load", kind: "BuildLoad" }),
        action({ id: "reply", kind: "Reply", replyPurpose: "ConfirmTender" }),
      ],
    });
    expect(rowLine(translate, message, format)).toEqual({ text: "Build the load · +1", tone: "muted" });
    expect(nextOpenAction(message.actions)?.id).toBe("load");
  });

  it("says why a message is where it is before what to do with it", () => {
    expect(rowLine(translate, messageRow({ status: "Quarantined" }), format)).toEqual({
      text: "Held back — looks like phishing",
      tone: "danger",
    });
    expect(rowLine(translate, messageRow({ novaRead: false }), format)?.text).toBe("Not read — Nova is off");
    expect(rowLine(translate, messageRow({ ...sender, awaitingReply: { since: 1, until: 2 } }), format)?.text).toBe(
      "Waiting on Luis to reply",
    );
    expect(
      rowLine(
        translate,
        messageRow({
          snooze: { until: 99, event: null, watchShipment: null, documentTypeId: null, documentTypeName: null, snoozedAt: 1, snoozedBy: null },
        }),
        format,
      )?.text,
    ).toBe("Snoozed · w99");
  });

  it("asks to mark handled once every step is done, and says nothing about a settled message", () => {
    expect(rowLine(translate, messageRow({ actions: [action({ done: true })] }), format)?.text).toBe("All done · mark handled");
    expect(rowLine(translate, messageRow({ status: "Actioned", needsReview: false, actions: [action()] }), format)).toBeNull();
    expect(rowLine(translate, messageRow({ actions: [] }), format)).toBeNull();
  });

  it("marks a step Nova can do in Nova's voice", () => {
    expect(rowLine(translate, messageRow({ actions: [action({ kind: "Reply", novaCanDo: true })] }), format)?.tone).toBe("nova");
  });
});

describe("snoozeText", () => {
  it("names the event a message waits on", () => {
    const base = { until: null, watchShipment: null, documentTypeId: null, documentTypeName: null, snoozedAt: 1, snoozedBy: null };
    expect(snoozeText(translate, { ...base, event: "ShipmentDelivered", watchShipment: SHIPMENT }, format)).toBe(
      "until S-104198 delivers",
    );
    expect(snoozeText(translate, { ...base, event: "DocumentAttached", documentTypeName: "Rate Confirmation" }, format)).toBe(
      "until a rate confirmation is attached",
    );
    expect(snoozeText(translate, { ...base, event: "InvoicePaid" }, format)).toBe("until the invoice is paid");
  });
});

describe("ruleActionPhrase", () => {
  it("ends the rule's sentence", () => {
    expect(ruleActionPhrase(translate, "FileOnShipment")).toBe("file it on the shipment");
  });
});
