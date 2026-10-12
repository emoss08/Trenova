import type { TriageAction } from "@/lib/graphql/inbox";
import { action } from "@/lib/triage/__tests__/message-fixture";
import { describe, expect, it } from "vitest";
import {
  actionBarMode,
  teachFor,
  whyRows,
  type BarMessage,
  type WhyMessage,
} from "../action-bar-mode";

const idle = { load: false, reply: false };

const sentReply = {
  id: "rep_1",
  status: "Sent" as const,
  sentAt: 2_000,
  draftedByNova: false,
};

function message(overrides: Partial<BarMessage> = {}): BarMessage {
  return {
    status: "InReview",
    needsReview: true,
    awaitingReply: null,
    classification: "Tender",
    novaRead: true,
    fromName: "Luis Ortega",
    fromAddress: "luis@drivers.example",
    actions: [],
    replyDraft: null,
    loadDraft: null,
    ...overrides,
  };
}

const file = action({ id: "file:a", kind: "File" });
const reply = action({ id: "reply", kind: "Reply" });
const load = action({ id: "load", kind: "BuildLoad", novaCanDo: true });

describe("actionBarMode", () => {
  it("draws nothing for a settled message", () => {
    expect(actionBarMode(message({ needsReview: false, status: "Actioned" }), idle)).toEqual({
      kind: "none",
    });
  });

  it("puts waiting on the sender ahead of everything, by their first name", () => {
    expect(
      actionBarMode(
        message({ awaitingReply: { since: 1, until: 9 }, actions: [file], status: "Quarantined" }),
        idle,
      ),
    ).toEqual({ kind: "waiting", firstName: "Luis", until: 9 });
  });

  it("holds a quarantined message back while it has a step open", () => {
    const block = action({ id: "block", kind: "BlockSender" });
    expect(actionBarMode(message({ status: "Quarantined", actions: [block] }), idle)).toEqual({
      kind: "held",
    });
    expect(
      actionBarMode(message({ status: "Quarantined", actions: [{ ...block, done: true }] }), idle)
        .kind,
    ).toBe("done");
  });

  it("names the next open step, where it is in the list, and explains only before any is done", () => {
    const first = actionBarMode(message({ actions: [file, reply] }), idle);
    expect(first).toMatchObject({
      kind: "next",
      action: file,
      position: 1,
      total: 2,
      primary: "doIt",
      busy: false,
      explain: true,
    });
    const second = actionBarMode(message({ actions: [{ ...file, done: true }, reply] }), idle);
    expect(second).toMatchObject({ kind: "next", action: reply, position: 2, explain: false });
  });

  it("offers Nova for a step it can do, and review once there is a draft", () => {
    expect(actionBarMode(message({ actions: [load] }), idle)).toMatchObject({
      primary: "novaDoIt",
    });
    expect(
      actionBarMode(
        message({
          actions: [{ ...load, loadDrafted: true }],
          loadDraft: { source: "", fields: [], draftedAt: 1, draftedByNova: true },
        }),
        idle,
      ),
    ).toMatchObject({ primary: "reviewDraft" });

    const novaReply = { ...reply, novaCanDo: true };
    expect(actionBarMode(message({ actions: [novaReply] }), idle)).toMatchObject({
      primary: "novaDoIt",
    });
    expect(
      actionBarMode(
        message({
          actions: [{ ...novaReply, reply: { ...sentReply, status: "Draft" } }],
          replyDraft: {
            id: "rep_d",
            messageId: "m",
            status: "Draft",
            fromAddress: "",
            toAddress: "",
            ccAddresses: [],
            subject: "",
            body: "",
            draftedByNova: true,
            templateId: null,
            failureText: "",
            suggestedAddress: null,
            sentBy: null,
            sentAt: null,
            createdAt: 1,
            updatedAt: 1,
          },
        }),
        idle,
      ),
    ).toMatchObject({ primary: "reviewReply" });
    expect(
      actionBarMode(
        message({ actions: [{ ...reply, reply: { ...sentReply, status: "Failed" } }] }),
        idle,
      ),
    ).toMatchObject({ primary: "reviewReply" });
  });

  it("does not offer Nova for a step it has no way to do itself", () => {
    expect(
      actionBarMode(message({ actions: [{ ...file, novaCanDo: true }] }), idle),
    ).toMatchObject({ primary: "doIt" });
  });

  it("is busy while Nova drafts the step that is next, and only that step", () => {
    expect(actionBarMode(message({ actions: [load, reply] }), { load: true, reply: false })).toMatchObject({
      busy: true,
    });
    expect(actionBarMode(message({ actions: [load, reply] }), { load: false, reply: true })).toMatchObject({
      busy: false,
    });
  });

  it("is all done when every step is, with the teach rule and wait-for-reply after a sent reply", () => {
    const steps: TriageAction[] = [
      { ...file, done: true },
      { ...reply, done: true, reply: sentReply },
    ];
    expect(actionBarMode(message({ actions: steps, classification: "ProofOfDelivery" }), idle)).toEqual({
      kind: "done",
      total: 2,
      only: null,
      teach: "FileOnShipment",
      waitForReply: true,
    });
  });

  it("names the one step when only one was asked for, and says nothing to do when none were", () => {
    const only = { ...file, done: true };
    expect(actionBarMode(message({ actions: [only] }), idle)).toMatchObject({
      kind: "done",
      total: 1,
      only,
      waitForReply: false,
    });
    expect(actionBarMode(message({ actions: [] }), idle)).toMatchObject({
      kind: "done",
      total: 0,
      only: null,
      teach: null,
    });
  });
});

