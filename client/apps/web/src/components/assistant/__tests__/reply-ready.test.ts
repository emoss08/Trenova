import { describe, expect, it } from "vitest";
import { ASSISTANT_REPLY_READY, isViewingThread, parseReplyReady } from "../reply-ready";

/*
Fixtures follow the notification contract: the data of a finished reply's
notice carries kind "assistant_reply_ready", threadId, turnId, a status of
Completed, Refused or Failed, and link — the path that opens the conversation.
*/
function notice(
  data: Record<string, unknown> | null,
  eventType: string | null = ASSISTANT_REPLY_READY,
) {
  return { eventType, data };
}

const complete = {
  kind: ASSISTANT_REPLY_READY,
  threadId: "athr_1",
  turnId: "atrn_1",
  status: "Completed",
  link: "/desk/t/athr_1",
};

describe("parseReplyReady", () => {
  it("reads a finished reply's notice", () => {
    expect(parseReplyReady(notice(complete))).toEqual({
      threadId: "athr_1",
      turnId: "atrn_1",
      status: "Completed",
      link: "/desk/t/athr_1",
    });
  });

  it("recognises the notice by the kind its data carries when the event type differs", () => {
    expect(parseReplyReady(notice(complete, "assistant.reply"))?.threadId).toBe("athr_1");
  });

  it("recognises the notice by its event type when the data leaves the kind out", () => {
    const { kind: _kind, ...rest } = complete;

    expect(parseReplyReady(notice(rest))?.threadId).toBe("athr_1");
  });

  it("ignores every other notification", () => {
    expect(parseReplyReady(notice({ threadId: "athr_1" }, "report_run_completed"))).toBeNull();
    expect(parseReplyReady(notice(null, "report_run_completed"))).toBeNull();
  });

  it("ignores a notice that does not say which conversation", () => {
    expect(parseReplyReady(notice({ ...complete, threadId: "" }))).toBeNull();
    expect(parseReplyReady(notice(null))).toBeNull();
  });

  it.each(["Refused", "Failed"] as const)("keeps a %s status", (status) => {
    expect(parseReplyReady(notice({ ...complete, status }))?.status).toBe(status);
  });

  it("reads a status it does not know as a finished reply rather than dropping the notice", () => {
    expect(parseReplyReady(notice({ ...complete, status: "Paused" }))?.status).toBe("Completed");
  });

  it("opens the conversation at the Desk when the notice carries no link", () => {
    const { link: _link, ...rest } = complete;

    expect(parseReplyReady(notice(rest))?.link).toBe("/desk/t/athr_1");
  });

  it.each(["https://example.com/desk/t/athr_1", "//example.com/x", "/\\example.com", "desk/t/x"])(
    "never follows a link that leaves the app (%s)",
    (link) => {
      expect(parseReplyReady(notice({ ...complete, link }))?.link).toBe("/desk/t/athr_1");
    },
  );

  it("tolerates a notice without a turn id", () => {
    const { turnId: _turnId, ...rest } = complete;

    expect(parseReplyReady(notice(rest))?.turnId).toBe("");
  });
});

describe("isViewingThread", () => {
  it("is true at the Desk on that conversation", () => {
    expect(
      isViewingThread("athr_1", {
        pathname: "/desk/t/athr_1",
        panelOpen: false,
        panelThreadId: null,
      }),
    ).toBe(true);
  });

  it("is false at the Desk on another conversation", () => {
    expect(
      isViewingThread("athr_1", {
        pathname: "/desk/t/athr_2",
        panelOpen: false,
        panelThreadId: null,
      }),
    ).toBe(false);
  });

  // The Desk has no corner panel, so a panel remembered as open on the thread
  // is not being looked at.
  it("ignores the panel's remembered state at the Desk", () => {
    expect(
      isViewingThread("athr_1", { pathname: "/desk", panelOpen: true, panelThreadId: "athr_1" }),
    ).toBe(false);
  });

  it("is true with the corner panel open on that conversation", () => {
    expect(
      isViewingThread("athr_1", {
        pathname: "/shipment-management/shipments",
        panelOpen: true,
        panelThreadId: "athr_1",
      }),
    ).toBe(true);
  });

  it("is false with the panel closed, even when it last showed that conversation", () => {
    expect(
      isViewingThread("athr_1", {
        pathname: "/shipment-management/shipments",
        panelOpen: false,
        panelThreadId: "athr_1",
      }),
    ).toBe(false);
  });

  it("is false with the panel open on another conversation", () => {
    expect(
      isViewingThread("athr_1", { pathname: "/", panelOpen: true, panelThreadId: "athr_2" }),
    ).toBe(false);
  });
});
