import { action } from "@/lib/triage/__tests__/message-fixture";
import { describe, expect, it } from "vitest";
import {
  INITIAL_PANEL,
  panelReducer,
  stepStart,
  type PanelState,
  type StepContext,
} from "../panel-state";

const steps = (...rows: [string, boolean][]) => rows.map(([id, done]) => ({ id, done }));
const nothing: StepContext = { hasLoadDraft: false, hasReplyDraft: false, replyInFlight: false };

describe("panelReducer", () => {
  it("opens one step at a time and closes it on a second press", () => {
    const one = panelReducer(INITIAL_PANEL, { type: "toggle", id: "file:a" });
    expect(one.open).toBe("file:a");
    const other = panelReducer(one, { type: "toggle", id: "reply" });
    expect(other.open).toBe("reply");
    expect(panelReducer(other, { type: "toggle", id: "reply" }).open).toBeNull();
    expect(panelReducer(other, { type: "open", id: "reply" }).open).toBe("reply");
    expect(panelReducer(other, { type: "close" }).open).toBeNull();
  });

  it("closes the open step when it is done, but not when another step finishes", () => {
    let state = panelReducer(INITIAL_PANEL, {
      type: "stepsChanged",
      steps: steps(["file:a", false], ["reply", false]),
    });
    state = panelReducer(state, { type: "open", id: "reply" });
    const otherDone = panelReducer(state, {
      type: "stepsChanged",
      steps: steps(["file:a", true], ["reply", false]),
    });
    expect(otherDone.open).toBe("reply");
    const replyDone = panelReducer(otherDone, {
      type: "stepsChanged",
      steps: steps(["file:a", true], ["reply", true]),
    });
    expect(replyDone.open).toBeNull();
  });

  it("lets a person reopen a done step to read what came of it, and keeps it open", () => {
    let state = panelReducer(INITIAL_PANEL, {
      type: "stepsChanged",
      steps: steps(["reply", true]),
    });
    state = panelReducer(state, { type: "open", id: "reply" });
    const again = panelReducer(state, { type: "stepsChanged", steps: steps(["reply", true]) });
    expect(again.open).toBe("reply");
  });

  it("closes a step that is no longer asked for", () => {
    let state = panelReducer(INITIAL_PANEL, {
      type: "stepsChanged",
      steps: steps(["block", false]),
    });
    state = panelReducer(state, { type: "open", id: "block" });
    expect(panelReducer(state, { type: "stepsChanged", steps: [] }).open).toBeNull();
  });

  it("keeps the same state when nothing about the steps changed", () => {
    const state = panelReducer(INITIAL_PANEL, {
      type: "stepsChanged",
      steps: steps(["link", false]),
    });
    expect(panelReducer(state, { type: "stepsChanged", steps: steps(["link", false]) })).toBe(
      state,
    );
  });

  it("opens the step Nova is drafting and tracks each draft on its own", () => {
    const loading = panelReducer(INITIAL_PANEL, {
      type: "draftStarted",
      draft: "load",
      stepId: "load",
    });
    expect(loading).toMatchObject({ open: "load", drafting: { load: true, reply: false } });
    const both = panelReducer(loading, { type: "draftStarted", draft: "reply", stepId: "reply" });
    expect(both.drafting).toEqual({ load: true, reply: true });
    expect(panelReducer(both, { type: "draftSettled", draft: "load" }).drafting).toEqual({
      load: false,
      reply: true,
    });
  });

  it("opens and closes the phone sheet without touching the open step", () => {
    const open: PanelState = { ...INITIAL_PANEL, open: "reply" };
    const sheet = panelReducer(open, { type: "sheet", open: true });
    expect(sheet).toMatchObject({ sheet: true, open: "reply" });
    expect(panelReducer(sheet, { type: "sheet", open: false }).sheet).toBe(false);
  });
});

describe("stepStart", () => {
  it("opens the link dialog for a step about which record", () => {
    expect(stepStart(action({ kind: "Link" }), nothing)).toBe("link");
  });

  it("has Nova draft a load it can, opens the empty form when it cannot, and reviews a draft", () => {
    const load = action({ id: "load", kind: "BuildLoad", novaCanDo: true });
    expect(stepStart(load, nothing)).toBe("draftLoad");
    expect(stepStart({ ...load, novaCanDo: false }, nothing)).toBe("compose");
    expect(stepStart(load, { ...nothing, hasLoadDraft: true })).toBe("toggle");
    expect(stepStart({ ...load, done: true }, nothing)).toBe("toggle");
  });

  it("has Nova draft a reply it can, opens a blank composer when it cannot, and reviews a draft", () => {
    const reply = action({ id: "reply", kind: "Reply", novaCanDo: true });
    expect(stepStart(reply, nothing)).toBe("draftReply");
    expect(stepStart({ ...reply, novaCanDo: false }, nothing)).toBe("compose");
    expect(stepStart(reply, { ...nothing, hasReplyDraft: true })).toBe("toggle");
    expect(stepStart(reply, { ...nothing, replyInFlight: true })).toBe("toggle");
    expect(stepStart({ ...reply, done: true }, nothing)).toBe("toggle");
  });

  it("opens every other step in place", () => {
    for (const kind of ["File", "CheckStops", "LogException", "BlockSender"] as const) {
      expect(stepStart(action({ kind }), nothing)).toBe("toggle");
    }
  });
});
