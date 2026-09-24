import { recordPath } from "@/config/record-links";
import type { Notification } from "@trenova/shared/types/notification";
import { describe, expect, it } from "vitest";
import {
  ACTION_COPY_ID,
  ACTION_COPY_LINK,
  ACTION_NEW_TAB,
  ACTION_OPEN,
  buildItemActions,
  CUSTOMER_LEDGER_PATH,
} from "../palette-actions";
import type { PaletteItem } from "../palette-model";
import { actionContext, command, page, record, t } from "./palette-test-fixtures";

function recordItem(overrides: Parameters<typeof record>[0] = {}): PaletteItem {
  const value = record(overrides);
  return { kind: "record", key: `record:${value.entityType}:${value.id}`, record: value };
}

function ids(item: PaletteItem, context = actionContext()): string[] {
  return buildItemActions(item, context, t).map((action) => action.id);
}

function notification(readAt: number | null): Notification {
  return {
    id: "ntf_1",
    organizationId: "org",
    businessUnitId: null,
    targetUserId: null,
    eventType: "x",
    priority: "medium",
    channel: "user",
    title: "Invoice ready",
    message: "",
    data: null,
    relatedEntities: null,
    source: "system",
    readAt,
    dismissedAt: null,
    createdAt: 1,
  };
}

describe("buildItemActions", () => {
  it("opens a shipment first and offers the keyed actions", () => {
    const actions = buildItemActions(recordItem({ metadata: { bol: "B-1" } }), actionContext(), t);

    expect(actions[0]).toMatchObject({ id: ACTION_OPEN, intent: { type: "navigate" } });
    expect(actions.map((action) => action.id)).toEqual([
      ACTION_OPEN,
      ACTION_NEW_TAB,
      ACTION_COPY_LINK,
      ACTION_COPY_ID,
      "copy-bol",
      "ask",
    ]);
  });

  it("copies the PRO number from metadata, falling back to the title", () => {
    const withPro = buildItemActions(
      recordItem({ title: "BOL-1", metadata: { proNumber: "PRO-9" } }),
      actionContext(),
      t,
    ).find((action) => action.id === ACTION_COPY_ID);
    const withoutPro = buildItemActions(recordItem({ title: "PRO-1" }), actionContext(), t).find(
      (action) => action.id === ACTION_COPY_ID,
    );

    expect(withPro?.intent).toMatchObject({ type: "copy", text: "PRO-9" });
    expect(withoutPro?.intent).toMatchObject({ type: "copy", text: "PRO-1" });
  });

  it("offers no BOL copy when the hit carries no BOL", () => {
    expect(ids(recordItem())).not.toContain("copy-bol");
  });

  it("leaves out asking the assistant for someone who cannot use it", () => {
    expect(ids(recordItem(), actionContext({ canUseAssistant: false }))).not.toContain("ask");
  });

  it("offers the customer ledger only to someone who can reach it", () => {
    const customer = recordItem({ entityType: "customer", id: "cus_1", title: "Acme" });

    const allowed = buildItemActions(
      customer,
      actionContext({ canReach: (path) => path === CUSTOMER_LEDGER_PATH }),
      t,
    ).find((action) => action.id === "customer-ledger");
    expect(allowed?.intent).toEqual({
      type: "navigate",
      href: `${CUSTOMER_LEDGER_PATH}?customerId=cus_1`,
    });

    expect(ids(customer, actionContext({ canReach: () => false }))).not.toContain(
      "customer-ledger",
    );
  });

  it("opens a worker's tabs through the record link registry", () => {
    const actions = buildItemActions(
      recordItem({ entityType: "worker", id: "wrk_1", title: "Dana Ruiz" }),
      actionContext(),
      t,
    );

    expect(actions.find((action) => action.id === "worker-pto")?.intent).toEqual({
      type: "navigate",
      href: recordPath("worker", "wrk_1", { tab: "pto" }),
    });
  });

  it("previews a document first rather than opening the record it is attached to", () => {
    const actions = buildItemActions(
      recordItem({ entityType: "document", id: "doc_1", title: "bol.pdf", href: "/parent" }),
      actionContext(),
      t,
    );

    expect(actions[0]?.intent).toEqual({
      type: "open-document",
      documentId: "doc_1",
      disposition: "view",
    });
    expect(actions.find((action) => action.id === "document-record")?.intent).toEqual({
      type: "navigate",
      href: "/parent",
    });
  });

  it("offers to unpin a page that is pinned and to pin one that is not", () => {
    const item: PaletteItem = { kind: "page", key: "page:/billing/invoices", page: page() };

    const pinAction = (pinned: boolean) =>
      buildItemActions(
        item,
        actionContext({ pinnedUrls: new Set(pinned ? ["/billing/invoices"] : []) }),
        t,
      ).find((action) => action.id === "toggle-pin")?.label;

    expect(pinAction(false)).toBe("Pin page");
    expect(pinAction(true)).toBe("Unpin page");
  });

  it("runs a command as its only action", () => {
    const item: PaletteItem = { kind: "command", key: "command:x", command: command() };

    expect(buildItemActions(item, actionContext(), t)).toEqual([
      expect.objectContaining({ id: ACTION_OPEN, intent: command().intent }),
    ]);
  });

  it("offers to mark a notification read only while it is unread", () => {
    const unread: PaletteItem = {
      kind: "notification",
      key: "n",
      notification: notification(null),
      href: "/billing/invoices",
    };
    const read: PaletteItem = { ...unread, notification: notification(100) };

    expect(ids(unread)).toContain("mark-read");
    expect(ids(read)).not.toContain("mark-read");
  });

  it("opens the notification center for a notification with nowhere to go", () => {
    const item: PaletteItem = {
      kind: "notification",
      key: "n",
      notification: notification(null),
      href: null,
    };

    expect(buildItemActions(item, actionContext(), t)[0]?.intent).toEqual({
      type: "open-dialog",
      dialog: "notifications",
    });
    expect(ids(item)).not.toContain(ACTION_NEW_TAB);
  });

  it("puts copying a number on Alt, clear of the browser's own Shift+C inspector", () => {
    const copyId = (mac: boolean) =>
      buildItemActions(recordItem(), actionContext({ mac }), t).find(
        (action) => action.id === ACTION_COPY_ID,
      )?.shortcut;

    expect(copyId(true)).toEqual(["⌥", "C"]);
    expect(copyId(false)).toEqual(["Alt", "C"]);
  });

  it("spells the modifier the way the platform does", () => {
    const newTab = (mac: boolean) =>
      buildItemActions(recordItem(), actionContext({ mac }), t).find(
        (action) => action.id === ACTION_NEW_TAB,
      )?.shortcut;

    expect(newTab(true)).toEqual(["⌘", "↵"]);
    expect(newTab(false)).toEqual(["Ctrl", "↵"]);
  });
});