describe("teachFor", () => {
  it("teaches from the first step the message asked for", () => {
    expect(teachFor([file], "RateConfirmation", true)).toBe("FileOnShipment");
    expect(teachFor([file], "Invoice", true)).toBe("FileAndQueuePay");
    expect(teachFor([load, reply], "Tender", true)).toBe("DraftLoad");
    expect(teachFor([reply], "StatusRequest", true)).toBe("DraftReply");
    expect(teachFor([action({ kind: "CheckStops" })], "DetentionDispute", true)).toBe("DraftReply");
  });

  it("teaches nothing a rule cannot do, or from a message Nova never read", () => {
    expect(teachFor([action({ kind: "Link" })], "Tender", true)).toBeNull();
    expect(teachFor([action({ kind: "BlockSender" })], "Other", true)).toBeNull();
    expect(teachFor([action({ kind: "LogException" })], "ProofOfDelivery", true)).toBeNull();
    expect(teachFor([file], null, true)).toBeNull();
    expect(teachFor([file], "Invoice", false)).toBeNull();
    expect(teachFor([], "Invoice", true)).toBeNull();
  });
});

describe("whyRows", () => {
  function why(overrides: Partial<WhyMessage> = {}): WhyMessage {
    return {
      failureText: "",
      status: "InReview",
      confidence: 0.68,
      matchReason: "",
      novaRead: true,
      mailbox: { reviewPolicy: "ReviewBelowConfidence", minConfidence: 0.75, name: "Intake" },
      ...overrides,
    };
  }

  it("leads with why the reading failed, and then shows no meter", () => {
    expect(
      whyRows(why({ failureText: "The body was cut off.", matchReason: "" }), null),
    ).toEqual([{ kind: "failure", text: "The body was cut off.", held: false }]);
    expect(
      whyRows(why({ failureText: "Held back: a sign-in link.", status: "Quarantined" }), null)[0],
    ).toEqual({ kind: "failure", text: "Held back: a sign-in link.", held: true });
  });

  it("measures the reading against the mailbox's bar only when the mailbox has one", () => {
    expect(whyRows(why(), null)[0]).toEqual({
      kind: "confidence",
      confidence: 0.68,
      bar: 0.75,
      under: true,
    });
    expect(whyRows(why({ confidence: 0.9 }), null)[0]).toMatchObject({ under: false });
    expect(
      whyRows(why({ mailbox: { reviewPolicy: "AlwaysReview", minConfidence: 0.9, name: "Claims" } }), null),
    ).toEqual([{ kind: "note", note: "alwaysReview", mailbox: "Claims" }]);
    expect(whyRows(why({ mailbox: null }), null)).toEqual([]);
  });

  it("says why the next step waits for a person, and how the message was matched", () => {
    expect(whyRows(why({ matchReason: "PO 88213 is on S-104198." }), load)).toEqual([
      { kind: "confidence", confidence: 0.68, bar: 0.75, under: true },
      { kind: "note", note: "loadByHand", mailbox: "Intake" },
      { kind: "match", text: "PO 88213 is on S-104198." },
    ]);
    expect(whyRows(why({ confidence: 0.9 }), file)[1]).toEqual({
      kind: "note",
      note: "filingWaits",
      mailbox: "Intake",
    });
    expect(whyRows(why({ novaRead: false }), file)).toEqual([
      { kind: "note", note: "notRead", mailbox: "Intake" },
    ]);
  });
});
