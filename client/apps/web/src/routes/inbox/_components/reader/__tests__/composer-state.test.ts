import { describe, expect, it } from "vitest";
import {
  canSend,
  composerReducer,
  initialComposer,
  type ComposerState,
  type ReplyOutcome,
} from "../composer-state";

const failedReply: ReplyOutcome = {
  id: "rep_1",
  status: "Failed",
  failureText: "edi@partner.example doesn't accept replies — their server returned 550.",
  suggestedAddress: "ops@partner.example",
  toAddress: "edi@partner.example",
};
const sentReply: ReplyOutcome = {
  id: "rep_2",
  status: "Sent",
  failureText: "",
  suggestedAddress: null,
  toAddress: "ops@partner.example",
};
const sendingReply: ReplyOutcome = { ...sentReply, status: "Sending" };

const writing: ComposerState = { phase: "writing", error: null };

describe("initialComposer", () => {
  it("opens on a blank composer, or where the last reply left off", () => {
    expect(initialComposer(null)).toEqual(writing);
    expect(initialComposer(sentReply)).toEqual({ phase: "sent", replyId: "rep_2" });
    expect(initialComposer(sendingReply)).toEqual({
      phase: "sending",
      to: "ops@partner.example",
      retrying: null,
    });
    expect(initialComposer(failedReply)).toEqual({
      phase: "failed",
      failure: {
        failureText: failedReply.failureText,
        suggestedAddress: "ops@partner.example",
        attemptedTo: "edi@partner.example",
      },
      error: null,
    });
  });

  it("treats a stored draft as something still being written", () => {
    expect(initialComposer({ ...sentReply, status: "Draft" })).toEqual(writing);
  });
});

describe("composerReducer", () => {
  it("sends once: a second press while sending does nothing", () => {
    const sending = composerReducer(writing, { type: "send", to: null });
    expect(sending).toEqual({ phase: "sending", to: null, retrying: null });
    expect(composerReducer(sending, { type: "send", to: null })).toBe(sending);
  });

  it("lands on sent when the provider took it", () => {
    const sending = composerReducer(writing, { type: "send", to: null });
    expect(composerReducer(sending, { type: "sendSettled", reply: sentReply })).toEqual({
      phase: "sent",
      replyId: "rep_2",
    });
  });

  it("waits for the provider's answer when the send is still going out", () => {
    const sending = composerReducer(writing, { type: "send", to: null });
    const queued = composerReducer(sending, { type: "sendSettled", reply: sendingReply });
    expect(queued).toEqual({ phase: "sending", to: "ops@partner.example", retrying: null });
    expect(composerReducer(queued, { type: "synced", latest: sentReply })).toEqual({
      phase: "sent",
      replyId: "rep_2",
    });
  });

  it("shows a refusal with the address to try instead, and retries there", () => {
    const sending = composerReducer(writing, { type: "send", to: null });
    const failed = composerReducer(sending, { type: "sendSettled", reply: failedReply });
    expect(failed).toMatchObject({
      phase: "failed",
      failure: { suggestedAddress: "ops@partner.example", attemptedTo: "edi@partner.example" },
    });

    const retrying = composerReducer(failed, { type: "send", to: "ops@partner.example" });
    expect(retrying).toMatchObject({ phase: "sending", to: "ops@partner.example" });
    expect(composerReducer(retrying, { type: "sendSettled", reply: sentReply })).toEqual({
      phase: "sent",
      replyId: "rep_2",
    });
  });

  it("keeps the refusal on screen when the retry itself errors, with the error", () => {
    const failed = initialComposer(failedReply);
    const retrying = composerReducer(failed, { type: "send", to: "ops@partner.example" });
    expect(composerReducer(retrying, { type: "sendErrored", error: "Network down" })).toEqual({
      phase: "failed",
      failure: {
        failureText: failedReply.failureText,
        suggestedAddress: "ops@partner.example",
        attemptedTo: "edi@partner.example",
      },
      error: "Network down",
    });
  });

  it("goes back to writing with the error when a first send errors before the provider", () => {
    const sending = composerReducer(writing, { type: "send", to: null });
    expect(composerReducer(sending, { type: "sendErrored", error: "Body is required" })).toEqual({
      phase: "writing",
      error: "Body is required",
    });
  });

  it("clears an old error when a new send starts", () => {
    const errored: ComposerState = { phase: "writing", error: "Body is required" };
    expect(composerReducer(errored, { type: "send", to: null })).toEqual({
      phase: "sending",
      to: null,
      retrying: null,
    });
  });

  it("drafts with Nova and comes back to writing, with the error if Nova could not", () => {
    const drafting = composerReducer(writing, { type: "draftRequested" });
    expect(drafting).toEqual({ phase: "drafting" });
    expect(composerReducer(drafting, { type: "send", to: null })).toBe(drafting);
    expect(composerReducer(drafting, { type: "draftReady" })).toEqual(writing);
    expect(composerReducer(drafting, { type: "draftFailed", error: "Nova is off" })).toEqual({
      phase: "writing",
      error: "Nova is off",
    });
  });

  it("does not start a draft over a reply that is going out or gone", () => {
    const sending = composerReducer(writing, { type: "send", to: null });
    expect(composerReducer(sending, { type: "draftRequested" })).toBe(sending);
    const sent: ComposerState = { phase: "sent", replyId: "rep_2" };
    expect(composerReducer(sent, { type: "draftRequested" })).toBe(sent);
  });

  it("shows a reply someone else sent meanwhile as sent", () => {
    expect(composerReducer(writing, { type: "synced", latest: sentReply })).toEqual({
      phase: "sent",
      replyId: "rep_2",
    });
  });

  it("ignores a server update that says nothing new", () => {
    expect(composerReducer(writing, { type: "synced", latest: null })).toBe(writing);
    const failed = initialComposer(failedReply);
    expect(composerReducer(failed, { type: "synced", latest: failedReply })).toBe(failed);
  });
});

describe("canSend", () => {
  it("needs words, and a composer that is not busy or done", () => {
    expect(canSend(writing, "Hi")).toBe(true);
    expect(canSend(writing, "   \n")).toBe(false);
    expect(canSend({ phase: "drafting" }, "Hi")).toBe(false);
    expect(canSend({ phase: "sending", to: null, retrying: null }, "Hi")).toBe(false);
    expect(canSend({ phase: "sent", replyId: "r" }, "Hi")).toBe(false);
    expect(canSend(initialComposer(failedReply), "Hi")).toBe(true);
  });
});
